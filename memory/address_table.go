package memory

import (
	"fmt"
)

func NewAddressTable(initialContext string) *AddressTable {
	return &AddressTable{
		currentContext:          initialContext,
		contextToSymbolsToAddrs: make(map[string]map[string]uint),
	}
}

type AddressTable struct {
	currentContext          string
	contextToSymbolsToAddrs map[string]map[string]uint
}

func (o *AddressTable) SetContext(context string) *AddressTable {
	o.currentContext = context
	return o
}

func (o *AddressTable) DeleteContext(context string) *AddressTable {
	delete(o.contextToSymbolsToAddrs, context)
	return o
}

func (o *AddressTable) AddSymbolInContext(symbolName string, address uint, context string) *AddressTable {
	symbolsToAddrs := o.contextToSymbolsToAddrs[context]
	if symbolsToAddrs == nil {
		symbolsToAddrs = make(map[string]uint)
	}

	symbolsToAddrs[symbolName] = address
	o.contextToSymbolsToAddrs[context] = symbolsToAddrs

	return o
}

func (o *AddressTable) DeleteSymbolFromContext(symbolName string, context string) *AddressTable {
	symbolsToAddrs, hasIt := o.contextToSymbolsToAddrs[context]
	if hasIt {
		delete(symbolsToAddrs, symbolName)
	}
	return o
}

func (o *AddressTable) DeleteSymbolInAllContexts(symbolName string) *AddressTable {
	for _, symbolsToAddrs := range o.contextToSymbolsToAddrs {
		delete(symbolsToAddrs, symbolName)
	}
	return o
}

func (o *AddressTable) CurrentContext() string {
	return o.currentContext
}

func (o *AddressTable) AddressOrExit(symbolName string) uint {
	symbolsToAddrs, hasIt := o.contextToSymbolsToAddrs[o.currentContext]
	if !hasIt {
		defaultExitFn(fmt.Errorf("the current context ('%s') is not in the lookup table",
			o.currentContext))
	}

	addr, hasIt := symbolsToAddrs[symbolName]
	if !hasIt {
		defaultExitFn(fmt.Errorf("failed to find the symbol '%s' in the table for '%s'",
			symbolName, o.currentContext))
	}

	return addr
}
