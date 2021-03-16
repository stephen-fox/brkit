package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"gitlab.com/stephen-fox/brkit/asm"
	"gitlab.com/stephen-fox/brkit/conv"
	"golang.org/x/arch/arm/armasm"
	"golang.org/x/arch/x86/x86asm"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const (
	asmSyntaxArg    = "s"
	inputFormatArg  = "i"
	outputFormatArg = "o"
	helpArg         = "h"

	intelSyntax = "intel"
	attSyntax   = "att"
	goSyntax    = "go"

	hexFormat = "hex"
	rawFormat = "raw"
	b64Format = "b64"

	prettyFormat      = "pretty"
	jsonHumanFormat   = "json"
	jsonVerboseFormat = "jsonv"

	x86_32Platform = "x86_32"
	x86_64Platform = "x86_64"
	armPlatform    = "arm"

	appName = "TODO"
	usage   = appName + `
TODO

The example hex string was written by Charles Stevenson (core@bokeoa.com):
http://shell-storm.org/shellcode/files/shellcode-55.php

usage:
` + appName + ` [options] ` + armPlatform + `|` + x86_32Platform + `|` + x86_64Platform + `

examples:
` + appName + ` "\x31\xc0\x40\x89\xc3\xcd\x80"
` + appName + ` "\x31\xc0" "\x40\x89" "\xc3\xcd\x80"

options:
`
)

func main() {
	inputFormat := flag.String(
		inputFormatArg,
		hexFormat,
		"The input data format")
	outputFormat := flag.String(
		outputFormatArg,
		prettyFormat,
		"")
	syntax := flag.String(
		asmSyntaxArg,
		intelSyntax,
		"The desired assembly syntax")
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

	if flag.NArg() != 1 {
		log.Fatalf("please specify a platform for decode for ('%s', '%s', '%s')",
			armPlatform, x86_32Platform, x86_32Platform)
	}

	config := asm.DecoderConfig{
		Disassemble: asm.DisassemblySyntax(*syntax),
	}
	platform := flag.Arg(0)
	switch platform {
	case armPlatform:
		config.ArchConfig = asm.ARMConfig{Mode: armasm.ModeARM}
	case x86_32Platform, x86_64Platform:
		bits := 32
		if platform == x86_64Platform {
			bits = 64
		}

		config.ArchConfig = asm.X86Config{Bits: bits}
	default:
		log.Fatalf("unsupported platform: '%s'", platform)
	}

	decoder, err := asm.NewDecoder(config)
	if err != nil {
		log.Fatalf("failed to create new decoder for - %s", err)
	}

	var decoded []byte
	switch *inputFormat {
	case b64Format:
		b64Str, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("failed to read base64 data from stdin - %s", err)
		}
		decoded = make([]byte, base64.StdEncoding.DecodedLen(len(b64Str)))
		_, err = base64.StdEncoding.Decode(decoded, b64Str)
	case hexFormat:
		decoded, err = conv.HexArrayToBytes(os.Stdin)
	case rawFormat:
		decoded, err = ioutil.ReadAll(os.Stdin)
	default:
		log.Fatalf("unknown data format: %s", *inputFormat)
	}
	if err != nil {
		log.Fatalf("failed to read %s data - %s", *inputFormat, err)
	}

	var jsonOut []string
	err = decoder.DecodeAll(decoded, func(inst asm.Inst) {
		switch *outputFormat {
		case prettyFormat:
			fmt.Println(inst.Dis)
		case jsonVerboseFormat:
			raw, err := json.MarshalIndent(&inst, "", "    ")
			if err != nil {
				log.Fatalf("failed to marshal instruction to json - %s", err)
			}
			jsonOut = append(jsonOut, string(raw))
		case jsonHumanFormat:
			jsonOut = append(jsonOut, inst.Dis)
		}
	})
	if err != nil {
		log.Fatalf("failed to decode instructions for '%s' - %s", platform, err)
	}

	if len(jsonOut) > 0 {
		switch *outputFormat {
		case jsonHumanFormat:
			raw, err := json.MarshalIndent(jsonOut, "", "    ")
			if err != nil {
				log.Fatalf("failed to marshal instructions summary slice to json - %s", err)
			}
			fmt.Printf("%s\n", raw)
		case jsonVerboseFormat:
			fmt.Printf("{[%s]}\n", strings.Join(jsonOut, ",\n"))
		}
	}
}

