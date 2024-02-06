package pattern

import (
	"log"
)

var (
	// DefaultExitFn is invoked by functions and methods ending in
	// the "OrExit" suffix when an error occurs.
	DefaultExitFn = func(err error) {
		log.Fatalln(err)
	}
)
