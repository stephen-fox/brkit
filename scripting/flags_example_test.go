package scripting_test

import (
	"flag"
	"log"
	"os"

	"gitlab.com/stephen-fox/brkit/process"
	"gitlab.com/stephen-fox/brkit/scripting"
)

// Note: The formatting of flag.PrintDefaults uses tabs after
// "-<arg>" *unless* it is a datatype. I.e.,
//
//	-V<tab>Log all process input and output
//	-s<space>int
func ExampleParseExploitArgs_help_output() {
	scripting.ParseExploitArgs(scripting.ParseExploitArgsConfig{
		ProcInfo: process.X86_64Info(),

		// Note: The following fields are only required for
		// this example code.
		OptOsArgs:      []string{"example", "-h"},
		OptMainFlagSet: flag.NewFlagSet("example", flag.ExitOnError),
		OptModMainFlagSet: func(flagSet *flag.FlagSet) {
			flagSet.SetOutput(os.Stdout)
		},
		OptExitFn: func(int) {},
	})

	// Output:
	// DESCRIPTION
	//   A brkit-based exploit.
	//
	// USAGE
	//   example -h
	//   example local EXE-PATH [options]
	//   example ssh SSH-SERVER-ADDRESS STD-PIPES-DIR-PATH [options]
	//   example remote ADDRESS [options]
	//
	// OPTIONS
	//   -V	Log all process input and output
	//   -h	Display this information
	//   -s int
	//     	Pause execution at the specified stage number
	//   -v	Enable verbose logging
}

func ExampleParseExploitArgs_verbose_logging() {
	_, args := scripting.ParseExploitArgs(scripting.ParseExploitArgsConfig{
		ProcInfo: process.X86_64Info(),
	})

	args.Verbose.Println("this logger will discard writes unless you specify the verbose logging flag")
}

func ExampleParseExploitArgs_stages() {
	_, args := scripting.ParseExploitArgs(scripting.ParseExploitArgsConfig{
		ProcInfo: process.X86_64Info(),
	})

	if args.Stages.Goto > 0 {
		log.Println("the StageCtl field will be set to the stage argument")
	}
}
