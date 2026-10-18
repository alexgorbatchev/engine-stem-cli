package stems

import (
	"github.com/alexgorbatchev/engine-stem-cli/internal/testfixture"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func createTestMP4(t *testing.T) string {
	t.Helper()
	t.Setenv("ENGINE_DJ_APP_PATH", testfixture.EngineBinary(t, []byte("synthetic-key123")))

	buf := make([]byte, 0, 1024)

	// ftyp box
	ftyp := make([]byte, 28)
	binary.BigEndian.PutUint32(ftyp[0:4], 28)
	copy(ftyp[4:8], "ftyp")
	copy(ftyp[8:12], "isom")
	buf = append(buf, ftyp...)

	// mdat box header (offset 28)
	mdatHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(mdatHeader[0:4], 8+64) // 64 bytes media data
	copy(mdatHeader[4:8], "mdat")
	buf = append(buf, mdatHeader...)

	// 64 bytes mdat payload (2 samples of 32 bytes)
	mdatPayload := make([]byte, 64)
	for i := range mdatPayload {
		mdatPayload[i] = byte(i)
	}
	buf = append(buf, mdatPayload...)

	// stbl box content
	stsz := make([]byte, 28)
	binary.BigEndian.PutUint32(stsz[0:4], 28)
	copy(stsz[4:8], "stsz")
	binary.BigEndian.PutUint32(stsz[12:16], 0) // variable size
	binary.BigEndian.PutUint32(stsz[16:20], 2) // 2 samples
	binary.BigEndian.PutUint32(stsz[20:24], 32)
	binary.BigEndian.PutUint32(stsz[24:28], 32)

	stco := make([]byte, 20)
	binary.BigEndian.PutUint32(stco[0:4], 20)
	copy(stco[4:8], "stco")
	binary.BigEndian.PutUint32(stco[12:16], 1)  // 1 chunk
	binary.BigEndian.PutUint32(stco[16:20], 36) // chunk offset at mdat payload start

	stsc := make([]byte, 28)
	binary.BigEndian.PutUint32(stsc[0:4], 28)
	copy(stsc[4:8], "stsc")
	binary.BigEndian.PutUint32(stsc[12:16], 1) // 1 entry
	binary.BigEndian.PutUint32(stsc[16:20], 1) // first chunk
	binary.BigEndian.PutUint32(stsc[20:24], 2) // 2 samples per chunk
	binary.BigEndian.PutUint32(stsc[24:28], 1) // sample desc idx

	stblContent := append(stsz, stco...)
	stblContent = append(stblContent, stsc...)

	stbl := make([]byte, 8+len(stblContent))
	binary.BigEndian.PutUint32(stbl[0:4], uint32(len(stbl)))
	copy(stbl[4:8], "stbl")
	copy(stbl[8:], stblContent)

	minfContent := stbl
	minf := make([]byte, 8+len(minfContent))
	binary.BigEndian.PutUint32(minf[0:4], uint32(len(minf)))
	copy(minf[4:8], "minf")
	copy(minf[8:], minfContent)

	mdiaContent := minf
	mdia := make([]byte, 8+len(mdiaContent))
	binary.BigEndian.PutUint32(mdia[0:4], uint32(len(mdia)))
	copy(mdia[4:8], "mdia")
	copy(mdia[8:], mdiaContent)

	trakContent := mdia
	trak := make([]byte, 8+len(trakContent))
	binary.BigEndian.PutUint32(trak[0:4], uint32(len(trak)))
	copy(trak[4:8], "trak")
	copy(trak[8:], trakContent)

	moovContent := trak
	moov := make([]byte, 8+len(moovContent))
	binary.BigEndian.PutUint32(moov[0:4], uint32(len(moov)))
	copy(moov[4:8], "moov")
	copy(moov[8:], moovContent)

	buf = append(buf, moov...)

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sample.m4a")
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("writing temp m4a: %v", err)
	}

	return path
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	origPath := createTestMP4(t)
	tmpDir := t.TempDir()
	encPath := filepath.Join(tmpDir, "encrypted.stems")
	decPath := filepath.Join(tmpDir, "decrypted.m4a")

	// Encrypt with verbose
	if err := Encrypt(origPath, encPath, true); err != nil {
		t.Fatalf("Encrypt error = %v", err)
	}

	// Decrypt with verbose
	if err := Decrypt(encPath, decPath, true); err != nil {
		t.Fatalf("Decrypt error = %v", err)
	}

	origBytes, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("reading orig: %v", err)
	}

	decBytes, err := os.ReadFile(decPath)
	if err != nil {
		t.Fatalf("reading decrypted: %v", err)
	}

	if len(origBytes) != len(decBytes) {
		t.Fatalf("decrypted length = %d, want %d", len(decBytes), len(origBytes))
	}
}

func TestEncryptDecryptErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist.m4a")
	invalidFile := filepath.Join(tmpDir, "invalid.m4a")

	if err := os.WriteFile(invalidFile, []byte("invalid content"), 0644); err != nil {
		t.Fatalf("writing invalid file: %v", err)
	}

	// Non-existent file
	if err := Encrypt(nonExistent, filepath.Join(tmpDir, "out.stems"), false); err == nil {
		t.Errorf("Encrypt non-existent expected error, got nil")
	}
	if err := Decrypt(nonExistent, filepath.Join(tmpDir, "out.m4a"), false); err == nil {
		t.Errorf("Decrypt non-existent expected error, got nil")
	}

	// Invalid file without mdat
	if err := Encrypt(invalidFile, filepath.Join(tmpDir, "out.stems"), false); err == nil {
		t.Errorf("Encrypt invalid file expected error, got nil")
	}
	if err := Decrypt(invalidFile, filepath.Join(tmpDir, "out.m4a"), false); err == nil {
		t.Errorf("Decrypt invalid file expected error, got nil")
	}
}
