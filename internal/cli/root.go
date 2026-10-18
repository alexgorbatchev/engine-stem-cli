// Package cli implements the command-line interface for engine-stem-cli.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is populated at build time or defaults to 1.0.0.
	Version = "1.0.0"

	// Flags
	flagVerbose bool
	flagQuiet   bool
	flagVersion bool
)

// RootCmd represents the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:   "engine-stem-cli",
	Short: "Engine DJ stems audio file utility",
	Long: `engine-stem-cli is a CLI tool for encrypting, decrypting, inspecting,
and extracting audio stems from Engine DJ .stems files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagVersion {
			fmt.Fprintln(cmd.OutOrStdout(), Version)
			return nil
		}
		return cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		PrintErr(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "", false, "Enable verbose diagnostic logging")
	RootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-error output")
	RootCmd.Flags().BoolVarP(&flagVersion, "version", "v", false, "Show version and build information")

	RootCmd.AddCommand(decryptCmd)
	RootCmd.AddCommand(encryptCmd)
	RootCmd.AddCommand(infoCmd)
	RootCmd.AddCommand(extractCmd)
}
