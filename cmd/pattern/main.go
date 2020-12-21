package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strings"
)

func main() {
	pattern := flag.String(
		"p",
		"",
		"The pattern string to search")
	patternTohex := flag.Bool(
		"tohex",
		false,
		"Encode pattern in hexadecimal encoding before searching it")
	fragment := flag.String(
		"f",
		"",
		"The fragment to find")
	wrongEndian := flag.Bool(
		"little",
		false,
		"Convert the fragment to little (wrong) endian before finding it")
	shorten := flag.Bool(
		"retry",
		false,
		"Repeatedly try shortening the fragment if it is not found in the pattern")

	flag.Parse()

	if len(*pattern) == 0 {
		log.Fatalln("please specify a pattern string")
	}

	if len(*fragment) == 0 {
		log.Fatalln("please specify a fragment string")
	}

	*fragment = strings.TrimPrefix(*fragment, "0x")
	decodedFragment := make([]byte, hex.DecodedLen(len(*fragment)))
	_, err := hex.Decode(decodedFragment, []byte(*fragment))
	if err != nil {
		log.Fatalf("failed to hex decode fragment string - %s", err)
	}

	*pattern = strings.TrimPrefix(*pattern, "0x")
	if *patternTohex {
		*pattern = fmt.Sprintf("%x", *pattern)
	}

	patternRaw, err := hex.DecodeString(*pattern)
	if err != nil {
		log.Fatalf("failed to hex decode pattern - %s", err)
	}

	if *wrongEndian {
		decodedFragmentLen := len(decodedFragment)
		temp := make([]byte, decodedFragmentLen)
		for i := range decodedFragment {
			temp[decodedFragmentLen-1-i] = decodedFragment[i]
		}
		decodedFragment = temp
	}

	for {
		if *shorten {
			log.Printf("trying fragment: '%s'", decodedFragment)
		}

		index := bytes.Index(patternRaw, decodedFragment)
		if index < 0 {
			newDecodedFragmentLen := len(decodedFragment)
			if !*shorten || newDecodedFragmentLen == 1 {
				break
			}
			decodedFragment = decodedFragment[0:newDecodedFragmentLen-1]
			continue
		}

		log.Printf("fragment offset is %d, result: '%s'",
			index, string(*pattern)[0:index])
		return
	}

	log.Fatalf("failed to find fragment '%x' in pattern", decodedFragment)
}
