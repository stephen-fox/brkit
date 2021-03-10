// Package memory provides functionality for reading and writing memory.
//
// Working with pointers and offsets
//
// One of the objectives of this library is to provide a simple API for storing
// pointers, or variables that point to a memory address, from another process.
// The Pointer struct accomplishes this by storing the byte representation
// in the endianness of the target process. These are created using
// a PointerMaker, which simplifies targeting a specific platform.
//
// This library also provides an AddressTable struct for organizing
// memory addresses and offsets in different contexts. It attempts
// to improve exploit development workflows by simplifying the management
// of offsets. It is not referenced by any other code in this library,
// and is meant to be purely a helper utility.
//
// Leaking and writing memory with format strings
//
// Misuse of the format family of C functions can spell disaster.
// This library provides functionality for building memory leaks
// and writes using format strings. Specifically, the types
// FormatStringLeaker, DPAFormatStringLeaker, and DPAFormatStringWriter
// provide a simple set of APIs for accomplishing this.
//
// The structure of a format string used in a format string attack is dependent
// on the objective of the user, the format specifier strategies available
// to the user.
//
// At a high level, these strategies include, but are not limited to,
// the following:
//	- Read data from memory using the direct parameter access (DPA),
//	  by specifying the memory location as an argument number to
//	  the format function
//	- Read data at the specified memory address by appending its raw
//	  bytes to the end of a format string, and referring the raw bytes
//	  using the DPA feature
//	- Write data to memory at a given argument number to the
//	  format function using DPA, and combining %c and %n specifiers
//	- Write data to memory by combining %c, %n, and DPA to specify
//	  an address in the format string as raw bytes
//
// This library takes special care to place target memory addresses
// at the end of the format string. This avoids situations when a null
// byte would unexpectedly terminate a format string. Format strings
// are also padded so that its arguments (such as a pointer) and the
// string itself align with the size of a pointer on the target system.
// This guarantees that the string will produce consistent leaks or
// writes. Error handling enforces an upper limit on the length of
// the arguments provided by the user to help prevent creation
// of an unreliable format string attack.
//
// Please refer to "Exploiting Format String Vulnerabilities" by Team Teso
// for an introduction to the subject:
// https://crypto.stanford.edu/cs155old/cs155-spring08/papers/formatstring-1.2.pdf
package memory
