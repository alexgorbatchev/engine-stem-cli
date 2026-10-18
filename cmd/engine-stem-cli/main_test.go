package main

import (
	"os"
	"testing"
)

func TestMainFunc(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"engine-stem-cli", "--version"}
	main()
}
