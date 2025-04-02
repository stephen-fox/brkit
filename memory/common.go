package memory

import "log"

// ProcessIO abstracts the input/output of a running software process.
type ProcessIO interface {
	// WriteLine appends a newline character to the specified []byte
	// and writes it to the process' input.
	WriteLine(p []byte) error

	// ReadUntil blocks and attempts to read from the process'
	// output until the specified []byte is found. It returns the
	// data read from the process, including the specified []byte.
	ReadUntil(p []byte) ([]byte, error)

	// PointerSizeBytes returns the size of a pointer in bytes
	// for the process' platform.
	PointerSizeBytes() int
}

var (
	// DefaultExitFn is invoked by functions and methods ending in
	// the "OrExit" suffix when an error occurs.
	DefaultExitFn = func(err error) {
		log.Fatalln(err)
	}
)
