package conv

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"unicode"
)

// HexArrayToBytes converts an array of hexadecimal characters into
// a []byte. It ignores C comments, which allows the function to parse
// blobs of data mixed with comments.
//
// While this was intended for converting a C array's contents to bytes,
// it can also be used to parse hex pairs from the command line using
// flag.Args.
func HexArrayToBytes(source io.Reader) ([]byte, error) {
	reader := NewHexArrayReader(source)
	buf := bytes.NewBuffer(nil)

	_, err := io.Copy(buf, reader)
	switch {
	case errors.Is(err, io.EOF):
		// OK.
	case err == nil:
		// OK.
	default:
		return nil, err
	}

	return buf.Bytes(), nil
}

// NewHexArrayReader returns an io.Reader implementation that converts
// a C array containing hex-encoded data into chunks of []byte which
// represent the hex-decoded array data.
func NewHexArrayReader(r io.Reader) io.Reader {
	return &hexArrayReader{
		bufferedSrc: bufio.NewReader(r),
	}
}

type hexArrayReader struct {
	bufferedSrc *bufio.Reader
	comments    [][]byte
}

func (o *hexArrayReader) Read(p []byte) (int, error) {
	avail := len(p)

	bytesWritten := 0

	tmp := bytes.Buffer{}

	commentBuf := bytes.Buffer{}

outer:
	for bytesWritten < avail {
		b, err := o.bufferedSrc.ReadByte()
		switch {
		case errors.Is(err, io.EOF):
			break outer
		case err == nil:
			// Keep going.
		default:
			return bytesWritten, fmt.Errorf("failed to read next byte from reader - %w", err)
		}

		if b == '/' {
			err := findComment(findCommentConfig{
				bufferedSrc: o.bufferedSrc,
				optDst:      &commentBuf,
			})
			if err != nil {
				return bytesWritten, err
			}

			if commentBuf.Len() > 0 {
				cp := make([]byte, commentBuf.Len())

				copy(cp, commentBuf.Bytes())

				o.comments = append(o.comments, cp)

				commentBuf.Reset()
			}

			continue
		}

		if !isHexChar(b) {
			continue
		}

		tmp.WriteByte(b)

		if tmp.Len() == 2 {
			_, err := hex.Decode(p[bytesWritten:], tmp.Bytes())
			if err != nil {
				return bytesWritten, fmt.Errorf("failed to hex-decode byte - %w", err)
			}

			bytesWritten++

			tmp.Reset()
		}
	}

	return bytesWritten, nil
}

func (o *hexArrayReader) LastComment() ([]byte, bool) {
	if len(o.comments) == 0 {
		return nil, false
	}

	head := o.comments[0]

	commentCopy := make([]byte, len(head))

	copy(commentCopy, head)

	if len(o.comments) > 1 {
		o.comments = o.comments[1:]
	} else {
		o.comments = nil
	}

	return commentCopy, true
}

type commentDebugWriter struct {
	buf *bytes.Buffer
}

func (o *commentDebugWriter) Write(p []byte) (int, error) {
	o.buf.Write(p)

	return len(p), nil
}

type findCommentConfig struct {
	bufferedSrc *bufio.Reader
	optDst      io.Writer
}

// findComment finds the remaining C syntax comment. It assumes that
// the reader has already processed the very first comment character
// (i.e., that the next byte read from the reader will be the second
// comment character).
func findComment(config findCommentConfig) error {

readAgain:
	secondChar, err := config.bufferedSrc.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read second start of comment char - %w", err)
	}

	switch secondChar {
	case '/':
		line, err := config.bufferedSrc.ReadBytes('\n')

		if config.optDst != nil && len(line) > 0 {
			line = bytes.TrimSpace(line)

			_, writeErr := config.optDst.Write(append(line, '\n'))
			if writeErr != nil {
				return fmt.Errorf("failed to write // comment to writer - %w", err)
			}
		}

		switch {
		case err == nil:
			_, err := discardWhitespace(config.bufferedSrc)
			if err != nil {
				return fmt.Errorf("failed to discard whitespace - %w", err)
			}

			nextChar, err := config.bufferedSrc.ReadByte()
			if err != nil {
				return err
			}

			if nextChar == '/' {
				goto readAgain
			}

			err = config.bufferedSrc.UnreadByte()
			if err != nil {
				return fmt.Errorf("failed to unread next byte - %w", err)
			}

			return nil
		default:
			return fmt.Errorf("failed to find newline char for line comment - %w", err)
		}
	case '*':
	readMultiLine:
		comment, err := config.bufferedSrc.ReadBytes('*')

		if config.optDst != nil && len(comment) > 0 {
			comment = bytes.TrimSuffix(comment, []byte{'*'})

			_, writeErr := config.optDst.Write(comment)
			if writeErr != nil {
				return fmt.Errorf("failed to write start of '/*' comment to writer - %w", writeErr)
			}
		}

		switch {
		case err == nil:
			nextChar, err := config.bufferedSrc.ReadByte()
			if err != nil {
				return fmt.Errorf("failed to check if next byte is end of multi-line comment - %w", err)
			}

			if nextChar == '/' {
				// End of comment reached.
				return nil
			}

			if config.optDst != nil {
				_, writeErr := config.optDst.Write([]byte{'*', nextChar})
				if writeErr != nil {
					return fmt.Errorf("failed to write trailing multiline comment chars to writer - %w", err)
				}
			}

			goto readMultiLine
		default:
			return fmt.Errorf("failed to find corresponding '*/' end of comment - %w", err)
		}
	default:
		return fmt.Errorf("unknown second start of comment char '%c'", secondChar)
	}
}

func discardWhitespace(bufferedSrc *bufio.Reader) ([]byte, error) {
	tmp := bytes.Buffer{}

	for {
		b, err := bufferedSrc.ReadByte()
		if err != nil {
			return tmp.Bytes(), err
		}

		if unicode.IsSpace(rune(b)) {
			tmp.WriteByte(b)

			continue
		}

		err = bufferedSrc.UnreadByte()
		if err != nil {
			return tmp.Bytes(), fmt.Errorf("failed to unread byte - %w", err)
		}

		return tmp.Bytes(), nil
	}
}

func isHexChar(b byte) bool {
	return (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F') || (b >= '0' && b <= '9')
}
