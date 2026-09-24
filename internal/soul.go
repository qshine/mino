package mino

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	assets "github.com/qshine/mino"
)

const soulLocation = "~/.mino/SOUL.md"
const maxSoulBytes = 64 << 10

// loadInstructions reads the user's identity once at startup, never project files.
func loadInstructions() (string, error) {
	directory, err := userDirectory()
	if err != nil {
		return "", err
	}
	path := filepath.Join(directory, "SOUL.md")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := createSoul(path); err != nil {
			return "", err
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return "", fmt.Errorf("Failed to check %s: %w", soulLocation, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s must be a regular file, not a directory or symbolic link", soulLocation)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("Failed to read %s: %w", soulLocation, err)
	}
	defer file.Close()
	if err := file.Chmod(0600); err != nil {
		return "", fmt.Errorf("Failed to secure %s permissions: %w", soulLocation, err)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxSoulBytes+1))
	if err != nil {
		return "", fmt.Errorf("Failed to read %s: %w", soulLocation, err)
	}
	if len(data) > maxSoulBytes || !utf8.Valid(data) || strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("%s must contain non-empty UTF-8 text of at most 64 KiB", soulLocation)
	}
	return string(data), nil
}

func createSoul(path string) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".soul-*.tmp")
	if err != nil {
		return fmt.Errorf("Failed to create %s: %w", soulLocation, err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.WriteString(assets.DefaultSoul); err != nil {
		return fmt.Errorf("Failed to write %s: %w", soulLocation, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Failed to close %s: %w", soulLocation, err)
	}
	// Publish a complete file without overwriting a user's or another process's file.
	if err := os.Link(file.Name(), path); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("Failed to save %s: %w", soulLocation, err)
	}
	return nil
}
