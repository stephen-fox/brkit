package asmkit

import (
	"fmt"

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
	Syntax     DisassemblySyntax
	ArchConfig interface{}
}

type X86Config struct {
	Bits int
}

type ARMConfig struct {
	Mode armasm.Mode
}

func NewDisassembler(config DisassemblerConfig) (*Disassembler, error) {
	switch assertedConfig := config.ArchConfig.(type) {
	case ARMConfig:
		var dissassemFn func(inst armasm.Inst) string
		switch config.Syntax {
		case SkipSyntax:
			// Do nothing.
		case ATTSyntax:
			dissassemFn = armasm.GNUSyntax
		default:
			return nil, fmt.Errorf("unsupported syntax type for arm: %s", config.Syntax)
		}

		return &Disassembler{
			disassOneInstFn: func(remainingInsts []byte) (Inst, error) {
				armInst, err := armasm.Decode(remainingInsts, assertedConfig.Mode)
				if err != nil {
					return Inst{}, err
				}

				var disassembly string
				if dissassemFn != nil {
					disassembly = dissassemFn(armInst)
				}

				instBin := copySlice(remainingInsts, armInst.Len)

				return Inst{
					Bin:  instBin,
					Len:  armInst.Len,
					Dis:  disassembly,
					Inst: armInst,
				}, nil
			},
		}, nil
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

		return &Disassembler{
			disassOneInstFn: func(remainingInsts []byte) (Inst, error) {
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
					Bin:  instBin,
					Len:  x86Inst.Len,
					Dis:  disassembly,
					Inst: x86Inst,
				}, nil
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported config type: %T", assertedConfig)
	}
}

func copySlice(src []byte, numBytes int) []byte {
	cp := make([]byte, numBytes)

	copy(cp, src[0:numBytes])

	return cp
}

type Disassembler struct {
	disassOneInstFn func(remainingInsts []byte) (Inst, error)
}

func (o *Disassembler) All(rawInstructions []byte, onDecodeFn func(Inst) error) error {
	index := 0

	for {
		if isDone(rawInstructions, index) {
			return nil
		}

		inst, err := o.disassOneInstFn(rawInstructions[index:])
		if err != nil {
			return fmt.Errorf("failed to decode instruction %d - %w - remaining data: 0x%x",
				index, err, rawInstructions[index:])
		}

		inst.Index = index

		err = onDecodeFn(inst)
		if err != nil {
			return fmt.Errorf("on decode function failed for instruction %d (%q) - %w",
				index, inst.Dis, err)
		}

		index += inst.Len
	}
}

func (o *Disassembler) Next(rawInstructions []byte) (Inst, error) {
	return o.disassOneInstFn(rawInstructions)
}

type Inst struct {
	Bin   []byte
	Len   int
	Index int
	Dis   string
	Inst  interface{}
}

func isDone(rawInstructions []byte, index int) bool {
	return index >= len(rawInstructions)-1
}
