package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func isolateConfig(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return filepath.Join(os.Getenv("HOME"), ".mino", "config.json")
}
func userDirectory() (string, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".mino")
	return dir, os.MkdirAll(dir, 0700)
}
func openHistory() (*Session, error) {
	dir, err := userDirectory()
	if err != nil {
		return nil, err
	}
	return OpenSession(dir)
}
