package asm

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

type DecoderConfig struct {
	Disassemble DisassemblySyntax
	ArchConfig  interface{}
}

type X86Config struct {
	Bits int
}

type ARMConfig struct {
	Mode armasm.Mode
}

func NewDecoder(config DecoderConfig) (*Decoder, error) {
	switch assertedConfig := config.ArchConfig.(type) {
	case ARMConfig:
		var dissassemFn func(inst armasm.Inst) string
		switch config.Disassemble {
		case SkipSyntax:
			// Do nothing.
		case ATTSyntax:
			dissassemFn = armasm.GNUSyntax
		default:
			return nil, fmt.Errorf("unsupported syntax type for arm: %s", config.Disassemble)
		}

		return &Decoder{
			decodeOneInstFn: func(singleInst []byte) (Inst, error) {
				armInst, err := armasm.Decode(singleInst, assertedConfig.Mode)
				if err != nil {
					return Inst{}, err
				}

				var disassembly string
				if dissassemFn != nil {
					disassembly = dissassemFn(armInst)
				}

				return Inst{
					Raw:  singleInst,
					Hex:  fmt.Sprintf("0x%x", singleInst[0:armInst.Len]),
					Len:  armInst.Len,
					Dis:  disassembly,
					Inst: armInst,
				}, nil
			},
		}, nil
	case X86Config:
		var disassemblyFn func(inst x86asm.Inst) string
		switch config.Disassemble {
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
			return nil, fmt.Errorf("unsupported syntax type for x86: %s", config.Disassemble)
		}

		return &Decoder{
			decodeOneInstFn: func(singleInst []byte) (Inst, error) {
				x86Inst, err := x86asm.Decode(singleInst, assertedConfig.Bits)
				if err != nil {
					return Inst{}, err
				}

				var disassembly string
				if disassemblyFn != nil {
					disassembly = disassemblyFn(x86Inst)
				}

				return Inst{
					Raw:  singleInst,
					Hex:  fmt.Sprintf("0x%x", singleInst[0:x86Inst.Len]),
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

type Decoder struct {
	decodeOneInstFn func(rawInstructions []byte) (Inst, error)
}

func (o Decoder) DecodeAll(rawInstructions []byte, onDecodeFn func(Inst)) error {
	index := 0
	for {
		if isDone(rawInstructions, index) {
			return nil
		}

		inst, err := o.decodeOneInstFn(rawInstructions[index:])
		if err != nil {
			return fmt.Errorf("failed to decode instruction %d - %w - remaining data: 0x%x",
				index, err, rawInstructions[index:])
		}

		inst.Index = index

		onDecodeFn(inst)

		index += inst.Len
	}
}

func (o Decoder) DecodeFirst(rawInstructions []byte) (Inst, error) {
	return o.decodeOneInstFn(rawInstructions)
}

type Inst struct {
	Raw   []byte
	Hex   string
	Len   int
	Index int
	Dis   string
	Inst  interface{}
}

func DecodeX86(rawInstructions []byte, bits int, onDecodeFn func(inst x86asm.Inst, index int)) error {
	index := 0
	for {
		if isDone(rawInstructions, index) {
			return nil
		}

		inst, err := x86asm.Decode(rawInstructions[index:], bits)
		if err != nil {
			return fmt.Errorf("failed to decode instruction %d - %w - remaining data: 0x%x",
				index, err, rawInstructions[index:])
		}

		onDecodeFn(inst, index)

		index += inst.Len
	}
}

func DecodeARM(rawInstructions []byte, onDecodeFn func(inst armasm.Inst, index int)) error {
	index := 0
	for {
		if isDone(rawInstructions, index) {
			return nil
		}

		inst, err := armasm.Decode(rawInstructions[index:], armasm.ModeARM)
		if err != nil {
			return fmt.Errorf("failed to decode instruction %d - %w - remaining data: 0x%x",
				index, err, rawInstructions[index:])
		}

		onDecodeFn(inst, index)

		index += inst.Len
	}
}

func isDone(rawInstructions []byte, index int) bool {
	return index >= len(rawInstructions)-1
}
