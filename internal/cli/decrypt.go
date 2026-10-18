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
	decryptOutput string
	decryptForce  bool
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt <input.stems> [output.m4a]",
	Short: "Decrypt an Engine DJ .stems file to standard audio format",
	Long:  "Decrypt converts an encrypted Engine DJ stem file into standard 8-channel audio.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]
		outputPath := decryptOutput

		if outputPath == "" {
			if len(args) == 2 {
				outputPath = args[1]
			} else {
				ext := filepath.Ext(inputPath)
				if strings.EqualFold(ext, ".stems") {
					outputPath = strings.TrimSuffix(inputPath, ext) + ".m4a"
				} else {
					outputPath = inputPath + ".m4a"
				}
			}
		}

		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			return fmt.Errorf("input file %q does not exist", inputPath)
		}

		if !decryptForce {
			if _, err := os.Stat(outputPath); err == nil {
				return fmt.Errorf("output file %q already exists (use --force / -f to overwrite)", outputPath)
			}
		}

		if !flagQuiet {
			PrintInfo(cmd.OutOrStdout(), fmt.Sprintf("Decrypting %s -> %s...", inputPath, outputPath))
		}

		if err := stems.Decrypt(inputPath, outputPath, flagVerbose); err != nil {
			return err
		}

		if !flagQuiet {
			PrintOK(cmd.OutOrStdout(), fmt.Sprintf("Successfully decrypted to %s", outputPath))
		}

		return nil
	},
}

func init() {
	decryptCmd.Flags().StringVarP(&decryptOutput, "output", "o", "", "Explicit output file path")
	decryptCmd.Flags().BoolVarP(&decryptForce, "force", "f", false, "Overwrite existing output file")
}
