package bstruct_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"gitlab.com/stephen-fox/brkit/bstruct"
)

func ExampleToBytesX86() {
	type example struct {
		Counter  uint16
		SomePtr  uint32
		Register uint32
	}

	buf := bytes.NewBuffer(nil)

	err := bstruct.ToBytesX86(bstruct.FieldWriterFn(buf), example{
		Counter:  666,
		SomePtr:  0xc0ded00d,
		Register: 0xfabfabdd,
	})
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("0x%x", buf.Bytes())

	// Output:
	// 0x9a020dd0dec0ddabbffa
}

func ExampleToBytes() {
	type example struct {
		Counter  uint16
		SomePtr  uint32
		Register uint32
	}

	buf := bytes.NewBuffer(nil)

	err := bstruct.ToBytes(
		binary.LittleEndian,
		bstruct.GoFieldOrder,
		bstruct.FieldWriterFn(buf), example{
			Counter:  666,
			SomePtr:  0xc0ded00d,
			Register: 0xfabfabdd,
		})
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("0x%x", buf.Bytes())

	// Output:
	// 0x9a020dd0dec0ddabbffa
}

func ExampleToBytes_with_logging() {
	type example struct {
		Counter  uint16
		SomePtr  uint32
		Register uint32
	}

	logger := log.New(os.Stdout, "", 0)

	err := bstruct.ToBytes(
		binary.LittleEndian,
		bstruct.GoFieldOrder,
		bstruct.FieldWriterFn(io.Discard, logger), example{
			Counter:  666,
			SomePtr:  0xc0ded00d,
			Register: 0xfabfabdd,
		})
	if err != nil {
		log.Fatalln(err)
	}

	// Output:
	// bstruct.fieldwriterfn - field: 0 | name: "Counter" | type: uint16 | value:
	// 00000000  9a 02                                             |..|
	// bstruct.fieldwriterfn - field: 1 | name: "SomePtr" | type: uint32 | value:
	// 00000000  0d d0 de c0                                       |....|
	// bstruct.fieldwriterfn - field: 2 | name: "Register" | type: uint32 | value:
	// 00000000  dd ab bf fa                                       |....|
}
