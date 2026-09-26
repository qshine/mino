package mino

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFirstRunSavesChosenModelAndSkipsPromptsOnRestart(t *testing.T) {
	path := isolateConfig(t)
	calls := 0
	cfg, err := loadConfig(func(label, fallback string, secret bool) (string, error) {
		calls++
		switch calls {
		case 1:
			if fallback != "https://api.openai.com/v1" || secret {
				t.Fatal("wrong base URL prompt")
			}
		case 2:
			if fallback != "" || secret {
				t.Fatal("model must not have a default")
			}
			return "chosen-model", nil
		case 3:
			if fallback != "" || !secret {
				t.Fatal("key must be secret with no default")
			}
			return " fake-key ", nil
		default:
			t.Fatal("unexpected prompt")
		}
		return "", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 || cfg != (config{"https://api.openai.com/v1", "fake-key", "chosen-model", 0}) {
		t.Fatal("configuration was not collected correctly")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]string
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["base_url"] != cfg.BaseURL || saved["api_key"] != cfg.APIKey || saved["model"] != cfg.Model || len(saved) != 3 {
		t.Fatal("JSON does not contain the three configuration fields")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}
	directory := filepath.Dir(path)
	dirInfo, err := os.Stat(directory)
	if err != nil || dirInfo.Mode().Perm() != 0700 {
		t.Fatal("config directory must be private")
	}
	entries, err := os.ReadDir(".")
	if err != nil || len(entries) != 0 {
		t.Fatal("configuration must not be stored in the working directory")
	}
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	// 手动放宽权限后，启动时应恢复为仅本人可读写。
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig(func(string, string, bool) (string, error) { t.Fatal("complete config must not prompt"); return "", nil })
	if err != nil || got != cfg {
		t.Fatalf("restart failed: %v", err)
	}
	info, err = os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("restart did not protect config permissions")
	}
	dirInfo, err = os.Stat(directory)
	if err != nil || dirInfo.Mode().Perm() != 0700 {
		t.Fatal("restart did not protect directory permissions")
	}
}

func TestPartialConfigOnlyPromptsMissingField(t *testing.T) {
	path := isolateConfig(t)
	writeTestConfig(t, path, `{"base_url":"https://gateway.example/v1/","api_key":"fake-key","model":" "}`)
	calls := 0
	cfg, err := loadConfig(func(label, fallback string, secret bool) (string, error) {
		calls++
		if label != "Model" || secret {
			t.Fatal("prompted a configured field")
		}
		return "custom-model", nil
	})
	if err != nil || calls != 1 || cfg.Model != "custom-model" || cfg.BaseURL != "https://gateway.example/v1" || cfg.APIKey != "fake-key" {
		t.Fatalf("partial config completion failed: %v", err)
	}
}

func TestMissingModelHasNoDefaultAndCannotBeSavedEmpty(t *testing.T) {
	path := isolateConfig(t)
	const before = `{"base_url":"https://api.openai.com/v1","api_key":"fake-key"}`
	writeTestConfig(t, path, before)
	_, err := loadConfig(func(label, fallback string, secret bool) (string, error) {
		if label != "Model" || fallback != "" || secret {
			t.Fatal("missing model must be requested without a default")
		}
		return "", nil
	})
	if err == nil || !strings.Contains(err.Error(), "model") {
		t.Fatalf("empty model error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != before {
		t.Fatal("empty model was saved")
	}
}

func TestConfigCancellationPreservesExistingFile(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "partial"}[existing], func(t *testing.T) {
			path := isolateConfig(t)
			const before = `{"base_url":"https://gateway.example/v1","api_key":"fake-key"}`
			if existing {
				writeTestConfig(t, path, before)
			}
			_, err := loadConfig(func(string, string, bool) (string, error) { return "", context.Canceled })
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v", err)
			}
			data, err := os.ReadFile(path)
			if existing {
				if err != nil || string(data) != before {
					t.Fatal("partial config was overwritten")
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatal("cancelled setup persisted a file")
			}
		})
	}
}

func TestInvalidConfigDoesNotOverwriteOrLeakSecrets(t *testing.T) {
	for _, before := range []string{
		`{"api_key":"fake-secret",`,
		`{"base_url":"https://user:fake-secret@example.com/v1","api_key":"fake-key","model":"test-model"}`,
	} {
		t.Run(before[:10], func(t *testing.T) {
			path := isolateConfig(t)
			writeTestConfig(t, path, before)
			_, err := loadConfig(func(string, string, bool) (string, error) {
				t.Fatal("invalid file must be fixed first")
				return "", nil
			})
			if err == nil || strings.Contains(err.Error(), "fake-secret") {
				t.Fatalf("unsafe config error = %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != before {
				t.Fatal("invalid config was overwritten")
			}
		})
	}
}

func TestConfigRejectsSymlinks(t *testing.T) {
	path := isolateConfig(t)
	target := filepath.Join(t.TempDir(), "other.json")
	if err := os.Mkdir(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"api_key":"fake-key"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(nil); err == nil {
		t.Fatal("config symlink was accepted")
	}
	if err := saveConfig(config{"https://api.openai.com/v1", "fake-key", "test-model", 0}); err == nil {
		t.Fatal("save replaced a symlink")
	}
}

func TestSaveFailureLeavesNoTemporarySecretFiles(t *testing.T) {
	path := isolateConfig(t)
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := saveConfig(config{"https://api.openai.com/v1", "fake-key", "test-model", 0}); err == nil {
		t.Fatal("saving over a directory succeeded")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatal("save failure left temporary files")
	}
}

func writeTestConfig(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

// 每个配置测试使用独立的用户目录，不能读写开发者的真实密钥。
func isolateConfig(t *testing.T) string {
	t.Helper()
	taskHome := t.TempDir()
	t.Setenv("HOME", taskHome)
	t.Chdir(t.TempDir())
	return filepath.Join(taskHome, ".mino", "config.json")
}
