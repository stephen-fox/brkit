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

const (
	execMode     = "exec"
	sshPipesMode = "ssh"
	dialMode     = "dial"
)

type ParseExploitArgsConfig struct {
	ProcInfo             process.Info
	OptExecArgs          []string
	OptName              string
	OptDescr             string
	OptOsArgs            []string
	OptMainFlagSet       *flag.FlagSet
	OptModMainFlagSet    func(*flag.FlagSet)
	OptOptionsFlagSet    *flag.FlagSet
	OptModOptionsFlagSet func(*flag.FlagSet)
	OptExitFn            func(int)
	OptModes             *ExploitModes
	OptLogger            *log.Logger
}

func (o ParseExploitArgsConfig) writeHelpAndExit(name string, output io.Writer) {
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

type ExploitModes struct {
	ExecEnabled     bool
	SshPipesEnabled bool
	DialEnabled     bool
}

func ParseExploitArgs(config ParseExploitArgsConfig) (*process.Process, ExploitArgs) {
	return ParseExploitArgsCtx(context.Background(), config)
}

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

type ExploitArgs struct {
	Stages  StageCtl
	Verbose *log.Logger
}

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

func parseExploitArgs(ctx context.Context, logger *log.Logger, config ParseExploitArgsConfig) (*process.Process, ExploitArgs, error) {
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
