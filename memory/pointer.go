package memory

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
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

func HexStringToPointerOrExit(hexStr string, pm PointerMaker, bits int) Pointer {
	res, err := HexStringToPointer(hexStr, pm, bits)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to convert '%s' to pointer - %w", hexStr, err))
	}
	return res
}

func HexStringToPointer(hexStr string, pm PointerMaker, bits int) (Pointer, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")

	i, err := strconv.ParseUint(hexStr, 16, bits)
	if err != nil {
		return nil, err
	}

	switch bits {
	case 32:
		return pm.U32(uint32(i)), nil
	case 64:
		return pm.U64(i), nil
	default:
		return nil, fmt.Errorf("unsupported bits %d", bits)
	}
}
