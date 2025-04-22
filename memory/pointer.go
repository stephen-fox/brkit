package memory

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// PointerMakerForX86_32 returns a new PointerMaker for a X86 32-bit system.
func PointerMakerForX86_32() PointerMaker {
	return PointerMaker{
		target:  binary.LittleEndian,
		bits:    32,
		ptrSize: 4,
	}
}

// PointerMakerForX86_64 returns a new PointerMaker for a X86 64-bit system.
func PointerMakerForX86_64() PointerMaker {
	return PointerMaker{
		target:  binary.LittleEndian,
		bits:    64,
		ptrSize: 8,
	}
}

// PointerMakerForOrExit calls PointerMakerFor, subsequently calling
// DefaultExitFn if an error occurs.
//
// Refer to PointerMakerFor for more information.
func PointerMakerForOrExit(endianness binary.ByteOrder, bits int, pointerSize int) PointerMaker {
	pm, err := PointerMakerFor(endianness, bits, pointerSize)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to create pointer maker - %w", err))
	}
	return pm
}

// PointerMakerFor returns a new PointerMaker for the specified endianness,
// bits, and pointer size in bytes.
func PointerMakerFor(endianness binary.ByteOrder, bits int, pointerSizeBytes int) (PointerMaker, error) {
	if endianness == nil {
		return PointerMaker{}, fmt.Errorf("endianness cannot be nil")
	}

	if bits <= 0 {
		return PointerMaker{}, fmt.Errorf("bits cannot be less than or equal to zero")
	}

	if pointerSizeBytes <= 0 {
		return PointerMaker{}, fmt.Errorf("pointer size cannot be less than or equal to zero")
	}

	return PointerMaker{
		target:  endianness,
		bits:    bits,
		ptrSize: pointerSizeBytes,
	}, nil
}

// PointerMaker helps with converting various representations of pointers
// to Pointer objects. Refer to Pointer's documentation for more information.
type PointerMaker struct {
	target  binary.ByteOrder
	bits    int
	ptrSize int
}

// ParseUintPrefixOrExit calls PointerMaker.ParseUintPrefix,
// subsequently calling DefaultExitFn if an error occurs.
//
// Refer to PointerMaker.ParseUintPrefix for more information.
func (o PointerMaker) ParseUintPrefixOrExit(str string, base int, prefix string) Pointer {
	p, err := o.ParseUintPrefix(str, base, prefix)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to parse uint string: %q - %w",
			str, err))
	}

	return p
}

// ParseUintPrefix trims the specified prefix from str and then
// parses the resulting string into a Pointer using ParseUint.
func (o PointerMaker) ParseUintPrefix(str string, base int, prefix string) (Pointer, error) {
	str = strings.TrimPrefix(str, prefix)

	return o.ParseUint(str, base)
}

// ParseUintOrExit calls PointerMaker.ParseUint, subsequently calling
// DefaultExitFn if an error occurs.
//
// Refer to PointerMaker.ParseUint for more information.
func (o PointerMaker) ParseUintOrExit(str string, base int) Pointer {
	p, err := o.ParseUint(str, base)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to parse uint string: %q - %w",
			str, err))
	}

	return p
}

// ParseUint parses a string into an unsigned integer and converts
// the resulting integer into a Pointer.
func (o PointerMaker) ParseUint(str string, base int) (Pointer, error) {
	u, err := strconv.ParseUint(str, base, o.bits)
	if err != nil {
		return Pointer{}, err
	}

	return o.FromUint(uint(u)), nil
}

// FromUint converts an unsigned integer memory address into a Pointer.
func (o PointerMaker) FromUint(address uint) Pointer {
	out := make([]byte, o.ptrSize)
	switch o.bits {
	case 16:
		o.target.PutUint16(out, uint16(address))
	case 32:
		o.target.PutUint32(out, uint32(address))
	case 64:
		o.target.PutUint64(out, uint64(address))
	default:
		panic(fmt.Sprintf("unsupported bits: %d", o.bits))
	}
	return Pointer{
		byteOrder: o.target,
		address:   address,
		bytes:     out,
	}
}

// FromHexStringOrExit calls PointerMaker.FromHexString, subsequently calling
// DefaultExitFn if an error occurs.
//
// Refer to PointerMaker.FromHexString for more information.
func (o PointerMaker) FromHexStringOrExit(hexStr string, sourceEndianness binary.ByteOrder) Pointer {
	p, err := o.FromHexString(hexStr, sourceEndianness)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to convert hex string to pointer - %w", err))
	}
	return p
}

// FromHexString converts a hex string to a Pointer according to the
// source endianness.
//
// The string can be prefixed with a "0x", which will be discarded
// prior to decoding.
func (o PointerMaker) FromHexString(hexStr string, sourceEndianness binary.ByteOrder) (Pointer, error) {
	return o.FromHexBytes([]byte(hexStr), sourceEndianness)
}

// FromHexBytesOrExit calls PointerMaker.FromHexBytes, subsequently calling
// DefaultExitFn if an error occurs.
//
// Refer to PointerMaker.FromHexBytes for more information.
func (o PointerMaker) FromHexBytesOrExit(hexBytes []byte, sourceEndianness binary.ByteOrder) Pointer {
	p, err := o.FromHexBytes(hexBytes, sourceEndianness)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to convert hex bytes to pointer - %w", err))
	}
	return p
}

