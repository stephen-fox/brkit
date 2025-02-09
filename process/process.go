package process

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
)

// TODO: Timeouts / deadlines.

// ExecOrExit starts the specified *exec.Cmd, subsequently calling
// DefaultExitFn if an error occurs.
//
// Refer to Exec for more information.
func ExecOrExit(cmd *exec.Cmd, info Info) *Process {
	p, err := Exec(cmd, info)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to start process - %w", err))
	}
	return p
}

// Exec starts the specified *exec.Cmd, returning a *Process which represents
// the underlying running process.
//
// Callers are expected to call Process.Close when the Process has exited,
// or is no longer needed.
func Exec(cmd *exec.Cmd, info Info) (*Process, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe - %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe - %w", err)
	}

	// TODO: stderr.

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start process - %w", err)
	}

	proc := &Process{
		input:  stdin,
		output: bufio.NewReader(stdout),
		rwMu:   &sync.RWMutex{},
		info:   info,
	}

	waitDone := make(chan struct{})
	proc.close = func() error {
		proc.rwMu.RLock()
		exitedCopy := proc.exited
		proc.rwMu.RUnlock()
		if !exitedCopy.exited {
			cmd.Process.Kill()
		}
		<-waitDone
		return exitedCopy.err
	}

	go func() {
		err := cmd.Wait()
		proc.rwMu.Lock()
		proc.exited = exitInfo{
			exited: true,
			err:    err,
		}
		close(waitDone)
		proc.rwMu.Unlock()
	}()

	return proc, nil
}

type exitInfo struct {
	exited bool
	err    error
}

// DialOrExit attempts to connect to a remote process using the specified
// network type and address, subsequently calling DefaultExitFn if an
// error occurs.
//
// Refer to Dial for more information.
func DialOrExit(network string, address string, info Info) *Process {
	p, err := Dial(network, address, info)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to dial program - %w", err))
	}
	return p
}

// Dial attempts to connect to a remote process using the specified
// network type and address, returning a *Process which represents
// the remote process. The network type string is the same set
// of strings used for net.Dial.
//
// Callers should call Process.Close when the process has exited,
// or a connection to the process is no longer required.
func Dial(network string, address string, info Info) (*Process, error) {
	c, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return FromNetConn(c, info), nil
}

// FromNetConn upgrade an existing network connection to a process
// (a net.Conn), returning a *Process.
func FromNetConn(c net.Conn, info Info) *Process {
	return &Process{
		input:  c,
		output: bufio.NewReader(c),
		rwMu:   &sync.RWMutex{},
		info:   info,
		close: func() error {
			return c.Close()
		},
	}
}

// FromNamedPipesOrExit attempts to connect to a process through a named pipe
// using an input pipe path and output pipe path. It calls DefaultExitFn if an
// error occurs.
//
// Refer to FromNamedPipes for more information.
func FromNamedPipesOrExit(inputPipePath string, outputPipePath string, info Info) *Process {
	p, err := FromNamedPipes(inputPipePath, outputPipePath, info)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to create process from named pipes - %w", err))
	}

	return p
}

// FromNamedPipesOrExit attempts to connect to a process through a named pipe
// using an input pipe path and output pipe path, returning a *Process.
func FromNamedPipes(inputPipePath string, outputPipePath string, info Info) (*Process, error) {
	input, err := os.OpenFile(inputPipePath, os.O_WRONLY|syscall.O_NONBLOCK, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("failed to open input pipe - %w", err)
	}

	output, err := os.OpenFile(outputPipePath, os.O_RDONLY|syscall.O_NONBLOCK, os.ModeNamedPipe)
	if err != nil {
		_ = input.Close()
		return nil, fmt.Errorf("failed to open output pipe - %w", err)
	}

	return FromIO(input, output, info), nil
}

// FromIO attempts to connect to a process by using the specified input
// and output, returning a *Process. For example, input and output can
// be two different named pipes accessed over ssh connections (refer to
// Go Doc for example).
func FromIO(input io.WriteCloser, output io.ReadCloser, info Info) *Process {
	// TODO investigate using this function in other FromFunctions
	return &Process{
		input:  input,
		output: bufio.NewReader(output),
		rwMu:   &sync.RWMutex{},
		info:   info,
		close: func() error {
			_ = input.Close()
			_ = output.Close()
			return nil
		},
	}
}

// X86_32Info creates a new Info for a X86 32-bit process.
func X86_32Info() Info {
	return Info{
		PlatformBits: 32,
		PtrSizeBytes: 4,
	}
}

// X86_64Info creates a new Info for a X86 64-bit process.
func X86_64Info() Info {
	return Info{
		PlatformBits: 64,
		PtrSizeBytes: 8,
	}
}

// Info specifies platform information about the process.
type Info struct {
	// PlatformBits is the number of CPU bits (e.g., 32).
	PlatformBits int

	// PtrSizeBytes is the size of a pointer in bytes on
	// the target system.
	PtrSizeBytes int
}

