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
	OptFlagSet *flag.FlagSet
	OptLogger  *log.Logger
}

func ParseExploitArgs(config ParseExploitArgsConfig) (*process.Process, ExploitArgs) {
	return ParseExploitArgsCtx(context.Background(), config)
}

func ParseExploitArgsCtx(ctx context.Context, config ParseExploitArgsConfig) (*process.Process, ExploitArgs) {
	logger := log.Default()
	if config.OptLogger != nil {
		logger = config.OptLogger
	}

	if logger.Flags() == log.LstdFlags {
		logger.SetFlags(0)
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
	} else {
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
	}

	if !flagSet.Parsed() {
		err := flagSet.Parse(os.Args[1:])
		if err != nil {
			return nil, ExploitArgs{}, err
		}
	}

	if tempArgs.help {
		name := filepath.Base(os.Args[0])

		flagSet.Output().Write([]byte(`DESCRIPTION
  Exploit ` + name + `.

USAGE
  ` + name + ` [options] local EXE-PATH
  ` + name + ` [options] ssh SSH-SERVER-ADDRESS STD-PIPES-DIR-PATH
  ` + name + ` [options] remote ADDRESS

OPTIONS
`))

		flagSet.PrintDefaults()

		os.Exit(1)

		return nil, ExploitArgs{}, errors.New("unreachable")
	}

	if flagSet.NArg() == 0 {
		return nil, ExploitArgs{}, errors.New(`please specify one of the following:
  local EXE-PATH
  ssh SSH-SERVER-ADDRESS STD-PIPES-DIR-PATH
  remote ADDRESS`)
	}

	var proc *process.Process
	var err error
	mode := flagSet.Arg(0)

	switch mode {
	case "local":
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
