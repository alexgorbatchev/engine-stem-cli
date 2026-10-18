package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexgorbatchev/engine-stem-cli/internal/stems"
	"github.com/spf13/cobra"
)

var (
	generateOutput  string
	generateModel   string
	generateBitrate string
	generateForce   bool
	generateDryRun  bool
)

var generateCmd = &cobra.Command{
	Use:   "generate <input-track> [output.stems]",
	Short: "Generate an Engine DJ .stems file from a source audio track",
	Long:  "Generate runs source separation on an audio track and packages it into an Engine DJ stem container.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]
		outputPath := generateOutput

		if outputPath == "" {
			if len(args) == 2 {
				outputPath = args[1]
			} else {
				ext := filepath.Ext(inputPath)
				outputPath = strings.TrimSuffix(inputPath, ext) + ".stems"
			}
		}

		out := cmd.OutOrStdout()

		if generateDryRun {
			if IsAgentMode() {
				fmt.Fprintf(out, "dry_run: generate %s -> %s (model: %s, bitrate: %s)\n", inputPath, outputPath, generateModel, generateBitrate)
			} else {
				fmt.Fprintf(out, "Dry run - Steps to generate %s from %s:\n", outputPath, inputPath)
				if generateModel != "" {
					fmt.Fprintf(out, "  1. Pure Go ONNX Model Inference (%s)\n", generateModel)
				} else {
					fmt.Fprintf(out, "  1. Pure Go ONNX Model Inference (default ONNX model)\n")
				}
				fmt.Fprintf(out, "  2. Encode 8-channel AAC-LC audio (bitrate: %s, matching source fidelity)\n", generateBitrate)
				fmt.Fprintf(out, "  3. AES-128-ECB sample encryption with PKCS#7 padding\n")
				fmt.Fprintf(out, "  4. Build MP4 container with stbl sample tables to %s\n", outputPath)
			}
			return nil
		}

		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			return fmt.Errorf("input file %q does not exist", inputPath)
		}

		if !generateForce {
			if _, err := os.Stat(outputPath); err == nil {
				return fmt.Errorf("output file %q already exists (use --force / -f to overwrite)", outputPath)
			}
		}

		if !flagQuiet {
			PrintInfo(out, fmt.Sprintf("Generating Engine DJ stems file %s from %s...", outputPath, inputPath))
		}

		opts := stems.GenerateOptions{
			InputPath:  inputPath,
			OutputPath: outputPath,
			ModelPath:  generateModel,
			Bitrate:    generateBitrate,
			Verbose:    flagVerbose,
		}

		if err := stems.GenerateStems(opts); err != nil {
			return err
		}

		if !flagQuiet {
			PrintOK(out, fmt.Sprintf("Successfully generated %s", outputPath))
		}

		return nil
	},
}

func init() {
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "Explicit output .stems file path")
	generateCmd.Flags().StringVarP(&generateModel, "model", "m", "", "Path to ONNX model file (.onnx)")
	generateCmd.Flags().StringVarP(&generateBitrate, "bitrate", "b", "640k", "Total AAC audio bitrate (default: 640k = ~160k/stem)")
	generateCmd.Flags().BoolVarP(&generateForce, "force", "f", false, "Overwrite existing output file")
	generateCmd.Flags().BoolVar(&generateDryRun, "dry-run", false, "Print generation pipeline summary without executing")

	RootCmd.AddCommand(generateCmd)
}
