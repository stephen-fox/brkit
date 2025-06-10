# brkit

[![GoDoc][godoc-badge]][godoc]

[godoc-badge]: https://pkg.go.dev/badge/gitlab.com/stephen-fox/brkit
[godoc]: https://pkg.go.dev/gitlab.com/stephen-fox/brkit

Package brkit provides functionality for binary research and exploitation.

brkit was originally developed as a collection of small command line tools.
It eventually expanded into a library that mimics the functionality of
[Python `pwntools`][pwntools].

[pwntools]: https://docs.pwntools.com/en/stable/

## Building an exploit

brkit is broken into several sub-packages, each representing a distinct set
of functionality. Users can use as much or as little of these libraries
as they like.

#### Examples

If you would like to jump straight into a realistic example, please check
out [Seung's solution][lactf-library] to a heap-based CTF challenge.
The sections below go into more detail about the brkit functionality
used in that example.

[lactf-library]: examples/lactf-2025-library/main.go

#### Scripting functionality

The `scripting` library provides tooling to automate an exploit program's
argument parsing, debugging, and connecting to a vulnerable process. Use of
this library is completely optional and the process connection code can be
utilized independently of this library.

In particular, the `ParseExploitArgs` function automatically adds several
useful commands and optional arguments to the exploit program. The following
commands can be specified by running `<exploit-name> <command>`:

- `exec` - Execute a process using Go's `os/exec` library
- `ssh` - Connect to a remote process using SSH and named pipes. This is
  useful for interacting with a process's standard file descriptors while
  the process is connected to a debugger
- `dial` - Connect to a remote process using Go's `net` library

ParseExploitArgs parses the above commands for you and returns
a `process.Process` object representing the vulnerable program
along with a struct containing optional functionality that is
controlled by the arguments:

```go
package main

import (
    "gitlab.com/stephen-fox/brkit/process"
    "gitlab.com/stephen-fox/brkit/scripting"
)

func main() {
    proc, args := scripting.ParseExploitArgs(scripting.ParseExploitArgsConfig{
        ProcInfo: process.X86_64Info(),
    })
    defer proc.Close()
}
```

Running the above program with `-h` will produce the following output:

```console
$ go run main.go -h
DESCRIPTION
   A brkit-based exploit.

USAGE
  example -h
  example exec EXE-PATH [options]
  example ssh SSH-SERVER-ADDRESS STD-PIPES-DIR-PATH [options]
  example dial ADDRESS [options]

OPTIONS
   -V Log all process input and output
   -h Display this information
   -s int
      Pause execution at the specified stage number
   -v Enable verbose logging
```

The `args` variable from the previous example snippet provides access to
a `scripting.StageCtl` object and a `log.Logger`. The `StageCtl` allows
users to define exploit stages and to pause execution at a particular
stage using command line arguments.

Here is an example of creating stages:

```go
package main

import (
    "gitlab.com/stephen-fox/brkit/process"
    "gitlab.com/stephen-fox/brkit/scripting"
)

func main() {
    proc, args := scripting.ParseExploitArgs(scripting.ParseExploitArgsConfig{
        ProcInfo: process.X86_64Info(),
    })
    defer proc.Close()

    args.Stages.Next("Example stage")

    args.Stages.Next("Another example stage")
}
```

Here is what happens when we execute the above program:

```console
$ go run tmpexample/main.go exec cat
[+] starting Stage 1: [Example stage]
[+] executed Stage 1: [Example stage]
[+] starting Stage 2: [Another example stage]
```

When using the `scripting.ParseExploitArgs` function, the exploit's
execution can be paused by specifying the `-s <stage-number>`
argument. Alternatively, the `args.Stages.Goto` field can be
set to the stage number that you would like to pause at.
To pause at the second stage in the previous example:

```go
// ...

func main() {
    proc, args := scripting.ParseExploitArgs(scripting.ParseExploitArgsConfig{
        ProcInfo: process.X86_64Info(),
    })
    defer proc.Close()

    args.Stages.Next.Goto = 2

    // ...
}
```

... which will produce the following output:

```console
$ go run tmpexample/main.go exec cat
[+] starting Stage 1: [Example stage]
[+] executed Stage 1: [Example stage]
[+] starting Stage 2: [Another example stage]
[+] press enter to continue
```

#### Interacting with a vulnerable process

The `process.Process` type abstracts reading from and writing to
a vulnerable process. A Process object can be instantiated using
the `scripting.ParseExploitArgs` function (as shown above) or by
calling one of the constructor-like functions in the `process`
library. The `process.Info` struct conveys critical attributes
like the width of a pointer:

```go
package main

import (
    "os"
    "os/exec"

    "gitlab.com/stephen-fox/brkit/process"
)

func main() {
    // Start a process using exec (in this case, cat):
    execProc, err := process.Exec(exec.Command("cat"), process.X86_64Info())

    // Connect to a process over the network:
    dialProc, err := process.Dial("tcp", "192.168.1.2:80", process.X86_64Info())

    // Construct a process from an io.Reader and io.Writer:
    r, w, _ := os.Pipe()
    ioProc := process.FromIO(r, w, process.X86_64Info())

    // A context.Context can also be supplied using process
    // library functions ending with the "Ctx" suffix.
}
```

The Process type implements the standard library's `io.Reader`,
`io.ReaderFrom`, `io.Writer`, and `io.Closer` interfaces.
In addition, several pwntools-like methods make it easy to send
and receive data:

```go
package main

import (
    "log"
    "os/exec"

    "gitlab.com/stephen-fox/brkit/process"
)

func main() {
    // Start a process using exec (in this case, cat):
    cat, _ := process.Exec(exec.Command("cat"), process.X86_64Info())

    // Optionally log all reads and writes to the process
    // in hexdump format:
    cat.SetLoggerR(log.Default())
    cat.SetLoggerW(log.Default())

    // This writes "hello world\n":
    cat.WriteLine([]byte("hello world"))

    // Block until a "\n" is read from the process.
    // (line will contain "hello world\n")
    line, _ := cat.ReadLine()

    cat.WriteLine([]byte("some more data"))

    // Block until "data\n" is read:
    cat.ReadUntil([]byte("data\n"))

    // Hook up the Go program's stdin and stdout to the process
    // and block until a read or write fails:
    cat.Interactive()
}
```

#### Representing process memory

The `memory` library provides several abstractions for working with
a process's memory. The `Pointer` type stores pointer variables in
the endianness and bit width of the target platform. Pointer objects
are created using the `PointerMaker` type:

```go
package main

import (
    "bytes"
    "os/exec"

    "gitlab.com/stephen-fox/brkit/memory"
    "gitlab.com/stephen-fox/brkit/process"
)

func main() {
    vulnProc, _ := process.Exec(exec.Command("vuln"), process.X86_64Info())
    defer vulnProc.Close()

    // Here is how a PointerMaker for a x86 64-bit CPU
    // can be instantiated:
    pointerMaker := memory.PointerMakerForX86_64()

    // To create a pointer from an unsigned integer:
    ptr := pointerMaker.FromUint(0xd00d8badf00d)

    // The pointer is written in the endianness of the target
    // platform. The payload below this comment becomes:
    //
    // 25 70 25 70 25 70 25 70  0d f0 ad 8b 0d d0 00 00  |%p%p%p%p........|
    payload := bytes.Repeat([]byte{'%', 'p'}, 4)
    payload = append(payload, ptr.Bytes()...)

    vulnProc.WriteLine(payload)
}
```

Both PointerMaker and Pointer implement checks to catch null pointers.
These checks aim to mitigate subtle mistakes or surprises in exploit
development, such as reading a null pointer from an external process
or leaving a Pointer variable unset in the exploit program itself.

Continuing from the previous example:

```go
// ...

func main() {
    // ...

    exampleFmtPtrLeak, _ := vulnProc.ReadLine()
    exampleFmtPtrLeak = bytes.TrimSpace(exampleFmtPtrLeak)

    // By default, the PointerMaker will not allow null pointers.
    // If you know that the vulnerable program may produce null
    // pointers and you would like to allow them, then use the
    // WithNullAllowed method:
    leakedPtr, _ := pointerMaker.WithNullAllowed(
        func(p memory.PointerMaker) (memory.Pointer, error) {
            return p.FromHexBytes(exampleFmtPtrLeak, binary.BigEndian)
        },
    )

    fmt.Println("leaked:", leakedPtr.HexString())

    // The badPtr variable below is inherently invalid because
    // its default value is null, which is not allowed by default.
    //
    // Calling badPtr.Bytes() below will cause this exploit program
    // to exit because Bytes checks if the Pointer value is null.
    var badPtr memory.Pointer

    payload = bytes.Repeat([]byte{0x41}, 8)
    payload = append(payload, badPtr.Bytes()...) // <-- Crashes here.

    vulnProc.WriteLine(payload)
}
```

#### Building exploit payloads

brkit provides several libraries that can be composed together to build
exploit payloads. First, there is the `iokit` library which provides the
`PayloadBuilder` type. The PayloadBuilder implements the "builder" style
pattern to make adding to and modifying a payload easy and self-descriptive.
The type's `Build` method transforms it into a sequence of bytes which
can be passed to a `process.Process` for writing.

The PayloadBuilder's many methods allow it to interoperate with pattern
string generators such as brkit's `pattern` library, `memory.Pointer`
objects, and various Go primitive types:

```go
package main

import (
    "encoding/binary"
    "os/exec"

    "gitlab.com/stephen-fox/brkit/iokit"
    "gitlab.com/stephen-fox/brkit/memory"
    "gitlab.com/stephen-fox/brkit/pattern"
    "gitlab.com/stephen-fox/brkit/process"
)

func main() {
    vulnProc, _ := process.Exec(exec.Command("vuln"), process.X86_64Info())
    defer vulnProc.Close()

    pm := memory.PointerMakerForX86_64()

    dbPattern := pattern.DeBruijn{}

    // Here is what the payload variable becomes:
    //
    // 41 41 41 41 41 41 41 41  41 41 41 41 41 41 41 41  |AAAAAAAAAAAAAAAA|
    // 7a 65 72 6f 63 6f 6f 6c  61 61 61 61 62 61 61 61  |zerocoolaaaabaaa|
    // 63 61 61 61 64 61 61 61  01 01 02 03 0d d0 de c0  |caaadaaa........|
    // 00 00 00 00 0d f0 ad fb  ee db ea 0d 0a           |.............|
    payload := iokit.NewPayloadBuilder().
        RepeatString("A", 8*2).
        String("zerocool").
        Pattern(&dbPattern, 16).
        Bytes([]byte{0x01, 0x01, 0x02, 0x03}).
        Uint64(0xc0ded00d, binary.LittleEndian).
        Pointer(pm.FromUint(0xdeadbeefbadf00d)).
        Byte('\n').
        Build()

    vulnProc.Write(payload)
}
```

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

## Goals

The overriding goal of this project is to help solve hacking CTF challenges,
specifically the binary exploitation variety. The project tries to achieve
the following goals:

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

## Special thanks

Several of the APIs in this library (namely the `process` sub-package) are
heavily inspired by:

- [pwntools](https://github.com/Gallopsled/pwntools)
- [pwn](https://github.com/Tnze/pwn) by Tnze
- [D3Ext](https://github.com/D3Ext) for the Go de Bruijn implementation

Lastly - a huge thank you to [Seung Kang](https://github.com/SeungKang) for
helping me maintain and improve this code base :3