func main2() {
	inputFormat := flag.String(
		inputFormatArg,
		hexFormat,
		"The input data format")
	outputFormat := flag.String(
		outputFormatArg,
		prettyFormat,
		"")
	syntax := flag.String(
		asmSyntaxArg,
		intelSyntax,
		"The desired assembly syntax")
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

	if flag.NArg() != 1 {
		log.Fatalf("please specify a platform for decode for ('%s', '%s', '%s')",
			armPlatform, x86_32Platform, x86_32Platform)
	}

	var err error
	var decoded []byte
	switch *inputFormat {
	case b64Format:
		b64Str, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("failed to read base64 data from stdin - %s", err)
		}
		decoded = make([]byte, base64.StdEncoding.DecodedLen(len(b64Str)))
		_, err = base64.StdEncoding.Decode(decoded, b64Str)
	case hexFormat:
		decoded, err = conv.HexArrayToBytes(os.Stdin)
	case rawFormat:
		decoded, err = ioutil.ReadAll(os.Stdin)
	default:
		log.Fatalf("unknown data format: %s", *inputFormat)
	}
	if err != nil {
		log.Fatalf("failed to read %s data - %s", *inputFormat, err)
	}

	var jsonOut []string
	platform := flag.Arg(0)
	switch platform {
	case armPlatform:
		var fn func(inst armasm.Inst) string
		switch *syntax {
		case attSyntax:
			fn = armasm.GNUSyntax
		case goSyntax, intelSyntax:
			log.Fatalf("%s syntax is not supported for arm", *syntax)
		default:
			log.Fatalf("unknown assembly syntax: '%s'", *syntax)
		}

		err = asm.DecodeARM(decoded, func(inst armasm.Inst, index int) {
			switch *outputFormat {
			case prettyFormat:
				fmt.Println(fn(inst))
			case jsonVerboseFormat:
				raw, err := json.MarshalIndent(&inst, "", "    ")
				if err != nil {
					log.Fatalf("failed to marshal instruction to json - %s", err)
				}
				jsonOut = append(jsonOut, string(raw))
			case jsonHumanFormat:
				jsonOut = append(jsonOut, fn(inst))
			}
		})
	case x86_32Platform, x86_64Platform:
		bits := 32
		if platform == x86_64Platform {
			bits = 64
		}

		var fn func(inst x86asm.Inst) string
		switch *syntax {
		case attSyntax:
			fn = func(inst x86asm.Inst) string {
				return x86asm.GNUSyntax(inst, 0, nil)
			}
		case goSyntax:
			fn = func(inst x86asm.Inst) string {
				return x86asm.GoSyntax(inst, 0, nil)
			}
		case intelSyntax:
			fn = func(inst x86asm.Inst) string {
				return x86asm.IntelSyntax(inst, 0, nil)
			}
		default:
			log.Fatalf("unknown assembly syntax: '%s'", *syntax)
		}

		err = asm.DecodeX86(decoded, bits, func(inst x86asm.Inst, index int) {
			switch *outputFormat {
			case prettyFormat:
				fmt.Println(fn(inst))
			case jsonVerboseFormat:
				raw, err := json.MarshalIndent(&inst, "", "    ")
				if err != nil {
					log.Fatalf("failed to marshal instruction to json - %s", err)
				}
				jsonOut = append(jsonOut, string(raw))
			case jsonHumanFormat:
				jsonOut = append(jsonOut, fn(inst))
			}
		})
	default:
		log.Fatalf("unknown platform: '%s'", platform)
	}
	if err != nil {
		log.Fatalf("failed to decode instructions for '%s' - %s", platform, err)
	}

	if len(jsonOut) > 0 {
		switch *outputFormat {
		case jsonHumanFormat:
			raw, err := json.MarshalIndent(jsonOut, "", "    ")
			if err != nil {
				log.Fatalf("failed to marshal instructions summary slice to json - %s", err)
			}
			fmt.Printf("%s\n", raw)
		case jsonVerboseFormat:
			fmt.Printf("{[%s]}\n", strings.Join(jsonOut, ",\n"))
		}
	}
}
