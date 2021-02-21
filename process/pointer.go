package process

import (
	"encoding/binary"
	"fmt"
)

func PointerMakerForX86() PointerMaker {
	return PointerMakerFor(binary.LittleEndian)
}

func PointerMakerFor(targetSystemEndianness binary.ByteOrder) PointerMaker {
	return PointerMaker{
		byteOrder: targetSystemEndianness,
	}
}

type PointerMaker struct {
	byteOrder binary.ByteOrder
}

func (o PointerMaker) U32(address uint32) Pointer {
	out := make([]byte, 4)
	o.byteOrder.PutUint32(out, address)
	return out
}

func (o PointerMaker) U64(address uint64) Pointer {
	out := make([]byte, 8)
	o.byteOrder.PutUint64(out, address)
	return out
}

type Pointer []byte

func (o Pointer) HexString() string {
	return fmt.Sprintf("0x%x", o)
}
