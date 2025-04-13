package conv_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"gitlab.com/stephen-fox/brkit/conv"
)

func ExampleCArrayToGoSlice() {
	// Note: This example is copied from IZ <guerrilla.sytes.net>
	// Title: "FreeBSD_x86-execve_sh-23b-iZ.c (Shellcode, execve /bin/sh, 23 bytes)"
	//
	// https://shell-storm.org/shellcode/files/shellcode-170.html
	example := strings.NewReader(`"\x31\xc0"                  /* xor %eax,%eax */
"\x50"                      /* push %eax */
"\x68\x2f\x2f\x73\x68"      /* push $0x68732f2f (//sh) */
"\x68\x2f\x62\x69\x6e"      /* push $0x6e69622f (/bin)*/

"\x89\xe3"                  /* mov %esp,%ebx */
"\x50"                      /* push %eax */
"\x54"                      /* push %esp */
"\x53"                      /* push %ebx */

"\x50"                      /* push %eax */
"\xb0\x3b"                  /* mov $0x3b,%al */
"\xcd\x80";                 /* int $0x80 */
`)

	output := bytes.NewBuffer(nil)

	err := conv.CArrayToGoSlice(example, output)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(output.String())

	// Output:
	// []byte{
	// 	0x31c0,       // xor %eax,%eax
	//	0x50,         // push %eax
	//	0x682f2f7368, // push $0x68732f2f (//sh)
	//	0x682f62696e, // push $0x6e69622f (/bin)
	//	0x89e3,       // mov %esp,%ebx
	//	0x50,         // push %eax
	//	0x54,         // push %esp
	//	0x53,         // push %ebx
	//	0x50,         // push %eax
	//	0xb03b,       // mov $0x3b,%al
	//	0xcd80,       // int $0x80
	// }
}
