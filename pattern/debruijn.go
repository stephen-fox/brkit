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

// Pattern generates the specified number of de Bruijn pattern
// string characters as a []byte. Each byte in the slice is
// a single, human-readable character in the pattern string.
func (o *DeBruijn) Pattern(numBytes int) ([]byte, error) {
	b := make([]byte, numBytes)

	_, err := o.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// ReadOrExit calls Read. It calls DefaultExitFn if an error occurs.
func (o *DeBruijn) ReadOrExit(b []byte) int {
	n, err := o.Read(b)
	if err != nil {
		DefaultExitFn(fmt.Errorf("pattern.debruijn: failed to read bytes - %w", err))
	}

	return n
}

// Read reads len(b) bytes of a de Bruijn pattern string into b.
//
// This method implements the io.Reader interface.
func (o *DeBruijn) Read(b []byte) (int, error) {
	err := o.generate(len(b))
	if err != nil {
		return 0, fmt.Errorf("failed to generate pattern data for read - %w", err)
	}

	n, err := o.buf.Read(b)

	return n, err
}

// WriteToNOrExit calls WriteToN. It calls DefaultExitFn if an error occurs.
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
	err := o.generate(n)
	if err != nil {
		return fmt.Errorf("failed to generate pattern data for write - %w", err)
	}

	_, err = io.CopyN(w, o.buf, int64(n))
	if err != nil {
		return err
	}

	o.numCalls++

	return nil
}

func (o *DeBruijn) generate(n int) error {
	if n <= 0 {
		return errors.New("n is less than or equal to zero")
	}

	if o.buf == nil {
		o.buf = bytes.NewBuffer(nil)
	}

	for o.buf.Len() < n {
		err := o._generate()
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

	return nil
}

// TODO: This method is incredibly inefficent and needs some halp.
func (o *DeBruijn) _generate() error {
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
