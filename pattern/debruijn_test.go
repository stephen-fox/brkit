package pattern

import (
	"bytes"
	"testing"
)

func TestDeBruijn_Duplicates(t *testing.T) {
	numBytes := 10_000_000 // 10 mb.

	t.Run("FourBytes", func(t *testing.T) {
		checkDeBruijnDuplicates(t, numBytes, 4)
	})

	t.Run("EightBytes", func(t *testing.T) {
		checkDeBruijnDuplicates(t, numBytes, 8)
	})
}

func checkDeBruijnDuplicates(t *testing.T, numBytes int, width int) {
	t.Helper()

	buf := bytes.NewBuffer(nil)

	deBruijn := DeBruijn{}

	err := deBruijn.WriteToN(buf, numBytes)
	if err != nil {
		t.Fatalf("failed to write - %s", err)
	}

	m := make(map[string]int)

	i := 0

	for buf.Len() > 0 {
		l := make([]byte, width)

		_, err := buf.Read(l)
		if err != nil {
			t.Fatalf("failed to read from buf - %s", err)
		}

		str := string(l)

		previousI, hasIt := m[str]
		if hasIt {
			t.Fatalf("already encountered %q at iteration %d (current iter: %d)",
				str, previousI, i)
		}

		m[str] = i

		i++
	}
}
