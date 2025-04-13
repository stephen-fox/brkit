# brkit

[![GoDoc][godoc-badge]][godoc]

[godoc-badge]: https://pkg.go.dev/badge/gitlab.com/stephen-fox/brkit
[godoc]: https://pkg.go.dev/gitlab.com/stephen-fox/brkit

Package brkit provides functionality for binary research and exploitation.

brkit was originally developed as a collection of small command line tools.
It eventually expanded into a library that mimics the functionality of
[Python `pwntools`][pwntools].

[pwntools]: https://docs.pwntools.com/en/stable/

## Goals

The overriding goal of this project is to help solve hacking CTF challenges,
specifically the binary exploitation variety. The library's APIs are
open-minded in the sense it can be used (responsibly) for non-CTF work.

Beyond that, the project tries to adhere to the following goals:

- Make developing exploits for low-level vulnerabilities more accessible
- Rely solely on the Go standard library. Use child Go modules as a last
  resort if external dependencies are unavoidable
- Leverage Go's type system as frequently as possible
- Provide APIs whose intent can be understood without a fancy IDE or having
  deep institutional knowledge of the codebase
- Focus on providing "LEGO-like" building blocks that can be easily bolted
  together (i.e., follow the [Unix philosophy][unix-philosophy] of small,
  composable tools)

[unix-philosophy]: https://en.wikipedia.org/wiki/Unix_philosophy

## APIs

brkit is broken into several sub-packages, each representing a distinct set
of functionality. To help with scripting, a set of proxy APIs are provided
which exit the program when an error occurs. These API names end with the
suffix `OrExit` to indicate this behavior.

The following subsections outline the various sub-packages and their usage.
Please refer to the Go doc documentation for detailed explanations and
usage examples.

#### `asmkit`

Package asmkit provides functionality for working with assembly.

Note: This library uses its own Go module, meaning users must explicitly
add the library's module to their `go.mod` file:

```sh
go get gitlab.com/stephen-fox/brkit/asmkit@latest
```

For example, the following hex-encoded shellcode can be disassembled
using the `Disassembler` type:

```go
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
```

#### `bstruct`

Package bstruct provides functionality for converting data structures
to binary.

The following example demonstrates how to convert a struct to binary data for
use on a x86 CPU:

```go
func ExampleToBytesX86() {
	type example struct {
		Counter  uint16
		SomePtr  uint32
		Register uint32
	}

	buf := bytes.NewBuffer(nil)

	bstruct.ToBytesX86OrExit(FieldWriterFn(buf), example{
		Counter:  666,
		SomePtr:  0xc0ded00d,
		Register: 0xfabfabdd,
	})

	fmt.Printf("0x%x", buf.Bytes())

	// Output:
	// 0x9a020dd0dec0ddabbffa
}
```

#### `conv`

Package conv provides functionality for converting binary-related data from
one format to another.

#### `iokit`

Package iokit provides additional input-output functionality that can be
useful when developing exploits.

One of the most useful types provided by this package is the `PayloadBuilder`,
which can compose types from the `memory` and `pattern` libraries:

```go
func ExamplePayloadBuilder() {
        pm := memory.PointerMakerForX86_64()

        examplePointer := pm.FromUint(0x7ffac0ded00d)

        examplePatternGen := &examplePatternGenerator{}

        payload := iokit.NewPayloadBuilder().
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
```

#### `memory`

Package memory provides functionality for reading and writing memory.

The memory library is useful for constructing memory leaks and writes, as well
as tracking memory addresses and pointers programmatically.

The `AddressTable` struct provides a small API for organizing memory offsets
in different contexts. For example, it can be used to track glibc symbol
offsets for different machines:

```go
func ExampleAddressTable() {
	offsets := memory.NewAddressTable("local").
		AddSymbolInContext("ioFileJumps", 0x00000000003ebc30, "local").
		AddSymbolInContext("ioFileJumps", 0x00000000003e82f0, "remote")

	addr := offsets.AddressOrExit("ioFileJumps")
	fmt.Printf("local ioFileJumps: 0x%x\n", addr)

	offsets.SetContext("remote")

	addr = offsets.AddressOrExit("ioFileJumps")
	fmt.Printf("remote ioFileJumps: 0x%x\n", addr)

	// Output:
	// local ioFileJumps: 0x3ebc30
	// remote ioFileJumps: 0x3e82f0
}
```

The `Pointer` struct is used for tracking variables that point to memory
addresses in a separate software process. It accomplishes this by storing
the pointed-to address as a []byte in the correct endianness (also known as
"wrong endian"), and as a unsigned integer. This makes mathematical operations
easy and reliable. A `Pointer` is created using a `PointerMaker`, which stores
platform-specific contexts like endianness and pointer size:

```go
func ExamplePointer_Uint_Math() {
	pm := memory.PointerMakerForX86_32()

	initial := pm.FromUint(0xdeadbeef)

	modified := pm.FromUint(initial.Uint()-0xef)

	fmt.Printf("0x%x", modified.Uint())

	// Output: 0xdeadbe00
}
```

#### Format string exploitation

