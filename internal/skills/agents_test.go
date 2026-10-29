package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAgent(t *testing.T, root, file, text string) string {
	t.Helper()
	path := filepath.Join(root, ".agents", "agents", file)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanAgentsMetadataAndRoleOnDemand(t *testing.T) {
	root := t.TempDir()
	c, problems, err := ScanAgents(root)
	if err != nil || len(problems) != 0 || len(c.Entries) != 0 || c.Prompt() != "" {
		t.Fatalf("missing directory: %+v, %q, %v", c, problems, err)
	}
	path := writeAgent(t, root, "reviewer.md", "---\nname: reviewer\ndescription: Reviews code.\ntools:\n  - read_file\n  - skill_view\nmodel: ignored\n---\nBODY_BEFORE")
	writeAgent(t, root, "explorer.md", "---\r\nname: explorer\r\ndescription: Explores.\r\n---\r\nEXPLORE")
	writeAgent(t, root, "notes.txt", "not a definition")
	if err := os.MkdirAll(filepath.Join(root, ".agents", "agents", "drafts.md"), 0700); err != nil {
		t.Fatal(err)
	}
	c, problems, err = ScanAgents(root)
	if err != nil || len(problems) != 0 {
		t.Fatalf("scan: %q, %v", problems, err)
	}
	if len(c.Entries) != 2 || c.Entries[0].Name != "explorer" || c.Entries[0].Tools != nil || strings.Join(c.Entries[1].Tools, ",") != "read_file,skill_view" {
		t.Fatalf("entries = %+v", c.Entries)
	}
	prompt := c.Prompt()
	if !strings.Contains(prompt, `"tools":["read_file","skill_view"]`) || strings.Contains(prompt, "BODY") || strings.Contains(prompt, "ignored") {
		t.Fatalf("prompt: %s", prompt)
	}
	// Only the body is read again: frozen metadata keeps the old tools.
	if err := os.WriteFile(path, []byte("---\nname: reviewer\ndescription: Changed.\ntools: [bash]\n---\nBODY_AFTER"), 0600); err != nil {
		t.Fatal(err)
	}
	e, body, err := c.Role("reviewer")
	if err != nil || body != "BODY_AFTER" || e.Description != "Reviews code." || len(e.Tools) != 2 {
		t.Fatalf("role: %+v, %q, %v", e, body, err)
	}
	if _, body, err := c.Role("explorer"); err != nil || body != "EXPLORE" {
		t.Fatalf("CRLF role: %q, %v", body, err)
	}
	if _, _, err := c.Role("missing"); err == nil || !strings.Contains(err.Error(), "/new") {
		t.Fatalf("missing role: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Role("reviewer"); err == nil {
		t.Fatal("read a deleted definition")
	}
}

func TestScanAgentsSkipsInvalidDefinitions(t *testing.T) {
	for _, tc := range []struct{ file, text, reason string }{
		{"review.md", "no front matter", "missing YAML front matter"},
		{"review.md", "---\nname: review\n---\nBody", "YAML strings"},
		{"review.md", "---\nname: other\ndescription: D\n---\nBody", "must match the file name"},
		{"Review.md", "---\nname: Review\ndescription: D\n---\nBody", "valid subagent name"},
		{"review.md", "---\nname: review\ndescription: D\ntools: bash\n---\nBody", "YAML list"},
		{"review.md", "---\nname: review\ndescription: D\ntools: []\n---\nBody", "1 to 64"},
		{"review.md", "---\nname: review\ndescription: D\ntools: [bash, bash]\n---\nBody", "listed twice"},
		{"review.md", "---\nname: review\ndescription: D\ntools: [spawn_agent]\n---\nBody", "cannot include spawn_agent"},
		{"review.md", "---\nname: review\ndescription: D\ntools: [memory_save]\n---\nBody", "cannot include memory_save"},
		{"review.md", "---\nname: review\ndescription: D\ntools: [\"a b\"]\n---\nBody", "tool name"},
		{"review.md", "---\nname: review\ndescription: D\ntools: [1]\n---\nBody", "YAML list"},
		{"review.md", "---\nname: review\ndescription: " + strings.Repeat("a", 1025) + "\n---\nBody", "1025 characters"},
		{"review.md", "---\nname: review\ndescription: D\n" + strings.Repeat("# x\n", 5000) + "---\nBody", "within 16 KiB"},
		{"review.md", "---\nname: review\ndescription: D\n---\n" + strings.Repeat("x", 65536), "at most 64 KiB"},
		{"review.md", "---\nname: review\ndescription: D\n---\n\xff", "UTF-8"},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			root := t.TempDir()
			writeAgent(t, root, "alpha.md", "---\nname: alpha\ndescription: Still loads\n---\n")
			writeAgent(t, root, tc.file, tc.text)
			c, problems, err := ScanAgents(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(c.Entries) != 1 || c.Entries[0].Name != "alpha" {
				t.Fatalf("entries = %+v", c.Entries)
			}
			if len(problems) != 1 || !strings.HasPrefix(problems[0], tc.file+": ") || !strings.Contains(problems[0], tc.reason) {
				t.Fatalf("problems = %q, want %s and %q", problems, tc.file, tc.reason)
			}
		})
	}
}

func TestScanAgentsLinksAndFullCatalog(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	target := writeAgent(t, outside, "linked.md", "---\nname: linked\ndescription: Outside\n---\nBody")
	if err := os.MkdirAll(filepath.Join(root, ".agents", "agents"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, ".agents", "agents", "linked.md")); err != nil {
		t.Fatal(err)
	}
	for i := range 40 {
		name := fmt.Sprintf("a%02d", i)
		writeAgent(t, root, name+".md", "---\nname: "+name+"\ndescription: "+strings.Repeat("x", 1000)+"\n---\nBody")
	}
	c, problems, err := ScanAgents(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(problems, "\n")
	if len(c.Entries) == 0 || len(c.Entries) == 40 || !strings.Contains(joined, "linked.md: skipped because the definition is a symbolic link") || !strings.Contains(joined, "a39.md: skipped because the subagent catalog is full") {
		t.Fatalf("%d entries, problems %q", len(c.Entries), problems)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	// A linked .agents/agents directory fails discovery, like a linked skills directory.
	if err := os.RemoveAll(filepath.Join(root, ".agents", "agents")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, ".agents", "agents"), filepath.Join(root, ".agents", "agents")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ScanAgents(root); err == nil {
		t.Fatal("scanned a linked definitions directory")
	}
	if _, _, err := c.Role("a00"); err == nil {
		t.Fatal("read through a linked definitions directory")
	}
}

func TestAgentCatalogValidate(t *testing.T) {
	for _, c := range []*AgentCatalog{
		nil,
		{Root: "relative"},
		{Root: "/p", Entries: []AgentEntry{{Name: "b", Description: "B"}, {Name: "a", Description: "A"}}},
		{Root: "/p", Entries: []AgentEntry{{Name: "a", Description: "A", Tools: []string{}}}},
		{Root: "/p", Entries: []AgentEntry{{Name: "a", Description: "A", Tools: []string{"spawn_agent"}}}},
	} {
		if err := c.Validate(); err == nil {
			t.Fatalf("accepted %+v", c)
		}
	}
}
