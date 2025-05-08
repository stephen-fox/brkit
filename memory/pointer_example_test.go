package memory_test

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"strings"

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

func ExamplePointerMaker_NullPtr() {
	pm := memory.PointerMakerForX86_32()

	pointer := pm.NullPtr()

	fmt.Println(pointer.HexString())

	// Output: 0x00000000
}

func ExamplePointerMaker_WithNullAllowed() {
	pm := memory.PointerMakerForX86_32()

	pointer, _ := pm.WithNullAllowed(func(allowNull memory.PointerMaker) (memory.Pointer, error) {
		return allowNull.FromUint(0x00), nil
	})

	fmt.Println(pointer.HexString())

	// Output: 0x00000000
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

	pointer := pm.NullPtr()

	if pointer.IsNull() {
		fmt.Println("pointer is null")
	}

	// Output: pointer is null
}

func ExamplePointer_zero_value() {
	// This example demonstrates a possible real-world scenario
	// where a user may declare a Pointer and assign its value
	// in a loop. If the loop never runs or does not otherwise
	// reach the assignment code, then the Pointer variable
	// will be null.
	//
	// The default behavior for the Pointer type's methods make
	// this mistake apparent by exiting the process if the Pointer
	// is null.

	pm := memory.PointerMakerForX86_64()

	var examplePtr memory.Pointer

	// This is a purposely-broken /proc/<pid>/maps output:
	exampleProcMaps := `55c8ded77000-55c8ded79000 r--p 00000000 fe:02 2360849                    /usr/bin/cat
55c8ded79000-55c8ded7e000 r-xp 00002000 fe:02 2360849                    /usr/bin/cat
`
	scanner := bufio.NewScanner(strings.NewReader(exampleProcMaps))

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "[heap]") {
			index := strings.Index(line, "-")
			if index > -1 {
				baseAddrStr := line[0:index]

				// This assignment is never reached.
				examplePtr = pm.FromHexStringOrExit(baseAddrStr, binary.BigEndian)

				break
			}
		}
	}

	// Since examplePtr is never assigned, the default value means it
	// is pointing at zero. To protect against this, examplePtr.Uint64
	// will write an error message to stderr and exit the process.
	heapOffset := examplePtr.Uint64() + 0x41

	_ = heapOffset
}
