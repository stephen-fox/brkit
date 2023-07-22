package conv

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// CArrayToGoSlice converts the contents of a C programming language
// read from r into a Go []byte declaration string.
//
// It preserves comments and sequences of bytes.
func CArrayToGoSlice(r io.Reader, w io.Writer) error {
	var blobs []Blob
	var longestLine int

	err := CArrayToBlobs(r, func(b Blob) error {
		if len(b.Bytes) > longestLine {
			longestLine = len(b.Bytes)
		}

		blobs = append(blobs, b)

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find blobs in reader - %w", err)
	}

	// numExtraChars represets how:
	// - we append "0x"
	// - we append a comma
	const numExtraChars = 3

	// Modify longest line to match the number of
	// characters we will add to the string plus
	// one space.
	longestLine = longestLine*2 + numExtraChars + 1

	out := byteSliceFormat{
		w:            bufio.NewWriter(w),
		noFormatting: true,
	}

	err = out.start()
	if err != nil {
		return err
	}

	err = out.add([]byte("\n"))
	if err != nil {
		return err
	}

	for _, blob := range blobs {
		err = out.add([]byte{'\t'})
		if err != nil {
			return err
		}

		var numCharsWritten int

		if len(blob.Bytes) > 0 {
			numCharsWritten, err = out.addBytesAsOneIndex(blob.Bytes, false)
			if err != nil {
				return err
			}
		}

		if blob.Comment != "" {
			blob.Comment = strings.TrimSpace(blob.Comment)
			numSpaces := longestLine - numCharsWritten
			err = out.add([]byte(strings.Repeat(" ", numSpaces) + "// " + blob.Comment))
			if err != nil {
				return err
			}
		}

		err = out.add([]byte{'\n'})
		if err != nil {
			return err
		}
	}

	err = out.end()
	if err != nil {
		return err
	}

	return nil
}

// BytesToGoSliceFormat converts a []byte into a Go []byte declaration string.
func BytesToGoSliceFormat(b []byte, noFormatting bool, w io.Writer) error {
	out := byteSliceFormat{
		w:            bufio.NewWriter(w),
		noFormatting: noFormatting,
	}

	err := out.start()
	if err != nil {
		return err
	}

	_, err = out.addBytePerIndex(b, true)
	if err != nil {
		return err
	}

	return out.end()
}

type byteSliceFormat struct {
	w              *bufio.Writer
	noFormatting   bool
	currentLineLen int
}

func (o *byteSliceFormat) start() error {
	_, err := o.w.Write([]byte("[]byte{"))
	return err
}

func (o *byteSliceFormat) addBytesAsOneIndex(decoded []byte, isLast bool) (int, error) {
	b := []byte(fmt.Sprintf("0x%x", decoded))
	n, err := o.w.Write(b)
	if err != nil {
		return n, err
	}

	o.currentLineLen += n

	if !isLast {
		n1, err := o.w.Write([]byte{','})
		n += n1
		if err != nil {
			return n, err
		}

		o.currentLineLen += n1
	}

	if !o.noFormatting && o.currentLineLen >= 62 {
		o.currentLineLen = 0

		n2, err := o.w.Write([]byte("\n\t"))
		n += n2
		if err != nil {
			return n, err
		}
	} else if !isLast {
		n2, err := o.w.Write([]byte{' '})
		n += n2
		if err != nil {
			return n, err
		}

		o.currentLineLen += n2
	}

	return n, nil
}

func (o *byteSliceFormat) addBytePerIndex(decoded []byte, isLast bool) (int, error) {
	decodedLen := len(decoded)
	n := 0

	for i, b := range decoded {
		needsComma := decodedLen > 1 && i != decodedLen-1

		n1, err := o.w.Write([]byte(fmt.Sprintf("0x%x", b)))
		n += n1
		if err != nil {
			return n, err
		}

		o.currentLineLen += n1

		if needsComma || isLast {
			n2, err := o.w.Write([]byte{','})
			n += n2
			if err != nil {
				return n, err
			}

			o.currentLineLen += n2
		}

		if !o.noFormatting && o.currentLineLen >= 62 {
			o.currentLineLen = 0
			n3, err := o.w.Write([]byte("\n\t"))
			n += n3
			if err != nil {
				return n, err
			}
		} else if needsComma {
			n3, err := o.w.Write([]byte{' '})
			n += n3
			if err != nil {
				return n, err
			}

			o.currentLineLen += n
		}
	}

	return n, nil
}

func (o *byteSliceFormat) add(b []byte) error {
	_, err := o.w.Write(b)
	return err
}

func (o *byteSliceFormat) end() error {
	_, err := o.w.Write([]byte{'}', '\n'})
	if err != nil {
		return err
	}

	return o.w.Flush()
}

// Blob represents a chunk of code and a comment.
type Blob struct {
	Bytes   []byte
	Comment string
}

func (o Blob) isEmpty() bool {
	return len(o.Bytes) == 0 && len(o.Comment) == 0
}

// CArrayToBlobs converts a C programming language array to
// to a series of Blob objects.
func CArrayToBlobs(source io.Reader, onBlobFn func(Blob) error) error {
	bufioReader := bufio.NewReader(source)
	needEndOfComment := false

readNextLine:
	line, err := bufioReader.ReadBytes('\n')
	if len(line) > 0 {
		b := bytes.TrimSpace(line)
		if len(b) == 0 {
			goto readNextLine
		}

		if needEndOfComment {
			comment, uncommented, foundEnd := findEndOfComment(b)
			needEndOfComment = foundEnd

			if comment != "" {
				err := onBlobFn(Blob{Comment: comment})
				if err != nil {
					return err
				}
			}

			if len(uncommented) == 0 {
				goto readNextLine
			}

			b = uncommented
		}

		blob, lookForEndOfComment, err := hexData(b)
		if err != nil {
			return err
		}

		needEndOfComment = lookForEndOfComment

		if blob.isEmpty() {
			goto readNextLine
		}

		err = onBlobFn(blob)
		if err != nil {
			return err
		}
	}

	switch {
	case err == nil:
		goto readNextLine
	case errors.Is(err, io.EOF):
		return nil
	default:
		return err
	}
}

func hexData(b []byte) (Blob, bool, error) {
	commentIndex := bytes.Index(b, []byte{'/'})
	hasComment := commentIndex > -1

	buf := bytes.NewBuffer(nil)

	for i, c := range b {
		if isHexChar(c) {
			buf.WriteByte(c)
		} else if hasComment && i == commentIndex {
			break
		}
	}

	var decoded []byte
	if buf.Len() > 0 {
		decoded = make([]byte, hex.DecodedLen(buf.Len()))

		_, err := hex.Decode(decoded, buf.Bytes())
		if err != nil {
			return Blob{}, false, err
		}
	}

	if !hasComment {
		return Blob{Bytes: decoded}, false, nil
	}

	commentB := b[commentIndex+1:]
	if len(commentB) == 0 {
		return Blob{Bytes: decoded}, false, nil
	}

	switch commentB[0] {
	case '/':
		return Blob{Bytes: decoded, Comment: string(commentB[1:])}, false, nil
	case '*':
		end := bytes.Index(commentB, []byte("*/"))
		if end < 0 {
			return Blob{Bytes: decoded, Comment: string(commentB[1:])}, true, nil
		}

		return Blob{Bytes: decoded, Comment: string(commentB[1:end])}, false, nil
	default:
		return Blob{}, false, fmt.Errorf("unknown start of comment character: '%c'", commentB[0])
	}
}

func findEndOfComment(b []byte) (string, []byte, bool) {
	end := bytes.Index(b, []byte("*/"))
	if end < 0 {
		return string(b), nil, false
	}

	return string(b[0:end]), b[end+2:], true
}
