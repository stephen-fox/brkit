package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const (
	outputFormatArg  = "o"
	prefixStrArg     = "prefix"
	littleArg        = "little"
	noNewLineArg     = "n"
	lenArg           = "len"
	helpArg          = "h"
	createStringMode = "new"
	appendStringMode = "append"

	hexOutputFormat = "hex"
	rawOutputFormat = "raw"

	appName = "app"
	usage   = appName + `
An application for working with bytes, and manipulating PoC exploit data.

usage:
  ` + appName + ` [mode] [options] value

examples:
  ` + appName + ` 0x080491e2
  ` + appName + ` -` + littleArg + ` 0x080491e2
  ` + appName + ` -` + lenArg + ` 40 ` + createStringMode + ` 0x080491e2

modes:
  ` + createStringMode + `, ` + appendStringMode + `

options:
`
)

func main() {
	outputFormat := flag.String(
		outputFormatArg,
		"hex",
		fmt.Sprintf("The output format (%s, %s)", hexOutputFormat, rawOutputFormat))
	createStringLen := flag.Int(
		lenArg,
		1,
		fmt.Sprintf("The length of a string created with '%s'", createStringMode))
	prefixString := flag.String(
		prefixStrArg,
		"",
		fmt.Sprintf("An existing string to prefix '%s' with (can be from stdin instead)", appendStringMode))
	toWrongEndian := flag.Bool(
		littleArg,
		false,
		"Convert the input value to little (wrong) endian")
	noNewLine := flag.Bool(
		noNewLineArg,
		false,
		"Do not append new line character to output")
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

	var mode string
	var value []byte
	switch flag.NArg() {
	case 0:
		log.Fatalf("please specify at least one argument")
	case 1:
		value = []byte(flag.Arg(0))
	case 2:
		mode = flag.Arg(0)
		value = []byte(flag.Arg(1))
	default:
		log.Fatalf("too many non-option arguments were specified")
	}

	if len(value) > 0 {
		value = bytes.TrimPrefix(value, []byte("0x"))
		temp := make([]byte, hex.DecodedLen(len(value)))
		_, err := hex.Decode(temp, value)
		if err != nil {
			log.Fatalf("failed to hex decode fragment string - %s", err)
		}
		value = temp
	} else {
		log.Println("reading value from stdin...")
		buff := bytes.NewBuffer(nil)
		_, err := io.Copy(buff, os.Stdin)
		if err != nil {
			log.Fatalf("failed to read data from stdin - %s", err)
		}
		value = buff.Bytes()
		log.Printf("read '0x%x'", value)
	}

	switch mode {
	case createStringMode:
		fmt.Print(strings.Repeat("A", *createStringLen))
	case appendStringMode:
		if len(*prefixString) == 0 {
			_, err := io.Copy(os.Stdout, os.Stdin)
			if err != nil {
				log.Fatalf("failed to copy from stdin - %s", err)
			}
		} else {
			fmt.Print(*prefixString)
		}
	}

	if *toWrongEndian {
		decodedFragmentLen := len(value)
		temp := make([]byte, decodedFragmentLen)
		for i := range value {
			temp[decodedFragmentLen-1-i] = value[i]
		}
		value = temp
	}

	switch *outputFormat {
	case "hex":
		fmt.Printf("0x%x", value)
	case "raw":
		fmt.Printf("%s", value)
	default:
		log.Fatalf("unknown output format: '%s'", outputFormat)
	}

	if !*noNewLine {
		fmt.Println()
	}
}
