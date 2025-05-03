package scripting

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"gitlab.com/stephen-fox/brkit/process"
)

type ParseExploitArgsConfig struct {
	ProcInfo   process.Info
	OptFlagSet *flag.FlagSet
	OptLogger  *log.Logger
}

func ParseExploitArgs(config ParseExploitArgsConfig) *process.Process {
	logger := log.Default()
	if config.OptLogger != nil {
		logger = config.OptLogger
	}

	if logger.Flags() == log.LstdFlags {
		logger.SetFlags(0)
	}

	proc, err := parseExploitArgs(config)
	if err != nil {
		logger.Fatalln("fatal:", err)
	}

	return proc
}

const usage = `please specify one of the following:
  local EXE-PATH
  ssh SSH-SERVER-ADDRESS STD-PIPES-DIR-PATH
  remote ADDRESS`

type ExploitArgs struct {
	Process     *process.Process
	Help        bool
	StageNumber int
	Verbose     *log.Logger
}

type tempExploitArgs struct {
	Help        bool
	StageNumber int
	Verbose     bool
}

func parseExploitArgs(config ParseExploitArgsConfig) (*process.Process, error) {
	var exploitArgs tempExploitArgs

	flagSet := flag.CommandLine
	if config.OptFlagSet != nil {
		flagSet = config.OptFlagSet
	} else {
		flagSet.BoolVar(&exploitArgs.Help, "h", false, "display this information")
	}

	if !flagSet.Parsed() {
		err := flagSet.Parse(os.Args[1:])
		if err != nil {
			return nil, err
		}
	}

	if flagSet.NArg() == 0 {
		return nil, errors.New(usage)
	}

	mode := flagSet.Arg(0)

	switch mode {
	case "local":
		exePath := flagSet.Arg(1)

		if exePath == "" {
			return nil, errors.New("please specify the local executable path as the last argument")
		}

		var additionalArgs []string
		if flagSet.NArg() > 2 {
			additionalArgs = flagSet.Args()[2:]
		}

		return process.ExecOrExit(
			exec.Command(exePath, additionalArgs...),
			config.ProcInfo), nil
	case "ssh":
		addr := flagSet.Arg(1)
		if addr == "" {
			return nil, errors.New("please specify the ssh server address to connect to as the first non-flag argument")
		}

		pipesDirPath := flagSet.Arg(2)
		if pipesDirPath == "" {
			return nil, errors.New("please specify the directory path containing the stdin and stdout pipe files as the second non-flag argument")
		}

		sshInput := process.ExecOrExit(exec.Command(
			"ssh", addr,
			"--",
			"cat", ">", pipesDirPath+"/stdin"),
			process.X86_64Info())

		sshOutput := process.ExecOrExit(exec.Command(
			"ssh", addr,
			"--",
			"cat", pipesDirPath+"/stdout"),
			process.X86_64Info())

		return process.FromIO(sshInput, sshOutput, process.X86_64Info()), nil
	case "remote":
		addr := flagSet.Arg(1)
		if addr == "" {
			return nil, errors.New("please specify the remote address as the last non-flag argument")
		}

		return process.DialOrExit("tcp", addr, config.ProcInfo), nil
	default:
		return nil, fmt.Errorf("unknown mode: %q - %s", mode, usage)
	}
}
