package main

import (
	"os"

	"github.com/flacial/llm/cmd"
)

func main() {
	// PNOTE: runs whatever in the cmd root.go file
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
