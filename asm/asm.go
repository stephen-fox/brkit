package asm

import (
	"fmt"
	"golang.org/x/arch/arm/armasm"
	"golang.org/x/arch/x86/x86asm"
)

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
