package asmkit_test

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"gitlab.com/stephen-fox/brkit/asmkit"
)

func ExampleDisassembler() {
	// exit(1) syscall shellcode by Charles Stevenson:
	// http://shell-storm.org/shellcode/files/shellcode-55.php
	hexEncodedInsts := "31c04089c3cd80"

	disass, err := asmkit.NewDisassembler(asmkit.DisassemblerConfig{
		Src:        hex.NewDecoder(strings.NewReader(hexEncodedInsts)),
		Syntax:     asmkit.IntelSyntax,
		ArchConfig: asmkit.X86Config{Bits: 32},
	})
	if err != nil {
		log.Fatalf("failed to create disassembler - %v", err)
	}

	for disass.Next() {
		fmt.Println(disass.Inst().Assembly)
	}

	err = disass.Err()
	if err != nil {
		log.Fatalf("disassembler failed - %v", err)
	}

	// Output:
	// xor eax, eax
	// inc eax
	// mov ebx, eax
	// int 0x80
}
