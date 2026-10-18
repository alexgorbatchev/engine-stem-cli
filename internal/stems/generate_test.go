package stems

import (
	"path/filepath"
	"testing"
)

func TestGenerateStemsValidation(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist.mp3")

	// Non-existent input file
	opts := GenerateOptions{
		InputPath:  nonExistent,
		OutputPath: filepath.Join(tmpDir, "out.stems"),
	}
	if err := GenerateStems(opts); err == nil {
		t.Errorf("GenerateStems non-existent input expected error, got nil")
	}

	// Non-existent model path
	origPath := createTestMP4(t)
	optsModel := GenerateOptions{
		InputPath:  origPath,
		OutputPath: filepath.Join(tmpDir, "out.stems"),
		ModelPath:  filepath.Join(tmpDir, "missing.onnx"),
	}
	if err := GenerateStems(optsModel); err == nil {
		t.Errorf("GenerateStems non-existent model expected error, got nil")
	}

	// Valid input file with default options
	optsValid := GenerateOptions{
		InputPath:  origPath,
		OutputPath: filepath.Join(tmpDir, "out.stems"),
		Bitrate:    "640k",
		Verbose:    true,
	}
	if err := GenerateStems(optsValid); err != nil {
		t.Fatalf("GenerateStems valid input error = %v", err)
	}
}
