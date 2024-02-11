package process

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"os/exec"
)

func ExampleExec() {
	cmd := exec.Command("cat")

	proc, err := Exec(cmd, X86_64Info())
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
	proc, err := Dial("tcp4", "192.168.1.2:8080", X86_64Info())
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

	proc := FromNetConn(c, X86_64Info())
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleFromNetConn_FromTLSConnection() {
	tlsConn, err := tls.Dial("tcp", "192.168.1.2", &tls.Config{
		ServerName: "example.com",
	})
	if err != nil {
		log.Fatalln(err)
	}

	proc := FromNetConn(tlsConn, X86_64Info())
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleFromNamedPipes() {
	proc, err := FromNamedPipes("/path/to/input.fifo", "/path/to/output.fifo", X86_64Info())
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

	sshInput := ExecOrExit(exec.Command("ssh", sshHost, "--", "cat", ">", inputPipePath), X86_64Info())
	sshOutput := ExecOrExit(exec.Command("ssh", sshHost, "--", "cat", outputPipePath), X86_64Info())

	proc := FromIO(sshInput, sshOutput, X86_64Info())
	defer proc.Close()

	proc.Write([]byte("hello world"))
}

func ExampleProcess_Close() {
	proc, err := Exec(exec.Command("cat"), X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()
}

func ExampleProcess_Read() {
	proc, err := Exec(exec.Command("cat", "/etc/passwd"), X86_64Info())
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

func ExampleProcess_WriteLine() {
	proc, err := Exec(exec.Command("cat"), X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	proc.WriteLine([]byte("hello world"))
}

func ExampleProcess_Write() {
	proc, err := Exec(exec.Command("cat"), X86_64Info())
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Close()

	proc.Write([]byte("hello world\n"))
}

func ExampleProcess_Interactive() {
	proc, err := Exec(exec.Command("cat"), X86_64Info())
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
