package memory

import "log"

func ExampleNewDPAFormatStringWriter() {
	writer, err := NewDPAFormatStringWriter(DPAFormatStringWriterConfig{
		MaxWrite:  999,
		DPAConfig: DPAFormatStringConfig{
			ProcessIO:    &fakeProcessIO{},
			MaxNumParams: 200,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	pm := PointerMakerForX86_32()

	writer.WriteLowerFourBytesAtOrExit(100, pm.FromUint(0xdeadbeef))
}

func ExampleDPAFormatStringWriter_WriteLowerFourBytesAt() {
	writer, err := NewDPAFormatStringWriter(DPAFormatStringWriterConfig{
		MaxWrite:  999,
		DPAConfig: DPAFormatStringConfig{
			ProcessIO:    &fakeProcessIO{},
			MaxNumParams: 200,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	pm := PointerMakerForX86_32()

	// Set the lower four bytes to 1000 (0x03E8).
	err = writer.WriteLowerFourBytesAt(1000, pm.FromUint(0xdeadbeef))
	if err != nil {
		log.Fatalln(err)
	}
}

func ExampleDPAFormatStringWriter_WriteLowerTwoBytesAt() {
	writer, err := NewDPAFormatStringWriter(DPAFormatStringWriterConfig{
		MaxWrite:  999,
		DPAConfig: DPAFormatStringConfig{
			ProcessIO:    &fakeProcessIO{},
			MaxNumParams: 200,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	pm := PointerMakerForX86_32()

	// Set the lower two bytes to 666 (0x029A)
	err = writer.WriteLowerTwoBytesAt(666, pm.FromUint(0xdeadbeef))
	if err != nil {
		log.Fatalln(err)
	}
}

func ExampleDPAFormatStringWriter_WriteLowestByteAt() {
	writer, err := NewDPAFormatStringWriter(DPAFormatStringWriterConfig{
		MaxWrite:  999,
		DPAConfig: DPAFormatStringConfig{
			ProcessIO:    &fakeProcessIO{},
			MaxNumParams: 200,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	pm := PointerMakerForX86_32()

	// Set the lowest byte to 255 (0xFF).
	err = writer.WriteLowestByteAt(255, pm.FromUint(0xdeadbeef))
	if err != nil {
		log.Fatalln(err)
	}
}
