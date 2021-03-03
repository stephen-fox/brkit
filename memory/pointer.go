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
		defaultExitFn(fmt.Errorf("failed to create pointer maker - %w", err))
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

func (o PointerMaker) Uint(address uint) Pointer {
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

func (o PointerMaker) HexStringOrExit(hexStr string, sourceEndianness binary.ByteOrder) Pointer {
	p, err := o.HexString(hexStr, sourceEndianness)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to convert hex string to pointer - %w", err))
	}
	return p
}

func (o PointerMaker) HexString(hexStr string, sourceEndianness binary.ByteOrder) (Pointer, error) {
	return o.HexBytes([]byte(hexStr), sourceEndianness)
}

func (o PointerMaker) HexBytesOrExit(hexBytes []byte, sourceEndianness binary.ByteOrder) Pointer {
	p, err := o.HexBytes(hexBytes, sourceEndianness)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to convert hex bytes to pointer - %w", err))
	}
	return p
}

func (o PointerMaker) HexBytes(hexBytes []byte, sourceEndianness binary.ByteOrder) (Pointer, error) {
	hexBytesNoPrefix := bytes.TrimPrefix(hexBytes, []byte("0x"))

	hexStrLen := len(hexBytesNoPrefix)
	if hexStrLen == 0 {
		return Pointer{}, fmt.Errorf("hex string cannot be zero-length")
	}

	maxLen := o.ptrSize * 2
	if hexStrLen > maxLen {
		return Pointer{}, fmt.Errorf("hex string cannot be longer than %d chars - it is %d chars long",
			maxLen, hexStrLen)
	}

	leadingZeros := maxLen - hexStrLen
	if leadingZeros > 0 {
		zeros := bytes.Repeat([]byte("0"), leadingZeros)
		if sourceEndianness.String() == binary.LittleEndian.String() {
			hexBytesNoPrefix = append(hexBytesNoPrefix, zeros...)
		} else {
			hexBytesNoPrefix = append(zeros, hexBytesNoPrefix...)
		}
	}

	decoded := make([]byte, o.ptrSize)
	_, err := hex.Decode(decoded, hexBytesNoPrefix)
	if err != nil {
		return Pointer{}, fmt.Errorf("failed to hex decode data - %w", err)
	}

	var canonicalBytes []byte
	if sourceEndianness.String() == o.byteOrder.String() {
		canonicalBytes = decoded
	} else {
		canonicalBytes = make([]byte, o.ptrSize)
		for i := 0; i < o.ptrSize; i++ {
			canonicalBytes[o.ptrSize-1-i] = decoded[i]
		}
	}

	var address uint
	switch o.ptrSize {
	case 2:
		address = uint(o.byteOrder.Uint16(canonicalBytes))
	case 4:
		address = uint(o.byteOrder.Uint32(canonicalBytes))
	case 8:
		address = uint(o.byteOrder.Uint64(canonicalBytes))
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
