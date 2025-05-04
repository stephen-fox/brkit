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

type ParseExploitArgsConfig struct {
	ProcInfo   process.Info
	OptName    string
	OptDescr   string
	OptOsArgs  []string
	OptModFlag func(*flag.FlagSet)
	OptFlagSet *flag.FlagSet
	OptExitFn  func(int)
	OptModes   *ExploitModes
	OptLogger  *log.Logger
}

func (config ParseExploitArgsConfig) usage(name string) string {
	const localUsage = "[options] local EXE-PATH"
	const sshPipesUsage = "[options] ssh SSH-SERVER-ADDRESS STD-PIPES-DIR-PATH"
	const remoteUsage = "[options] remote ADDRESS"

	var prefix string
	if name == "" {
		prefix = "  "
	} else {
		prefix = "  " + name + " "
	}

	var usage string

	if config.OptModes == nil {
		usage = prefix + localUsage + `
` + prefix + sshPipesUsage + `
` + prefix + remoteUsage
	} else {
		if config.OptModes.LocalEnabled {
			usage += prefix + localUsage
		}

		if config.OptModes.SshPipesEnabled {
			if usage != "" {
				usage += "\n"
			}

			usage += prefix + sshPipesUsage
		}

		if config.OptModes.RemoteEnabled {
			if usage != "" {
				usage += "\n"
			}

			usage += prefix + remoteUsage
		}
	}

	return usage
}

type ExploitModes struct {
	LocalEnabled    bool
	SshPipesEnabled bool
	RemoteEnabled   bool
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

type tempExploitArgs struct {
	help              bool
	stageNumber       int
	verboseLogging    bool
	enableProcLogging bool
}

func (o tempExploitArgs) toExploitArgs(logger *log.Logger) ExploitArgs {
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
	var tempArgs tempExploitArgs

	flagSet := flag.CommandLine
	if config.OptFlagSet != nil {
		flagSet = config.OptFlagSet
	}

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

	if config.OptModFlag != nil {
		config.OptModFlag(flagSet)
	}

	var osArgs []string
	if config.OptOsArgs == nil {
		osArgs = os.Args
	} else {
		osArgs = config.OptOsArgs
	}

	if !flagSet.Parsed() {
		err := flagSet.Parse(osArgs[1:])
		if err != nil {
			return nil, ExploitArgs{}, err
		}
	}

	if tempArgs.help {
		var name string
		if config.OptName == "" {
			name = filepath.Base(osArgs[0])
		} else {
			name = config.OptName
		}

		description := "A brkit-based exploit."
		if config.OptDescr != "" {
			description = config.OptDescr
		}

		flagSet.Output().Write([]byte(`DESCRIPTION
  ` + description + `

USAGE
` + config.usage(name) + `

OPTIONS
`))

		flagSet.PrintDefaults()

		if config.OptExitFn == nil {
			os.Exit(1)
		} else {
			config.OptExitFn(1)
		}

		return nil, ExploitArgs{}, nil
	}

	if flagSet.NArg() == 0 {
		return nil, ExploitArgs{}, errors.New(`please specify one of the following commands:
` + config.usage(""))
	}

	var proc *process.Process
	var err error
	mode := flagSet.Arg(0)

	switch mode {
	case "local":
		if config.OptModes != nil && !config.OptModes.LocalEnabled {
			return nil, ExploitArgs{}, errors.New("local mode is explicitly disabled")
		}

		exePath := flagSet.Arg(1)

		if exePath == "" {
			return nil, ExploitArgs{}, errors.New("please specify the local executable path as the last argument")
		}

		var additionalArgs []string
		if flagSet.NArg() > 2 {
			additionalArgs = flagSet.Args()[2:]
		}

		proc, err = process.Exec(
			exec.CommandContext(ctx, exePath, additionalArgs...),
			config.ProcInfo)
		if err != nil {
			return nil, ExploitArgs{}, fmt.Errorf("failed to exec start process - %w", err)
		}
	case "ssh":
		if config.OptModes != nil && !config.OptModes.SshPipesEnabled {
			return nil, ExploitArgs{}, errors.New("ssh pipes mode is explicitly disabled")
		}

		addr := flagSet.Arg(1)
		if addr == "" {
			return nil, ExploitArgs{}, errors.New("please specify the ssh server address to connect to as the first non-flag argument")
		}

		pipesDirPath := flagSet.Arg(2)
		if pipesDirPath == "" {
			return nil, ExploitArgs{}, errors.New("please specify the directory path containing the stdin and stdout pipe files as the second non-flag argument")
		}

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

		proc = process.FromIOCtx(ctx, sshInput, sshOutput, config.ProcInfo)
	case "remote":
		if config.OptModes != nil && !config.OptModes.RemoteEnabled {
			return nil, ExploitArgs{}, errors.New("remote mode is explicitly disabled")
		}

		addr := flagSet.Arg(1)
		if addr == "" {
			return nil, ExploitArgs{}, errors.New("please specify the remote address as the last non-flag argument")
		}

		proc, err = process.DialCtx(ctx, "tcp", addr, config.ProcInfo)
		if err != nil {
			return nil, ExploitArgs{}, fmt.Errorf("failed to dial target - %w", err)
		}
	default:
		return nil, ExploitArgs{}, fmt.Errorf("unknown mode: %q", mode)
	}

	if tempArgs.enableProcLogging {
		proc.SetLoggerR(log.New(logger.Writer(), "[<] ", logger.Flags()))
		proc.SetLoggerW(log.New(logger.Writer(), "[>] ", logger.Flags()))
	}

	return proc, tempArgs.toExploitArgs(logger), nil
}
