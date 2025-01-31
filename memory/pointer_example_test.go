package memory

import (
	"encoding/binary"
	"fmt"
	"log"
)

func ExamplePointerMaker_FromParseUintPrefix() {
	pm := PointerMakerForX86_32()

	pointer, err := pm.FromParseUintPrefix("0xdeadbeef", 16, "0x")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xefbeadde
}

func ExamplePointerMaker_FromParseUint() {
	pm := PointerMakerForX86_32()

	pointer, err := pm.FromParseUint("deadbeef", 16)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xefbeadde
}

func ExamplePointerMaker_FromHexString() {
	pm := PointerMakerForX86_32()

	pointer, err := pm.FromHexString("0xdeadbeef", binary.BigEndian)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xefbeadde
}

func ExamplePointerMaker_FromHexBytes() {
	pm := PointerMakerForX86_32()

	pointer, err := pm.FromHexBytes([]byte("0xdeadbeef"), binary.BigEndian)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xefbeadde
}

func ExamplePointerMaker_FromUint() {
	pm := PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	fmt.Println(pointer.HexString())

	// Output: 0xefbeadde
}

func ExamplePointerMaker_FromRawBytes() {
	pm := PointerMakerForX86_32()

	pointer, err := pm.FromRawBytes([]byte{0xde, 0xad, 0xbe, 0xef}, binary.BigEndian)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(pointer.HexString())

	// Output: 0xefbeadde
}

func ExamplePointer_Bytes() {
	pm := PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	fmt.Printf("0x%x", pointer.Bytes())

	// Output: 0xefbeadde
}

func ExamplePointer_Uint() {
	pm := PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	fmt.Printf("0x%x", pointer.Uint())

	// Output: 0xdeadbeef
}

func ExamplePointer_Uint_Math() {
	pm := PointerMakerForX86_32()

	initial := pm.FromUint(0xdeadbeef)

	modified := pm.FromUint(initial.Uint()-0xef)

	fmt.Printf("0x%x", modified.Uint())

	// Output: 0xdeadbe00
}

func ExamplePointer_HexString() {
	pm := PointerMakerForX86_32()

	pointer := pm.FromUint(0xdeadbeef)

	fmt.Println(pointer.HexString())

	// Output: 0xefbeadde
}
