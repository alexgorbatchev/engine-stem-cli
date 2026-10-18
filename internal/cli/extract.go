package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alexgorbatchev/engine-stem-cli/internal/mp4"
	"github.com/alexgorbatchev/engine-stem-cli/internal/stems"
	"github.com/spf13/cobra"
)

var (
	extractOutputDir string
	extractFormat    string
	extractDryRun    bool
)

var extractCmd = &cobra.Command{
	Use:   "extract <file> [output-dir]",
	Short: "Extract 4 stereo stems (vocals, drums, bass, other) via FFmpeg",
	Long: `Extract decrypts the target file (if encrypted) and splits the 8 audio channels
into 4 stereo audio files (vocals, drums, bass, other) using FFmpeg.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]
		outDir := extractOutputDir

		if outDir == "" {
			if len(args) == 2 {
				outDir = args[1]
			} else {
				base := filepath.Base(inputPath)
				ext := filepath.Ext(base)
				outDir = strings.TrimSuffix(base, ext) + "-stems"
			}
		}

		format := strings.ToLower(extractFormat)
		if format != "wav" && format != "m4a" {
			return fmt.Errorf("invalid format %q: must be 'wav' or 'm4a'", extractFormat)
		}

		data, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("reading file %q: %w", inputPath, err)
		}

		meta, err := mp4.InspectContainer(data)
		if err != nil {
			return fmt.Errorf("inspecting file: %w", err)
		}

		m4aPath := inputPath
		var tempM4A string

		if meta.IsEncrypted {
			tempFile, err := os.CreateTemp("", "engine-stem-cli-decrypted-*.m4a")
			if err != nil {
				return fmt.Errorf("creating temp file: %w", err)
			}
			tempM4A = tempFile.Name()
			tempFile.Close()
			defer os.Remove(tempM4A)

			if !flagQuiet {
				PrintInfo(cmd.OutOrStdout(), fmt.Sprintf("Decrypting %s for stem extraction...", inputPath))
			}
			if err := stems.Decrypt(inputPath, tempM4A, flagVerbose); err != nil {
				return fmt.Errorf("decrypting input: %w", err)
			}
			m4aPath = tempM4A
		}

		vocalsPath := filepath.Join(outDir, "vocals."+format)
		drumsPath := filepath.Join(outDir, "drums."+format)
		bassPath := filepath.Join(outDir, "bass."+format)
		otherPath := filepath.Join(outDir, "other."+format)

		filterComplex := "[0:a]pan=stereo|c0=c0|c1=c1[s1];[0:a]pan=stereo|c0=c2|c1=c3[s2];[0:a]pan=stereo|c0=c4|c1=c5[s3];[0:a]pan=stereo|c0=c6|c1=c7[s4]"

		ffmpegArgs := []string{
			"-y", "-hide_banner",
			"-strict", "-2",
			"-i", m4aPath,
			"-filter_complex", filterComplex,
			"-map", "[s1]", vocalsPath,
			"-map", "[s2]", drumsPath,
			"-map", "[s3]", bassPath,
			"-map", "[s4]", otherPath,
		}

		ffmpegCmdStr := fmt.Sprintf("ffmpeg %s", strings.Join(ffmpegArgs, " "))

		if extractDryRun {
			if IsAgentMode() {
				fmt.Fprintf(cmd.OutOrStdout(), "dry_run: %s\n", ffmpegCmdStr)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Dry run - FFmpeg command:\n%s\n", ffmpegCmdStr)
			}
			return nil
		}

		ffmpegBin, err := exec.LookPath("ffmpeg")
		if err != nil {
			PrintErr(os.Stderr, "FFmpeg executable not found in PATH")
			if !IsAgentMode() {
				fmt.Fprintf(os.Stderr, "To extract stems manually, run:\n  %s\n", ffmpegCmdStr)
			}
			return fmt.Errorf("ffmpeg not found in PATH: %w", err)
		}

		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("creating output directory %q: %w", outDir, err)
		}

		if !flagQuiet {
			PrintInfo(cmd.OutOrStdout(), fmt.Sprintf("Extracting stems into %s/...", outDir))
		}

		c := exec.Command(ffmpegBin, ffmpegArgs...)
		if flagVerbose {
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
		}

		if err := c.Run(); err != nil {
			return fmt.Errorf("executing ffmpeg: %w", err)
		}

		if !flagQuiet {
			PrintOK(cmd.OutOrStdout(), "Successfully extracted stems:")
			if IsAgentMode() {
				fmt.Fprintf(cmd.OutOrStdout(), "* %s\n* %s\n* %s\n* %s\n", vocalsPath, drumsPath, bassPath, otherPath)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n  - %s\n  - %s\n  - %s\n", vocalsPath, drumsPath, bassPath, otherPath)
			}
		}

		return nil
	},
}

func init() {
	extractCmd.Flags().StringVarP(&extractOutputDir, "output-dir", "o", "", "Output directory for stems")
	extractCmd.Flags().StringVar(&extractFormat, "format", "wav", "Output audio format (wav or m4a)")
	extractCmd.Flags().BoolVar(&extractDryRun, "dry-run", false, "Print FFmpeg command without executing")
}
