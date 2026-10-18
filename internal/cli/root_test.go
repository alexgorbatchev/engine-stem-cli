package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenamedCommand(t *testing.T) {
	resetFlags()
	flagVersion = false
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	t.Cleanup(func() {
		RootCmd.SetOut(nil)
		RootCmd.SetErr(nil)
		flagVersion = false
		if err := RootCmd.Flags().Set("help", "false"); err != nil {
			t.Error(err)
		}
	})
	RootCmd.SetArgs([]string{"--help"})
	if err := RootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "engine-stem-cli [command]") || strings.Contains(buf.String(), "enginedj-stems") {
		t.Fatalf("unexpected command identity in help: %s", buf.String())
	}
}

func TestExecute(t *testing.T) {
	RootCmd.SetArgs([]string{"--version"})
	Execute()
}
