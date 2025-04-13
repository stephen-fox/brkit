package asmkit

import (
	"errors"
	"fmt"
	"io"

	"golang.org/x/arch/arm/armasm"
	"golang.org/x/arch/x86/x86asm"
)

const (
	SkipSyntax  DisassemblySyntax = ""
	ATTSyntax   DisassemblySyntax = "att"
	GoSyntax    DisassemblySyntax = "go"
	IntelSyntax DisassemblySyntax = "intel"
)

type DisassemblySyntax string

type DisassemblerConfig struct {
	Src        io.Reader
	Syntax     DisassemblySyntax
	ArchConfig interface{}
}

type X86Config struct {
	Bits int
}

type ArmConfig struct {
	Mode ArmMode
}

// A Mode is an instruction execution mode.
//
// This type allows callers to interoperate with golang.org/x/arch/x86/x86asm
// without forcing them to rely on it directly.
type ArmMode int

func (o ArmMode) toArmasmMode() armasm.Mode {
	return armasm.Mode(o)
}

const (
	ModeARM   = ArmMode(armasm.ModeARM)
	ModeThumb = ArmMode(armasm.ModeThumb)
)

func NewDisassembler(config DisassemblerConfig) (*Disassembler, error) {
	if config.Src == nil {
		return nil, errors.New("source reader is nil")
	}

	var disassOneInstFn func(remainingInsts []byte) (Inst, error)

	switch assertedConfig := config.ArchConfig.(type) {
	case ArmConfig:
		var dissassemFn func(inst armasm.Inst) string
		switch config.Syntax {
		case SkipSyntax:
			// Do nothing.
		case ATTSyntax:
			dissassemFn = armasm.GNUSyntax
		default:
			return nil, fmt.Errorf("unsupported syntax type for arm: %q", config.Syntax)
		}

		disassOneInstFn = func(remainingInsts []byte) (Inst, error) {
			armInst, err := armasm.Decode(remainingInsts, assertedConfig.Mode.toArmasmMode())
			if err != nil {
				return Inst{}, err
			}

			var disassembly string
			if dissassemFn != nil {
				disassembly = dissassemFn(armInst)
			}

			instBin := copySlice(remainingInsts, armInst.Len)

			return Inst{
				Binary:      instBin,
				Len:         armInst.Len,
				Disass:      disassembly,
				ArchLibInst: armInst,
			}, nil
		}
	case X86Config:
		var disassemblyFn func(inst x86asm.Inst) string
		switch config.Syntax {
		case SkipSyntax:
			// Do nothing.
		case ATTSyntax:
			disassemblyFn = func(inst x86asm.Inst) string {
				return x86asm.GNUSyntax(inst, 0, nil)
			}
		case GoSyntax:
			disassemblyFn = func(inst x86asm.Inst) string {
				return x86asm.GoSyntax(inst, 0, nil)
			}
		case IntelSyntax:
			disassemblyFn = func(inst x86asm.Inst) string {
				return x86asm.IntelSyntax(inst, 0, nil)
			}
		default:
			return nil, fmt.Errorf("unsupported syntax type for x86: %q", config.Syntax)
		}

		disassOneInstFn = func(remainingInsts []byte) (Inst, error) {
			x86Inst, err := x86asm.Decode(remainingInsts, assertedConfig.Bits)
			if err != nil {
				return Inst{}, err
			}

			var disassembly string
			if disassemblyFn != nil {
				disassembly = disassemblyFn(x86Inst)
			}

			instBin := copySlice(remainingInsts, x86Inst.Len)

			return Inst{
				Binary:      instBin,
				Len:         x86Inst.Len,
				Disass:      disassembly,
				ArchLibInst: x86Inst,
			}, nil
		}
	default:
		return nil, fmt.Errorf("unsupported config type: %T", assertedConfig)
	}

	var optCommentReader lastComment

	cm, isCommentReader := config.Src.(lastComment)
	if isCommentReader {
		optCommentReader = cm
	}

	return &Disassembler{
		reader:          config.Src,
		optComments:     optCommentReader,
		disassOneInstFn: disassOneInstFn,
		hasMore:         true,
	}, nil
}

func copySlice(src []byte, numBytes int) []byte {
	cp := make([]byte, numBytes)

	copy(cp, src[0:numBytes])

	return cp
}

type Disassembler struct {
	reader          io.Reader
	optComments     lastComment
	disassOneInstFn func(remainingInsts []byte) (Inst, error)
	last            Inst
	buf             []byte
	readerDone      bool
	index           int
	hasMore         bool
	err             error
}

type lastComment interface {
	LastComment() ([]byte, bool)
}

func (o *Disassembler) All(onDecodeFn func(Inst) error) error {
	for o.Next() {
		last := o.Inst()

		err := onDecodeFn(last)
		if err != nil {
			return fmt.Errorf("on decode function failed (%q) - %w",
				last.Disass, err)
		}
	}

	err := o.Err()
	if err != nil {
		return err
	}

	return nil
}

func (o *Disassembler) Err() error {
	if o.err != nil {
		return o.err
	}

	return nil
}

func (o *Disassembler) Inst() Inst {
	return o.last
}

func (o *Disassembler) Next() bool {
	return o.next()
}

func (o *Disassembler) next() bool {
	if o.err != nil || !o.hasMore {
		return false
	}

	inst, hasMore, err := o.parseNext()
	if err != nil {
		o.err = fmt.Errorf("instruction %d - %w", o.index, err)

		return false
	}

	o.last = inst

	o.hasMore = hasMore

	return true
}

func (o *Disassembler) parseNext() (Inst, bool, error) {
	err := o.read()
	if err != nil {
		return Inst{}, false, err
	}

	if len(o.buf) == 0 {
		return Inst{}, false, nil
	}

	inst, err := o.disassOneInstFn(o.buf)
	if err != nil {
		return Inst{}, false, fmt.Errorf("disassembly failed - %w", err)
	}

	if o.optComments != nil {
		comment, hasComment := o.optComments.LastComment()
		if hasComment {
			inst.Comment = string(comment)
		}
	}

	o.buf = o.buf[inst.Len:]

	o.index++

	return inst, len(o.buf) > 0, nil
}

func (o *Disassembler) read() error {
	if o.readerDone {
		return nil
	}

	const readSizeBytes = 1024

	if len(o.buf) < readSizeBytes {
		b := make([]byte, readSizeBytes)

		n, err := o.reader.Read(b)
		switch {
		case err == nil:
			o.buf = append(o.buf, b[0:n]...)
		case errors.Is(err, io.EOF):
			o.buf = append(o.buf, b[0:n]...)

			o.readerDone = true
		default:
			o.readerDone = true

			return err
		}
	}

	return nil
}

type Inst struct {
	Binary      []byte
	Len         int
	Index       int
	Disass      string
	Comment     string
	ArchLibInst interface{}
}

func isDone(rawInstructions []byte, index int) bool {
	return index >= len(rawInstructions)-1
}
