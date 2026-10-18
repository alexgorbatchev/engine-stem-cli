package cli

import (
	"bytes"
	"encoding/binary"
	"github.com/alexgorbatchev/engine-stem-cli/internal/testfixture"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestMP4File(t *testing.T) string {
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
	path := filepath.Join(tmpDir, "test.m4a")
	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatalf("writing test m4a: %v", err)
	}
	return path
}

func resetFlags() {
	decryptOutput = ""
	decryptForce = false
	encryptOutput = ""
	encryptForce = false
	infoJSON = false
	extractOutputDir = ""
	extractFormat = "wav"
	extractDryRun = false
	generateOutput = ""
	generateModel = ""
	generateBitrate = "640k"
	generateForce = false
	generateDryRun = false
	flagVerbose = false
	flagQuiet = false
}

func TestIsAgentMode(t *testing.T) {
	tests := []struct {
		envVal string
		want   bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{" True ", true},
		{"yes", true},
		{"YES", true},
		{" Yes ", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"", false},
		{"2", false},
		{"random", false},
	}

	for _, tt := range tests {
		t.Run("env="+tt.envVal, func(t *testing.T) {
			t.Setenv("AGENT", tt.envVal)
			if got := IsAgentMode(); got != tt.want {
				t.Errorf("IsAgentMode() with AGENT=%q = %v, want %v", tt.envVal, got, tt.want)
			}
		})
	}
}

func TestDualModeOutputHelpers(t *testing.T) {
	// Test Human Mode
	t.Run("HumanMode", func(t *testing.T) {
		t.Setenv("AGENT", "")
		buf := new(bytes.Buffer)

		PrintOK(buf, "success message")
		if !strings.Contains(buf.String(), "[OK] success message") {
			t.Errorf("expected [OK] tag, got %q", buf.String())
		}

		buf.Reset()
		PrintInfo(buf, "info message")
		if !strings.Contains(buf.String(), "[INFO] info message") {
			t.Errorf("expected [INFO] tag, got %q", buf.String())
		}

		buf.Reset()
		PrintWarn(buf, "warn message")
		if !strings.Contains(buf.String(), "[WARN] warn message") {
			t.Errorf("expected [WARN] tag, got %q", buf.String())
		}

		buf.Reset()
		PrintErr(buf, "error message")
		if !strings.Contains(buf.String(), "[ERROR] error message") {
			t.Errorf("expected [ERROR] tag, got %q", buf.String())
		}

		buf.Reset()
		RenderDivider(buf, '=')
		if buf.Len() == 0 {
			t.Errorf("expected divider line in human mode")
		}

		buf.Reset()
		pairs := []KeyValue{
			{Key: "File", Value: "test.m4a"},
			{Key: "Sample Count", Value: "2"},
		}
		RenderKeyValues(buf, "Test Title", pairs)
		out := buf.String()
		if !strings.Contains(out, "Test Title") || !strings.Contains(out, "PROPERTY") || !strings.Contains(out, "VALUE") {
			t.Errorf("expected formatted table in human mode, got: %s", out)
		}
	})

	// Test Agent Mode
	t.Run("AgentMode", func(t *testing.T) {
		t.Setenv("AGENT", "1")
		buf := new(bytes.Buffer)

		PrintOK(buf, "success message")
		if !strings.Contains(buf.String(), "OK: success message") {
			t.Errorf("expected OK: prefix, got %q", buf.String())
		}

		buf.Reset()
		PrintInfo(buf, "info message")
		if !strings.Contains(buf.String(), "INFO: info message") {
			t.Errorf("expected INFO: prefix, got %q", buf.String())
		}

		buf.Reset()
		PrintWarn(buf, "warn message")
		if !strings.Contains(buf.String(), "WARN: warn message") {
			t.Errorf("expected WARN: prefix, got %q", buf.String())
		}

		buf.Reset()
		PrintErr(buf, "error message")
		if !strings.Contains(buf.String(), "ERR: error message") {
			t.Errorf("expected ERR: prefix, got %q", buf.String())
		}

		buf.Reset()
		RenderDivider(buf, '=')
		if buf.Len() != 0 {
			t.Errorf("expected no divider line in agent mode, got %q", buf.String())
		}

		buf.Reset()
		pairs := []KeyValue{
			{Key: "File", Value: "test.m4a"},
			{Key: "Sample Count", Value: "2"},
		}
		RenderKeyValues(buf, "Test Title", pairs)
		out := buf.String()
		if !strings.Contains(out, "file:test.m4a") || !strings.Contains(out, "sample_count:2") {
			t.Errorf("expected flat key-values in agent mode, got: %s", out)
		}
		if strings.Contains(out, "Test Title") || strings.Contains(out, "Property") {
			t.Errorf("agent mode should not contain table headers or title, got: %s", out)
		}
	})
}

