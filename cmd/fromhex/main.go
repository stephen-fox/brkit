package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const (
	outputFormatArg = "o"
	verboseArg      = "v"
	helpArg         = "h"

	hexFormat = "hex"
	rawFormat = "raw"
	b64Format = "b64"

	appName = "fromhex"
	usage   = appName + `
Encodes a hex-encoded binary data (e.g., "\x31\xc0\x40\x89\xc3\xcd\x80") into
another encoding. The hex string can be supplied via stdin, as a single command
line argument, or as several command line arguments. The motivation behind
this tool was to help convert shellcode strings to various encodings.

The example hex string was written by Charles Stevenson (core@bokeoa.com):
http://shell-storm.org/shellcode/files/shellcode-55.php

usage:
` + appName + ` [options] [hex-string]

examples:
` + appName + ` "\x31\xc0\x40\x89\xc3\xcd\x80"
` + appName + ` "\x31\xc0" "\x40\x89" "\xc3\xcd\x80"

options:
`
)

func main() {
	outputEncoding := flag.String(
		outputFormatArg,
		rawFormat,
		fmt.Sprintf("The output encoding type (%s)", supportedIOEncodingStr()))
	verbose := flag.Bool(
		verboseArg,
		false,
		"Enable verbose logging")
	help := flag.Bool(
		helpArg,
		false,
		"Display this help page")

	flag.Parse()

	if *help {
		os.Stderr.WriteString(usage)
		flag.PrintDefaults()
		os.Exit(1)
	}

	var sourceName string
	var scannerSource io.Reader
	nArgs := flag.NArg()
	switch nArgs {
	case 0:
		sourceName = "stdin"
		scannerSource = os.Stdin
	case 1:
		sourceName = "first cli argument"
		scannerSource = strings.NewReader(flag.Arg(0))
	default:
		sourceName = "concatenated cli arguments"
		concat := bytes.NewBuffer(nil)
		for i := 0; i < nArgs; i++ {
			concat.WriteString(flag.Arg(i))
		}
		scannerSource = concat
	}

	hexASCII := bytes.NewBuffer(nil)
	scanner := bufio.NewScanner(scannerSource)
	scanner.Split(bufio.ScanBytes)
	for scanner.Scan() {
		switch scanner.Bytes()[0] {
		case 'x', '\\', '"', '\n', ' ':
			continue
		}

		if *verbose {
			log.Printf("read: 0x%x\n", scanner.Bytes())
		}

		hexASCII.Write(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to read from %s - %s", sourceName, err)
	}

	decoded, err := hex.DecodeString(hexASCII.String())
	if err != nil {
		log.Fatalf("failed to hex decode data from %s - %s", sourceName, err)
	}

	switch *outputEncoding {
	case hexFormat:
		fmt.Printf("0x%x", decoded)
	case rawFormat:
		fmt.Printf("%s", decoded)
	case b64Format:
		fmt.Print(base64.StdEncoding.EncodeToString(decoded))
	default:
		log.Fatalf("unknown output format: '%s'", *outputEncoding)
	}
}

func supportedIOEncodingStr() string {
	return fmt.Sprintf("'%s', '%s', '%s'", b64Format, hexFormat, rawFormat)
}
