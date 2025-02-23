package iokit

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"gitlab.com/stephen-fox/brkit/memory"
)

func ExampleNewPayloadBuilder() {
	pm := memory.PointerMakerForX86_64()

	examplePointer := pm.FromUint(0x7ffac0ded00d)

	examplePatternGen := &examplePatternGenerator{}

	payload := NewPayloadBuilder().
		RepeatString("A", 8*2).
		String("zerocool").
		Pattern(examplePatternGen, 16).
		Bytes([]byte{0xa8, 0xac, 0x20, 0xff, 0x42, 0x7f, 0x00, 0x00}).
		Uint64(0xc0ded00d, binary.LittleEndian).
		Pointer(examplePointer).
		Build()

	fmt.Print(hex.Dump(payload))

	// Output:
	// 00000000  41 41 41 41 41 41 41 41  41 41 41 41 41 41 41 41  |AAAAAAAAAAAAAAAA|
	// 00000010  7a 65 72 6f 63 6f 6f 6c  41 30 42 30 43 30 44 30  |zerocoolA0B0C0D0|
	// 00000020  45 30 46 30 41 31 41 32  a8 ac 20 ff 42 7f 00 00  |E0F0A1A2.. .B...|
	// 00000030  0d d0 de c0 00 00 00 00  0d d0 de c0 fa 7f 00 00  |................|
}

type examplePatternGenerator struct{}

func (o *examplePatternGenerator) Pattern(int) ([]byte, error) {
	return []byte("A0B0C0D0E0F0A1A2"), nil
}
