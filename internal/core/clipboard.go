package core

import "github.com/atotto/clipboard"

// ReadClipboard returns the current clipboard contents. It lives here so the
// TUI never imports a driver directly and headless mode can reuse it later.
func ReadClipboard() (string, error) {
	return clipboard.ReadAll()
}

// WriteClipboard replaces the clipboard contents.
func WriteClipboard(value string) error {
	return clipboard.WriteAll(value)
}