func TestInfoCmd(t *testing.T) {
	resetFlags()
	path := createTestMP4File(t)

	// Test text info output in Human Mode
	t.Run("HumanMode", func(t *testing.T) {
		t.Setenv("AGENT", "")
		buf := new(bytes.Buffer)
		RootCmd.SetOut(buf)
		RootCmd.SetErr(buf)
		RootCmd.SetArgs([]string{"info", path})

		if err := RootCmd.Execute(); err != nil {
			t.Fatalf("info command failed: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "PROPERTY") || !strings.Contains(out, "VALUE") || !strings.Contains(out, "Sample Count") {
			t.Errorf("unexpected human info output: %s", out)
		}
	})

	// Test text info output in Agent Mode
	t.Run("AgentMode", func(t *testing.T) {
		t.Setenv("AGENT", "1")
		buf := new(bytes.Buffer)
		RootCmd.SetOut(buf)
		RootCmd.SetErr(buf)
		RootCmd.SetArgs([]string{"info", path})

		if err := RootCmd.Execute(); err != nil {
			t.Fatalf("info command failed: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "sample_count:2") || !strings.Contains(out, "channels:8 (4 stereo stem pairs)") {
			t.Errorf("unexpected agent info output: %s", out)
		}
	})

	// Test JSON info output
	t.Run("JSON", func(t *testing.T) {
		t.Setenv("AGENT", "")
		resetFlags()
		buf := new(bytes.Buffer)
		RootCmd.SetOut(buf)
		RootCmd.SetErr(buf)
		RootCmd.SetArgs([]string{"info", "--json", path})

		if err := RootCmd.Execute(); err != nil {
			t.Fatalf("info --json command failed: %v", err)
		}

		jsonOut := buf.String()
		if !strings.Contains(jsonOut, `"SampleCount": 2`) || !strings.Contains(jsonOut, `"ChannelCount": 8`) {
			t.Errorf("unexpected JSON info output: %s", jsonOut)
		}
	})
}

func TestEncryptDecryptCmd(t *testing.T) {
	t.Setenv("AGENT", "")
	resetFlags()
	origPath := createTestMP4File(t)
	tmpDir := t.TempDir()
	stemsPath := filepath.Join(tmpDir, "output.stems")
	m4aPath := filepath.Join(tmpDir, "output.m4a")

	// Encrypt command
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"encrypt", "-o", stemsPath, origPath})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("encrypt command failed: %v", err)
	}

	if _, err := os.Stat(stemsPath); os.IsNotExist(err) {
		t.Fatalf("encrypt command did not create stems file")
	}

	// Decrypt command
	buf.Reset()
	RootCmd.SetArgs([]string{"decrypt", "-o", m4aPath, stemsPath})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("decrypt command failed: %v", err)
	}

	if _, err := os.Stat(m4aPath); os.IsNotExist(err) {
		t.Fatalf("decrypt command did not create m4a file")
	}
}

