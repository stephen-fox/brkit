package memory

import "fmt"

func ExampleAddressTable() {
	offsets := NewAddressTable("local").
		AddSymbolInContext("ioFileJumps", 0x00000000003ebc30, "local").
		AddSymbolInContext("ioFileJumps", 0x00000000003e82f0, "remote")

	addr := offsets.AddressOrExit("ioFileJumps")
	fmt.Printf("local ioFileJumps: 0x%x\n", addr)

	offsets.SetContext("remote")

	addr = offsets.AddressOrExit("ioFileJumps")
	fmt.Printf("remote ioFileJumps: 0x%x\n", addr)

	// Output:
	// local ioFileJumps: 0x3ebc30
	// remote ioFileJumps: 0x3e82f0
}
