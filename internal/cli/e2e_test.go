package cli

import (
	"bytes"
	"encoding/json"
	"github.com/alexgorbatchev/engine-stem-cli/internal/testfixture"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexgorbatchev/engine-stem-cli/internal/mp4"
)

func ensureKeyAvailable(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv("ENGINE_DJ_APP_PATH", testfixture.EngineBinary(t, []byte("synthetic-key123")))
}

func TestE2ERoundtrip(t *testing.T) {
	// Look for testdata/after_dark_30s.mp3 relative to project root
	sampleMP3 := filepath.Join("..", "..", "testdata", "after_dark_30s.mp3")
	if _, err := os.Stat(sampleMP3); os.IsNotExist(err) {
		t.Skipf("skipping E2E test: sample file %s not found", sampleMP3)
	}

	ffmpegBin, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skipf("skipping E2E test: ffmpeg not found in PATH")
	}

	tmpDir := t.TempDir()
	ensureKeyAvailable(t, tmpDir)

	m4a8chPath := filepath.Join(tmpDir, "sample_8ch.m4a")
	stemsPath := filepath.Join(tmpDir, "sample.stems")
	decryptedPath := filepath.Join(tmpDir, "decrypted.m4a")
	extractDir := filepath.Join(tmpDir, "extracted-stems")

	// Step 1: Merge 30s MP3 into 8-channel AAC M4A using FFmpeg
	t.Log("Step 1: Merging 30s MP3 into 8-channel AAC M4A...")
	ffmpegArgs := []string{
		"-y", "-hide_banner",
		"-i", sampleMP3,
		"-filter_complex", "[0:a][0:a][0:a][0:a]amerge=inputs=4[a]",
		"-map", "[a]",
		"-c:a", "aac",
		"-b:a", "640k",
		m4a8chPath,
	}

	cmd := exec.Command(ffmpegBin, ffmpegArgs...)
	if err := cmd.Run(); err != nil {
		t.Fatalf("ffmpeg 8-channel merge failed: %v", err)
	}

	m4aInfo, err := os.Stat(m4a8chPath)
	if err != nil || m4aInfo.Size() == 0 {
		t.Fatalf("merged 8-channel M4A file is empty or missing")
	}

	// Step 2: Encrypt 8-channel M4A to .stems file using CLI encrypt command
	t.Log("Step 2: Encrypting M4A to Engine DJ .stems container...")
	resetFlags()
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"encrypt", "-o", stemsPath, m4a8chPath})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("encrypt command failed: %v", err)
	}

	stemsInfo, err := os.Stat(stemsPath)
	if err != nil || stemsInfo.Size() == 0 {
		t.Fatalf(".stems file is empty or missing")
	}

	// Step 3: Run info command and verify encryption status & container atom metadata
	t.Log("Step 3: Running info command on .stems file...")
	resetFlags()
	buf.Reset()
	RootCmd.SetArgs([]string{"info", "--json", stemsPath})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	var meta mp4.Metadata
	if err := json.Unmarshal(buf.Bytes(), &meta); err != nil {
		t.Fatalf("parsing info JSON output: %v", err)
	}

	if !meta.IsEncrypted {
		t.Errorf("info command reported IsEncrypted = false, want true")
	}
	if meta.ChannelCount != 8 {
		t.Errorf("info command reported ChannelCount = %d, want 8", meta.ChannelCount)
	}
	if meta.SampleCount == 0 {
		t.Errorf("info command reported 0 samples")
	}

	// Step 4: Decrypt .stems file back to M4A using CLI decrypt command
	t.Log("Step 4: Decrypting .stems file back to M4A...")
	resetFlags()
	buf.Reset()
	RootCmd.SetArgs([]string{"decrypt", "-o", decryptedPath, stemsPath})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("decrypt command failed: %v", err)
	}

	decInfo, err := os.Stat(decryptedPath)
	if err != nil {
		t.Fatalf("decrypted M4A file missing")
	}

	if decInfo.Size() != m4aInfo.Size() {
		t.Errorf("decrypted file size %d != original merged file size %d", decInfo.Size(), m4aInfo.Size())
	}

	// Step 5: Extract 4 stereo stem WAV files from .stems container
	t.Log("Step 5: Extracting stems from .stems file...")
	resetFlags()
	buf.Reset()
	RootCmd.SetArgs([]string{"extract", "-o", extractDir, stemsPath})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("extract command failed: %v", err)
	}

	// Step 6: Verify all 4 extracted WAV stem files exist and are non-empty
	stems := []string{"vocals.wav", "drums.wav", "bass.wav", "other.wav"}
	for _, stem := range stems {
		stemPath := filepath.Join(extractDir, stem)
		info, err := os.Stat(stemPath)
		if err != nil {
			t.Fatalf("extracted stem file %s missing: %v", stem, err)
		}
		if info.Size() < 1000000 {
			t.Errorf("extracted stem file %s size %d is suspiciously small", stem, info.Size())
		}
	}

	// Step 7: Verify extracted stem audio stream format via ffprobe
	ffprobeBin, err := exec.LookPath("ffprobe")
	if err == nil {
		vocalsWav := filepath.Join(extractDir, "vocals.wav")
		probeCmd := exec.Command(ffprobeBin, "-v", "error",
			"-show_entries", "stream=codec_name,sample_rate,channels",
			"-of", "default=noprint_wrappers=1", vocalsWav)

		probeOut, err := probeCmd.Output()
		if err != nil {
			t.Fatalf("ffprobe failed on extracted WAV: %v", err)
		}

		outStr := string(probeOut)
		if !strings.Contains(outStr, "codec_name=pcm_s16le") {
			t.Errorf("expected pcm_s16le codec, got ffprobe output: %s", outStr)
		}
		if !strings.Contains(outStr, "sample_rate=44100") {
			t.Errorf("expected sample_rate=44100, got ffprobe output: %s", outStr)
		}
		if !strings.Contains(outStr, "channels=2") {
			t.Errorf("expected channels=2, got ffprobe output: %s", outStr)
		}
	}

	t.Log("E2E Round-trip test passed successfully!")
}