func TestEncryptDecryptCmdAutoOutput(t *testing.T) {
	t.Setenv("AGENT", "")
	resetFlags()
	origPath := createTestMP4File(t)
	tmpDir := t.TempDir()

	// Copy origPath to a file ending in .m4a inside tmpDir
	m4aFile := filepath.Join(tmpDir, "track.m4a")
	content, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("reading orig: %v", err)
	}
	if err := os.WriteFile(m4aFile, content, 0644); err != nil {
		t.Fatalf("writing m4a: %v", err)
	}

	// Test encrypt auto-output path (track.m4a -> track.stems)
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"encrypt", m4aFile})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("encrypt auto-output failed: %v", err)
	}

	expectedStems := filepath.Join(tmpDir, "track.stems")
	if _, err := os.Stat(expectedStems); os.IsNotExist(err) {
		t.Fatalf("expected auto-generated stems file at %s", expectedStems)
	}

	// Test decrypt auto-output path (track.stems -> track.m4a)
	os.Remove(m4aFile)

	buf.Reset()
	RootCmd.SetArgs([]string{"decrypt", expectedStems})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("decrypt auto-output failed: %v", err)
	}

	if _, err := os.Stat(m4aFile); os.IsNotExist(err) {
		t.Fatalf("expected auto-generated m4a file at %s", m4aFile)
	}
}

func TestCmdErrorPaths(t *testing.T) {
	t.Setenv("AGENT", "")
	resetFlags()
	origPath := createTestMP4File(t)
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist.m4a")

	// Non-existent file errors
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)

	RootCmd.SetArgs([]string{"encrypt", nonExistent})
	if err := RootCmd.Execute(); err == nil {
		t.Errorf("encrypt non-existent expected error, got nil")
	}

	RootCmd.SetArgs([]string{"decrypt", nonExistent})
	if err := RootCmd.Execute(); err == nil {
		t.Errorf("decrypt non-existent expected error, got nil")
	}

	RootCmd.SetArgs([]string{"info", nonExistent})
	if err := RootCmd.Execute(); err == nil {
		t.Errorf("info non-existent expected error, got nil")
	}

	RootCmd.SetArgs([]string{"extract", nonExistent})
	if err := RootCmd.Execute(); err == nil {
		t.Errorf("extract non-existent expected error, got nil")
	}

	// Existing output file without --force error
	existingOut := filepath.Join(tmpDir, "existing.m4a")
	os.WriteFile(existingOut, []byte("data"), 0644)

	RootCmd.SetArgs([]string{"encrypt", "-o", existingOut, origPath})
	if err := RootCmd.Execute(); err == nil {
		t.Errorf("encrypt existing output without --force expected error, got nil")
	}

	RootCmd.SetArgs([]string{"decrypt", "-o", existingOut, origPath})
	if err := RootCmd.Execute(); err == nil {
		t.Errorf("decrypt existing output without --force expected error, got nil")
	}

	// Existing output file WITH --force
	RootCmd.SetArgs([]string{"encrypt", "-o", existingOut, "-f", origPath})
	if err := RootCmd.Execute(); err != nil {
		t.Errorf("encrypt existing output with --force failed: %v", err)
	}

	// Invalid format in extract
	RootCmd.SetArgs([]string{"extract", "--format", "invalid", origPath})
	if err := RootCmd.Execute(); err == nil {
		t.Errorf("extract invalid format expected error, got nil")
	}
}

