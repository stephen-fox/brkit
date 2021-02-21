package process

import (
	"bytes"
	"testing"
)

func TestPointerMakerForX86_U32(t *testing.T) {
	pm := PointerMakerForX86()
	raw := pm.U32(0xdeadbeef)
	exp := []byte{0xef, 0xbe, 0xad, 0xde}
	if !bytes.Equal(raw, exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, raw)
	}
}

func TestPointerMakerForX86_U64(t *testing.T) {
	pm := PointerMakerForX86()
	raw := pm.U64(0x00000000deadbeef)
	exp := []byte{0xef, 0xbe, 0xad, 0xde, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(raw, exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, raw)
	}
}
