package iokit

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"unicode/utf8"
)

// Buffer wraps the bytes.Buffer type, adding additional features such as
// logging and helper methods.
type Buffer struct {
	// Buf is the internal bytes.Buffer. It is automatically
	// instantiated by the struct's methods if it is nil.
	Buf *bytes.Buffer

	// OptLoggerR is an optional logger that, when non-nil,
	// will recieve hexdump-style output when read-type
	// methods are called.
	OptLoggerR *log.Logger

	// OptLoggerW is an optional logger that, when non-nil,
	// will recieve hexdump-style output when write-type
	// methods are called.
	OptLoggerW *log.Logger
}

// Bytes calls Buf.Bytes.
func (o *Buffer) Bytes() []byte {
	if o.Buf == nil {
		return nil
	}

	return o.Buf.Bytes()
}

// Cap calls Buf.Cap.
func (o *Buffer) Cap() int {
	if o.Buf == nil {
		return 0
	}

	return o.Buf.Cap()
}

// Grow calls Buf.Grow.
func (o *Buffer) Grow(n int) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	o.Buf.Grow(n)
}

// Next calls Buf.Next.
func (o *Buffer) Next(n int) []byte {
	if o.Buf == nil {
		return nil
	}

	return o.Buf.Next(n)
}

// Reset calls Buf.Reset.
func (o *Buffer) Reset() {
	if o.Buf != nil {
		o.Buf.Reset()
	}
}

// Truncate calls Buf.Truncate.
func (o *Buffer) Truncate(n int) {
	if o.Buf != nil {
		o.Buf.Truncate(n)
	}
}

// Len calls Buf.Len.
func (o *Buffer) Len() int {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	return o.Buf.Len()
}

// UnreadByteOrExit calls UnreadByte. It calls DefaultExitFn if an error occurs.
func (o *Buffer) UnreadByteOrExit() {
	err := o.UnreadByte()
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to unread byte - %w", err))
	}
}

// UnreadByte calls Buf.UnreadByte.
func (o *Buffer) UnreadByte() error {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	err := o.Buf.UnreadByte()
	if err != nil {
		return err
	}

	return nil
}

// UnreadRuneOrExit calls UnreadRune. It calls DefaultExitFn if an error occurs.
func (o *Buffer) UnreadRuneOrExit() {
	err := o.UnreadRune()
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to unread rune - %w", err))
	}
}

// UnreadRune calls Buf.UnreadRune.
func (o *Buffer) UnreadRune() error {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	err := o.Buf.UnreadRune()
	if err != nil {
		return err
	}

	return nil
}

// ReadOrExit calls Read. It calls DefaultExitFn if an error occurs.
func (o *Buffer) ReadOrExit(b []byte) int {
	n, err := o.Read(b)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to read - %w", err))
	}

	return n
}

// Read calls Buf.Read.
func (o *Buffer) Read(b []byte) (int, error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	n, err := o.Buf.Read(b)

	if o.OptLoggerR != nil {
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

		o.OptLoggerR.Println("iokit.buffer: read:\n" + hexDump)
	}

	return n, err
}

// ReadByteOrExit calls ReadByte. It calls DefaultExitFn if an error occurs.
func (o *Buffer) ReadByteOrExit() byte {
	b, err := o.ReadByte()
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to read byte - %w", err))
	}

	return b
}

// ReadByte calls Buf.ReadByte.
func (o *Buffer) ReadByte() (byte, error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	b, err := o.Buf.ReadByte()
	if err != nil {
		return b, err
	}

	if o.OptLoggerR != nil {
		hexDump := hex.Dump([]byte{b})

		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerR.Println("iokit.buffer: read:\n" + hexDump)
	}

	return b, nil
}

// ReadBytesOrExit calls ReadBytes. It calls DefaultExitFn if an error occurs.
func (o *Buffer) ReadBytesOrExit(delim byte) []byte {
	line, err := o.ReadBytes(delim)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to read bytes - %w", err))
	}

	return line
}

// ReadBytes calls Buf.ReadBytes.
func (o *Buffer) ReadBytes(delim byte) (line []byte, err error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	line, err = o.Buf.ReadBytes(delim)

	if o.OptLoggerR != nil {
		var hexDump string
		if len(line) > 0 {
			hexDump = hex.Dump(line)
		}

		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerR.Println("iokit.buffer: read bytes:\n" + hexDump)
	}

	return line, err
}

// ReadFromOrExit calls ReadFrom. It calls DefaultExitFn if an error occurs.
func (o *Buffer) ReadFromOrExit(r io.Reader) int64 {
	n, err := o.ReadFrom(r)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to read from - %w", err))
	}

	return n
}

// ReadFrom calls Buf.ReadFrom.
func (o *Buffer) ReadFrom(r io.Reader) (int64, error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	var hexDumpOutput *bytes.Buffer
	var hexDumper io.WriteCloser

	if o.OptLoggerR != nil {
		hexDumpOutput = bytes.NewBuffer(nil)
		hexDumper = hex.Dumper(hexDumpOutput)

		r = io.TeeReader(r, hexDumper)
	}

	n, err := o.Buf.ReadFrom(r)

	if o.OptLoggerR != nil {
		// Flush remaining bytes to the hex dump buffer.
		_ = hexDumper.Close()

		hexDump := hexDumpOutput.String()
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerR.Println("iokit.buffer: read from:\n" + hexDump)
	}

	return n, err
}

