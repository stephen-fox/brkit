package scripting

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"gitlab.com/stephen-fox/brkit/process"
)

// Various exploit modes command strings.
const (
	execMode     = "exec"
	sshPipesMode = "ssh"
	dialMode     = "dial"
)

// ParseExploitArgsConfig configures the ParseExploitArgs function.
//
// All fields starting with "Opt" are optional (i.e., are ignored
// when set to their default values).
type ParseExploitArgsConfig struct {
	// ProcInfo is the process.Info to configure the process with.
	ProcInfo process.Info

	// OptExecArgs specifies additional arguments to pass to the
	// child process when spawning it using exec-based modes.
	//
	// This field is ignored if it is empty or a non-exec mode is used.
	OptExecArgs []string

	// OptName overrides the exploit command name.
	//
	// This field is ignored if set to an empty string.
	OptName string

	// OptDescr overrides the exploit's description, which is
	// displayed when the "-h" argument is provided.
	//
	// This field is ignored if set to an empty string.
	OptDescr string

	// OptOsArgs overrides the arguments to be parsed. Normally,
	// os.Args is used. Refer to flag.Parse for more information
	// on this behavior.
	//
	// This field is ignored if the slice is zero length.
	OptOsArgs []string

	// OptMainFlagSet overrides the default flag.FlagSet. Normally,
	// flag.CommandLine is used.
	//
	// This field is ignored if set to nil.
	OptMainFlagSet *flag.FlagSet

	// OptModMainFlagSet specifies a function that receives the main
	// flag.FlagSet prior to parsing arguments.
	//
	// This field is ignored if set to nil.
	OptModMainFlagSet func(*flag.FlagSet)

	// OptOptionsFlagSet overrides the options flag.FlagSet (flags
	// that appear after the exploit mode arguments).
	//
	// This field is ignored if set to nil.
	OptOptionsFlagSet *flag.FlagSet

	// OptModOptionsFlagSet specifies a function that receives
	// the options flag.FlagSet (flags that appear after the
	// exploit mode arguments)) prior to parsing those arguments.
	//
	// This field is ignored if set to nil.
	OptModOptionsFlagSet func(*flag.FlagSet)

	// OptExitFn overrides the function that exits the exploit
	// program (normally os.Exit).
	//
	// This field is ignored if set to nil.
	OptExitFn func(int)

	// OptModes specifies which exploit modes are enabled.
	//
	// This field is ignored if set to nil.
	OptModes *ExploitModes

	// OptLogger specifies the log.Logger to use.
	//
	// This field is ignored if set to nil.
	OptLogger *log.Logger
}

// writeHelpAndExit generates the "-h" output and writes it to the output
// io.Writer variable.
func (o ParseExploitArgsConfig) writeHelpAndExit(name string, output io.Writer) {
	if output == os.Stderr {
		stdoutInfo, err := os.Stdout.Stat()
		if err == nil && stdoutInfo.Mode()&os.ModeNamedPipe != 0 {
			output = os.Stdout
		}
	}

	description := "A brkit-based exploit."
	if o.OptDescr != "" {
		description = o.OptDescr
	}

	output.Write([]byte(`DESCRIPTION
  ` + description + `

USAGE
` + o.usage(name) + `

OPTIONS
`))

	optionsFlagSet := flag.NewFlagSet("", flag.ExitOnError)
	optionsFlagSet.SetOutput(output)

	newExploitFlags(optionsFlagSet)

	optionsFlagSet.PrintDefaults()

	if o.OptExitFn == nil {
		os.Exit(1)
	} else {
		o.OptExitFn(1)
	}
}

