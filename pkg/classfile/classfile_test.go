package classfile

import "testing"

func TestMagicHeaderValidation(t *testing.T) {
	invalidData := []byte{0x12, 0x34, 0x56, 0x78}
	_, err := Parse(invalidData)
	if err == nil {
		t.Error("expected error for invalid magic bytes, got nil")
	}

	validHeader := []byte{
		0xCA, 0xFE, 0xBA, 0xBE, // Magic
		0x00, 0x00, 0x00, 0x32, // Minor=0, Major=50
		0x00, 0x01, // Constant pool count = 1
	}
	cf, err := Parse(validHeader)
	if err != nil {
		t.Fatalf("unexpected error parsing valid header: %v", err)
	}
	if cf.Magic != 0xCAFEBABE {
		t.Errorf("expected magic 0xCAFEBABE, got 0x%X", cf.Magic)
	}
}
