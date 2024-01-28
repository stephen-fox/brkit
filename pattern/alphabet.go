package pattern

import (
	"bytes"
	"strconv"
)

type AlphabetPattern struct {
	AlphabetIndex int
	CurrentNum    uint8
}

func (o *AlphabetPattern) NBytes(length int) []byte {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	result := bytes.NewBuffer(nil)

	for i := 0; i < length; i++ {
		if i%2 == 0 {
			result.WriteByte(letters[o.AlphabetIndex])

			if o.AlphabetIndex < len(letters)-1 {
				o.AlphabetIndex++
			} else {
				o.AlphabetIndex = 0
				o.CurrentNum++
			}
		} else {
			result.WriteString(strconv.Itoa(int(o.CurrentNum)))
		}
	}

	return result.Bytes()
}
