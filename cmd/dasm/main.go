// dasm is a simple diassembler for CPU instructions in various
// encoding formats.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"gitlab.com/stephen-fox/brkit/asmkit"
	"gitlab.com/stephen-fox/brkit/conv"
)

const (
	asmSyntaxArg    = "s"
	inputFormatArg  = "i"
	outputFormatArg = "o"
	helpArg         = "h"

	intelSyntax = "intel"
	attSyntax   = "att"
	goSyntax    = "go"

	syntaxes = "'" + intelSyntax + "', '" + attSyntax + "', " + goSyntax + "'"

	x86_32Platform = "x86_32"
	x86_64Platform = "x86_64"
	armPlatform    = "arm"

	hexFormat = "hex"
	rawFormat = "raw"
	b64Format = "b64"

	prettyDisassFormat      = "pretty"
	jsonDisassFormat        = "json"
	jsonVerboseDisassFormat = "jsonv"
	goDisassFormat          = "go"

	inputFormats = "'" + rawFormat + "', '" + hexFormat +
		"', '" + b64Format + "'"

	outputFormats = "'" + rawFormat + "', '" + hexFormat +
		"', '" + b64Format + "', '" + prettyDisassFormat +
		"', '" + jsonDisassFormat + "', '" + jsonVerboseDisassFormat +
		"', '" + goDisassFormat + "'"

	appName = "dasm"

	usage = `DESCRIPTION
  ` + appName + ` is a simple diassembler for CPU instructions in various
  encoding formats. It was originally created to gauge the
  trustworthiness of shellcode from other users.

USAGE
  ` + appName + ` [options] ` + armPlatform + `|` + x86_32Platform + `|` + x86_64Platform + ` < some-file

SUPPORTED INPUT FORMATS
  ` + inputFormats + `

  Note: '` + hexFormat + `' means both hex chars or a C-style hex array body

SUPPORTED OUTPUT FORMATS
  ` + outputFormats + `

  Notes:
    - '` + goDisassFormat + `' means a Go []byte
    - '` + hexFormat + `' and '` + b64Format + `' will contain the instructions in binary
      format (not the human-readble assembly instructions)

EXAMPLES:
  Note: The following examples use shellcode written by Charles Stevenson
  (core@bokeoa.com):
  http://shell-storm.org/shellcode/files/shellcode-55.php

  Disassemble shellcode:
    $ echo "\x31\xc0\x40\x89\xc3\xcd\x80" > exit-1.hex
    $ ` + appName + ` ` + x86_32Platform + ` < exit-1.hex
    xor eax, eax
    inc eax
    mov ebx, eax
    int 0x80

  Disassemble the previous example into a Go []byte:
    $ ` + appName + ` -` + outputFormatArg + ` ` + goDisassFormat + ` ` + x86_32Platform + ` < exit-1.hex
    []byte {
        0x31, 0xc0, // xor eax, eax
        0x40, // inc eax
        0x89, 0xc3, // mov ebx, eax
        0xcd, 0x80, // int 0x80
    }

OPTIONS
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
	help := flag.Bool(
		helpArg,
		false,
		"Display this information")

	inputFormat := flag.String(
		inputFormatArg,
		hexFormat,
		"The input data `format`. Supported input formats are:\n"+inputFormats+"\n")

	outputFormat := flag.String(
		outputFormatArg,
		prettyDisassFormat,
		"The output data `format`. Suppported output formats are:\n"+outputFormats+"\n")

	syntax := flag.String(
		asmSyntaxArg,
		intelSyntax,
		"The desired assembly `syntax`. Supported syntaxes are:\n"+syntaxes+"\n")

	flag.Parse()

	if *help {
		out := os.Stderr

		stdoutInfo, err := os.Stdout.Stat()
		if err == nil && stdoutInfo.Mode()&os.ModeNamedPipe != 0 {
			out = os.Stdout
		}

		flag.CommandLine.SetOutput(out)
		out.WriteString(usage)
		flag.PrintDefaults()
		os.Exit(1)
	}

	if flag.NArg() != 1 {
		return fmt.Errorf("please specify a platform for decode for ('%s', '%s', '%s')",
			armPlatform, x86_32Platform, x86_64Platform)
	}

	config := asmkit.DisassemblerConfig{
		Syntax: asmkit.AssemblySyntax(*syntax),
	}

	platform := flag.Arg(0)

	switch platform {
	case armPlatform:
		config.ArchConfig = asmkit.ArmConfig{Mode: asmkit.ModeARM}
	case x86_32Platform, x86_64Platform:
		bits := 32
		if platform == x86_64Platform {
			bits = 64
		}

		config.ArchConfig = asmkit.X86Config{Bits: bits}
	default:
		return fmt.Errorf("unsupported platform: %q", platform)
	}

	switch *inputFormat {
	case rawFormat:
		config.Src = os.Stdin
	case hexFormat:
		config.Src = conv.NewHexArrayReader(os.Stdin)
	case b64Format:
		config.Src = base64.NewDecoder(base64.StdEncoding, os.Stdin)
	default:
		return fmt.Errorf("unknown input format: %q", *inputFormat)
	}

	disassembler, err := asmkit.NewDisassembler(config)
	if err != nil {
		return fmt.Errorf("failed to create new disassembler - %w", err)
	}

	output := bytes.NewBuffer(nil)
	var writer instWriter

	switch *outputFormat {
	case prettyDisassFormat:
		writer = &disassWriter{
			w: output,
		}
	case hexFormat:
		writer = &encoderWriter{
			encoder: hex.NewEncoder(output),
			w:       output,
		}
	case b64Format:
		writer = &encoderWriter{
			encoder: base64.NewEncoder(base64.StdEncoding, output),
			w:       output,
		}
	case jsonDisassFormat:
		writer = &jsonDisassWriter{
			indent: "  ",
			w:      output,
		}
	case jsonVerboseDisassFormat:
		writer = &jsonVerboseWriter{
			indent: "  ",
			w:      output,
		}
	case goDisassFormat:
		writer = &goByteSliceWriter{
			w: output,
		}
	default:
		return fmt.Errorf("unsupported output format: %q",
			*outputFormat)
	}

	err = disassembler.All(func(inst asmkit.Inst) error {
		return writer.Write(inst)
	})
	if err != nil {
		return fmt.Errorf("failed to decode instructions for %q - %w",
			platform, err)
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to write remaining data to output - %w", err)
	}

	_, err = io.Copy(os.Stdout, output)
	if err != nil {
		return err
	}

	return nil
}

type instWriter interface {
	Write(asmkit.Inst) error
	Flush() error
}

var _ instWriter = (*disassWriter)(nil)

type disassWriter struct {
	w io.Writer
}

func (o *disassWriter) Write(inst asmkit.Inst) error {
	_, err := o.w.Write([]byte(inst.Assembly))
	if err != nil {
		return err
	}

	if inst.Comment != "" {
		_, err = o.w.Write([]byte([]byte(" ;" + inst.Comment)))
		if err != nil {
			return err
		}
	}

	_, err = o.w.Write([]byte([]byte{'\n'}))
	if err != nil {
		return err
	}

	return nil
}

func (o *disassWriter) Flush() error {
	return nil
}

var _ instWriter = (*encoderWriter)(nil)

type encoderWriter struct {
	encoder io.Writer
	w       io.Writer
}

func (o *encoderWriter) Write(inst asmkit.Inst) error {
	_, err := o.encoder.Write([]byte(inst.Binary))
	if err != nil {
		return err
	}

	return nil
}

func (o *encoderWriter) Flush() error {
	closer, ok := o.encoder.(io.Closer)
	if ok {
		err := closer.Close()
		if err != nil {
			return err
		}
	}

	_, err := o.w.Write([]byte{'\n'})
	if err != nil {
		return err
	}

	return nil
}

var _ instWriter = (*jsonDisassWriter)(nil)

type jsonDisassWriter struct {
	indent string
	w      io.Writer
	buf    []string
}

func (o *jsonDisassWriter) Write(inst asmkit.Inst) error {
	o.buf = append(o.buf, inst.Assembly)

	return nil
}

func (o *jsonDisassWriter) Flush() error {
	enc := json.NewEncoder(o.w)

	enc.SetIndent("", o.indent)

	err := enc.Encode(o.buf)
	if err != nil {
		return err
	}

	return nil
}

var _ instWriter = (*jsonVerboseWriter)(nil)

type jsonVerboseWriter struct {
	indent string
	w      io.Writer
	buf    []json.RawMessage
}

func (o *jsonVerboseWriter) Write(inst asmkit.Inst) error {
	item, err := json.MarshalIndent(&inst, "", o.indent)
	if err != nil {
		return err
	}

	o.buf = append(o.buf, item)

	return nil
}

func (o *jsonVerboseWriter) Flush() error {
	enc := json.NewEncoder(o.w)

	enc.SetIndent("", o.indent)

	err := enc.Encode(o.buf)
	if err != nil {
		return err
	}

	return nil
}

var _ instWriter = (*goByteSliceWriter)(nil)

type goByteSliceWriter struct {
	isInit bool
	w      io.Writer
}

func (o *goByteSliceWriter) Write(inst asmkit.Inst) error {
	if !o.isInit {
		o.isInit = true

		_, err := o.w.Write([]byte("[]byte {\n"))
		if err != nil {
			return err
		}
	}

	_, err := o.w.Write([]byte([]byte{'\t'}))
	if err != nil {
		return err
	}

	for _, b := range inst.Binary {
		_, err = fmt.Fprintf(o.w, "0x%x, ", b)
		if err != nil {
			return err
		}
	}

	_, err = o.w.Write([]byte([]byte("// " + inst.Assembly)))
	if err != nil {
		return err
	}

	if inst.Comment != "" {
		_, err = o.w.Write([]byte([]byte(" ;" + inst.Comment)))
		if err != nil {
			return err
		}
	}

	_, err = o.w.Write([]byte([]byte{'\n'}))
	if err != nil {
		return err
	}

	return nil
}

func (o *goByteSliceWriter) Flush() error {
	_, err := o.w.Write([]byte([]byte{'}', '\n'}))
	if err != nil {
		return err
	}

	return nil
}
