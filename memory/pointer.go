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
	return out
}

func (o PointerMaker) HexString(hexStr string, sourceEndianness binary.ByteOrder) (Pointer, error) {
	return o.HexBytes([]byte(hexStr), sourceEndianness)
}

func (o PointerMaker) HexBytes(hexBytes []byte, sourceEndianness binary.ByteOrder) (Pointer, error) {
	hexBytesNoPrefix := bytes.TrimPrefix(hexBytes, []byte("0x"))

	hexStrLen := len(hexBytesNoPrefix)
	if hexStrLen == 0 {
		return nil, fmt.Errorf("hex string cannot be zero-length")
	}

	maxLen := o.ptrSize * 2
	if hexStrLen > maxLen {
		return nil, fmt.Errorf("hex string cannot be longer than %d chars - it is %d chars long",
			maxLen, hexStrLen)
	}

	numZeros := maxLen - hexStrLen
	if numZeros > 0 {
		zeros := bytes.Repeat([]byte("0"), numZeros)
		if sourceEndianness.String() == binary.LittleEndian.String() {
			hexBytesNoPrefix = append(hexBytesNoPrefix, zeros...)
		} else {
			hexBytesNoPrefix = append(zeros, hexBytesNoPrefix...)
		}
	}

	decoded := make([]byte, o.ptrSize)
	_, err := hex.Decode(decoded, hexBytesNoPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to hex decode data - %w", err)
	}

	if sourceEndianness.String() == o.byteOrder.String() {
		return decoded, nil
	}

	wrongEndian := make([]byte, o.ptrSize)
	for i := 0; i < o.ptrSize; i++ {
		wrongEndian[o.ptrSize-1-i] = decoded[i]
	}
	return wrongEndian, nil
}

type Pointer []byte

func (o Pointer) HexString() string {
	return fmt.Sprintf("0x%x", o)
}
