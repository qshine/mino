package mino

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/mcp"
)

func mcpCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var output bytes.Buffer
	err := runCLI(context.Background(), "dev", append([]string{"mcp"}, args...), strings.NewReader(""), &output, io.Discard)
	return output.String(), err
}

func TestMCPCommandManagesPrivateConfig(t *testing.T) {
	path := filepath.Join(filepath.Dir(isolateConfig(t)), "mcp.json")
	if out, err := mcpCommand(t, "list"); err != nil || !strings.Contains(out, "No MCP servers") {
		t.Fatalf("empty list: %q %v", out, err)
	}
	if out, err := mcpCommand(t, "add", "context7", "https://mcp.context7.com/mcp"); err != nil || !strings.Contains(out, "Added MCP server context7") {
		t.Fatalf("add: %q %v", out, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("config file: %v %v", info, err)
	}
	if _, err := mcpCommand(t, "add", "local", "--cwd", t.TempDir(), "--", "true", "--flag"); err != nil {
		t.Fatalf("stdio add: %v", err)
	}
	config, err := mcp.Load(path)
	if err != nil || config.Servers["local"].Command == "true" || !filepath.IsAbs(config.Servers["local"].Command) || config.Servers["local"].Args[0] != "--flag" {
		t.Fatalf("stdio config: %+v %v", config.Servers["local"], err)
	}
	for _, args := range [][]string{
		{"add", "context7", "https://other.example/mcp"},      // duplicate
		{"add", "plain", "http://example.com/mcp"},            // not https
		{"add", "Bad_Name", "https://example.com/mcp"},        // invalid name
		{"add", "key", "https://example.com/mcp", "--bearer"}, // secrets need a terminal
		{"add", "hdr", "https://example.com/mcp", "--header", "Host"},
		{"add", "mixed", "--env", "TOKEN", "https://example.com/mcp"},
		{"add", "nocmd", "--", "definitely-not-a-command-xyz"},
		{"remove", "missing"},
	} {
		if _, err := mcpCommand(t, args...); err == nil {
			t.Errorf("accepted %v", args)
		}
	}

	// Values entered elsewhere stay hidden from list output.
	config.Servers["context7"] = mcp.Server{Type: "http", URL: "https://mcp.context7.com/mcp", Headers: map[string]string{"Authorization": "Bearer top-secret"}}
	if err := mcp.Save(path, config); err != nil {
		t.Fatal(err)
	}
	out, err := mcpCommand(t, "list")
	if err != nil || !strings.Contains(out, "headers=Authorization (values hidden)") || strings.Contains(out, "top-secret") || !strings.Contains(out, "local  stdio") {
		t.Fatalf("list: %q %v", out, err)
	}
	if _, err := mcpCommand(t, "remove", "context7"); err != nil {
		t.Fatal(err)
	}
	if config, _ := mcp.Load(path); len(config.Servers) != 1 {
		t.Fatalf("remove: %+v", config)
	}
	if out, err := mcpCommand(t, "help"); err != nil || !strings.Contains(out, "mino mcp add NAME URL") {
		t.Fatalf("help: %q %v", out, err)
	}
}
