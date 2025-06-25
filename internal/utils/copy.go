package utils

import "golang.design/x/clipboard"

func CopyToClipboard(input string) error {
	err := clipboard.Init()
	if err != nil {
		return err
	}

	clipboard.Write(clipboard.FmtText, []byte(input))

	return nil
}
