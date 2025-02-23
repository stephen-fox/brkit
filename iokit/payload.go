package iokit

import (
	"encoding/binary"
	"fmt"
)

// NewPayloadBuilder instantiates a new PayloadBuilder.
func NewPayloadBuilder() *PayloadBuilder {
	return &PayloadBuilder{
		buf: Buffer{},
	}
}

// PayloadBuilder helps build payloads and other binary sequences
// by implementing the "builder pattern".
type PayloadBuilder struct {
	buf Buffer
	err error
}

// Uint32 writes an unsigned 64-bit integer in the specified
// binary.ByteOrder (or endianness) to the payload.
func (o *PayloadBuilder) Uint32(u uint32, order binary.ByteOrder) *PayloadBuilder {
	b := make([]byte, 4)

	order.PutUint32(b, u)

	o.Bytes(b)

	return o
}

// Uint64 writes an unsigned 64-bit integer in the specified
// binary.ByteOrder (or endianness) to the payload.
func (o *PayloadBuilder) Uint64(u uint64, order binary.ByteOrder) *PayloadBuilder {
	b := make([]byte, 8)

	order.PutUint64(b, u)

	o.Bytes(b)

	return o
}

// PatternGenerator abstracts pattern string generators.
type PatternGenerator interface {
	// Pattern generates a pattern string as a []byte. Each byte
	// in the slice is a human-readable character.
	Pattern(numBytes int) ([]byte, error)
}

// Pattern writes the specified number of bytes from the PatternGenerator
// to the payload.
func (o *PayloadBuilder) Pattern(generator PatternGenerator, numBytes int) *PayloadBuilder {
	if o.err != nil {
		return o
	}

	b, err := generator.Pattern(numBytes)
	if err != nil {
		o.err = err
		return o
	}

	o.Bytes(b)

	return o
}

// Byter abstracts types that can be represented as a []byte.
type Byter interface {
	// Bytes returns the object as a []byte.
	Bytes() []byte
}

// Pointer writes a raw pointer as a []byte to the payload.
func (o *PayloadBuilder) Pointer(pointer Byter) *PayloadBuilder {
	return o.Byter(pointer)
}

// Byter writes the specified Byter's []byte to the payload.
func (o *PayloadBuilder) Byter(b Byter) *PayloadBuilder {
	if o.err != nil {
		return o
	}

	_, err := o.buf.Write(b.Bytes())
	if err != nil {
		o.err = err
	}

	return o
}

// Bytes writes the specified []byte to the payload.
func (o *PayloadBuilder) Bytes(b []byte) *PayloadBuilder {
	if o.err != nil {
		return o
	}

	_, err := o.buf.Write(b)
	if err != nil {
		o.err = err
	}

	return o
}

// Byte writes the specified byte to the payload.
func (o *PayloadBuilder) Byte(b byte) *PayloadBuilder {
	if o.err != nil {
		return o
	}

	err := o.buf.WriteByte(b)
	if err != nil {
		o.err = err
	}

	return o
}

// String writes the specified string to the payload.
func (o *PayloadBuilder) String(str string) *PayloadBuilder {
	if o.err != nil {
		return o
	}

	_, err := o.buf.WriteString(str)
	if err != nil {
		o.err = err
	}

	return o
}

// RepeatString repeatedly writes the specified string to the payload.
func (o *PayloadBuilder) RepeatString(str string, count int) *PayloadBuilder {
	if o.err != nil {
		return o
	}

	_, err := o.buf.RepeatString(str, count)
	if err != nil {
		o.err = err
	}

	return o
}

// RepeatBytes repeatedly writes the specified []byte to the payload.
func (o *PayloadBuilder) RepeatBytes(b []byte, count int) *PayloadBuilder {
	if o.err != nil {
		return o
	}

	_, err := o.buf.RepeatBytes(b, count)
	if err != nil {
		o.err = err
	}

	return o
}

// TrimEnd trims the last n bytes from the payload.
func (o *PayloadBuilder) TrimEnd(n int) *PayloadBuilder {
	if o.err != nil {
		return o
	}

	o.buf.TrimEnd(n)

	return o
}

// Build returns the payload as a []byte.
func (o *PayloadBuilder) Build() []byte {
	if o.err != nil {
		DefaultExitFn(fmt.Errorf("failed to build payload - %w", o.err))
	}

	return o.buf.Bytes()
}
