package crypto

import (
	"bytes"
	"github.com/alexgorbatchev/engine-stem-cli/internal/testfixture"
	"path/filepath"
	"testing"
)

func TestCipherEncryptDecrypt(t *testing.T) {
	t.Setenv("ENGINE_DJ_APP_PATH", testfixture.EngineBinary(t, []byte("synthetic-key123")))
	c, err := NewCipher()
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}

	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello world")},
		{"exact block", []byte("0123456789abcdef")},
		{"multi block", []byte("the quick brown fox jumps over the lazy dog")},
		{"binary payload", []byte{0x00, 0xff, 0xaa, 0x55, 0x12, 0x34, 0x56, 0x78}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted := c.EncryptSample(tt.data)
			if len(tt.data) > 0 && len(encrypted)%16 != 0 {
				t.Fatalf("EncryptSample() output length %d is not a multiple of 16", len(encrypted))
			}

			decrypted := c.DecryptSample(encrypted)
			if !bytes.Equal(decrypted, tt.data) {
				t.Fatalf("DecryptSample() = %v, want %v", decrypted, tt.data)
			}
		})
	}
}

func TestPKCS7Padding(t *testing.T) {
	tests := []struct {
		inputLen  int
		wantedPad byte
	}{
		{0, 16},
		{1, 15},
		{15, 1},
		{16, 16},
	}

	for _, tt := range tests {
		input := make([]byte, tt.inputLen)
		padded := addPKCS7Padding(input)
		if len(padded)%16 != 0 {
			t.Errorf("addPKCS7Padding(len %d) result len %d not multiple of 16", tt.inputLen, len(padded))
		}
		if padded[len(padded)-1] != tt.wantedPad {
			t.Errorf("addPKCS7Padding(len %d) pad byte = %d, want %d", tt.inputLen, padded[len(padded)-1], tt.wantedPad)
		}
		unpadded := removePKCS7Padding(padded)
		if !bytes.Equal(unpadded, input) {
			t.Errorf("removePKCS7Padding() = len %d, want len %d", len(unpadded), len(input))
		}
	}
}

func TestRemovePKCS7PaddingInvalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want []byte
	}{
		{"empty", []byte{}, []byte{}},
		{"padLen zero", []byte{1, 2, 3, 0}, []byte{1, 2, 3, 0}},
		{"padLen too big", []byte{1, 2, 3, 20}, []byte{1, 2, 3, 20}},
		{"corrupted pad byte", []byte{1, 2, 3, 4, 3, 3}, []byte{1, 2, 3, 4, 3, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removePKCS7Padding(tt.data)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("removePKCS7Padding(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestDecryptSampleInvalidLength(t *testing.T) {
	t.Setenv("ENGINE_DJ_APP_PATH", testfixture.EngineBinary(t, []byte("synthetic-key123")))
	c, err := NewCipher()
	if err != nil {
		t.Fatalf("NewCipher error = %v", err)
	}

	// Non-multiple of 16 should be returned unchanged
	data := []byte{1, 2, 3, 4, 5}
	got := c.DecryptSample(data)
	if !bytes.Equal(got, data) {
		t.Errorf("DecryptSample(unaligned) = %v, want %v", got, data)
	}
}

func TestDeriveKeyAndScanBinary(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "missing.app")

	_, err := scanBinaryForKey(nonExistent)
	if err == nil {
		t.Errorf("scanBinaryForKey expected error for missing binary, got nil")
	}

	mockBinary := testfixture.EngineBinary(t, []byte("synthetic-key123"))

	extractedKey, err := scanBinaryForKey(mockBinary)
	if err != nil {
		t.Fatalf("scanBinaryForKey on mock binary error = %v", err)
	}

	if len(extractedKey) != 16 {
		t.Errorf("extractedKey len = %d, want 16", len(extractedKey))
	}

	// Test DeriveKey with ENGINE_DJ_APP_PATH env var pointing to mock binary
	t.Setenv("ENGINE_DJ_APP_PATH", mockBinary)
	cachedKeyMu.Lock()
	cachedKey = nil
	cachedKeyMu.Unlock()
	key, err := DeriveKey()
	if err != nil {
		t.Fatalf("DeriveKey with ENGINE_DJ_APP_PATH error = %v", err)
	}

	if !bytes.Equal(extractedKey, key) {
		t.Errorf("extractedKey %v != DeriveKey %v", extractedKey, key)
	}

	// Test SetEngineAppPath
	SetEngineAppPath(tmpDir)
	if len(engineBinaryPaths) == 0 {
		t.Errorf("engineBinaryPaths empty after SetEngineAppPath")
	}
}

func BenchmarkDeriveKey(b *testing.B) {
	mockBinary := testfixture.EngineBinary(b, []byte("synthetic-key123"))
	b.Setenv("ENGINE_DJ_APP_PATH", mockBinary)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DeriveKey(); err != nil {
			b.Fatal(err)
		}
	}
}
