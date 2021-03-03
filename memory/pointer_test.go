package memory

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestPointerMakerForX86_32_Uint(t *testing.T) {
	pm := PointerMakerForX68_32()
	pointer := pm.Uint(0xdeadbeef)
	exp := []byte{0xef, 0xbe, 0xad, 0xde}
	if !bytes.Equal(pointer, exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer)
	}
}

func TestPointerMakerForX68_32_HexBytes(t *testing.T) {
	exp := []byte{0xef, 0xbe, 0xad, 0x00}

	pm := PointerMakerForX68_32()
	pointer, err := pm.HexBytes([]byte("0xadbeef"), binary.BigEndian)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pointer, exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer)
	}

	pointer, err = pm.HexBytes([]byte("0xefbead"), binary.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pointer, exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer)
	}
}

func TestPointerMakerForX86_64_Uint(t *testing.T) {
	pm := PointerMakerForX68_64()
	raw := pm.Uint(0x00000000deadbeef)
	exp := []byte{0xef, 0xbe, 0xad, 0xde, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(raw, exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, raw)
	}
}

func TestPointerMakerForX68_64_HexBytes(t *testing.T) {
	exp := []byte{0xef, 0xbe, 0xad, 0xde, 0x00, 0x00, 0x00, 0x00}

	pm := PointerMakerForX68_64()
	pointer, err := pm.HexBytes([]byte("0x00000000deadbeef"), binary.BigEndian)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pointer, exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer)
	}

	pointer, err = pm.HexBytes([]byte("0xefbeadde00000000"), binary.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pointer, exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer)
	}
}
