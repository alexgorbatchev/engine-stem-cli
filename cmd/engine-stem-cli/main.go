// Command engine-stem-cli is a CLI tool for encrypting, decrypting, inspecting,
// and extracting audio stems from Engine DJ .stems files.
package main

import (
	"github.com/alexgorbatchev/engine-stem-cli/internal/cli"
)

func main() {
	cli.Execute()
}
