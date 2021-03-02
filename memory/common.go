package memory

import "log"

type ProcessIO interface {
	WriteLine(p []byte) error
	ReadUntil(p []byte) ([]byte, error)
}

var (
	defaultExitFn = func(err error) {
		log.Fatalln(err)
	}
)
