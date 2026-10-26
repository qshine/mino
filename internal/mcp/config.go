// Package mcp connects Mino to user-configured MCP servers and adapts their tools.
package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"unicode/utf8"
)

const maxConfigBytes = 64 << 10
const maxServers = 16

// Config is ~/.mino/mcp.json. It may contain credentials, so it is private,
// lives only in the user's Mino directory, and is never read from a project.
type Config struct {
	Servers map[string]Server `json:"servers"`
}

// Server is one MCP endpoint. Header and environment values are secrets: they
// are sent only to that server and never displayed, logged, or given to the model.
type Server struct {
	Type    string            `json:"type"` // "http" or "stdio"
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Dir     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Sandbox bool              `json:"sandbox,omitempty"` // stdio only: confine the process with network allowed
}

var serverName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var headerName = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")
var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidName keeps server names short and free of "_", which separates the
// server from the tool in exposed names such as mcp_context7_query-docs.
func ValidName(name string) bool { return len(name) <= 32 && serverName.MatchString(name) }

// Headers that the transport owns or that would let a config impersonate protocol state.
var reservedHeaders = []string{"accept", "connection", "content-length", "content-type", "host", "mcp-protocol-version", "mcp-session-id", "last-event-id", "transfer-encoding"}

// Reserved environment names keep the minimal process environment Mino supplies.
var reservedEnv = []string{"HOME", "LANG", "PATH"}

func validText(text string, limit int) bool {
	return text != "" && len(text) <= limit && utf8.ValidString(text) && !strings.ContainsAny(text, "\x00\r\n")
}

// ValidateURL accepts HTTPS, or plain HTTP only on the loopback interface for local
// servers and tests. Credentials belong in headers, never in the URL.
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || !validText(raw, 2048) || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return errors.New("MCP URL must be an absolute https URL without credentials or a fragment")
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if ip := net.ParseIP(host); host == "localhost" || ip != nil && ip.IsLoopback() {
			return nil
		}
	}
	return errors.New("MCP URL must use https (plain http is allowed only for localhost)")
}

func ValidHeaderName(name string) bool {
	return len(name) <= 128 && headerName.MatchString(name) && !slices.Contains(reservedHeaders, strings.ToLower(name))
}

func ValidEnvName(name string) bool {
	return len(name) <= 128 && envName.MatchString(name) && !slices.Contains(reservedEnv, name)
}

func (s Server) Validate() error {
	switch s.Type {
	case "http":
		if s.Command != "" || len(s.Args) != 0 || s.Dir != "" || len(s.Env) != 0 || s.Sandbox {
			return errors.New("An http server uses only url and headers")
		}
		if err := ValidateURL(s.URL); err != nil {
			return err
		}
		seen := map[string]bool{}
		for name, value := range s.Headers {
			if !ValidHeaderName(name) || seen[strings.ToLower(name)] || !validText(value, 8<<10) {
				return fmt.Errorf("Invalid or duplicate header %q", name)
			}
			seen[strings.ToLower(name)] = true
		}
	case "stdio":
		if s.URL != "" || len(s.Headers) != 0 {
			return errors.New("A stdio server uses only command, args, cwd, and env")
		}
		if !validText(s.Command, 4096) || !filepath.IsAbs(s.Command) || filepath.Clean(s.Command) != s.Command {
			return errors.New("A stdio server command must be a clean absolute path")
		}
		if !validText(s.Dir, 4096) || !filepath.IsAbs(s.Dir) || filepath.Clean(s.Dir) != s.Dir {
			return errors.New("A stdio server cwd must be a clean absolute path")
		}
		if len(s.Args) > 64 {
			return errors.New("A stdio server accepts at most 64 arguments")
		}
		for _, arg := range s.Args {
			if len(arg) > 4096 || !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
				return errors.New("Server arguments must be UTF-8 without NUL")
			}
		}
		for name, value := range s.Env {
			if !ValidEnvName(name) || len(value) > 8<<10 || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
				return fmt.Errorf("Invalid environment variable %q", name)
			}
		}
	default:
		return errors.New(`Server type must be "http" or "stdio"`)
	}
	return nil
}

func (c Config) Validate() error {
	if len(c.Servers) > maxServers {
		return fmt.Errorf("At most %d MCP servers can be configured", maxServers)
	}
	for name, server := range c.Servers {
		if !ValidName(name) {
			return fmt.Errorf("Invalid MCP server name %q: use lowercase letters, digits, and single hyphens (at most 32 characters)", name)
		}
		if err := server.Validate(); err != nil {
			return fmt.Errorf("MCP server %s: %w", name, err)
		}
	}
	return nil
}

// Destination is the safe description of where tool arguments go.
func (s Server) Destination() string {
	if s.Type == "http" {
		return s.URL
	}
	return s.Command
}

// Load reads the private config. A missing file means no servers.
func Load(path string) (Config, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return Config{Servers: map[string]Server{}}, nil
	}
	if err != nil {
		return Config{}, errors.New("Cannot open ~/.mino/mcp.json; it must be a regular file, not a symbolic link")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return Config{}, errors.New("~/.mino/mcp.json must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return Config{}, errors.New("~/.mino/mcp.json may contain credentials and must be private (chmod 600)")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil || len(data) > maxConfigBytes {
		return Config{}, errors.New("~/.mino/mcp.json must be at most 64 KiB")
	}
	var c Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil || decoder.More() {
		return Config{}, errors.New("~/.mino/mcp.json is not valid MCP configuration JSON")
	}
	if c.Servers == nil {
		c.Servers = map[string]Server{}
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save publishes the config atomically with mode 0600.
func Save(path string, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("~/.mino/mcp.json must be a regular file, not a symbolic link")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".mcp-*.tmp")
	if err != nil {
		return fmt.Errorf("Failed to create MCP config: %w", err)
	}
	defer os.Remove(file.Name())
	_, err = file.Write(append(data, '\n'))
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(file.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("Failed to save MCP config: %w", err)
	}
	return nil
}
