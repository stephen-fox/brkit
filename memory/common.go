package memory

import "log"

var (
	defaultExitFn = func(err error) {
		log.Fatalln(err)
	}
)
