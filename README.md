# brkit

[![GoDoc][godoc-badge]][godoc]

[godoc-badge]: https://pkg.go.dev/gitlab.com/stephen-fox/brkit?status.svg
[godoc]: https://pkg.go.dev/gitlab.com/stephen-fox/brkit

Package brkit provides functionality for binary research.

## Use case
This library was originally developed as a collection of small command line
utilities. It eventually expanded into a library that mimics the functionality
of Python `pwntools`. I developed this library to further my understanding of
binary-level vulnerability research and exploit development. The overriding
goal of this project is to help with solving hacking CTF challenges. The API is
open-minded in the sense it could be used (responsibly) for non-CTF work.

## APIs
brkit is broken into several subpackages, each representing a distinct set
of functionality. The following subsections will outline their usage.
Please refer to the GoDoc documentation for detailed explanations
and usage examples.

#### `memory`
Package memory provides functionality for reading and writing memory.

This library is useful for constructing memory leaks and writes, as well as
tracking memory addresses and pointers programmatically. The `AddressTable`
struct provides a small API for organizing memory offsets in different contexts.
For example, it can be used to track glibc symbol offsets for
different machines:

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

The `ProcessIO` interface type fulfills a similar role as the `io.ReadWriter`.
It abstracts a process' input/output and other important attributes. Normally,
this is provided by the `process.Process` type - but can be implemented
different as desired.

This library also provides functions for automating the creation of format
string attacks, primarily through the direct parameter access (DPA) feature.
The `SetupFormatStringLeakViaDPA` function accomplishes this by first leaking
an oracle string within a newly created format string. This oracle is replaced
with an address provided by the caller. All of this is done before returning
to the caller:

```go
func ExampleSetupFormatStringLeakViaDPA() {
	leaker, err := memory.SetupFormatStringLeakViaDPA(DPAFormatStringConfig{
		ProcessIO:    &fakeProcessIO{},
		MaxNumParams: 200,
	})
	if err != nil {
		log.Fatalln(err)
	}

	pm := memory.PointerMakerForX86_64()

	log.Printf("read: 0x%x", leaker.MemoryAtOrExit(pm.FromUint(0x00000000deadbeef)))
}
```

Creation of format string attacks that can write memory is handled in a similar
fashion. The `SetupDPAFormatStringWriter` function leaks the DPA argument
number of an oracle string, and then replaces it with a caller-supplied address.
By abusing certain format specifiers (which is discussed in the GoDoc), callers
can effectively overwrite the lower four, two, or single bytes:

```go
func ExampleDPAFormatStringWriter_WriteLowerFourBytesAt() {
	writer, err := memory.SetupDPAFormatStringWriter(DPAFormatStringWriterConfig{
		MaxWrite:  999,
		DPAConfig: DPAFormatStringConfig{
			ProcessIO:    &fakeProcessIO{},
			MaxNumParams: 200,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	pm := memory.PointerMakerForX86_32()

	// Set the lower four bytes to 1000 (0x03E8).
	err = writer.WriteLowerFourBytesAt(1000, pm.FromUint(0xdeadbeef))
	if err != nil {
		log.Fatalln(err)
	}
}
```

#### `process`
Package process provides functionality for working with running
software processes.

TODO

## Scriptability
TODO

## Command line utilities
Several command line utilities are included to aid in binary research efforts.

#### `fromhex`
Encodes a hex-encoded binary data (e.g., "\x31\xc0\x40\x89\xc3\xcd\x80") into
another encoding.

#### `pattern`
Find repeating string patterns in strings. Useful for finding where an input
string begins to overwrite program state (e.g., stack-based buffer overflows).

#### `stringer`
An application for working with strings of bytes, and manipulating data.

## Special thanks
Several of the APIs in this library (namely the `process` subpackage) are
heavily inspired by:

- [pwntools](https://github.com/Gallopsled/pwntools)
- [pwn](https://github.com/Tnze/pwn) by Tnze

Thank you!
