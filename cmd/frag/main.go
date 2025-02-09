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
		"The pattern string to search")

	fragment := flag.String(
		"f",
		"",
		"The fragment to find")

	wrongEndian := flag.Bool(
		"r",
		false,
		"Reverse fragment's endianness")

	shorten := flag.Bool(
		"retry",
		false,
		"Repeatedly try shortening the fragment if it is not found in the pattern")

	quiet := flag.Bool(
		"q",
		false,
		"Only output the range without any visualization")

	flag.Parse()

	if len(*patternStr) == 0 {
		return errors.New("please specify a pattern string")
	}

	if len(*fragment) == 0 {
		return errors.New("please specify a fragment string")
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

		if *quiet {
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
