package process_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"os/exec"

	"gitlab.com/stephen-fox/brkit/process"
)

func ExampleExec() {
	cmd := exec.Command("cat")

	proc, err := process.Exec(cmd, process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	err = proc.WriteLine([]byte("hello world"))
	if err != nil {
		log.Fatalln(err)
	}

	line, err := proc.ReadLine()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("%s", line)
}

func ExampleDial() {
	proc, err := process.Dial("tcp4", "192.168.1.2:8080", process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleDialCtx() {
	ctx := context.Background()

	proc, err := process.DialCtx(ctx, "tcp4", "192.168.1.2:8080", process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleFromNetConn() {
	c, err := net.Dial("tcp", "192.168.1.2:8080")
	if err != nil {
		log.Fatalln(err)
	}

	proc := process.FromNetConn(c, process.X86_64Info())
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleFromNetConnCtx() {
	c, err := net.Dial("tcp", "192.168.1.2:8080")
	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()

	proc := process.FromNetConnCtx(ctx, c, process.X86_64Info())
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleFromNetConn_from_tls_connection() {
	tlsConn, err := tls.Dial("tcp", "192.168.1.2", &tls.Config{
		ServerName: "example.com",
	})
	if err != nil {
		log.Fatalln(err)
	}

	proc := process.FromNetConn(tlsConn, process.X86_64Info())
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleFromNamedPipes() {
	proc, err := process.FromNamedPipes(
		"/path/to/input.fifo",
		"/path/to/output.fifo",
		process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	proc.Write([]byte("hello world"))
}

func ExampleFromNamedPipesCtx() {
	ctx := context.Background()

	proc, err := process.FromNamedPipesCtx(
		ctx,
		"/path/to/input.fifo",
		"/path/to/output.fifo",
		process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	proc.Write([]byte("hello world"))
}

func ExampleFromIO() {
	flag.Parse()

	sshHost := flag.Arg(1)
	inputPipePath := flag.Arg(2)
	outputPipePath := flag.Arg(3)

	sshInput := process.ExecOrExit(
		exec.Command("ssh", sshHost, "--", "cat", ">", inputPipePath),
		process.X86_64Info())

	sshOutput := process.ExecOrExit(
		exec.Command("ssh", sshHost, "--", "cat", outputPipePath),
		process.X86_64Info())

	proc := process.FromIO(sshInput, sshOutput, process.X86_64Info())
	defer proc.Close()

	proc.Write([]byte("hello world"))
}

func ExampleFromIOCtx() {
	flag.Parse()

	sshHost := flag.Arg(1)
	inputPipePath := flag.Arg(2)
	outputPipePath := flag.Arg(3)

	sshInput := process.ExecOrExit(
		exec.Command("ssh", sshHost, "--", "cat", ">", inputPipePath),
		process.X86_64Info())

	sshOutput := process.ExecOrExit(
		exec.Command("ssh", sshHost, "--", "cat", outputPipePath),
		process.X86_64Info())

	ctx := context.Background()

	proc := process.FromIOCtx(ctx, sshInput, sshOutput, process.X86_64Info())
	defer proc.Close()

	proc.Write([]byte("hello world"))
}

func ExampleProcess_Close() {
	proc, err := process.Exec(exec.Command("cat"), process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()
}

func ExampleProcess_Read() {
	proc, err := process.Exec(exec.Command("cat", "/etc/passwd"), process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	b := make([]byte, 1024)

	n, err := proc.Read(b)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("read %d bytes: %s", n, b[0:n])
}

func ExampleProcess_ReadFrom() {
	proc, err := process.Exec(exec.Command("cat"), process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	_, err = proc.ReadFrom(bytes.NewReader([]byte("hello world")))
	if err != nil {
		log.Fatalln(err)
	}
}

func ExampleProcess_WriteLine() {
	proc, err := process.Exec(exec.Command("cat"), process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleProcess_Write() {
	proc, err := process.Exec(exec.Command("cat"), process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	proc.Write([]byte("hello world\n"))
}

func ExampleProcess_Interactive() {
	proc, err := process.Exec(exec.Command("cat"), process.X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	// Anything typed into stdin will be written to the process' stdin.
	err = proc.Interactive()
	if err != nil {
		log.Fatalln(err)
	}
}
