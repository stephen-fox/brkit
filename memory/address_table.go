package memory

import (
	"fmt"
)

// NewAddressTable creates a new instance of an *AddressTable with
// the specified initial context. Refer to AddressTable's documentation
// for more information.
func NewAddressTable(initialContext string) *AddressTable {
	return &AddressTable{
		currentContext:          initialContext,
		contextToSymbolsToAddrs: make(map[string]map[string]uint),
	}
}

// AddressTable helps organize memory addresses and offsets for symbols
// in different contexts. A context can be (but is not limited to) the
// name of the target environment.
//
// For example, imagine you are writing an exploit for a piece of software
// running remotely, and libc is a target. The version of libc on your test
// machine might be different from the version on the target machine.
// As a result, you will likely need to modify the offsets of libc symbols
// when testing the exploit in one environment or another.
//
// Rather than manually commenting out variables. you can use an AddressTable
// to track the offsets of symbols for your "test" and "target" environments.
// You can switch environments (contexts) by simply specifying a different
// initial context in the argument to NewAddressTable. This is far less
// error-prone, and results in only one line of code needing to be changed
// when switching between environments.
type AddressTable struct {
	currentContext          string
	contextToSymbolsToAddrs map[string]map[string]uint
}

// SetContext sets the current context to the specified value.
func (o *AddressTable) SetContext(context string) *AddressTable {
	o.currentContext = context
	return o
}

// DeleteContext deletes the specified context.
func (o *AddressTable) DeleteContext(context string) *AddressTable {
	delete(o.contextToSymbolsToAddrs, context)
	return o
}

// AddSymbolInContext adds or sets the address of a symbol for
// the specified context.
func (o *AddressTable) AddSymbolInContext(symbolName string, address uint, context string) *AddressTable {
	symbolsToAddrs := o.contextToSymbolsToAddrs[context]
	if symbolsToAddrs == nil {
		symbolsToAddrs = make(map[string]uint)
	}

	symbolsToAddrs[symbolName] = address
	o.contextToSymbolsToAddrs[context] = symbolsToAddrs

	return o
}

// DeleteSymbolFromContext deletes a symbol from the specified context.
func (o *AddressTable) DeleteSymbolFromContext(symbolName string, context string) *AddressTable {
	symbolsToAddrs, hasIt := o.contextToSymbolsToAddrs[context]
	if hasIt {
		delete(symbolsToAddrs, symbolName)
	}
	return o
}

// DeleteSymbolInAllContexts deletes the specified symbol from all contexts.
func (o *AddressTable) DeleteSymbolInAllContexts(symbolName string) *AddressTable {
	for _, symbolsToAddrs := range o.contextToSymbolsToAddrs {
		delete(symbolsToAddrs, symbolName)
	}
	return o
}

// CurrentContext returns the current context.
func (o *AddressTable) CurrentContext() string {
	return o.currentContext
}

// AddressOrExit returns the address of the specified symbol for the
// currently selected context.
//
// If the context or the symbol do not exist, then DefaultExitFn is invoked.
func (o *AddressTable) AddressOrExit(symbolName string) uint {
	symbolsToAddrs, hasIt := o.contextToSymbolsToAddrs[o.currentContext]
	if !hasIt {
		DefaultExitFn(fmt.Errorf("the current context ('%s') is not in the lookup table",
			o.currentContext))
	}

	addr, hasIt := symbolsToAddrs[symbolName]
	if !hasIt {
		DefaultExitFn(fmt.Errorf("failed to find the symbol '%s' in the table for '%s'",
			symbolName, o.currentContext))
	}

	return addr
}
