package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecisionOrder(t *testing.T) {
	p := Default()
	cases := []struct {
		call Call
		want Decision
	}{
		{Call{Tool: "read_file", ReadOnly: true, Target: "src/main.go"}, Allow},
		{Call{Tool: "read_file", ReadOnly: true, Target: ".env"}, Deny},
		{Call{Tool: "read_file", ReadOnly: true, Target: "config/.env.local"}, Deny},
		{Call{Tool: "write_file", Target: ".git/hooks/pre-commit"}, Deny},
		{Call{Tool: "edit_file", Target: "deploy/server.pem"}, Deny},
		{Call{Tool: "read_file", ReadOnly: true, Target: "docs/env.md"}, Allow},
		{Call{Tool: "read_file", ReadOnly: true, Target: "src/.git"}, Allow}, // a file named .git is not the .git/ directory
		{Call{Tool: "write_file", Target: "src/main.go"}, Ask},
		{Call{Tool: "bash"}, Ask},
		{Call{Tool: "mcp_context7_query-docs"}, Ask},
		{Call{Tool: "skill_view", ReadOnly: true}, Allow},
		// macOS file systems are case-insensitive by default.
		{Call{Tool: "read_file", ReadOnly: true, Target: ".ENV"}, Deny},
		{Call{Tool: "read_file", ReadOnly: true, Target: ".Git/config"}, Deny},
		{Call{Tool: "edit_file", Target: "keys/ID_RSA"}, Deny},
		{Call{Tool: "read_file", ReadOnly: true, Target: "deploy/Server.PEM"}, Deny},
	}
	for _, c := range cases {
		if got, _ := p.Decide(c.call); got != c.want {
			t.Errorf("%+v: got %v, want %v", c.call, got, c.want)
		}
	}
}

func writePolicy(t *testing.T, text string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(text), perm); err != nil {
		t.Fatal(err)
	}
	os.Chmod(path, perm)
	return path
}

func TestPolicyFile(t *testing.T) {
	if p, err := Load(filepath.Join(t.TempDir(), "missing.json")); err != nil || !p.Offered("bash") {
		t.Fatalf("missing policy: %v", err)
	}
	p, err := Load(writePolicy(t, `{"bash":"deny","protected_paths":["secrets/","*.sqlite"],"allow_tools":["mcp_context7_query-docs"],"deny_tools":["mcp_tickets_create"]}`, 0600))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		call Call
		want Decision
	}{
		{Call{Tool: "bash"}, Deny},
		{Call{Tool: "read_file", ReadOnly: true, Target: "secrets/token.txt"}, Deny},
		{Call{Tool: "read_file", ReadOnly: true, Target: "data/app.sqlite"}, Deny},
		{Call{Tool: "read_file", ReadOnly: true, Target: ".env"}, Deny}, // defaults still apply
		{Call{Tool: "mcp_context7_query-docs"}, Allow},
		{Call{Tool: "mcp_context7_resolve-library-id"}, Ask},
		{Call{Tool: "mcp_tickets_create"}, Deny},
	}
	for _, c := range cases {
		if got, _ := p.Decide(c.call); got != c.want {
			t.Errorf("%+v: got %v, want %v", c.call, got, c.want)
		}
	}
	if p.Offered("bash") || p.Offered("mcp_tickets_create") || !p.Offered("read_file") {
		t.Fatal("offered tools")
	}
	if unknown := p.Unknown([]string{"bash", "mcp_context7_query-docs"}); len(unknown) != 1 || unknown[0] != "mcp_tickets_create" {
		t.Fatalf("unknown=%v", unknown)
	}
	if !strings.Contains(p.Summary(), "Bash denied") {
		t.Fatal(p.Summary())
	}
	for _, bad := range []string{
		`{"allow_tools":["bash"]}`,
		`{"allow_tools":["write_file"]}`,
		`{"bash":"allow"}`,
		`{"protected_paths":["a/b"]}`,
		`{"protected_paths":["["]}`,
		`{"protected_paths":[".."]}`,
		`{"unknown":true}`,
		`{"bash":"deny"} {}`,
	} {
		if _, err := Load(writePolicy(t, bad, 0600)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	if _, err := Load(writePolicy(t, `{}`, 0666)); err == nil || !strings.Contains(err.Error(), "writable") {
		t.Fatalf("group-writable policy accepted: %v", err)
	}
	link := filepath.Join(t.TempDir(), "link.json")
	os.Symlink(writePolicy(t, `{}`, 0600), link)
	if _, err := Load(link); err == nil {
		t.Fatal("symlinked policy accepted")
	}
}