// usage generates the usage strings that appear in the "-h" output
// and in the missing command error.
func (o ParseExploitArgsConfig) usage(optName string) string {
	const helpUsage = "-h"
	const localUsage = execMode + " EXE-PATH [options]"
	const sshPipesUsage = sshPipesMode + " SSH-SERVER-ADDRESS STD-PIPES-DIR-PATH [options]"
	const remoteUsage = dialMode + " ADDRESS [options]"

	var prefix string
	if optName == "" {
		prefix = "  "
	} else {
		prefix = "  " + optName + " "
	}

	usage := prefix + helpUsage

	if o.OptModes == nil {
		usage += `
` + prefix + localUsage + `
` + prefix + sshPipesUsage + `
` + prefix + remoteUsage
	} else {
		usage += "\n"

		if o.OptModes.ExecEnabled {
			usage += prefix + localUsage
		}

		if o.OptModes.SshPipesEnabled {
			if usage != "" {
				usage += "\n"
			}

			usage += prefix + sshPipesUsage
		}

		if o.OptModes.DialEnabled {
			if usage != "" {
				usage += "\n"
			}

			usage += prefix + remoteUsage
		}
	}

	return usage
}

// ExploitModes configures which modes are available to the exploit.
// Enabled modes are reflected in the auto-generated help documentation.
//
// Refer to ParseExploitArgs documentation for details.
type ExploitModes struct {
	// ExecEnabled enables exec mode if set to true.
	ExecEnabled bool

	// SshPipesEnabled enables ssh pipes mode if set to true.
	SshPipesEnabled bool

	// DialEnabled enables dial mod if set to true.
	DialEnabled bool
}

// ParseExploitArgs adds useful arguments to an exploit program.
// This function works by parsing the exploit program's arguments
// using several predefined command arguments and options. Additional
// required information is specified using the ParseExploitArgsConfig
// type. The previously-named type also allows argument parsing
// behavior to be overriden using optional struct fields.
//
// The general argument structure expected by this function is:
//
//	program-name MODE POSITIONAL-ARGS [options]
//
// # Help
//
// Help documentation is auto-generated as well and can be viewed
// by executing the program with the "-h" argument. The "-h" can
// be specified before or after the mode arguments. Documentation
// is written to standard error by default. If standard output is
// a pipe, then the documentation is written to standard output
// instead. Refer to the Go Doc examples for a sample of the
// help documentation.
//
// # Modes
//
// Several modes (also known as non-flag arguments or commands) are
// made available by default. These modes are (required positional
// arguments are capitalized strings):
//
//   - exec EXE-PATH - Executes the vulnerable program using the fork+exec
//     style of execution
//   - ssh SSH-SERVER-ADDR PIPES-DIR-PATH - Connects to the vulnerable
//     process over SSH using two named pipes (or FIFOs). The SSH server
//     is connected to using the "ssh" program found in the PATH
//     environment variable. The SSH server address is the first
//     positional argument. The pipes' parent directory is specified
//     using the second positional  argument. The pipe files must be
//     named "stdin" and "stdout"
//   - dial ADDRESS - Connect to the vulnerable process over the network.
//     The address string must be of the format: HOST:PORT. For example,
//     "my-ctf.net:80" or "192.168.1.2:80"
//
// # Options
//
// Optional arguments may be specified after the mode arguments.
// These arguments appear after the mode arguments to make it
// easy to quickly modify them between executions of the exploit
// program (e.g., by avoiding constant left-arrowing / word jumping).
//
// The following arguments are parsed by default:
//
//   - -h - Writes the auto-generated help documentation to standard
//     error and exits
//   - -v - Enables the verbose logger returned in ExploitArgs
//   - -V - Enables logging of reads and writes made to and from
//     the vulnerable processs
//   - -s - Sets the stage number to pause the exploit's exection
//     at in the StageCtl returned in ExploitArgs
func ParseExploitArgs(config ParseExploitArgsConfig) (*process.Process, ExploitArgs) {
	return ParseExploitArgsCtx(context.Background(), config)
}

// ParseExploitArgsCtx parses the exploit program's arguments
// and passes the provided context.Context to the resulting
// process.Process. The context.Context is used to cancel
// the process.
//
// Refer to ParseExploitArgs for details on which argument
// strings are expected and their behavior.
func ParseExploitArgsCtx(ctx context.Context, config ParseExploitArgsConfig) (*process.Process, ExploitArgs) {
	logger := log.Default()
	if config.OptLogger != nil {
		logger = config.OptLogger
	} else {
		logger.SetFlags(0)
		logger.SetPrefix("[+] ")
	}

	proc, args, err := parseExploitArgs(ctx, logger, config)
	if err != nil {
		logger.Fatalln("fatal:", err)
	}

	return proc, args
}