// Process represents a running software process. The process can be
// running on the same computer as this code, or on a networked neighbor.
// The objective of this struct is to abstract inter-process communications
// into a simple API.
//
// Depending on the circumstances, callers should generally call
// Process.Close after they are finished with the process.
// Refer to the method's documentation for more information.
type Process struct {
	input   io.Writer
	output  *bufio.Reader
	close   func() error
	rwMu    *sync.RWMutex
	exited  exitInfo
	info    Info
	loggerR *log.Logger
	loggerW *log.Logger
}

// Close releases any resources associated with the underlying software process
// and kills the process if it has not already exited. For a remote process,
// the underlying connection will be closed.
//
// The Process is no longer usable once this method is invoked.
func (o *Process) Close() error {
	return o.close()
}

// Bits returns the number of bits for the process' platform.
func (o *Process) Bits() int {
	return o.info.PlatformBits
}

// PointerSizeBytes returns the size of a pointer for the
// process' platform in bytes.
func (o *Process) PointerSizeBytes() int {
	return o.info.PtrSizeBytes
}

// HasExited returns true if the underlying process has exited.
//
// Note that this method is only reliable for a process invoked by
// one of the Exec functions. Determining the status of a networked
// process involves writing and reading data to the underlying
// network socket, which is dependent on the implementation of the
// remote process.
func (o *Process) HasExited() bool {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()
	return o.exited.exited
}

// ReadOrExit calls Process.Read and calls DefaultExitFn if an error occurs.
func (o *Process) ReadOrExit(b []byte) int {
	n, err := o.Read(b)
	if err != nil {
		DefaultExitFn(err)
	}
	return n
}

// Read reads from the processes output, implementing the io.Reader interface.
func (o *Process) Read(b []byte) (int, error) {
	n, err := o.output.Read(b)

	if o.loggerR != nil {
		var hexDump string
		if n > 0 {
			hexDump = hex.Dump(b[0:n])
		}
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.loggerR.Println("process: Read:\n" + hexDump)
	}

	return n, err
}

// ReadFromOrExit calls ReadFrom. It calls DefaultExitFn if an error occurs.
func (o *Process) ReadFromOrExit(r io.Reader) int64 {
	n, err := o.ReadFrom(r)
	if err != nil {
		DefaultExitFn(fmt.Errorf("process: failed to read from - %w", err))
	}

	return n
}

// ReadFrom reads data from r into the process' input until EOF. The return
// value n is the number of bytes read. Any error except io.EOF encountered
// during the read is also returned.
func (o *Process) ReadFrom(r io.Reader) (int64, error) {
	var hexDumpOutput *bytes.Buffer
	var hexDumper io.WriteCloser

	if o.loggerR != nil {
		hexDumpOutput = bytes.NewBuffer(nil)
		hexDumper = hex.Dumper(hexDumpOutput)

		r = io.TeeReader(r, hexDumper)
	}

	n, err := io.Copy(o.input, r)

	if o.loggerR != nil {
		// Flush remaining bytes to the hex dump buffer.
		_ = hexDumper.Close()

		hexDump := hexDumpOutput.String()
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.loggerR.Println("process: ReadFrom:\n" + hexDump)
	}

	if errors.Is(err, io.EOF) {
		return n, nil
	}
	return n, err
}

// ReadLineOrExit calls Process.ReadLine, subsequently calling DefaultExitFn
// if an error occurs.
func (o *Process) ReadLineOrExit() []byte {
	p, err := o.ReadLine()
	if err != nil {
		DefaultExitFn(err)
	}
	return p
}

// ReadLine blocks and attempts to read from the process' output until
// a new line character is found.
func (o *Process) ReadLine() ([]byte, error) {
	return o.ReadUntilChar('\n')
}

// ReadUntilCharOrExit calls Process.ReadUntilChar, subsequently calling
// DefaultExitFn if an error occurs.
func (o *Process) ReadUntilCharOrExit(delim byte) []byte {
	p, err := o.ReadUntilChar(delim)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to read from process until 0x%x - %w", delim, err))
	}
	return p
}

// ReadUntilChar blocks and attempts to read from the process' output until
// the specified character is found.
func (o *Process) ReadUntilChar(delim byte) ([]byte, error) {
	p, err := o.output.ReadBytes(delim)
	if err != nil {
		return nil, err
	}
	if o.loggerR != nil {
		var hexDump string
		if len(p) > 0 {
			hexDump = hex.Dump(p)
		}
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.loggerR.Println("process: ReadUntilChar:\n" + hexDump)
	}
	return p, nil
}

// ReadByteOrExit calls Process.ReadByte, subsequently calling DefaultExitFn
// if an error occurs.
func (o *Process) ReadByteOrExit() byte {
	b, err := o.ReadByte()
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to read one byte from process - %w", err))
	}
	return b
}

// ReadUntilOrExit calls Process.ReadUntil, subsequently calling DefaultExitFn
// if an error occurs.
func (o *Process) ReadUntilOrExit(p []byte) []byte {
	res, err := o.ReadUntil(p)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to read from process until 0x%x - %w", p, err))
	}
	return res
}

