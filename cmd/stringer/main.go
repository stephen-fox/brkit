package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	outputFormatArg  = "o"
	noNewLineArg     = "n"
	helpArg          = "h"

	inputFormatArg  = "i"
	wrongEndianArg  = "wendian"
	repeatStringArg = "repeat"

	hexFormat = "hex"
	rawFormat = "raw"
	b64Format = "b64"

	appName = "app"
	usage   = appName + `
An application for working with strings of bytes, and manipulating data.

usage:
  ` + appName + ` [main options] [string [string options]...]

examples:
  ` + appName + ` 0x080491e2
  ` + appName + ` 0x080491e2 -` + wrongEndianArg + `
  ` + appName + ` A -` + repeatStringArg + ` 184 -` + inputFormatArg + ` ` + rawFormat + ` 0x080491e2 -` + wrongEndianArg + `

main options:
`
)

func main() {
	outputEncoding := flag.String(
		outputFormatArg,
		hexFormat,
		fmt.Sprintf("The output encoding type (%s)", supportedIOEncodingStr()))
	noNewLine := flag.Bool(
		noNewLineArg,
		false,
		"Do not append new line character to output")
	help := flag.Bool(
		helpArg,
		false,
		"Display this help page")
	flag.Parse()

	// TODO: Read from stdin support?
	if *help {
		os.Stderr.WriteString(usage)
		flag.PrintDefaults()
		os.Stderr.WriteString("\nstring options:\n")
		newStringFlagsConfig().set.PrintDefaults()
		os.Exit(1)
	}

	remainingArgs := flag.Args()

	i := 0
	var values []byte
	for {
		i++
		value, next, keepGoing, err := processNextString(remainingArgs)
		if err != nil {
			log.Fatalf("failed to process value %d - %s", i, err)
		}

		values = append(values, value...)
		if !keepGoing {
			break
		}

		remainingArgs = next
	}

	switch *outputEncoding {
	case hexFormat:
		fmt.Printf("%x", values)
	case rawFormat:
		fmt.Printf("%s", values)
	case b64Format:
		fmt.Print(base64.StdEncoding.EncodeToString(values))
	default:
		log.Fatalf("unknown output format: '%s'", *outputEncoding)
	}

	if !*noNewLine {
		fmt.Println()
	}
}

func newStringFlagsConfig() *stringFlagsConfig {
	set := flag.NewFlagSet("string manipulation options", flag.ExitOnError)
	return &stringFlagsConfig{
		set:            set,
		inputEncoding:  set.String(
			inputFormatArg,
			hexFormat,
			fmt.Sprintf("The input encoding type (%s)", supportedIOEncodingStr())),
		repeatString:   set.Uint(
			repeatStringArg,
			0,
			"Create a new string n bytes long"),
		pattern:       set.Uint(
			"pattern",
			0,
			"Create a pattern string (not well tested, sorry)"),
		swapEndianness: set.Bool(
			wrongEndianArg,
			false,
			"Swap the endianness of the resulting string"),
	}
}

func supportedIOEncodingStr() string {
	return fmt.Sprintf("'%s', '%s', '%s'", b64Format, hexFormat, rawFormat)
}

type stringFlagsConfig struct {
	set            *flag.FlagSet
	inputEncoding  *string
	repeatString   *uint
	pattern        *uint
	swapEndianness *bool
}

func processNextString(remainingOSArgs []string) ([]byte, []string, bool, error) {
	remainingOSArgsLen := len(remainingOSArgs)
	if remainingOSArgsLen == 0 {
		return nil, nil, false, fmt.Errorf("please specify an input value")
	}
	stringFlags := newStringFlagsConfig()
	stringFlags.set.Parse(remainingOSArgs[1:])

	var value []byte
	var err error
	switch *stringFlags.inputEncoding {
	case rawFormat:
		value = []byte(remainingOSArgs[0])
	case b64Format:
		value, err = base64.StdEncoding.DecodeString(remainingOSArgs[0])
		if err != nil {
			return nil, nil, false, fmt.Errorf("failed to base64 decode value - %s", err)
		}
	default:
		value, err = hex.DecodeString(strings.TrimPrefix(remainingOSArgs[0], "0x"))
		if err != nil {
			return nil, nil, false, fmt.Errorf("failed to hex decode value - %s", err)
		}
	}

	if *stringFlags.pattern > 0 {
		value = pattern(int(*stringFlags.pattern))
	}

	if *stringFlags.repeatString > 0 {
		value = bytes.Repeat(value, int(*stringFlags.repeatString))
	}

	if *stringFlags.swapEndianness {
		decodedLen := len(value)
		temp := make([]byte, decodedLen)
		for i := range value {
			temp[decodedLen-1-i] = value[i]
		}
		value = temp
	}

	return value, stringFlags.set.Args(), len(stringFlags.set.Args()) > 0, nil
}

func pattern(length int) []byte {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := bytes.NewBuffer(nil)
	alphabetIndex := 0
	var set uint8
	for i := 0; i < length; i++ {
		if i%2 == 0 {
			result.WriteString(string(letters[alphabetIndex]))
			if alphabetIndex < len(letters)-1 {
				alphabetIndex++
			} else {
				alphabetIndex = 0
				set++
			}
		} else {
			result.WriteString(fmt.Sprintf("%d", set))
		}
	}

	return result.Bytes()
}
