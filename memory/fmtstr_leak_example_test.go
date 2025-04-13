package memory_test

import (
	"log"

	"gitlab.com/stephen-fox/brkit/memory"
)

func ExampleNewDPAFormatStringLeaker() {
	leaker, err := memory.NewDPAFormatStringLeaker(memory.DPAFormatStringConfig{
		ProcessIO:    &fakeProcessIO{},
		MaxNumParams: 200,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("read: 0x%x", leaker.RawPointerAtParamOrExit(10))
}

func ExampleDPAFormatStringLeaker_FindParamNumber() {
	leaker, err := memory.NewDPAFormatStringLeaker(memory.DPAFormatStringConfig{
		ProcessIO:    &fakeProcessIO{},
		MaxNumParams: 200,
	})
	if err != nil {
		log.Fatalln(err)
	}

	paramNum, foundIt, err := leaker.FindParamNumber([]byte{0x7f, 0x41, 0x41, 0x41, 0x41, 0x41})
	if err != nil {
		log.Fatalln(err)
	}

	if !foundIt {
		log.Fatalln("failed to find target")
	}

	log.Printf("target is at param. number: %d", paramNum)
}

func ExampleDPAFormatStringLeaker_RawPointerAtParam() {
	leaker, err := memory.NewDPAFormatStringLeaker(memory.DPAFormatStringConfig{
		ProcessIO:    &fakeProcessIO{},
		MaxNumParams: 200,
	})
	if err != nil {
		log.Fatalln(err)
	}

	raw, err := leaker.RawPointerAtParam(10)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("read 0x%x", raw)
}

func ExampleSetupFormatStringLeakViaDPA() {
	leaker, err := memory.SetupFormatStringLeakViaDPA(memory.DPAFormatStringConfig{
		ProcessIO:    &fakeProcessIO{},
		MaxNumParams: 200,
	})
	if err != nil {
		log.Fatalln(err)
	}

	pm := memory.PointerMakerForX86_64()

	log.Printf("read: 0x%x", leaker.MemoryAtOrExit(pm.FromUint(0x00000000deadbeef)))
}

func ExampleFormatStringLeaker_MemoryAt() {
	leaker, err := memory.SetupFormatStringLeakViaDPA(memory.DPAFormatStringConfig{
		ProcessIO:    &fakeProcessIO{},
		MaxNumParams: 200,
	})
	if err != nil {
		log.Fatalln(err)
	}

	pm := memory.PointerMakerForX86_64()

	raw, err := leaker.MemoryAt(pm.FromUint(0x00000000deadbeef))
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("read: 0x%x", raw)
}

type fakeProcessIO struct {
	ptrSizeBytes  int
	expectedBytes []byte
	expectedErr   error
}

func (o fakeProcessIO) WriteLine([]byte) error {
	return nil
}

func (o fakeProcessIO) ReadUntil([]byte) ([]byte, error) {
	return o.expectedBytes, o.expectedErr
}

func (o fakeProcessIO) PointerSizeBytes() int {
	return o.ptrSizeBytes
}
