package cryptoutil

import (
	"encoding/base64"
	"testing"
)

func TestEncryptBUPTPasswordShape(t *testing.T) {
	got, err := EncryptBUPTPassword("password")
	if err != nil {
		t.Fatalf("encrypt password: %v", err)
	}
	first, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("decode outer base64: %v", err)
	}
	second, err := base64.StdEncoding.DecodeString(string(first))
	if err != nil {
		t.Fatalf("decode inner base64: %v", err)
	}
	if len(second)%16 != 0 {
		t.Fatalf("ciphertext length = %d, want multiple of 16", len(second))
	}
}
