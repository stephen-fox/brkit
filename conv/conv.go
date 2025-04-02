package conv

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
)

// HexArrayToBytes converts an array of hexadecimal characters into
// a []byte. It ignores C comments, which allows the function to parse
// blobs of data mixed with comments.
//
// While this was intended for converting a C array's contents to bytes, it can
// also be used to parse hex pairs from the command line using flag.Args.
func HexArrayToBytes(source io.Reader) ([]byte, error) {
	bufferedSource := bufio.NewReader(source)
	buf := bytes.NewBuffer(nil)

OUTER:
	for {
		b, err := bufferedSource.ReadByte()
		switch err {
		case io.EOF:
			break OUTER
		case nil:
			// Nothing to do.
		default:
			return nil, fmt.Errorf("failed to read next byte from reader - %w", err)
		}

		if b == '/' {
			peekedSocChar, err := bufferedSource.Peek(1)
			if err != nil {
				return nil, fmt.Errorf("failed to peek second start of comment char - %w", err)
			}
			switch peekedSocChar[0] {
			case '/':
				_, err = bufferedSource.ReadBytes('\n')
				switch err {
				case io.EOF:
					break OUTER
				case nil:
					continue OUTER
				default:
					return nil, fmt.Errorf("failed to find newline char for line comment - %w", err)
				}
			case '*':
				_, err := bufferedSource.ReadByte()
				if err != nil {
					return nil, fmt.Errorf("failed to discard peeked char - %w", err)
				}
				for {
					_, err = bufferedSource.ReadBytes('*')
					if err != nil {
						return nil, fmt.Errorf("failed to find corresponding '*' comment char - %w", err)
					}

					nextChar, err := bufferedSource.ReadByte()
					if err != nil {
						return nil, fmt.Errorf("failed to peek next end of multi-line comment char - %w", err)
					}

					if nextChar == '/' {
						continue OUTER
					}
				}
			default:
				return nil, fmt.Errorf("unknown second start of comment char '%c'", peekedSocChar[0])
			}
		}

		if isHexChar(b) {
			buf.WriteByte(b)
		}
	}

	hexDecoded := make([]byte, hex.DecodedLen(buf.Len()))
	_, err := hex.Decode(hexDecoded, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to hex decode bytes - %w", err)
	}
	return hexDecoded, nil
}

func isHexChar(b byte) bool {
	return (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F') || (b >= '0' && b <= '9')
}
