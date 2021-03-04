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

// TODO: Process cleanup func is not accessible.
//
// TODO: Return cleanup func separately to make it clear
//  end user needs to call it?

func StartOrExit(cmd *exec.Cmd) *Process {
	p, err := Start(cmd)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to start process - %w", err))
	}
	return p
}

func Start(cmd *exec.Cmd) (*Process, error) {
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

func DialOrExit(network string, address string) *Process {
	p, err := Dial(network, address)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to dial program - %w", err))
	}
	return p
}

func Dial(network string, address string) (*Process, error) {
	c, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return FromNetConn(c), nil
}

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

type Process struct {
	input  io.Writer
	output *bufio.Reader
	done   func() error
	rwMu   *sync.RWMutex
	exited exitInfo
	logger *log.Logger
}

func (o Process) HasExited() bool {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()
	return o.exited.exited
}

func (o Process) ReadLineOrExit() []byte {
	p, err := o.ReadLine()
	if err != nil {
		defaultExitFn(err)
	}
	return p
}

func (o Process) ReadLine() ([]byte, error) {
	return o.ReadUntilChar('\n')
}

func (o Process) ReadUntilCharOrExit(delim byte) []byte {
	p, err := o.ReadUntilChar(delim)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to read from process until 0x%x - %w", delim, err))
	}
	return p
}

func (o Process) ReadByteOrExit() byte {
	b, err := o.ReadByte()
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to read one byte from process - %w", err))
	}
	return b
}

func (o Process) ReadUntilOrExit(p []byte) []byte {
	res, err := o.ReadUntil(p)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to read from process until 0x%x - %w", p, err))
	}
	return res
}

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

func (o Process) ReadByte() (byte, error) {
	return o.output.ReadByte()
}

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

func (o Process) WriteLineOrExit(p []byte) {
	err := o.WriteLine(p)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to write line to process - %w", err))
	}
}

func (o Process) WriteLine(p []byte) error {
	if o.logger != nil {
		o.logger.Printf("writing line 0x%x", p)
	}
	_, err := o.input.Write(append(p, '\n'))
	return err
}

func (o *Process) SetLogger(logger *log.Logger) {
	o.logger = logger
}

func (o *Process) InteractiveOrExit() {
	err := o.Interactive()
	if err != nil {
		defaultExitFn(fmt.Errorf("process interaction failed - %w", err))
	}
}

func (o *Process) Interactive() error {
	done := make(chan error, 2)

	go func() {
		_, err := io.Copy(os.Stdout, o.output)
		done <- fmt.Errorf("failed to copy output reader to stdout - %w", err)
	}()

	go func() {
		_, err := io.Copy(o.input, os.Stdin)
		done <- fmt.Errorf("failed to copy stdin to input writer - %w", err)
	}()

	return <-done
}
