package memory_test

import (
	"encoding/binary"
	"fmt"
	"log"

	"gitlab.com/stephen-fox/brkit/memory"
)

func ExamplePointerMaker_ParseUintPrefix() {
	pm := memory.PointerMakerForX86_32()

	pointer, err := pm.ParseUintPrefix("0xdeadbeef", 16, "0x")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xdeadbeef
}

func ExamplePointerMaker_ParseUint() {
	pm := memory.PointerMakerForX86_32()

	pointer, err := pm.ParseUint("deadbeef", 16)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xdeadbeef
}

func ExamplePointerMaker_FromHexString() {
	pm := memory.PointerMakerForX86_32()

	pointer, err := pm.FromHexString("0xdeadbeef", binary.BigEndian)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xdeadbeef
}

func ExamplePointerMaker_FromHexBytes() {
	pm := memory.PointerMakerForX86_32()

	pointer, err := pm.FromHexBytes([]byte("0xdeadbeef"), binary.BigEndian)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xdeadbeef
}

func ExamplePointerMaker_FromUint() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	fmt.Println(pointer.HexString())

	// Output: 0xdeadbeef
}

func ExamplePointerMaker_FromRawBytes() {
	pm := memory.PointerMakerForX86_32()

	pointer, err := pm.FromRawBytes([]byte{0xde, 0xad, 0xbe, 0xef}, binary.BigEndian)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xdeadbeef
}

func ExamplePointer_Bytes() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	fmt.Printf("0x%x", pointer.Bytes())

	// Output: 0xefbeadde
}

func ExamplePointer_Uint() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	fmt.Printf("0x%x", pointer.Uint())

	// Output: 0xdeadbeef
}

func ExamplePointer_Uint_math() {
	pm := memory.PointerMakerForX86_32()

	initial := pm.FromUint(0xdeadbeef)

	modified := pm.FromUint(initial.Uint() - 0xef)

	fmt.Printf("0x%x", modified.Uint())

	// Output: 0xdeadbe00
}

func ExamplePointer_HexString() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	fmt.Println(pointer.HexString())

	// Output: 0xdeadbeef
}

func ExamplePointer_Offset_addition() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	adjustedPointer := pointer.Offset(0x100)

	fmt.Printf("0x%x", adjustedPointer.Uint())

	// Output: 0xdeadbfef
}

func ExamplePointer_Offset_subtract() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	adjustedPointer := pointer.Offset(-0x100)

	fmt.Printf("0x%x", adjustedPointer.Uint())

	// Output: 0xdeadbdef
}

func ExamplePointer_IsNull() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.FromUint(0x00)

	if pointer.IsNull() {
		fmt.Println("pointer is null")
	}

	// Output: pointer is null
}

func ExamplePointer_NonNullOrExit() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	pointer.NonNullOrExit()

	fmt.Println("pointer is non-null")

	// Output: pointer is non-null
}
