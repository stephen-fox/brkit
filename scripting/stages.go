package scripting

import (
	"fmt"
	"log"
)

// StageCtl allows users to implement stages in their exploit which
// are reflected as output to a log.Logger (log.Default by default).
type StageCtl struct {
	// Goto optionally specifies a stage number to pause
	// execution at until a newline is received on stdin.
	// For example, setting this field to 2 means that
	// the second stage will block until a newline
	// is provided.
	//
	// The stage number is incremented by one each time
	// Next is called.
	Goto int

	// Logger may be specified to override the logging
	// behavior. By default, StageCtl uses the logger
	// returned by log.Default.
	Logger *log.Logger

	num      int
	prevDesc string
}

// Next increments the stage counter by one and writes a log
// message containing an optional description.
func (o *StageCtl) Next(description ...string) {
	logger := log.Default()
	if o.Logger != nil {
		logger = o.Logger
	}

	if o.num > 0 {
		logger.Printf("executed Stage %d: [%s]",
			o.num, o.prevDesc)
	}

	o.num++
	if len(description) > 0 {
		o.prevDesc = description[0]
	}

	logger.Printf("starting Stage %d: %s",
		o.num, description)

	if o.Goto == 0 || o.Goto > o.num {
		return
	}

	logger.Printf("press enter to continue")
	fmt.Scanln()
}
