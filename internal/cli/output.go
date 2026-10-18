package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
)

// IsAgentMode returns true if AGENT=1, true, or yes.
func IsAgentMode() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AGENT")))
	return v == "1" || v == "true" || v == "yes"
}

// PrintOK formats success messages.
func PrintOK(w io.Writer, msg string) {
	if IsAgentMode() {
		fmt.Fprintf(w, "OK: %s\n", msg)
	} else {
		fmt.Fprintf(w, "[OK] %s\n", msg)
	}
}

// PrintInfo formats informational messages.
func PrintInfo(w io.Writer, msg string) {
	if IsAgentMode() {
		fmt.Fprintf(w, "INFO: %s\n", msg)
	} else {
		fmt.Fprintf(w, "[INFO] %s\n", msg)
	}
}

// PrintWarn formats warning messages.
func PrintWarn(w io.Writer, msg string) {
	if IsAgentMode() {
		fmt.Fprintf(w, "WARN: %s\n", msg)
	} else {
		fmt.Fprintf(w, "[WARN] %s\n", msg)
	}
}

// PrintErr formats error messages.
func PrintErr(w io.Writer, msg string) {
	if IsAgentMode() {
		fmt.Fprintf(w, "ERR: %s\n", msg)
	} else {
		fmt.Fprintf(w, "[ERROR] %s\n", msg)
	}
}

// RenderDivider prints a full-width terminal divider in human mode and nothing in agent mode.
func RenderDivider(w io.Writer, char rune) {
	if IsAgentMode() {
		return
	}
	width := 80
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if termWidth, _, err := term.GetSize(fd); err == nil && termWidth > 0 {
			width = termWidth
		}
	}
	fmt.Fprintln(w, strings.Repeat(string(char), width))
}

// KeyValue represents a single metadata property.
type KeyValue struct {
	Key   string
	Value string
}

// RenderKeyValues outputs tabular data as a clean table in human mode, or flat key-value pairs in agent mode.
func RenderKeyValues(w io.Writer, title string, pairs []KeyValue) {
	if IsAgentMode() {
		for _, pair := range pairs {
			normalizedKey := strings.ToLower(strings.ReplaceAll(pair.Key, " ", "_"))
			fmt.Fprintf(w, "%s:%s\n", normalizedKey, pair.Value)
		}
		return
	}

	RenderDivider(w, '=')
	fmt.Fprintf(w, " %s\n", title)
	RenderDivider(w, '-')

	table := tablewriter.NewTable(w, tablewriter.WithHeader([]string{"Property", "Value"}))
	for _, pair := range pairs {
		_ = table.Append([]string{pair.Key, pair.Value})
	}
	_ = table.Render()
	RenderDivider(w, '=')
}
