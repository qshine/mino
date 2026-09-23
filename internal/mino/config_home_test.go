package mino

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigIgnoresWorkingDirectoryConfig(t *testing.T) {
	path := isolateConfig(t)
	for _, name := range []string{"miniagent.json", "config.json"} {
		if err := os.WriteFile(name, []byte(`{"base_url":"https://wrong.example/v1","api_key":"wrong-key","model":"wrong-model"}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := loadConfig(nil); err == nil {
		t.Fatal("loaded credentials from the working directory")
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		t.Fatal("startup did not create ~/.mino")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("saved incomplete configuration")
	}
}

func TestConfigRejectsInvalidHomeDirectory(t *testing.T) {
	for _, home := range []string{"", "relative-home"} {
		t.Run(home, func(t *testing.T) {
			isolateConfig(t)
			t.Setenv("HOME", home)
			if _, err := loadConfig(nil); err == nil {
				t.Fatal("invalid home directory was accepted")
			}
			if err := saveConfig(config{"https://api.openai.com/v1", "fake-key", "test-model"}); err == nil {
				t.Fatal("saved with invalid home directory")
			}
			entries, err := os.ReadDir(".")
			if err != nil || len(entries) != 0 {
				t.Fatal("invalid home directory created files in the working directory")
			}
		})
	}
}

func TestConfigRejectsUnsafeDirectory(t *testing.T) {
	for _, kind := range []string{"file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			path := isolateConfig(t)
			directory := filepath.Dir(path)
			target := t.TempDir()
			if err := os.Chmod(target, 0755); err != nil {
				t.Fatal(err)
			}
			var err error
			if kind == "file" {
				err = os.WriteFile(directory, []byte("keep this file"), 0600)
			} else {
				err = os.Symlink(target, directory)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := loadConfig(nil); err == nil {
				t.Fatal("invalid config directory was accepted")
			}
			if err := saveConfig(config{"https://api.openai.com/v1", "fake-key", "test-model"}); err == nil {
				t.Fatal("saved into an invalid config directory")
			}
			entries, err := os.ReadDir(target)
			if err != nil || len(entries) != 0 {
				t.Fatal("config followed the directory symlink")
			}
			info, err := os.Stat(target)
			if err != nil || info.Mode().Perm() != 0755 {
				t.Fatal("config changed permissions on the symlink target")
			}
		})
	}
}
