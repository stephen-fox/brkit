package scripting

import (
	"fmt"
	"log"
)

type StageCtl struct {
	Goto     int
	Logger   *log.Logger
	num      int
	prevDesc string
}

func (o *StageCtl) Next(description ...string) {
	logger := log.Default()
	if o.Logger != nil {
		logger = o.Logger
	}

	if o.num > 0 {
		logger.Printf("executed Stage %d: [%s] ", o.num, o.prevDesc)
	}

	o.num++
	if len(description) > 0 {
		o.prevDesc = description[0]
	}

	logger.Printf("starting Stage %d: %s ", o.num, description)

	if o.Goto == 0 || o.Goto > o.num {
		return
	}

	logger.Printf("press enter to continue")
	fmt.Scanln()
}