// ExploitArgs contains the various values generated by parsing
// the exploit's arguments.
//
// Refer to ParseExploitArgs documentation for more details.
type ExploitArgs struct {
	// Stages is an auto-generates StageCtl. It is
	// automatically configured to stop at the stage
	// specified by the stage argument.
	Stages StageCtl

	// Verbose is a log.Logger that defaults to
	// discarding its output. It will only write
	// log messages if the verbose argument is
	// provided. The ParseExploitArgsConfig type
	// provides a mechanism to configure the
	// log.Logger on which this one is based.
	Verbose *log.Logger
}

// newExploitFlags creates a new exploitFlags for the provided flag.FlagSet.
func newExploitFlags(flagSet *flag.FlagSet) *exploitFlags {
	var tempArgs exploitFlags

	flagSet.BoolVar(
		&tempArgs.help,
		"h",
		false,
		"Display this information")

	flagSet.BoolVar(
		&tempArgs.enableProcLogging,
		"V",
		false,
		"Log all process input and output")

	flagSet.BoolVar(
		&tempArgs.verboseLogging,
		"v",
		false,
		"Enable verbose logging")

	flagSet.IntVar(
		&tempArgs.stageNumber,
		"s",
		0,
		"Pause execution at the specified stage number")

	return &tempArgs
}

// exploitFlags contains the intermediate result of parsing the exploit
// program's arguments. This struct is used to generate the ExploitArgs
// struct.
type exploitFlags struct {
	help              bool
	stageNumber       int
	verboseLogging    bool
	enableProcLogging bool
}

func (o exploitFlags) toExploitArgs(logger *log.Logger) ExploitArgs {
	args := ExploitArgs{
		Stages:  StageCtl{Goto: o.stageNumber},
		Verbose: log.New(io.Discard, "", 0),
	}

	if o.verboseLogging {
		args.Verbose = log.New(logger.Writer(), "[!] ", logger.Flags())
	}

	return args
}

