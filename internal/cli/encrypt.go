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
	encryptOutput string
	encryptForce  bool
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt <input.m4a> [output.stems]",
	Short: "Encrypt an 8-channel audio file to Engine DJ .stems format",
	Long:  "Encrypt converts an 8-channel audio file into an Engine DJ compatible stem container.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]
		outputPath := encryptOutput

		if outputPath == "" {
			if len(args) == 2 {
				outputPath = args[1]
			} else {
				ext := filepath.Ext(inputPath)
				if strings.EqualFold(ext, ".m4a") || strings.EqualFold(ext, ".mp4") {
					outputPath = strings.TrimSuffix(inputPath, ext) + ".stems"
				} else {
					outputPath = inputPath + ".stems"
				}
			}
		}

		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			return fmt.Errorf("input file %q does not exist", inputPath)
		}

		if !encryptForce {
			if _, err := os.Stat(outputPath); err == nil {
				return fmt.Errorf("output file %q already exists (use --force / -f to overwrite)", outputPath)
			}
		}

		if !flagQuiet {
			PrintInfo(cmd.OutOrStdout(), fmt.Sprintf("Encrypting %s -> %s...", inputPath, outputPath))
		}

		if err := stems.Encrypt(inputPath, outputPath, flagVerbose); err != nil {
			return err
		}

		if !flagQuiet {
			PrintOK(cmd.OutOrStdout(), fmt.Sprintf("Successfully encrypted to %s", outputPath))
		}

		return nil
	},
}

func init() {
	encryptCmd.Flags().StringVarP(&encryptOutput, "output", "o", "", "Explicit output file path")
	encryptCmd.Flags().BoolVarP(&encryptForce, "force", "f", false, "Overwrite existing output file")
}