// FromHexBytes converts a hex encoded byte sequence to a Pointer according
// to the source endianness.
//
// The byte sequence can be prefixed with a "0x", which will be discarded
// prior to decoding..
func (o PointerMaker) FromHexBytes(hexBytes []byte, sourceEndianness binary.ByteOrder) (Pointer, error) {
	hexBytesNoPrefix := bytes.TrimPrefix(hexBytes, []byte("0x"))

	decoded := make([]byte, hex.DecodedLen(len(hexBytesNoPrefix)))
	_, err := hex.Decode(decoded, hexBytesNoPrefix)
	if err != nil {
		return Pointer{}, fmt.Errorf("failed to hex decode data - %w", err)
	}

	return o.FromRawBytes(decoded, sourceEndianness)
}

// FromRawBytesOrExit calls PointerMaker.FromRawBytes, subsequently calling
// DefaultExitFn if an error occurs.
//
// Refer to PointerMaker.FromRawBytes for more information.
func (o PointerMaker) FromRawBytesOrExit(raw []byte, sourceEndianness binary.ByteOrder) Pointer {
	p, err := o.FromRawBytes(raw, sourceEndianness)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to convert raw bytes to pointer - %w", err))
	}
	return p
}

// FromRawBytes converts a raw []byte into a Pointer given its
// source endianness.
func (o PointerMaker) FromRawBytes(raw []byte, sourceEndianness binary.ByteOrder) (Pointer, error) {
	rawLen := len(raw)
	if rawLen == 0 {
		return Pointer{}, errors.New("pointer bytes cannot be zero-length")
	}

	if rawLen > o.ptrSize {
		return Pointer{}, fmt.Errorf("pointer bytes cannot be longer than pointer size of %d - it is %d bytes long",
			o.ptrSize, rawLen)
	}

	leadingZeros := o.ptrSize - rawLen
	if leadingZeros > 0 {
		zeros := bytes.Repeat([]byte{0x00}, leadingZeros)
		if sourceEndianness.String() == binary.LittleEndian.String() {
			raw = append(raw, zeros...)
		} else {
			raw = append(zeros, raw...)
		}
	}

	var canonicalBytes []byte
	if sourceEndianness.String() == o.target.String() {
		canonicalBytes = raw
	} else {
		canonicalBytes = make([]byte, o.ptrSize)
		for i := 0; i < o.ptrSize; i++ {
			canonicalBytes[o.ptrSize-1-i] = raw[i]
		}
	}

	var address uint
	switch o.ptrSize {
	case 2:
		address = uint(sourceEndianness.Uint16(raw))
	case 4:
		address = uint(sourceEndianness.Uint32(raw))
	case 8:
		address = uint(sourceEndianness.Uint64(raw))
	default:
		return Pointer{}, fmt.Errorf("unsupported pointer size: %d", o.ptrSize)
	}

	return Pointer{
		byteOrder: o.target,
		address:   address,
		bytes:     canonicalBytes,
	}, nil
}

// Pointer provides a canonical representation of a memory address pointer.
// A pointer is a variable that points to a memory address.
//
// Pointers are created with a PointerMaker.
type Pointer struct {
	byteOrder binary.ByteOrder
	address   uint
	bytes     []byte
}

// Bytes returns the pointer as a []byte in the endianness of the target
// platform with the correct padding.
func (o Pointer) Bytes() []byte {
	return o.bytes
}

// Uint returns the pointer as a unsigned integer.
//
// This is useful for performing math on the pointed-to address.
func (o Pointer) Uint() uint {
	return o.address
}

// Offset returns a new Pointer after applying the given amount.
func (o Pointer) Offset(amount int64) Pointer {
	adjusted := o.address
	switch {
	case amount == 0:
		return o
	case amount > 0:
		adjusted = adjusted + uint(amount)
	case amount < 0:
		adjusted = adjusted - uint(amount*-1)
	}

	numBytes := len(o.bytes)
	b := make([]byte, numBytes)

	switch numBytes {
	case 1:
		b[0] = uint8(adjusted)
	case 2:
		o.byteOrder.PutUint16(b, uint16(adjusted))
	case 4:
		o.byteOrder.PutUint32(b, uint32(adjusted))
	case 8:
		o.byteOrder.PutUint64(b, uint64(adjusted))
	default:
		panic(fmt.Sprintf("unsupported pointer length: %d", numBytes))
	}

	return Pointer{
		byteOrder: o.byteOrder,
		address:   adjusted,
		bytes:     b,
	}
}

// HexString returns a hex-encoded string representing the pointer,
// prefixed with the "0x" string.
//
// This method renders the pointer as a human would expect to
// see it; it does not respect the platform's byte ordering.
func (o Pointer) HexString() string {
	initial := fmt.Sprintf("%x", o.address)

	needsToBeLen := len(o.bytes) * 2
	initialLen := len(initial)

	if initialLen < needsToBeLen {
		return "0x" + strings.Repeat("0", needsToBeLen-initialLen) + initial
	}

	return "0x" + initial
}
