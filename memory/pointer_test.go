package memory

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestPointerMakerForX86_32_Uint(t *testing.T) {
	pm := PointerMakerForX86_32()
	pointer := pm.FromUint(0xdeadbeef)
	exp := []byte{0xef, 0xbe, 0xad, 0xde}
	if !bytes.Equal(pointer.Bytes(), exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer.Bytes())
	}
}

func TestPointerMakerForX68_32_FromHexBytes(t *testing.T) {
	exp := []byte{0xef, 0xbe, 0xad, 0x00}

	pm := PointerMakerForX86_32()
	pointer, err := pm.FromHexBytes([]byte("0xadbeef"), binary.BigEndian)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pointer.Bytes(), exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer.Bytes())
	}

	pointer, err = pm.FromHexBytes([]byte("0xefbead"), binary.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pointer.Bytes(), exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer.Bytes())
	}
}

func TestPointerMakerForX86_64_FromUint(t *testing.T) {
	pm := PointerMakerForX86_64()
	pointer := pm.FromUint(0x00000000deadbeef)
	exp := []byte{0xef, 0xbe, 0xad, 0xde, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(pointer.Bytes(), exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer.Bytes())
	}
}

func TestPointerMakerForX68_64_FromHexBytes(t *testing.T) {
	exp := []byte{0xef, 0xbe, 0xad, 0xde, 0x00, 0x00, 0x00, 0x00}

	pm := PointerMakerForX86_64()
	pointer, err := pm.FromHexBytes([]byte("0x00000000deadbeef"), binary.BigEndian)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pointer.Bytes(), exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer.Bytes())
	}

	pointer, err = pm.FromHexBytes([]byte("0xefbeadde00000000"), binary.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pointer.Bytes(), exp) {
		t.Fatalf("expected 0x%x - got 0x%x", exp, pointer.Bytes())
	}
}

func TestPointer_Uint(t *testing.T) {
	pm := PointerMakerForX86_64()
	pointer, err := pm.FromHexBytes([]byte("0x00000000deadbeef"), binary.BigEndian)
	if err != nil {
		t.Fatal(err)
	}

	address := pointer.Uint()
	if address != 0xdeadbeef {
		t.Fatalf("expected 0xdeadbeef - got %x", address)
	}
}