// ReadRuneOrExit calls ReadRune. It calls DefaultExitFn if an error occurs.
func (o *Buffer) ReadRuneOrExit() (r rune, size int) {
	var err error
	r, size, err = o.ReadRune()
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to read rune - %w", err))
	}

	return r, size
}

// ReadRune calls Buf.ReadRune.
func (o *Buffer) ReadRune() (r rune, size int, err error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	r, size, err = o.Buf.ReadRune()
	if err != nil {
		return r, size, err
	}

	if o.OptLoggerR != nil {
		b := make([]byte, size)

		utf8.EncodeRune(b, r)

		hexDump := hex.Dump(b)
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerR.Println("iokit.buffer: read rune:\n" + hexDump)
	}

	return r, size, err
}

// ReadStringOrExit calls ReadString. It calls DefaultExitFn if an error occurs.
func (o *Buffer) ReadStringOrExit(delim byte) string {
	line, err := o.ReadString(delim)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to read string - %w", err))
	}

	return line
}

// ReadString calls Buf.ReadString.
func (o *Buffer) ReadString(delim byte) (line string, err error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	line, err = o.Buf.ReadString(delim)

	if o.OptLoggerR != nil {
		hexDump := hex.Dump([]byte(line))
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerR.Println("iokit.buffer: read string:\n" + hexDump)
	}

	return line, err
}

// WriteOrExit calls Write. It calls DefaultExitFn if an error occurs.
func (o *Buffer) WriteOrExit(b []byte) int {
	n, err := o.Write(b)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to write - %w", err))
	}

	return n
}

// Write calls Buf.Write.
func (o *Buffer) Write(b []byte) (int, error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	if o.OptLoggerW != nil {
		hexDump := hex.Dump(b)
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerW.Printf("iokit.buffer: write:\n%s", hexDump)
	}

	return o.Buf.Write(b)
}

// WriteStringOrExit calls WriteString. It calls DefaultExitFn if an error occurs.
func (o *Buffer) WriteStringOrExit(s string) int {
	n, err := o.WriteString(s)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to write string - %w", err))
	}

	return n
}

// WriteString calls Buf.WriteString.
func (o *Buffer) WriteString(s string) (int, error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	if o.OptLoggerW != nil {
		hexDump := hex.Dump([]byte(s))
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerW.Printf("iokit.buffer: write value:\n%s", hexDump)
	}

	return o.Buf.WriteString(s)
}

// WriteByteOrExit calls WriteByte. It calls DefaultExitFn if an error occurs.
func (o *Buffer) WriteByteOrExit(b byte) {
	err := o.WriteByte(b)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to write byte - %w", err))
	}
}

// WriteByte calls Buf.WriteByte.
func (o *Buffer) WriteByte(b byte) error {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	if o.OptLoggerW != nil {
		hexDump := hex.Dump([]byte{b})
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerW.Printf("iokit.buffer: write byte:\n%s", hexDump)
	}

	return o.Buf.WriteByte(b)
}

// WriteRuneOrExit calls WriteRune. It calls DefaultExitFn if an error occurs.
func (o *Buffer) WriteRuneOrExit(r rune) int {
	n, err := o.WriteRune(r)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to write rune - %w", err))
	}

	return n
}

// WriteRune calls Buf.WriteRune.
func (o *Buffer) WriteRune(r rune) (int, error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	if o.OptLoggerW != nil {
		b := make([]byte, utf8.RuneLen(r))

		utf8.EncodeRune(b, r)

		hexDump := hex.Dump(b)
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerW.Printf("iokit.buffer: write rune:\n%s", hexDump)
	}

	return o.Buf.WriteRune(r)
}

// WriteToOrExit calls WriteTo. It calls DefaultExitFn if an error occurs.
func (o *Buffer) WriteToOrExit(w io.Writer) int64 {
	n, err := o.WriteTo(w)
	if err != nil {
		DefaultExitFn(fmt.Errorf("iokit.buffer: failed to write to - %w", err))
	}

	return n
}

// WriteTo calls Buf.WriteTo.
func (o *Buffer) WriteTo(w io.Writer) (int64, error) {
	if o.Buf == nil {
		o.Buf = bytes.NewBuffer(nil)
	}

	var hexDumpOutput *bytes.Buffer
	var hexDumper io.WriteCloser

	if o.OptLoggerW != nil {
		hexDumpOutput = bytes.NewBuffer(nil)
		hexDumper = hex.Dumper(hexDumpOutput)

		w = io.MultiWriter(hexDumper, w)
	}

	n, err := o.Buf.WriteTo(w)

	if o.OptLoggerW != nil {
		// Flush remaining bytes to the hex dump buffer.
		_ = hexDumper.Close()

		hexDump := hexDumpOutput.String()
		if len(hexDump) <= 1 {
			// hex.Dump always adds a newline.
			hexDump = "<empty-value>"
		} else {
			hexDump = hexDump[0 : len(hexDump)-1]
		}

		o.OptLoggerW.Println("iokit.buffer: write to:\n" + hexDump)
	}

	return n, err
}
