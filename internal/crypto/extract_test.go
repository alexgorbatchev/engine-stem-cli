package crypto

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"github.com/alexgorbatchev/engine-stem-cli/internal/testfixture"
)

func TestScanExtractsBinaryValues(t *testing.T) {
	tests := []struct{ name, key string }{
		{"first", "synthetic-key123"},
		{"changed", "different-key456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := []byte(tt.key)
			got, err := scanBinaryForKey(testfixture.EngineBinary(t, want))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatal("extracted key does not match synthetic initializer")
			}
		})
	}
}

func TestRejectUnsupportedBinary(t *testing.T) {
	path := testfixture.EngineBinary(t, []byte("synthetic-key123"))
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"not Mach-O", func([]byte) []byte { return []byte("not an executable") }},
		{"truncated", func(b []byte) []byte { return b[:12] }},
		{"Intel only", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[4:], 0x1000007); return b }},
		{"missing imports", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[16:], 1); return b }},
		{"unrelated import", func(b []byte) []byte { b[0x842] = 'x'; return b }},
		{"missing text", func(b []byte) []byte { b[104] = 'x'; return b }},
		{"invalid symbol index", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[184+68:], 100); return b }},
		{"unsupported branch in initializer", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[0x414:], 0x14000002); return b }},
		{"missing key reference", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[0x404:], 0x91000508); return b }},
		{"truncated text payload", func(b []byte) []byte { return b[:0x500] }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := extractKey(tt.mutate(bytes.Clone(valid))); err == nil {
				t.Fatal("unsupported binary unexpectedly supplied a key")
			}
		})
	}
}

func TestUniversalBinary(t *testing.T) {
	path := testfixture.EngineBinary(t, []byte("synthetic-key123"))
	thin, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 4096+len(thin))
	for i, v := range []uint32{0xcafebabe, 1, 0x100000c, 0, 4096, uint32(len(thin)), 12} {
		binary.BigEndian.PutUint32(data[i*4:], v)
	}
	copy(data[4096:], thin)
	key, err := extractKey(data)
	if err != nil || !bytes.Equal(key, []byte("synthetic-key123")) {
		t.Fatalf("universal binary extraction failed: %v", err)
	}
	binary.BigEndian.PutUint32(data[8:], 0x1000007)
	if _, err := extractKey(data); err == nil {
		t.Fatal("universal binary without ARM64 accepted")
	}
}

func TestDeriveKeyFailureAndCacheIsolation(t *testing.T) {
	cachedKeyMu.Lock()
	previousKey, previousPaths := cachedKey, engineBinaryPaths
	cachedKey, engineBinaryPaths = nil, nil
	cachedKeyMu.Unlock()
	t.Cleanup(func() {
		cachedKeyMu.Lock()
		cachedKey, engineBinaryPaths = previousKey, previousPaths
		cachedKeyMu.Unlock()
	})
	t.Setenv("ENGINE_DJ_APP_PATH", "")
	if _, err := DeriveKey(); err == nil {
		t.Fatal("missing installation supplied a key")
	}
	if _, err := NewCipher(); err == nil {
		t.Fatal("cipher initialized without a key")
	}
	want := []byte("synthetic-key123")
	path := testfixture.EngineBinary(t, want)
	engineBinaryPaths = []string{path}
	t.Setenv("ENGINE_DJ_APP_PATH", path+".missing")
	key, err := DeriveKey()
	if err != nil {
		t.Fatal(err)
	}
	key[0] ^= 0xff
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	cached, err := DeriveKey()
	if err != nil || !bytes.Equal(cached, want) {
		t.Fatal("cached key changed through the returned slice")
	}
}