func TestExtractCmdDryRun(t *testing.T) {
	resetFlags()
	path := createTestMP4File(t)
	tmpDir := t.TempDir()

	// Human Mode
	t.Run("HumanMode", func(t *testing.T) {
		t.Setenv("AGENT", "")
		buf := new(bytes.Buffer)
		RootCmd.SetOut(buf)
		RootCmd.SetErr(buf)
		RootCmd.SetArgs([]string{"extract", "--dry-run", "-o", tmpDir, path})

		if err := RootCmd.Execute(); err != nil {
			t.Fatalf("extract --dry-run failed: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "ffmpeg -y") || !strings.Contains(out, "vocals.wav") {
			t.Errorf("unexpected dry run output: %s", out)
		}
	})

	// Agent Mode
	t.Run("AgentMode", func(t *testing.T) {
		t.Setenv("AGENT", "1")
		buf := new(bytes.Buffer)
		RootCmd.SetOut(buf)
		RootCmd.SetErr(buf)
		RootCmd.SetArgs([]string{"extract", "--dry-run", "-o", tmpDir, path})

		if err := RootCmd.Execute(); err != nil {
			t.Fatalf("extract --dry-run in agent mode failed: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "dry_run: ffmpeg -y") || !strings.Contains(out, "vocals.wav") {
			t.Errorf("unexpected dry run output in agent mode: %s", out)
		}
	})
}

func TestEncryptDecryptAutoOutputNonStandardExt(t *testing.T) {
	t.Setenv("AGENT", "")
	resetFlags()
	origPath := createTestMP4File(t)
	tmpDir := t.TempDir()

	// File with non-standard extension .raw
	rawFile := filepath.Join(tmpDir, "track.raw")
	content, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("reading orig: %v", err)
	}
	if err := os.WriteFile(rawFile, content, 0644); err != nil {
		t.Fatalf("writing raw file: %v", err)
	}

	// Encrypt track.raw -> track.raw.stems
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"encrypt", rawFile})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("encrypt track.raw failed: %v", err)
	}

	expectedStems := filepath.Join(tmpDir, "track.raw.stems")
	if _, err := os.Stat(expectedStems); os.IsNotExist(err) {
		t.Fatalf("expected auto-generated %s", expectedStems)
	}

	// Decrypt track.raw.stems -> track.raw.stems.m4a
	buf.Reset()
	RootCmd.SetArgs([]string{"decrypt", rawFile + ".stems"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("decrypt track.raw.stems failed: %v", err)
	}

	expectedM4A := filepath.Join(tmpDir, "track.raw.m4a")
	if _, err := os.Stat(expectedM4A); os.IsNotExist(err) {
		t.Fatalf("expected auto-generated %s", expectedM4A)
	}
}

func TestExtractCmdAutoDir(t *testing.T) {
	t.Setenv("AGENT", "")
	resetFlags()
	path := createTestMP4File(t)

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"extract", "--dry-run", path})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("extract auto-dir failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "test-stems/vocals.wav") {
		t.Errorf("expected test-stems directory in dry run output: %s", out)
	}
}

func TestGenerateCmdDryRun(t *testing.T) {
	resetFlags()
	path := createTestMP4File(t)

	// Human Mode
	t.Run("HumanMode", func(t *testing.T) {
		t.Setenv("AGENT", "")
		buf := new(bytes.Buffer)
		RootCmd.SetOut(buf)
		RootCmd.SetErr(buf)
		RootCmd.SetArgs([]string{"generate", "--dry-run", path})

		if err := RootCmd.Execute(); err != nil {
			t.Fatalf("generate --dry-run failed: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "Pure Go ONNX Model Inference") || !strings.Contains(out, "Encode 8-channel AAC-LC audio") {
			t.Errorf("unexpected dry run generate output: %s", out)
		}
	})

	// Agent Mode
	t.Run("AgentMode", func(t *testing.T) {
		t.Setenv("AGENT", "1")
		buf := new(bytes.Buffer)
		RootCmd.SetOut(buf)
		RootCmd.SetErr(buf)
		RootCmd.SetArgs([]string{"generate", "--dry-run", path})

		if err := RootCmd.Execute(); err != nil {
			t.Fatalf("generate --dry-run in agent mode failed: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "dry_run: generate") {
			t.Errorf("unexpected agent dry run generate output: %s", out)
		}
	})
}

func TestExtractCmdDryRunEncrypted(t *testing.T) {
	t.Setenv("AGENT", "")
	resetFlags()
	origPath := createTestMP4File(t)
	tmpDir := t.TempDir()

	// Encrypt to stems file
	stemsPath := filepath.Join(tmpDir, "encrypted.stems")
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"encrypt", "-o", stemsPath, origPath})
	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Make sample 1 in stems file look encrypted for InspectContainer
	stemsData, err := os.ReadFile(stemsPath)
	if err != nil {
		t.Fatalf("reading stems: %v", err)
	}
	stemsData[84] = 0xe4
	if err := os.WriteFile(stemsPath, stemsData, 0644); err != nil {
		t.Fatalf("writing stems: %v", err)
	}

	// Extract dry-run on encrypted stems file (tests decryption step during extract)
	buf.Reset()
	RootCmd.SetArgs([]string{"extract", "--dry-run", "--verbose", stemsPath})
	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("extract dry-run encrypted failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Decrypting") || !strings.Contains(out, "ffmpeg -y") {
		t.Errorf("unexpected dry run encrypted output: %s", out)
	}
}
