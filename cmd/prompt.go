package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/flacial/llm/internal/log"
)

func getPromptContent(cliArgs []string, promptFilePath string) (string, error) {
	var finalPrompt string
	var stdinContent string

	stats, _ := os.Stdin.Stat()
	// Check if stdin is piped
	// We doing bitwise ops baby!
	if (stats.Mode() & os.ModeCharDevice) == 0 {
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("error reading stdin: %w", err)
		}

		stdinContent = strings.TrimSpace(string(stdinBytes))
	}

	fileContent := ""
	if promptFilePath != "" {
		fileBytes, err := os.ReadFile(promptFilePath)
		if err != nil {
			return "", fmt.Errorf("error reading prompt file %q: %w", promptFilePath, err)
		}

		fileContent = strings.TrimSpace(string(fileBytes))
	}

	cliPrompt := strings.TrimSpace(strings.Join(cliArgs, " "))

	// Determine final prompt based on this order: file > stdin > cli
	if fileContent != "" {
		finalPrompt = fileContent

		if cliPrompt != "" || stdinContent != "" {
			log.Logger.Warn().Msg("Warning: File content takes precedence. CLI arguments and stdin will be ignored.")
		}
	} else if stdinContent != "" && cliPrompt != "" {
		log.Logger.Info().Msg("Using stdin content and CLI prompt")
		finalPrompt = cliPrompt + "\n\n" + stdinContent
	} else if stdinContent != "" {
		log.Logger.Info().Msg("Using stdin content")
		finalPrompt = stdinContent
	} else if cliPrompt != "" {
		log.Logger.Info().Msg("Using CLI prompt")
		finalPrompt = cliPrompt
	} else {
		return "", errors.New("no prompt provided. Use 'llm \"your prompt\"', pipe input, or specify a file with -f")
	}

	if finalPrompt == "" {
		return "", errors.New("prompt cannot be empty after combining inputs")
	}

	return finalPrompt, nil
}
