package memory

import "log"

// ProcessIO abstracts the input/output of a running software process.
type ProcessIO interface {
	// WriteLine writes the specified []byte to the process
	// and appends a new line.
	WriteLine(p []byte) error

	// ReadUntil ReadUntil blocks and attempts to read from the
	// process' output until the specified []byte is found,
	// returning the data read, including the specified []byte.
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
