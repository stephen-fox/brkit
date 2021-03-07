package memory

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

func PointerMakerForX68_32() PointerMaker {
	return PointerMaker{
		byteOrder: binary.LittleEndian,
		bits:      32,
		ptrSize:   4,
	}
}

func PointerMakerForX68_64() PointerMaker {
	return PointerMaker{
		byteOrder: binary.LittleEndian,
		bits:      64,
		ptrSize:   8,
	}
}

func PointerMakerForOrExit(endianness binary.ByteOrder, bits int, pointerSize int) PointerMaker {
	pm, err := PointerMakerFor(endianness, bits, pointerSize)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to create pointer maker - %w", err))
	}
	return pm
}

func PointerMakerFor(endianness binary.ByteOrder, bits int, pointerSize int) (PointerMaker, error) {
	if endianness == nil {
		return PointerMaker{}, fmt.Errorf("endianness cannot be nil")
	}

	if bits <= 0 {
		return PointerMaker{}, fmt.Errorf("bits cannot be less than or equal to zero")
	}

	if pointerSize <= 0 {
		return PointerMaker{}, fmt.Errorf("pointer size cannot be less than or equal to zero")
	}

	return PointerMaker{
		byteOrder: endianness,
		bits:      bits,
		ptrSize:   pointerSize,
	}, nil
}

type PointerMaker struct {
	byteOrder binary.ByteOrder
	bits      int
	ptrSize   int
}

func (o PointerMaker) FromUint(address uint) Pointer {
	out := make([]byte, o.ptrSize)
	switch o.bits {
	case 16:
		o.byteOrder.PutUint16(out, uint16(address))
	case 32:
		o.byteOrder.PutUint32(out, uint32(address))
	case 64:
		o.byteOrder.PutUint64(out, uint64(address))
	default:
		panic(fmt.Sprintf("unsupported bits: %d", o.bits))
	}
	return Pointer{
		byteOrder: o.byteOrder,
		address:   address,
		bytes:     out,
	}
}

func (o PointerMaker) FromHexStringOrExit(hexStr string, sourceEndianness binary.ByteOrder) Pointer {
	p, err := o.FromHexString(hexStr, sourceEndianness)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to convert hex string to pointer - %w", err))
	}
	return p
}

func (o PointerMaker) FromHexString(hexStr string, sourceEndianness binary.ByteOrder) (Pointer, error) {
	return o.FromHexBytes([]byte(hexStr), sourceEndianness)
}

func (o PointerMaker) FromHexBytesOrExit(hexBytes []byte, sourceEndianness binary.ByteOrder) Pointer {
	p, err := o.FromHexBytes(hexBytes, sourceEndianness)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to convert hex bytes to pointer - %w", err))
	}
	return p
}

func (o PointerMaker) FromHexBytes(hexBytes []byte, sourceEndianness binary.ByteOrder) (Pointer, error) {
	hexBytesNoPrefix := bytes.TrimPrefix(hexBytes, []byte("0x"))

	decoded := make([]byte, hex.DecodedLen(len(hexBytesNoPrefix)))
	_, err := hex.Decode(decoded, hexBytesNoPrefix)
	if err != nil {
		return Pointer{}, fmt.Errorf("failed to hex decode data - %w", err)
	}

	return o.FromRawBytes(decoded, sourceEndianness)
}

func (o PointerMaker) FromRawBytesOrExit(raw []byte, sourceEndianness binary.ByteOrder) Pointer {
	p, err := o.FromRawBytes(raw, sourceEndianness)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to convert raw bytes to pointer - %w", err))
	}
	return p
}

func (o PointerMaker) FromRawBytes(raw []byte, sourceEndianness binary.ByteOrder) (Pointer, error) {
	rawLen := len(raw)
	if rawLen == 0 {
		return Pointer{}, fmt.Errorf("pointer bytes slice cannot be zero-length")
	}

	if rawLen > o.ptrSize {
		return Pointer{}, fmt.Errorf("slice cannot be longer than pointer size of %d - it is %d bytes long",
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
	if sourceEndianness.String() == o.byteOrder.String() {
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
		byteOrder: o.byteOrder,
		address:   address,
		bytes:     canonicalBytes,
	}, nil
}

type Pointer struct {
	byteOrder binary.ByteOrder
	address   uint
	bytes     []byte
}

func (o Pointer) Bytes() []byte {
	return o.bytes
}

func (o Pointer) Uint() uint {
	return o.address
}

func (o Pointer) HexString() string {
	return fmt.Sprintf("0x%x", o.Bytes())
}
