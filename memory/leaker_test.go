package memory

import (
	"bytes"
	"testing"
)

func TestPrependStringWithCharUntilLen(t *testing.T) {
	res := prependStringWithCharUntilLen([]byte("AAA"), 'B', 8)

	exp := []byte("BBBBBAAA")
	if !bytes.Equal(res, exp) {
		t.Fatalf("expected '%s' - got '%s'", exp, res)
	}
}

func TestAppendStringWithCharUntilLen(t *testing.T) {
	res := appendStringWithCharUntilLen([]byte("AAA"), 'B', 8)

	exp := []byte("AAABBBBB")
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

func TestRandomStringOfCharsAndNums(t *testing.T) {
	result := make(map[string]struct{})
	for i := 0; i < 10; i++ {
		numBytes := 8
		data, err := randomStringOfCharsAndNums(numBytes)
		if err != nil {
			t.Fatal(err)
		}

		if len(data) != numBytes {
			t.Fatalf("expected %d characters - got %d", numBytes, len(data))
		}

		str := string(data)
		_, hasIt := result[str]
		if hasIt {
			t.Fatalf("value '%s' was already generated", str)
		}

		result[str] = struct{}{}
	}
}
