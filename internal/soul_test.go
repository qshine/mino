package mino

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadInstructionsCreatesPrivateSoulAndIgnoresProjectFiles(t *testing.T) {
	path := filepath.Join(filepath.Dir(isolateConfig(t)), "SOUL.md")
	for _, name := range []string{"AGENTS.md", "SOUL.md"} {
		if err := os.WriteFile(name, []byte("project-only instructions"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := loadInstructions()
	if err != nil || !strings.Contains(got, "You are Mino") || strings.Contains(got, "project-only") {
		t.Fatalf("default instructions = %q, error = %v", got, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != got {
		t.Fatalf("saved SOUL.md does not match instructions: %v", err)
	}
	for name, mode := range map[string]os.FileMode{filepath.Dir(path): 0700, path: 0600} {
		info, err := os.Stat(name)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("incorrect permissions for %s: %v", name, err)
		}
	}
}

func TestLoadInstructionsPreservesUserSoulAndReadsEdits(t *testing.T) {
	path := filepath.Join(filepath.Dir(isolateConfig(t)), "SOUL.md")
	if err := os.Mkdir(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"You are Mino. 请简短回答。\n", "You are Mino. Explain with examples.\n"} {
		if err := os.WriteFile(path, []byte(want), 0600); err != nil {
			t.Fatal(err)
		}
		t.Chdir(t.TempDir())
		if err := os.Mkdir("AGENTS.md", 0700); err != nil {
			t.Fatal(err)
		}
		got, err := loadInstructions()
		if err != nil || got != want {
			t.Fatalf("user instructions = %q, error = %v", got, err)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != want {
			t.Fatal("loading instructions overwrote the user's SOUL.md")
		}
	}
}

func TestLoadInstructionsRejectsInvalidSoul(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":     []byte(" \n\t"),
		"too large": []byte(strings.Repeat("x", (64<<10)+1)),
		"not UTF-8": {0xff, 0xfe},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(filepath.Dir(isolateConfig(t)), "SOUL.md")
			if err := os.Mkdir(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadInstructions(); err == nil || !strings.Contains(err.Error(), "SOUL.md") {
				t.Fatalf("invalid SOUL.md error = %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != string(data) {
				t.Fatal("invalid SOUL.md was overwritten")
			}
		})
	}
}

func TestLoadInstructionsRejectsUnsafeSoul(t *testing.T) {
	for _, kind := range []string{"directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(filepath.Dir(isolateConfig(t)), "SOUL.md")
			if err := os.Mkdir(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "private.txt")
			if err := os.WriteFile(target, []byte("private target"), 0644); err != nil {
				t.Fatal(err)
			}
			var err error
			if kind == "directory" {
				err = os.Mkdir(path, 0700)
			} else {
				err = os.Symlink(target, path)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := loadInstructions(); err == nil {
				t.Fatal("accepted a non-regular SOUL.md")
			}
			info, err := os.Stat(target)
			if err != nil || info.Mode().Perm() != 0644 {
				t.Fatal("loading SOUL.md changed the symlink target")
			}
		})
	}
}

func TestLoadInstructionsConcurrentInitialization(t *testing.T) {
	path := filepath.Join(filepath.Dir(isolateConfig(t)), "SOUL.md")
	results := make(chan error, 8)
	for range cap(results) {
		go func() {
			_, err := loadInstructions()
			results <- err
		}()
	}
	for range cap(results) {
		if err := <-results; err != nil {
			t.Errorf("concurrent initialization: %v", err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "You are Mino") {
		t.Fatal("concurrent initialization did not save a complete identity")
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(files) != 1 || files[0].Name() != "SOUL.md" {
		t.Fatalf("initialization left temporary files: %v, %v", files, err)
	}
}
