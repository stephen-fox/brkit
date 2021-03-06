package process

import (
	"crypto/tls"
	"log"
	"net"
	"os/exec"
)

func ExampleExec() {
	cmd := exec.Command("cat")

	proc, err := Exec(cmd)
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Cleanup()

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
	proc, err := Dial("tcp4", "192.168.1.2:8080")
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Cleanup()

	proc.WriteLine([]byte("hello world"))
}

func ExampleFromNetConn() {
	c, err := net.Dial("tcp", "192.168.1.2:8080")
	if err != nil {
		log.Fatalln(err)
	}

	proc := FromNetConn(c)

	proc.WriteLine([]byte("hello world"))
}

func ExampleFromNetConn_FromTLSConnection() {
	tlsConn, err := tls.Dial("tcp", "192.168.1.2", &tls.Config{
		ServerName: "example.com",
	})
	if err != nil {
		log.Fatalln(err)
	}

	proc := FromNetConn(tlsConn)

	proc.WriteLine([]byte("hello world"))
}

func ExampleProcess_Cleanup() {
	proc, err := Exec(exec.Command("cat"))
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Cleanup()
}

func ExampleProcess_WriteLine() {
	proc, err := Exec(exec.Command("cat"))
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Cleanup()

	proc.WriteLine([]byte("hello world"))
}

func ExampleProcess_Write() {
	proc, err := Exec(exec.Command("cat"))
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Cleanup()

	proc.WriteLine([]byte("hello world\n"))
}

func ExampleProcess_Interactive() {
	proc, err := Exec(exec.Command("cat"))
	if err != nil {
		log.Fatalln(err)
	}
	defer proc.Cleanup()

	// Anything typed into stdin will be written to the process' stdin.
	err = proc.Interactive()
	if err != nil {
		log.Fatalln(err)
	}
}
