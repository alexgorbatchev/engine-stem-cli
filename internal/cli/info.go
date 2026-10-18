package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexgorbatchev/engine-stem-cli/internal/mp4"
	"github.com/spf13/cobra"
)

var infoJSON bool

var infoCmd = &cobra.Command{
	Use:   "info <file>",
	Short: "Inspect audio file metadata and stem properties",
	Long:  "Inspect displays file properties, channel layout, and encryption status for Engine DJ stem files.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file %q: %w", filePath, err)
		}

		meta, err := mp4.InspectContainer(data)
		if err != nil {
			return err
		}

		if infoJSON {
			output, err := json.MarshalIndent(meta, "", "  ")
			if err != nil {
				return fmt.Errorf("formatting JSON: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(output))
			return nil
		}

		encStatus := "Decrypted / Standard M4A"
		if meta.IsEncrypted {
			encStatus = "Encrypted (Engine DJ Stems)"
		}

		pairs := []KeyValue{
			{Key: "File", Value: filePath},
			{Key: "File Size", Value: fmt.Sprintf("%d bytes", meta.FileSize)},
			{Key: "mdat Offset", Value: fmt.Sprintf("%d (size: %d bytes)", meta.MdatOffset, meta.MdatSize)},
			{Key: "moov Offset", Value: fmt.Sprintf("%d (size: %d bytes)", meta.MoovOffset, meta.MoovSize)},
			{Key: "Sample Count", Value: fmt.Sprintf("%d", meta.SampleCount)},
			{Key: "Chunk Count", Value: fmt.Sprintf("%d", meta.ChunkCount)},
			{Key: "Encryption", Value: encStatus},
			{Key: "Channels", Value: fmt.Sprintf("%d (4 stereo stem pairs)", meta.ChannelCount)},
		}

		RenderKeyValues(cmd.OutOrStdout(), "Audio Stem Metadata", pairs)
		return nil
	},
}

func init() {
	infoCmd.Flags().BoolVar(&infoJSON, "json", false, "Output metadata in JSON format")
}
