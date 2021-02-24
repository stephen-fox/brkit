package memory

import (
	"bytes"
	"testing"
)

func TestPadStartOfStringWithCharUntilLen(t *testing.T) {
	res := prependStringWithCharUntilLen([]byte("AAA"), 'B', 8)

	exp := []byte("BBBBBAAA")
	if !bytes.Equal(res, exp) {
		t.Fatalf("expected '%s' - got '%s'", exp, res)
	}
}

func TestStackAlignedLen(t *testing.T) {
	formatStr := []byte("|%1000$p|")

	res := stringLenMemoryAligned(formatStr, 8)
	exp := 16
	if res != exp {
		t.Fatalf("expected %d - got %d", exp, res)
	}

	res = stringLenMemoryAligned(formatStr, 4)
	exp = 12
	if res != exp {
		t.Fatalf("expected %d - got %d", exp, res)
	}
}