The memory library also provides functions for automating the creation of
format string attacks, primarily through the direct parameter access (DPA)
feature. The `SetupFormatStringLeakViaDPA` function accomplishes this by
first leaking an oracle string within a newly created format string. This
oracle is replaced with an address provided by the caller. All of this is
done before returning to the caller.

The `ProcessIO` interface type fulfills a similar role as the `io.ReadWriter`.
It abstracts a process' input/output and other important attributes. Normally,
this is provided by the `process.Process` type - but can be implemented
different as desired.

This allows for format string exploitation automation:

```go
func ExampleSetupFormatStringLeakViaDPA() {
	leaker := memory.SetupFormatStringLeakViaDPAOrExit(DPAFormatStringConfig{
		ProcessIO:    &fakeProcessIO{},
		MaxNumParams: 200,
	})

	pm := memory.PointerMakerForX86_64()

	log.Printf("read: 0x%x", leaker.MemoryAtOrExit(pm.FromUint(0x00000000deadbeef)))
}
```

Creation of format string attacks that can write memory is handled in a similar
fashion. The `SetupDPAFormatStringWriter` function leaks the DPA argument
number of an oracle string, and then replaces it with a caller-supplied
address. By abusing certain format specifiers (which is discussed in the
Go doc), callers can effectively overwrite the lower four, two, or
single bytes:

```go
func ExampleDPAFormatStringWriter_WriteLowerFourBytesAt() {
	writer := memory.SetupDPAFormatStringWriterOrExit(DPAFormatStringWriterConfig{
		MaxWrite:  999,
		DPAConfig: DPAFormatStringConfig{
			ProcessIO:    &fakeProcessIO{},
			MaxNumParams: 200,
		},
	})

	pm := memory.PointerMakerForX86_32()

	// Set the lower four bytes to 1000 (0x03E8).
	writer.WriteLowerFourBytesAtOrExit(1000, pm.FromUint(0xdeadbeef))
}
```

#### `pattern`

Package pattern provides functionality for generating pattern strings.

The following example demonstrates how to generate a de Bruijn pattern string:

```go
func ExampleDeBruijn_WriteToN() {
	db := &pattern.DeBruijn{}

	db.WriteToNOrExit(os.Stdout, 16)
	os.Stdout.WriteString("\n")

	db.WriteToNOrExit(os.Stdout, 16)
	os.Stdout.WriteString("\n")

	db.WriteToNOrExit(os.Stdout, 16)

	// Output:
	// aaaabaaacaaadaaa
	// eaaafaaagaaahaaa
	// iaaajaaakaaalaaa
}
```

#### `process`

Package process provides functionality for working with running
software processes.

A software process is represented by the `Process` struct. This abstracts
interaction with a process, regardless of it being a process started by the
library, or an existing one running on another machine across the network.
Several constructor-like functions aid in the instantiation of a new `Process`.
For example, a new process can exec'ed like so:

```go
func ExampleExec() {
	cmd := exec.Command("cat")

	proc := process.ExecOrExit(cmd, process.X86_64Info())
	defer proc.Close()

	proc.WriteLineOrExit([]byte("hello world"))

	line := proc.ReadLineOrExit()

	log.Printf("%s", line)
}
```

If the process has a TCP listener, it can be connected to like so:

```go
func ExampleDial() {
	proc := process.DialOrExit("tcp4", "192.168.1.2:8080", process.X86_64Info())
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}
```

These functions accept an `Info` struct which stores information about the
process, such as its bits. These can be instantiated by specifying their
field values, or by calling the constructor-like helper functions.

## Command line utilities

Several command line utilities are included to aid in binary research efforts.

#### `dasm`

A very simple disassembler that supports various encoding formats.

#### `frag`

Finds fragments in pattern strings. Useful for understanding how a payload
overwrites process state (e.g., finding the offset of a payload fragment in
a variable that was overwritten by a stack-based buffer overflow).

#### `fromhex`

Decodes hex-encoded data (e.g., "\x31\xc0\x40\x89\xc3\xcd\x80") and encodes
the underlying binary data into another encoding.

#### `stringer`

A string creation and manipulation tool capable of creating pattern strings
and arbitrary binary data.

## Installing command line utilities

Since this is a Go (Golang) project, the preferred method of installation
is using `go install`. This automates downloading and building Go applications
from source in a secure manner. By default, this copies applications
into `~/go/bin/`.

You must first [install Go](https://golang.org/doc/install). After
installing Go, simply run the following command to install one of
the applications:

```sh
# Note: Be sure to replace '<app-name>'.
go install gitlab.com/stephen-fox/brkit/cmd/<app-name>@latest
# If successful, the resulting exectuable should be in "~/go/bin/".
```

## Special thanks

Several of the APIs in this library (namely the `process` sub-package) are
heavily inspired by:

- [pwntools](https://github.com/Gallopsled/pwntools)
- [pwn](https://github.com/Tnze/pwn) by Tnze
- [D3Ext](https://github.com/D3Ext) for the Go de Bruijn implementation

Lastly - a huge thank you to [Seung Kang](https://github.com/SeungKang) for
helping me maintain and improve this code base :3
