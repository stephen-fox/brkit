package scripting_test

import (
	"log"
	"os"

	"gitlab.com/stephen-fox/brkit/scripting"
)

func ExampleStageCtl() {
	optionalLogger := log.New(os.Stdout, "[+] ", 0)

	stages := scripting.StageCtl{
		Logger: optionalLogger,
	}

	// Here is an example of a "named" stage:
	stages.Next("hello from stage one")

	// ... and an "unnamed" stage:
	stages.Next("")

	// Output:
	// [+] starting Stage 1: [hello from stage one]
	// [+] executed Stage 1: [hello from stage one]
	// [+] starting Stage 2: []
}