// parseExploitArgs parses the arguments passed to the exploit program.
func parseExploitArgs(ctx context.Context, logger *log.Logger, config ParseExploitArgsConfig) (*process.Process, ExploitArgs, error) {
	if config.ProcInfo.PlatformBits == 0 {
		return nil, ExploitArgs{}, errors.New("the provided process.ProcInfo's PlatformBits is zero")
	}

	if config.ProcInfo.PtrSizeBytes == 0 {
		return nil, ExploitArgs{}, errors.New("the provided process.ProcInfo's PtrSizeBytes is zero")
	}

	mainFlagSet := flag.CommandLine
	if config.OptMainFlagSet != nil {
		mainFlagSet = config.OptMainFlagSet
	}

	help := mainFlagSet.Bool("h", false, "Display this information")

	if config.OptModMainFlagSet != nil {
		config.OptModMainFlagSet(mainFlagSet)
	}

	var osArgs []string
	if len(config.OptOsArgs) == 0 {
		osArgs = os.Args
	} else {
		osArgs = config.OptOsArgs
	}

	var name string
	if config.OptName == "" {
		name = filepath.Base(osArgs[0])
	} else {
		name = config.OptName
	}

	if !mainFlagSet.Parsed() {
		err := mainFlagSet.Parse(osArgs[1:])
		if err != nil {
			return nil, ExploitArgs{}, err
		}
	}

	if *help {
		config.writeHelpAndExit(name, mainFlagSet.Output())

		return nil, ExploitArgs{}, nil
	}

	if mainFlagSet.NArg() == 0 {
		return nil, ExploitArgs{}, errors.New(`please specify one of the following commands:
` + config.usage(""))
	}

	var startProcFn func() (*process.Process, error)
	var remainingArgs []string
	mode := mainFlagSet.Arg(0)

	switch mode {
	case execMode:
		if config.OptModes != nil && !config.OptModes.ExecEnabled {
			return nil, ExploitArgs{}, errors.New(mode + " mode is explicitly disabled")
		}

		exePath := mainFlagSet.Arg(1)
		if exePath == "" {
			return nil, ExploitArgs{}, errors.New("please specify the local executable path as the last argument")
		}

		startProcFn = func() (*process.Process, error) {
			proc, err := process.Exec(
				exec.CommandContext(ctx, exePath, config.OptExecArgs...),
				config.ProcInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to exec process - %w", err)
			}

			return proc, nil
		}

		remainingArgs = mainFlagSet.Args()[1:]
	case sshPipesMode:
		if config.OptModes != nil && !config.OptModes.SshPipesEnabled {
			return nil, ExploitArgs{}, errors.New(mode + "mode is explicitly disabled")
		}

		addr := mainFlagSet.Arg(1)
		if addr == "" {
			return nil, ExploitArgs{}, errors.New("please specify the ssh server address to connect to as the first non-flag argument")
		}

		pipesDirPath := mainFlagSet.Arg(2)
		if pipesDirPath == "" {
			return nil, ExploitArgs{}, errors.New("please specify the directory path containing the stdin and stdout pipe files as the second non-flag argument")
		}

		startProcFn = func() (*process.Process, error) {
			sshInput := process.ExecOrExit(exec.CommandContext(
				ctx,
				"ssh", addr,
				"--",
				"cat", ">", pipesDirPath+"/stdin"),
				process.X86_64Info())

			sshOutput := process.ExecOrExit(exec.CommandContext(
				ctx,
				"ssh", addr,
				"--",
				"cat", pipesDirPath+"/stdout"),
				process.X86_64Info())

			return process.FromIOCtx(ctx, sshInput, sshOutput, config.ProcInfo), nil
		}

		remainingArgs = mainFlagSet.Args()[3:]
	case dialMode:
		if config.OptModes != nil && !config.OptModes.DialEnabled {
			return nil, ExploitArgs{}, errors.New(mode + " mode is explicitly disabled")
		}

		addr := mainFlagSet.Arg(1)
		if addr == "" {
			return nil, ExploitArgs{}, errors.New("please specify the remote address as the last non-flag argument")
		}

		startProcFn = func() (*process.Process, error) {
			proc, err := process.DialCtx(ctx, "tcp", addr, config.ProcInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to dial target - %w", err)
			}

			return proc, nil
		}

		remainingArgs = mainFlagSet.Args()[2:]
	default:
		return nil, ExploitArgs{}, fmt.Errorf("unknown mode: %q", mode)
	}

	if len(remainingArgs) == 0 {
		proc, err := startProcFn()
		if err != nil {
			return nil, ExploitArgs{}, err
		}

		var tmp exploitFlags

		return proc, tmp.toExploitArgs(logger), nil
	}

	var optionsFlagSet *flag.FlagSet
	if config.OptOptionsFlagSet == nil {
		optionsFlagSet = flag.NewFlagSet("", flag.ExitOnError)
	} else {
		optionsFlagSet = config.OptOptionsFlagSet
	}

	if config.OptModOptionsFlagSet != nil {
		config.OptModOptionsFlagSet(optionsFlagSet)
	}

	tempArgs := newExploitFlags(optionsFlagSet)

	optionsFlagSet.Parse(remainingArgs)

	if tempArgs.help {
		config.writeHelpAndExit(name, optionsFlagSet.Output())

		return nil, ExploitArgs{}, nil
	}

	proc, err := startProcFn()
	if err != nil {
		return nil, ExploitArgs{}, err
	}

	if tempArgs.enableProcLogging {
		proc.SetLoggerR(log.New(logger.Writer(), "[<] ", logger.Flags()))
		proc.SetLoggerW(log.New(logger.Writer(), "[>] ", logger.Flags()))
	}

	return proc, tempArgs.toExploitArgs(logger), nil
}
