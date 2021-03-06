package process

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
)

// TODO: Timeouts / deadlines.

// ExecOrExit starts the specified *exec.Cmd. Refer to Exec for
// more information.
//
// Any underlying errors result in a call to DefaultExitFn.
func ExecOrExit(cmd *exec.Cmd) *Process {
	p, err := Exec(cmd)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to start process - %w", err))
	}
	return p
}

// Exec starts the specified *exec.Cmd, returning a *Process which represents
// the underlying running process.
//
// Callers are expected to call Process.Cleanup when the Process has exited,
// or is no longer needed.
func Exec(cmd *exec.Cmd) (*Process, error) {
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
	}

	waitDone := make(chan struct{})
	proc.done = func() error {
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
// network type and address. Refer to Dial for more information.
//
// Any underlying errors result in a call to DefaultExitFn.
func DialOrExit(network string, address string) *Process {
	p, err := Dial(network, address)
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
// Callers should call Process.Cleanup when the process has exited,
// or a connection to the process is no longer required.
func Dial(network string, address string) (*Process, error) {
	c, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return FromNetConn(c), nil
}

// FromNetConn upgrade an existing network connection to a process
// (a net.Conn), returning a *Process.
func FromNetConn(c net.Conn) *Process {
	return &Process{
		input:  c,
		output: bufio.NewReader(c),
		rwMu:   &sync.RWMutex{},
		done: func() error {
			return c.Close()
		},
	}
}

// Process represents a running software process. The process can be
// running on the same computer as this code, or on a networked neighbor.
// The objective of this struct is to abstract inter-process communications
// into a simple API.
//
// Depending on the circumstances, callers should generally call
// Process.Cleanup after they are finished with the process.
// Refer to the method's documentation for more information.
type Process struct {
	input  io.Writer
	output *bufio.Reader
	done   func() error
	rwMu   *sync.RWMutex
	exited exitInfo
	logger *log.Logger
}

// Cleanup, generally speaking, releases any resources associated with
// the underlying software process and kills the process if it has not
// already exited. For a remote process, the underlying connection
// is closed.
//
// The Process is no longer usable once this method is invoked,
func (o Process) Cleanup() error {
	return o.done()
}

// HasExited returns true if the underlying process has exited.
//
// Note that this method is only reliable for a process invoked by
// one of the Exec functions. Determining the status of a networked
// process involves writing and reading data to the underlying
// network socket, which is dependent on the implementation of the
// remote process.
func (o Process) HasExited() bool {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()
	return o.exited.exited
}

// ReadLineOrExit calls Process.ReadLine, subsequently calling DefaultExitFn
// if an error occurs.
func (o Process) ReadLineOrExit() []byte {
	p, err := o.ReadLine()
	if err != nil {
		DefaultExitFn(err)
	}
	return p
}

// ReadLine blocks and attempts to read from the process' output until
// a new line character is found.
func (o Process) ReadLine() ([]byte, error) {
	return o.ReadUntilChar('\n')
}

// ReadUntilCharOrExit calls Process.ReadUntilChar, subsequently calling
// DefaultExitFn if an error occurs.
func (o Process) ReadUntilCharOrExit(delim byte) []byte {
	p, err := o.ReadUntilChar(delim)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to read from process until 0x%x - %w", delim, err))
	}
	return p
}

// ReadUntilChar blocks and attempts to read from the process' output until
// the specified character is found.
func (o Process) ReadUntilChar(delim byte) ([]byte, error) {
	p, err := o.output.ReadBytes(delim)
	if err != nil {
		return nil, err
	}
	if o.logger != nil {
		o.logger.Printf("ReadUntilChar read: '%s'", p)
	}
	return p, nil
}

// ReadByteOrExit calls Process.ReadByte, subsequently calling DefaultExitFn
// if an error occurs.
func (o Process) ReadByteOrExit() byte {
	b, err := o.ReadByte()
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to read one byte from process - %w", err))
	}
	return b
}

// ReadUntilOrExit calls Process.ReadUntil, subsequently calling DefaultExitFn
// if an error occurs.
func (o Process) ReadUntilOrExit(p []byte) []byte {
	res, err := o.ReadUntil(p)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to read from process until 0x%x - %w", p, err))
	}
	return res
}

// ReadUntil blocks and attempts to read from the process' output until the
// specified []byte is found.
func (o Process) ReadUntil(p []byte) ([]byte, error) {
	if o.logger != nil {
		o.logger.Printf("ReadUntil: 0x%x", p)
	}

	buff := bytes.NewBuffer(nil)
	for {
		b, err := o.ReadByte()
		if err != nil {
			return nil, err
		}

		buff.WriteByte(b)
		if o.logger != nil {
			o.logger.Printf("ReadUntil buff is now: %s", buff.Bytes())
		}
		if bytes.Contains(buff.Bytes(), p) {
			if o.logger != nil {
				o.logger.Printf("ReadUntil buff contains target")
			}
			return buff.Bytes(), nil
		}
		if o.logger != nil {
			o.logger.Printf("ReadUntil buff does not contain target")
		}
	}
}

// ReadByte blocks and attempts to read one byte from the process' output.
func (o Process) ReadByte() (byte, error) {
	return o.output.ReadByte()
}

// WriteLineOrExit calls Process.WriteLine, subsequently calling DefaultExitFn
// if an error occurs.
func (o Process) WriteLineOrExit(p []byte) {
	err := o.WriteLine(p)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to write line to process - %w", err))
	}
}

// WriteLine blocks and attempts to write the specified []byte
// with an appended new line character to the process' input.
func (o Process) WriteLine(p []byte) error {
	_, err := o.input.Write(append(p, '\n'))
	return err
}

// WriteOrExit calls Process.Write, subsequently calling DefaultExitFn
// if an error occurs.
func (o Process) WriteOrExit(p []byte) {
	_, err := o.input.Write(p)
	if err != nil {
		DefaultExitFn(err)
	}
}

// Write blocks and attempts to write the specified []byte to the
// process' input.
func (o Process) Write(p []byte) error {
	if o.logger != nil {
		o.logger.Printf("writing line 0x%x", p)
	}
	_, err := o.input.Write(p)
	return fmt.Errorf("failed to write 0x%x - %w", p, err)
}

// SetLogger sets the *log.Logger for debugging purposes.
func (o *Process) SetLogger(logger *log.Logger) {
	o.logger = logger
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
		_, err := io.Copy(o.input, os.Stdin)
		if err != nil {
			done <- fmt.Errorf("failed to copy stdin to input writer - %w", err)
		} else {
			done <- nil
		}
	}()

	return <-done
}
