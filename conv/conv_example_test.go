package conv

import (
	"bytes"
	"fmt"
	"log"
)

func ExampleHexArrayToBytes() {
	// exit(1) syscall shellcode by Charles Stevenson:
	// http://shell-storm.org/shellcode/files/shellcode-55.php
	cArrayContents := []byte(
`/*  _exit(1); linux/x86 by core */
// 7 bytes _exit(1) ... 'cause we're nice >:) by core
"\x31\xc0"              // xor  %eax,%eax
"\x40"                  // inc  %eax
"\x89\xc3"              // mov  %eax,%ebx
"\xcd\x80"              // int  $0x80
`)

	exit1Bytes, err := HexArrayToBytes(bytes.NewReader(cArrayContents))
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("0x%x\n", exit1Bytes)

	// Output: 0x31c04089c3cd80
}
