package pattern

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
)

// DeBruijn generates a pattern string using a de Bruijn sequence.
//
// This code is heavily based on work by D3Ext:
// https://gist.github.com/D3Ext/845bdc6a22bbdd50fe409d78b7d59b96
type DeBruijn struct {
	// OptLogger logs the pattern string if specified.
	OptLogger *log.Logger
	t         int
	p         int
	n         int
	buf       *bytes.Buffer
	numCalls  int
}

const (
	deBruijnChars    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345689"
	deBruijnCharsLen = len(deBruijnChars)
)

// WriteToNOrExit calls WriteToN and calls DefaultExitFn if an error occurs.
func (o *DeBruijn) WriteToNOrExit(w io.Writer, n int) {
	err := o.WriteToN(w, n)
	if err != nil {
		DefaultExitFn(fmt.Errorf("pattern.debruijn: failed to write pattern string number %d of size %d - %w",
			o.numCalls, n, err))
	}
}

// WriteToN writes n bytes of a de Bruijn pattern string to w.
// Subsequent calls to WriteToN will resume the de Bruijn sequence.
func (o *DeBruijn) WriteToN(w io.Writer, n int) error {
	if n <= 0 {
		return errors.New("n is less than or equal to zero")
	}

	if o.buf == nil {
		o.buf = bytes.NewBuffer(nil)
	}

	for o.buf.Len() < n {
		err := o.generate()
		if err != nil {
			return err
		}
	}

	if o.OptLogger != nil {
		// a b c d e
		// 0 1 2 3 4
		// [0 : 5]
		o.OptLogger.Println("pattern string "+
			strconv.Itoa(o.numCalls)+":",
			string(o.buf.Bytes()[0:n]))
	}

	_, err := io.CopyN(w, o.buf, int64(n))
	if err != nil {
		return err
	}

	o.numCalls++

	return nil
}

func (o *DeBruijn) generate() error {
	o.n += 4

	if o.t == 0 {
		o.t = 1
	}

	if o.p == 0 {
		o.p = 1
	}

	// TODO: Just write to buf directly?
	a := make([]byte, deBruijnCharsLen*o.n)
	var seq []byte
	var db func(int, int)

	db = func(t, p int) {
		o.t = t
		o.p = p

		if t > o.n {
			if o.n%p == 0 {
				seq = append(seq, a[1:p+1]...)
			}
		} else {
			a[t] = a[t-p]

			db(t+1, p)

			for j := int(a[t-p] + 1); j < deBruijnCharsLen; j++ {
				a[t] = byte(j)

				db(t+1, t)
			}
		}
	}

	db(o.t, o.p)

	for _, i := range seq {
		err := o.buf.WriteByte(deBruijnChars[i])
		if err != nil {
			return err
		}
	}

	// return b + b[0:o.n-1]
	cp := o.buf.Bytes()[0 : o.n-1]

	_, err := o.buf.Write(cp)
	if err != nil {
		return err
	}

	return nil
}
