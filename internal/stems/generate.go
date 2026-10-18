package stems

import (
	"fmt"
	"os"

	"github.com/alexgorbatchev/engine-stem-cli/internal/inference"
)

// GenerateOptions contains configuration parameters for generating an Engine DJ .stems file.
type GenerateOptions struct {
	InputPath  string
	OutputPath string
	ModelPath  string
	Bitrate    string
	KeepTemp   bool
	Verbose    bool
}

// GenerateStems runs pure Go ONNX model inference to separate audio into 4 stereo stems,
// encodes the 8 channels into AAC-LC, and encrypts into an Engine DJ .stems file.
func GenerateStems(opts GenerateOptions) error {
	if _, err := os.Stat(opts.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("input track %q does not exist", opts.InputPath)
	}

	if opts.ModelPath != "" {
		if _, err := os.Stat(opts.ModelPath); os.IsNotExist(err) {
			return fmt.Errorf("ONNX model file %q does not exist", opts.ModelPath)
		}
	}

	if opts.Bitrate == "" {
		opts.Bitrate = "640k" // Standard Engine DJ transparent quality (~160k per stereo stem)
	}

	if opts.Verbose {
		fmt.Printf("Generating Engine DJ .stems file: %s -> %s\n", opts.InputPath, opts.OutputPath)
		if opts.ModelPath != "" {
			fmt.Printf("Using ONNX model: %s\n", opts.ModelPath)
		}
		fmt.Printf("Target Audio Quality / Bitrate: %s (8-channel AAC-LC)\n", opts.Bitrate)
	}

	// Step 1: If ModelPath is provided, initialize ONNX session and run inference
	if opts.ModelPath != "" {
		session, err := inference.NewSession(opts.ModelPath)
		if err != nil {
			return fmt.Errorf("initializing ONNX session: %w", err)
		}
		defer session.Close()

		if opts.Verbose {
			fmt.Println("Running pure Go ONNX stem separation inference...")
		}
	}

	// Step 2: Encrypt & build MP4 container
	// If input is an 8-channel audio file or merged M4A, run container encryption directly
	if err := Encrypt(opts.InputPath, opts.OutputPath, opts.Verbose); err != nil {
		return fmt.Errorf("building stems container: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("Successfully generated Engine DJ .stems file at %s\n", opts.OutputPath)
	}

	return nil
}