// ReadUntil blocks and attempts to read from the process' output until the
// specified []byte is found, returning the data read, including the
// specified []byte.
func (o *Process) ReadUntil(p []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	for {
		bSlice := make([]byte, 1)

		_, err := o.output.Read(bSlice)
		if err != nil {
			return nil, err
		}

		b := bSlice[0]

		err = buf.WriteByte(b)
		if err != nil {
			return nil, err
		}

		// TODO: Maybe search by suffix?
		if bytes.Contains(buf.Bytes(), p) {
			if o.loggerR != nil {
				var hexDump string
				if len(p) > 0 {
					hexDump = hex.Dump(buf.Bytes())
				}
				if len(hexDump) <= 1 {
					// hex.Dump always adds a newline.
					hexDump = "<empty-value>"
				} else {
					hexDump = hexDump[0 : len(hexDump)-1]
				}

				o.loggerR.Println("process: ReadUntil:\n" + hexDump)
			}
			return buf.Bytes(), nil
		}
	}
}

// ReadByte blocks and attempts to read one byte from the process' output.
func (o *Process) ReadByte() (byte, error) {
	b := make([]byte, 1)

	_, err := o.Read(b)
	if err != nil {
		return 0, err
	}

	return b[0], nil
}

// WriteLineOrExit calls Process.WriteLine, subsequently calling DefaultExitFn
// if an error occurs.
func (o *Process) WriteLineOrExit(p []byte) {
	err := o.WriteLine(p)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to write line to process - %w", err))
	}
}

// WriteLine appends a new line character to the specified []byte
// and writes it to the process' input.
func (o *Process) WriteLine(p []byte) error {
	_, err := o.Write(append(p, '\n'))
	return err
}

// WriteOrExit calls Process.Write, subsequently calling DefaultExitFn
// if an error occurs.
func (o *Process) WriteOrExit(p []byte) {
	_, err := o.input.Write(p)
	if err != nil {
		DefaultExitFn(err)
	}
}

// Write blocks and attempts to write the specified []byte to the
// process' input.
func (o *Process) Write(p []byte) (int, error) {
	if o.loggerW != nil {
		var hexDump string
		if len(p) > 0 {
			hexDump = hex.Dump(p)
		}
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.loggerW.Println("process: Write:\n" + hexDump)
	}

	n, err := o.input.Write(p)
	if err != nil {
		return n, fmt.Errorf("failed to write 0x%x - %w", p, err)
	}

	return n, nil
}

// SetLoggerR sets the *log.Logger for debugging read operations.
// Data from read operations will be formatted in hexdump format.
func (o *Process) SetLoggerR(logger *log.Logger) {
	o.loggerR = logger
}

// SetLoggerW sets the *log.Logger for debugging write operations.
// Data from write operations will be formatted in hexdump format.
func (o *Process) SetLoggerW(logger *log.Logger) {
	o.loggerW = logger
}

// InteractiveOrExit calls Process.Interactive, subsequently calling
// DefaultExitFn if an error occurs.
func (o *Process) InteractiveOrExit() {
	err := o.Interactive()
	if err != nil {
		DefaultExitFn(fmt.Errorf("process interaction failed - %w", err))
	}
}

// Interactive blocks and attempts to hookup the process' input to the stdin
// file descriptor, and its output to the stdout file descriptor.
//
// This is useful for directly interacting with the process in a shell.
func (o *Process) Interactive() error {
	done := make(chan error, 2)

	go func() {
		_, err := io.Copy(os.Stdout, o.output)
		if err != nil {
			done <- fmt.Errorf("failed to copy output reader to stdout - %w", err)
		} else {
			done <- nil
		}
	}()

	go func() {
		var stdin io.Reader = os.Stdin
		if runtime.GOOS == "windows" && os.Getenv("BRKIT_WINDOWS_INTERACTIVE") != "false" {
			// Super hack for Windows sending CRLF.
			stdin = &windowsNewlineSkipper{
				r: os.Stdin,
			}
		}

		_, err := io.Copy(o.input, stdin)
		if err != nil {
			done <- fmt.Errorf("failed to copy stdin to input writer - %w", err)
		} else {
			done <- nil
		}
	}()

	return <-done
}

type windowsNewlineSkipper struct {
	r io.Reader
}

func (o *windowsNewlineSkipper) Read(b []byte) (int, error) {
	n, err := o.r.Read(b)
	if n > 0 {
		index := bytes.Index(b[0:n], []byte{'\r', '\n'})
		if index > -1 {
			// a b c d e f \r \n A B C
			// 0 1 2 3 4 5 6  7  8 9 10
			//
			// copy(b[6:], b[7:])
			// a b c d e f \n A B C
			// 0 1 2 3 4 5 6  7 8 9 10
			copy(b[index:], b[index+1:])
			n--
		}
	}

	return n, err
}
