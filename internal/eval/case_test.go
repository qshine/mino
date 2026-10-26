package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validCase = `name: demo
description: A scripted bash call.
prompt: List files.
approve: [bash]
workspace:
  files:
    notes.md: "hello\n"
model:
  - call: {tool: bash, arguments: {command: ls}}
    expect_input_contains: List files
  - text: Done.
graders:
  - final_text_contains: Done
  - final_text_matches: "^Done"
  - tool_called: bash
  - tool_not_called: write_file
  - tool_order: [bash]
  - result_status: {call: 1, status: completed}
  - file_exists: notes.md
  - file_contains: {path: notes.md, text: hello}
  - max_requests: 2
  - max_tool_calls: 1
  - final_text_not_contains: Error
`

func TestParseAcceptsEveryGrader(t *testing.T) {
	c, err := Parse("demo", []byte(validCase))
	if err != nil {
		t.Fatal(err)
	}
	if c.Approve.All || !c.Approve.Allows("bash") || c.Approve.Allows("write_file") {
		t.Fatalf("approve=%+v", c.Approve)
	}
	if len(c.Script) != 2 || c.Script[0].Call == nil || string(c.Script[0].Call.Arguments) != `{"command":"ls"}` || c.Script[0].ExpectInputContains != "List files" || c.Script[1].Text != "Done." {
		t.Fatalf("script=%+v", c.Script)
	}
	if len(c.Graders) != 11 || c.Graders[10].Kind != "final_text_not_contains" || c.ReplayOnly || c.Graders[1].Pattern == nil || c.Graders[5].Call != 1 || c.Graders[7].Path != "notes.md" || c.Graders[9].Limit != 1 {
		t.Fatalf("graders=%+v", c.Graders)
	}
	if c.Workspace.Files["notes.md"] != "hello\n" {
		t.Fatalf("workspace=%+v", c.Workspace)
	}
	if plain, err := Parse("demo", []byte("name: demo\nprompt: hi\ngraders:\n  - max_requests: 1\n")); err != nil || !plain.Approve.All || plain.Script != nil || plain.ReplayOnly {
		t.Fatalf("defaults: %+v %v", plain, err)
	}
	if only, err := Parse("demo", []byte("name: demo\nprompt: hi\nlive: false\nmodel:\n  - text: ok\ngraders:\n  - max_requests: 1\n")); err != nil || !only.ReplayOnly {
		t.Fatalf("replay-only: %+v %v", only, err)
	}
}

func TestParseRejectsInvalidCases(t *testing.T) {
	base := "name: demo\nprompt: hi\ngraders:\n  - max_requests: 1\n"
	for name, doc := range map[string]string{
		"missing prompt":        "name: demo\ngraders:\n  - max_requests: 1\n",
		"name mismatch":         strings.Replace(base, "name: demo", "name: other", 1),
		"bad name":              strings.Replace(base, "name: demo", "name: Demo_1", 1),
		"unknown field":         base + "extra: 1\n",
		"unknown grader":        "name: demo\nprompt: hi\ngraders:\n  - judge: good\n",
		"two-key grader":        "name: demo\nprompt: hi\ngraders:\n  - max_requests: 1\n    max_tool_calls: 1\n",
		"no graders":            "name: demo\nprompt: hi\n",
		"empty tool_order":      "name: demo\nprompt: hi\ngraders:\n  - tool_order: []\n",
		"bad regex":             "name: demo\nprompt: hi\ngraders:\n  - final_text_matches: \"(\"\n",
		"result call zero":      "name: demo\nprompt: hi\ngraders:\n  - result_status: {call: 0, status: completed}\n",
		"result extra field":    "name: demo\nprompt: hi\ngraders:\n  - result_status: {call: 1, status: completed, tool: bash}\n",
		"approve maybe":         base + "approve: maybe\n",
		"runs too many":         base + "runs: 33\n",
		"seed escapes":          base + "workspace:\n  files:\n    ../x: a\n",
		"seed absolute":         base + "workspace:\n  files:\n    /etc/x: a\n",
		"seed too large":        base + "workspace:\n  files:\n    big.txt: \"" + strings.Repeat("a", 64<<10+1) + "\"\n",
		"step both":             base + "model:\n  - text: a\n    call: {tool: bash}\n",
		"step neither":          base + "model:\n  - expect_input_contains: a\n",
		"step unknown field":    base + "model:\n  - text: a\n    tool: bash\n",
		"call no tool":          base + "model:\n  - call: {arguments: {}}\n",
		"too many steps":        base + "model:\n" + strings.Repeat("  - text: a\n", 17),
		"too many graders":      "name: demo\nprompt: hi\ngraders:\n" + strings.Repeat("  - max_requests: 1\n", 33),
		"file path escapes":     "name: demo\nprompt: hi\ngraders:\n  - file_exists: ../secret\n",
		"not yaml":              "name: [",
		"live not bool":         base + "live: maybe\n",
		"replay-only no script": base + "live: false\n",
		"empty not_contains":    "name: demo\nprompt: hi\ngraders:\n  - final_text_not_contains: \"\"\n",
	} {
		if _, err := Parse("demo", []byte(doc)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestLoadDirectoryIsSortedAndBounded(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"beta", "alpha"} {
		if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte("name: "+name+"\nprompt: hi\ngraders:\n  - max_requests: 1\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("ignored"), 0600); err != nil {
		t.Fatal(err)
	}
	cases, err := Load(dir)
	if err != nil || len(cases) != 2 || cases[0].Name != "alpha" || cases[1].Name != "beta" {
		t.Fatalf("cases=%+v err=%v", cases, err)
	}
	if single, err := Load(filepath.Join(dir, "beta.yaml")); err != nil || len(single) != 1 {
		t.Fatalf("single=%+v err=%v", single, err)
	}
	if _, err := Load(t.TempDir()); err == nil || !strings.Contains(err.Error(), "No eval cases") {
		t.Fatalf("empty dir: %v", err)
	}
	if _, err := Load(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing path accepted")
	}
	broken := filepath.Join(dir, "broken.yaml")
	if err := os.WriteFile(broken, []byte("name: broken\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "broken.yaml") {
		t.Fatalf("broken case: %v", err)
	}
	os.Remove(broken)
	many := t.TempDir()
	for i := 0; i <= maxCases; i++ {
		name := "c" + strings.Repeat("a", i/26) + string(rune('a'+i%26))
		if err := os.WriteFile(filepath.Join(many, name+".yaml"), []byte("name: "+name+"\nprompt: hi\ngraders:\n  - max_requests: 1\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Load(many); err == nil || !strings.Contains(err.Error(), "Too many") {
		t.Fatalf("limit: %v", err)
	}
}
