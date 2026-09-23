package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const configLocation = "~/.mino/config.json"

type config struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

func loadConfig(prompt func(label, fallback string, secret bool) (string, error)) (config, error) {
	path, err := configPath()
	if err != nil {
		return config{}, err
	}
	if err := checkConfigFile(path); err != nil {
		return config{}, err
	}
	var cfg config
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return config{}, fmt.Errorf("Invalid JSON in %s. Fix the file and restart.", configLocation)
		}
		if err := os.Chmod(path, 0600); err != nil {
			return config{}, fmt.Errorf("Failed to secure config file permissions: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return config{}, fmt.Errorf("Failed to read config: %w", err)
	}
	cfg.normalize()
	if cfg.BaseURL != "" {
		if err := validateBaseURL(cfg.BaseURL); err != nil {
			return config{}, err
		}
	}
	changed := false
	for _, field := range []struct {
		value           *string
		label, fallback string
		secret          bool
	}{
		{&cfg.BaseURL, "API URL", "https://api.openai.com/v1", false},
		{&cfg.Model, "Model", "", false},
		{&cfg.APIKey, "API Key", "", true},
	} {
		if *field.value != "" {
			continue
		}
		if prompt == nil {
			return config{}, fmt.Errorf("Missing %s. Start Mino in a terminal to complete setup.", field.label)
		}
		value, err := prompt(field.label, field.fallback, field.secret)
		if err != nil {
			return config{}, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			value = field.fallback
		}
		*field.value = value
		changed = true
	}
	if err := cfg.validate(); err != nil {
		return config{}, err
	}
	if changed {
		if err := saveConfig(cfg); err != nil {
			return config{}, err
		}
	}
	return cfg, nil
}

func (c *config) normalize() {
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.Model = strings.TrimSpace(c.Model)
}

func (c *config) validate() error {
	c.normalize()
	if c.APIKey == "" {
		return fmt.Errorf("api_key must not be empty in %s", configLocation)
	}
	if c.Model == "" {
		return fmt.Errorf("model must not be empty in %s", configLocation)
	}
	if err := validateBaseURL(c.BaseURL); err != nil {
		return err
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	return nil
}

func validateBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") ||
		u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return fmt.Errorf("base_url in %s must be an absolute HTTP(S) URL without credentials, query parameters, or fragments", configLocation)
	}
	// 仅允许本地调试使用明文 HTTP，远程地址必须通过 HTTPS 保护密钥。
	if u.Scheme == "http" && u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback() {
		return fmt.Errorf("base_url in %s must use HTTPS for remote servers; HTTP is only allowed for local servers", configLocation)
	}
	return nil
}

// 配置位置只取决于用户主目录，不随启动时的工作目录变化。
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Failed to locate your home directory: %w", err)
	}
	if !filepath.IsAbs(home) {
		return "", fmt.Errorf("Your home directory must be an absolute path")
	}
	directory := filepath.Join(home, ".mino")
	if err := os.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("Failed to create ~/.mino: %w", err)
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return "", fmt.Errorf("Failed to check ~/.mino: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("~/.mino must be a directory, not a file or symbolic link")
	}
	if err := os.Chmod(directory, 0700); err != nil {
		return "", fmt.Errorf("Failed to secure ~/.mino permissions: %w", err)
	}
	return filepath.Join(directory, "config.json"), nil
}

func checkConfigFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Failed to check config file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular file, not a directory or symbolic link", configLocation)
	}
	return nil
}

func saveConfig(cfg config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := checkConfigFile(path); err != nil {
		return err
	}
	// 临时文件默认是 0600；写完再重命名，避免中断后留下半份配置。
	file, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("Failed to create config file: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("Failed to write config: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Failed to close config file: %w", err)
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("Failed to save config: %w", err)
	}
	return nil
}

func loadInstructions() (string, error) {
	data, err := os.ReadFile("AGENTS.md")
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("Failed to read AGENTS.md: %w", err)
	}
	return string(data), nil
}
