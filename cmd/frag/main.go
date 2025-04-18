// frag finds fragments in pattern strings. Useful for understanding
// how a payload overwrites process state (e.g., finding the offset
// of a payload fragment in a variable that was overwritten by
// a stack-based buffer overflow).
package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	patternStr := flag.String(
		"p",
		"",
		"The string to search (will be hex-decoded if it starts with 0x)")

	patternStrFile := flag.String(
		"P",
		"",
		"Load the string to search from a file")

	fragment := flag.String(
		"f",
		"",
		"The fragment to find (will be hex-decoded if it starts with 0x)")

	wrongEndian := flag.Bool(
		"r",
		false,
		"Reverse fragment's endianness")

	shorten := flag.Bool(
		"retry",
		false,
		"Repeatedly try shortening the fragment if it is not found in the pattern")

	displayVisual := flag.Bool(
		"v",
		false,
		"Display a visualization of the fragment's location in the string")

	flag.Parse()

	if flag.NArg() > 0 {
		return errors.New("it looks like you specified a non-flag argument, are you missing a -<arg>?")
	}

	if *patternStr == "" && *patternStrFile == "" {
		return errors.New("please specify a pattern string")
	}

	if *fragment == "" {
		return errors.New("please specify a fragment string")
	}

	if *patternStrFile != "" {
		patternBytes, err := os.ReadFile(*patternStrFile)
		if err != nil {
			return fmt.Errorf("failed to read pattern string from file %q - %w",
				*patternStrFile, err)
		}

		*patternStr = string(patternBytes)
	}

	var fragmentBinary []byte
	var err error

	if strings.HasPrefix(*fragment, "0x") {
		fragmentBinary, err = hex.DecodeString(strings.TrimPrefix(*fragment, "0x"))
		if err != nil {
			return fmt.Errorf("failed to hex decode fragment string - %v", err)
		}
	} else {
		fragmentBinary = []byte(*fragment)
	}

	var patternBinary []byte
	if strings.HasPrefix(*patternStr, "0x") {
		patternBinary, err = hex.DecodeString(strings.TrimPrefix(*patternStr, "0x"))
		if err != nil {
			return fmt.Errorf("failed to hex decode pattern string - %v", err)
		}
	} else {
		patternBinary = []byte(*patternStr)
	}

	if *wrongEndian {
		decodedFragmentLen := len(fragmentBinary)
		temp := make([]byte, decodedFragmentLen)

		for i := range fragmentBinary {
			temp[decodedFragmentLen-1-i] = fragmentBinary[i]
		}

		fragmentBinary = temp
	}

	for {
		if *shorten {
			log.Printf("shortening fragment to: '%s'...",
				strings.TrimSpace(hex.Dump(fragmentBinary)))
		}

		index := bytes.Index(patternBinary, fragmentBinary)
		if index < 0 {
			newDecodedFragmentLen := len(fragmentBinary)
			if !*shorten || newDecodedFragmentLen == 1 {
				break
			}

			fragmentBinary = fragmentBinary[0 : newDecodedFragmentLen-1]

			continue
		}

		fragmentLen := len(fragmentBinary)
		endIndex := index + fragmentLen

		infoStr := fmt.Sprintf("%d:%d (%d bytes)",
			index, endIndex, fragmentLen)

		if !*displayVisual {
			fmt.Println(infoStr)

			return nil
		}

		fmt.Printf("%s\n", *patternStr)

		spaces := strings.Repeat(" ", index)

		fmt.Printf("%s%s\n",
			spaces, strings.Repeat("^", len(fragmentBinary)))

		fmt.Printf("%s%s\n",
			spaces, infoStr)

		return nil
	}

	return fmt.Errorf("failed to find fragment (hexdump of fragment:\n%s)",
		strings.TrimSpace(hex.Dump(fragmentBinary)))
}
