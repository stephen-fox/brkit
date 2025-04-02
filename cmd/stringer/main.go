// stringer is a string creation and manipulation tool capable of creating
// pattern strings and arbitrary binary data.
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

	"gitlab.com/stephen-fox/brkit/pattern"
)

const (
	outputFormatArg  = "o"
	printPatternsArg = "printpatterns"
	patternIndexArg  = "pindex"
	patternSetArg    = "pset"
	helpArg          = "h"

	inputFormatArg  = "i"
	patternArg      = "pattern"
	wrongEndianArg  = "wendian"
	repeatStringArg = "repeat"
	commentArg      = "comment"

	hexFormat      = "hex"
	rawFormat      = "raw"
	b64Format      = "b64"
	fromFileFormat = "file"

	appName = "stringer"
	usage   = appName + `
A string creation and manipulation tool capable of creating pattern strings and
arbitrary binary data.

usage:
  ` + appName + ` [main options] [string [string manipulation options]...]

examples:
  ` + appName + ` 0x41 -` + patternArg + ` 200
  ` + appName + ` 0x080491e2 -` + wrongEndianArg + `
  ` + appName + ` 0x41 -` + repeatStringArg + ` 184 0x080491e2 -` + wrongEndianArg + `
  ` + appName + ` -` + outputFormatArg + ` ` + hexFormat + ` /tmp/example.txt -` + inputFormatArg + ` ` + fromFileFormat + `

main options:
`
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	outputEncoding := flag.String(
		outputFormatArg,
		rawFormat,
		fmt.Sprintf("The output encoding type (%s)", outputTypesStr()))

	printPatternStrings := flag.Bool(
		printPatternsArg,
		false,
		"Print pattern strings to stderr for future reference")

	help := flag.Bool(
		helpArg,
		false,
		"Display this help page")

	flag.Parse()

	// TODO: Read from stdin support?
	if *help {
		os.Stderr.WriteString(usage)
		flag.PrintDefaults()

		os.Stderr.WriteString("\nstring manipulation options:\n")
		newStringFlagsConfig().set.PrintDefaults()

		os.Exit(1)
	}

	remainingArgs := flag.Args()

	i := 0

	var values []byte

	db := &pattern.DeBruijn{}
	if *printPatternStrings {
		db.OptLogger = log.Default()
	}

	for {
		i++

		result, err := processNextString(remainingArgs, db)
		if err != nil {
			return fmt.Errorf("failed to process value %d - %s", i, err)
		}

		values = append(values, result.value...)
		if len(result.remainingArgs) == 0 {
			break
		}

		remainingArgs = result.remainingArgs
	}

	switch *outputEncoding {
	case hexFormat:
		fmt.Printf("%x", values)
	case rawFormat:
		fmt.Printf("%s", values)
	case b64Format:
		fmt.Print(base64.StdEncoding.EncodeToString(values))
	default:
		return fmt.Errorf("unknown output format: '%s'", *outputEncoding)
	}

	return nil
}

func newStringFlagsConfig() *stringFlagsConfig {
	set := flag.NewFlagSet("string manipulation options", flag.ExitOnError)

	return &stringFlagsConfig{
		set: set,
		inputEncoding: set.String(
			inputFormatArg,
			hexFormat,
			fmt.Sprintf("The input encoding type (%s)", inputTypesStr())),
		repeatString: set.Uint(
			repeatStringArg,
			0,
			"Create a new string n bytes long"),
		pattern: set.Uint(
			patternArg,
			0,
			"Create a pattern string n bytes long (not well tested, sorry)"),
		swapEndianness: set.Bool(
			wrongEndianArg,
			false,
			"Swap the endianness of the resulting string"),
		comment: set.String(
			commentArg,
			"",
			"Specify a comment for this value"),
	}
}

func inputTypesStr() string {
	return fmt.Sprintf("'%s', '%s', '%s', '%s'",
		b64Format, hexFormat, rawFormat, fromFileFormat)
}

func outputTypesStr() string {
	return fmt.Sprintf("'%s', '%s', '%s'",
		b64Format, hexFormat, rawFormat)
}

type stringFlagsConfig struct {
	set            *flag.FlagSet
	inputEncoding  *string
	repeatString   *uint
	pattern        *uint
	swapEndianness *bool
	comment        *string
}

type processNextStringResult struct {
	value         []byte
	isPatternStr  bool
	remainingArgs []string
}

func processNextString(remainingOSArgs []string, db *pattern.DeBruijn) (*processNextStringResult, error) {
	remainingOSArgsLen := len(remainingOSArgs)
	if remainingOSArgsLen == 0 {
		return nil, fmt.Errorf("please specify an input value")
	}

	stringFlags := newStringFlagsConfig()
	stringFlags.set.Parse(remainingOSArgs[1:])

	inputValue := remainingOSArgs[0]

	var value []byte
	var err error

	switch *stringFlags.inputEncoding {
	case rawFormat:
		value = []byte(inputValue)
	case b64Format:
		value, err = base64.StdEncoding.DecodeString(inputValue)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode value - %s", err)
		}
	case fromFileFormat:
		value, err = os.ReadFile(inputValue)
		if err != nil {
			return nil, fmt.Errorf("failed to read input from file '%s' - %w",
				inputValue, err)
		}
	default:
		value, err = hex.DecodeString(strings.TrimPrefix(inputValue, "0x"))
		if err != nil {
			return nil, fmt.Errorf("failed to hex decode value - %s", err)
		}
	}

	if *stringFlags.pattern > 0 {
		buf := bytes.NewBuffer(nil)

		err = db.WriteToN(buf, int(*stringFlags.pattern))
		if err != nil {
			return nil, fmt.Errorf("failed to write pattern string - %w", err)
		}

		value = buf.Bytes()
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

	return &processNextStringResult{
		value:         value,
		isPatternStr:  *stringFlags.pattern > 0,
		remainingArgs: stringFlags.set.Args(),
	}, nil
}
