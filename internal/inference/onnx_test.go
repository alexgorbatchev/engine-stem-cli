package inference

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSessionNonExistentModel(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "missing.onnx")

	session, err := NewSession(nonExistent)
	if err == nil {
		if session != nil {
			session.Close()
		}
		t.Errorf("NewSession expected error for missing model, got nil")
	}
}

func TestSessionCloseNil(t *testing.T) {
	s := &Session{}
	if err := s.Close(); err != nil {
		t.Errorf("Close on empty session returned error: %v", err)
	}
}

func TestNewSessionInvalidModelData(t *testing.T) {
	tmpDir := t.TempDir()
	invalidModel := filepath.Join(tmpDir, "invalid.onnx")
	os.WriteFile(invalidModel, []byte("not an onnx file"), 0644)

	session, err := NewSession(invalidModel)
	if err == nil {
		if session != nil {
			session.Close()
		}
		t.Errorf("NewSession expected error for invalid ONNX binary, got nil")
	}
}

func TestSeparateDataValidation(t *testing.T) {
	s := &Session{}
	audioData := []float32{0.1, 0.2, 0.3}

	// Mismatched length (len=3, channels=2, numSamples=2 -> expected 4)
	_, err := s.Separate(audioData, 2, 2)
	if err == nil {
		t.Errorf("Separate expected error for mismatched audio length, got nil")
	}
}
