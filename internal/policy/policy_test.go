package policy

import "testing"

func TestDecisionOrder(t *testing.T) {
	p := New()
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

func TestModes(t *testing.T) {
	edit := Call{Tool: "edit_file", Target: "src/main.go"}
	protected := Call{Tool: "write_file", Target: ".env"}
	bash, mcp := Call{Tool: "bash"}, Call{Tool: "mcp_docs_post"}
	cases := []struct {
		mode                           Mode
		edit, protected, bash, mcpTool Decision
	}{
		{Manual, Ask, Deny, Ask, Ask},
		{AcceptEdits, Allow, Deny, Ask, Ask},
		{Bypass, Allow, Deny, Allow, Allow},
	}
	p := New()
	if p.Mode() != Manual {
		t.Fatalf("new policy mode = %v", p.Mode())
	}
	for _, c := range cases {
		p.SetMode(c.mode)
		for call, want := range map[Call]Decision{edit: c.edit, protected: c.protected, bash: c.bash, mcp: c.mcpTool} {
			if got, _ := p.Decide(call); got != want {
				t.Errorf("%v %+v: got %v, want %v", c.mode, call, got, want)
			}
		}
	}
	for _, name := range []string{"manual", "accept-edits", "bypass"} {
		if mode, ok := ParseMode(name); !ok || mode.String() != name {
			t.Errorf("ParseMode(%q) = %v, %v", name, mode, ok)
		}
	}
	for _, name := range []string{"", "auto", "Bypass", "plan"} {
		if _, ok := ParseMode(name); ok {
			t.Errorf("accepted mode %q", name)
		}
	}
}
