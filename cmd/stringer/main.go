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
	printPatternsArg = "printpatterns"
	patternIndexArg  = "pindex"
	patternSetArg    = "pset"
	helpArg          = "h"

	inputFormatArg  = "i"
	patternArg      = "pattern"
	wrongEndianArg  = "wendian"
	repeatStringArg = "repeat"
	commentArg      = "comment"

	hexFormat = "hex"
	rawFormat = "raw"
	b64Format = "b64"

	appName = "stringer"
	usage   = appName + `
An application for working with strings of bytes, and manipulating data.

usage:
  ` + appName + ` [main options] [string [string manipulation options]...]

examples:
  ` + appName + ` A -` + patternArg + ` 200
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
	printPatternStrings := flag.Bool(
		printPatternsArg,
		false,
		"Print pattern strings to stderr for future reference")
	patternIndex := flag.Uint(
		patternIndexArg,
		0,
		"The initial pattern index value")
	patternSet := flag.Uint(
		patternSetArg,
		0,
		"The initial pattern set value")
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
	pg := &patternGenerator{
		alphabetIndex: int(*patternIndex),
		set:           uint8(*patternSet),
	}
	for {
		i++
		result, err := processNextString(remainingArgs, pg)
		if err != nil {
			log.Fatalf("failed to process value %d - %s", i, err)
		}

		if *printPatternStrings && result.isPatternStr {
			os.Stderr.WriteString(fmt.Sprintf("pattern str @ %d: %x\n",
				i, result.value))
		}

		values = append(values, result.value...)
		if len(result.remainingArgs) == 0 {
			break
		}

		remainingArgs = result.remainingArgs
	}

	if pg.alphabetIndex > 0 || pg.set > 0 {
		log.Printf("pattern ended at index %d, set %d", pg.alphabetIndex, pg.set)
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
}

func newStringFlagsConfig() *stringFlagsConfig {
	set := flag.NewFlagSet("string manipulation options", flag.ExitOnError)
	return &stringFlagsConfig{
		set: set,
		inputEncoding: set.String(
			inputFormatArg,
			hexFormat,
			fmt.Sprintf("The input encoding type (%s)", supportedIOEncodingStr())),
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

func supportedIOEncodingStr() string {
	return fmt.Sprintf("'%s', '%s', '%s'", b64Format, hexFormat, rawFormat)
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

func processNextString(remainingOSArgs []string, pg *patternGenerator) (*processNextStringResult, error) {
	remainingOSArgsLen := len(remainingOSArgs)
	if remainingOSArgsLen == 0 {
		return nil, fmt.Errorf("please specify an input value")
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
			return nil, fmt.Errorf("failed to base64 decode value - %s", err)
		}
	default:
		value, err = hex.DecodeString(strings.TrimPrefix(remainingOSArgs[0], "0x"))
		if err != nil {
			return nil, fmt.Errorf("failed to hex decode value - %s", err)
		}
	}

	if *stringFlags.pattern > 0 {
		value = pg.pattern(int(*stringFlags.pattern))
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

type patternGenerator struct {
	alphabetIndex int
	set           uint8
}

func (o *patternGenerator) pattern(length int) []byte {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := bytes.NewBuffer(nil)
	for i := 0; i < length; i++ {
		if i%2 == 0 {
			result.WriteString(string(letters[o.alphabetIndex]))
			if o.alphabetIndex < len(letters)-1 {
				o.alphabetIndex++
			} else {
				o.alphabetIndex = 0
				o.set++
			}
		} else {
			result.WriteString(fmt.Sprintf("%d", o.set))
		}
	}

	return result.Bytes()
}
