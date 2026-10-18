// Package inference provides pure Go ONNX model execution for stem separation.
package inference

import (
	"fmt"
	"os"

	ort "github.com/yalue/onnxruntime_go"
)

// Session manages an ONNX Runtime inference session for audio stem separation.
type Session struct {
	modelPath string
	session   *ort.DynamicAdvancedSession
	inputName string
	outputNames []string
}

// InitONNXRuntime initializes the global ONNX Runtime library if not already initialized.
func InitONNXRuntime(sharedLibPath string) error {
	if ort.IsInitialized() {
		return nil
	}
	if sharedLibPath != "" {
		ort.SetSharedLibraryPath(sharedLibPath)
	}
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("initializing ONNX runtime environment: %w", err)
	}
	return nil
}

// NewSession creates a new ONNX inference session for a model file.
func NewSession(modelPath string) (*Session, error) {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file %q does not exist", modelPath)
	}

	if !ort.IsInitialized() {
		if err := InitONNXRuntime(""); err != nil {
			return nil, err
		}
	}

	inputNames := []string{"input"}
	outputNames := []string{"output"}

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		inputNames,
		outputNames,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("creating ONNX session for %q: %w", modelPath, err)
	}

	return &Session{
		modelPath:   modelPath,
		session:     session,
		inputName:   "input",
		outputNames: outputNames,
	}, nil
}

// Close destroys the ONNX inference session.
func (s *Session) Close() error {
	if s.session != nil {
		if err := s.session.Destroy(); err != nil {
			return fmt.Errorf("destroying ONNX session: %w", err)
		}
		s.session = nil
	}
	return nil
}

// Separate runs ONNX model inference on audio tensor data.
// Input tensor shape: [1, 2, num_samples] (normalized float32 audio [-1.0, 1.0]).
// Output tensor shape: [1, 4, 2, num_samples] (4 stereo stems: Vocals, Drums, Bass, Other).
func (s *Session) Separate(audioData []float32, channels, numSamples int) ([][][]float32, error) {
	if len(audioData) != channels*numSamples {
		return nil, fmt.Errorf("audio data length %d mismatch with channels=%d, numSamples=%d",
			len(audioData), channels, numSamples)
	}

	inputShape := ort.NewShape(1, int64(channels), int64(numSamples))
	inputTensor, err := ort.NewTensor(inputShape, audioData)
	if err != nil {
		return nil, fmt.Errorf("creating input tensor: %w", err)
	}
	defer inputTensor.Destroy()

	outShape := ort.NewShape(1, 4, int64(channels), int64(numSamples))
	outData := make([]float32, 1*4*channels*numSamples)
	outTensor, err := ort.NewTensor(outShape, outData)
	if err != nil {
		return nil, fmt.Errorf("creating output tensor: %w", err)
	}
	defer outTensor.Destroy()

	inputs := []ort.ArbitraryTensor{inputTensor}
	outputs := []ort.ArbitraryTensor{outTensor}

	if err := s.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("running ONNX inference: %w", err)
	}

	// Unpack output float32 slice into [4][2][numSamples]
	stems := make([][][]float32, 4)
	for i := 0; i < 4; i++ {
		stems[i] = make([][]float32, channels)
		for ch := 0; ch < channels; ch++ {
			stems[i][ch] = make([]float32, numSamples)
			base := (i*channels + ch) * numSamples
			copy(stems[i][ch], outData[base:base+numSamples])
		}
	}

	return stems, nil
}
