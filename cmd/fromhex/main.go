// fromhex decodes hex-encoded data (e.g., "\x31\xc0\x40\x89\xc3\xcd\x80") and
// encodes the underlying binary data into another encoding.
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"gitlab.com/stephen-fox/brkit/conv"
)

const (
	outputFormatArg = "o"
	noFmtArg        = "n"
	helpArg         = "h"

	hexFormat = "hex"
	rawFormat = "raw"
	b64Format = "b64"
	goFormat  = "go"

	appName = "fromhex"
	usage   = appName + `
Decodes hex-encoded data (e.g., "\x31\xc0\x40\x89\xc3\xcd\x80") and encodes
the underlying binary data into another encoding.

The hex string can be supplied via stdin, as a single command line argument,
or as several command line arguments. Data can be provided as a C-style array
variable's contents. C comments are automatically discarded. The motivation
behind this tool was to help convert shellcode strings to various encodings.

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
		fmt.Sprintf("The output encoding type\n(%s)", supportedIOEncodingStr()))
	noFormatting := flag.Bool(
		noFmtArg,
		false,
		"Do not format output data (i.e., apply spacing or newlines)")
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

	log.SetFlags(0)

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

	decoded, err := conv.HexArrayToBytes(scannerSource)
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
	case goFormat:
		decodedLen := len(decoded)
		currentLineLen := 0
		bufioWriter := bufio.NewWriter(os.Stdout)

		_, err := bufioWriter.WriteString("[]byte{")
		if err != nil {
			log.Fatalf("fatal: %s", err)
		}

		for i, b := range decoded {
			needsComma := decodedLen > 1 && i != decodedLen-1

			bufioWriter.WriteString("0x")
			bufioWriter.WriteString(fmt.Sprintf("%x", b))
			currentLineLen += 4

			if needsComma {
				bufioWriter.WriteByte(',')
				currentLineLen++
			}

			if !*noFormatting && currentLineLen >= 62 {
				currentLineLen = 0
				bufioWriter.WriteString("\n\t")
			} else if needsComma {
				bufioWriter.WriteByte(' ')
				currentLineLen++
			}
		}

		bufioWriter.WriteString("}\n")

		err = bufioWriter.Flush()
		if err != nil {
			log.Fatalf("fatal: %s", err)
		}
	default:
		log.Fatalf("unknown output format: '%s'", *outputEncoding)
	}
}

func supportedIOEncodingStr() string {
	return fmt.Sprintf("'%s', '%s', '%s', '%s'",
		b64Format, hexFormat, goFormat, rawFormat)
}
