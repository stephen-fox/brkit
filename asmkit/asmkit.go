package asmkit

import (
	"errors"
	"fmt"
	"io"

	"golang.org/x/arch/arm/armasm"
	"golang.org/x/arch/x86/x86asm"
)

const (
	// SkipSyntax tells the assembler/disassembler to skip assembly
	// or disassembly.
	SkipSyntax AssemblySyntax = "skip"

	// ATTSyntax tells the assembler/disassembler to use AT&T syntax.
	// You should feel bad for using this setting:
	//
	// https://outerproduct.net/2021-02-13_att-asm.html
	ATTSyntax AssemblySyntax = "att"

	// GoSyntax tells the assembler/Disassembler to use Go syntax.
	GoSyntax AssemblySyntax = "go"

	// IntelSyntax tells the assembler/disassembler to use Intel syntax.
	IntelSyntax AssemblySyntax = "intel"
)

// AssemblySyntax is the assembly syntax to use.
type AssemblySyntax string

// DisassemblerConfig configures a Disassembler.
type DisassemblerConfig struct {
	// Src is the io.Reader to read binary CPU instructions from.
	Src io.Reader

	// Syntax is the AssemblySyntax to disassemble into.
	Syntax AssemblySyntax

	// ArchConfig is the architecture-specific configuration
	// to use.
	//
	// This can be ArmConfig or X86Config.
	ArchConfig interface{}
}

// X86Config configures the assembler/disassembler for a x86 CPU.
type X86Config struct {
	// Bits is the number of bits (e.g., 32 or 64).
	Bits int
}

// ArmConfig configures the assembler/disassembler for an ARM CPU.
type ArmConfig struct {
	// Mode is the ARM mode.
	//
	// Refer to the golang.org/x/arch/arm/armasm Go library
	// for more information.
	Mode ArmMode
}

// ArmMode is an instruction execution mode for ARM.
//
// This type allows callers to interoperate with golang.org/x/arch/arm/armasm
// without forcing them to rely on it directly.
type ArmMode int

func (o ArmMode) toArmasmMode() armasm.Mode {
	return armasm.Mode(o)
}

// These ArmModes provide interoperability with golang.org/x/arch/arm/armasm.
const (
	ModeARM   = ArmMode(armasm.ModeARM)
	ModeThumb = ArmMode(armasm.ModeThumb)
)

// NewDisassembler instantiates a new Disassembler.
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
				Assembly:    disassembly,
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
				Assembly:    disassembly,
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

// Disassembler disassembles CPU instructions using a bufio.Scanner-like API.
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

// All disassembles all CPU instructions from the underlying io.Reader,
// calling onDecodeFn for each instruction it successfully diassembles.
func (o *Disassembler) All(onDecodeFn func(Inst) error) error {
	for o.Next() {
		last := o.Inst()

		err := onDecodeFn(last)
		if err != nil {
			return fmt.Errorf("on decode function failed (%q) - %w",
				last.Assembly, err)
		}
	}

	err := o.Err()
	if err != nil {
		return err
	}

	return nil
}

// Err returns the last error or nil if no error has occurred.
func (o *Disassembler) Err() error {
	if o.err != nil {
		return o.err
	}

	return nil
}

// Inst returns the last-parsed instruction.
func (o *Disassembler) Inst() Inst {
	return o.last
}

// Next disassembles the next CPU instruction from the underlying io.Reader
// and returns true if an instruction was successfully parsed. It returns
// false if an instruction could not be read or disassembly failed.
//
// Callers should check the error returned by Err if this method returns
// false.
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

// Inst represents a CPU instruction.
type Inst struct {
	// Binary is the instruction in binary format.
	Binary []byte

	// Len is the length of the instruction in bytes.
	Len int

	// Index is the zero-based index number of the instruction.
	Index int

	// Assembly is the instruction in human-readable format.
	Assembly string

	// Comment is an optional comment.
	//
	// If this Inst was generated by a Disassembler and the
	// underlying io.Reader provides a method to retreive
	// the last comment, then this field will contain any
	// associated comments.
	Comment string

	// ArchLibInst is the instruction object provided by
	// the golang.org/x/arch library.
	ArchLibInst interface{}
}
