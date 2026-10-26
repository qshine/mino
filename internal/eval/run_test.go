package eval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func call(tool, arguments string) Step {
	return Step{Call: &ScriptCall{Tool: tool, Arguments: json.RawMessage(arguments)}}
}

func parseGraders(t *testing.T, yaml string) []Grader {
	t.Helper()
	c, err := Parse("demo", []byte("name: demo\nprompt: hi\ngraders:\n"+yaml))
	if err != nil {
		t.Fatal(err)
	}
	return c.Graders
}

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func detail(outcomes []Outcome) string {
	var parts []string
	for _, o := range outcomes {
		if !o.Passed {
			parts = append(parts, o.Grader+": "+o.Detail)
		}
	}
	return strings.Join(parts, "; ")
}

func TestReplayApprovedBashPasses(t *testing.T) {
	home := isolateHome(t)
	c := Case{Name: "demo", Prompt: "Say hello", Approve: Approval{All: true}, Script: []Step{call("bash", `{"command":"echo hi"}`), {Text: "The command printed hi."}}, Graders: parseGraders(t, `
  - tool_called: bash
  - tool_order: [bash]
  - result_status: {call: 1, status: completed}
  - final_text_contains: printed hi
  - final_text_matches: "^The command"
  - max_requests: 2
  - max_tool_calls: 1
  - tool_not_called: write_file
  - final_text_not_contains: secret
`)}
	result, err := RunCase(context.Background(), c, Options{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.Requests != 2 || result.ToolCalls != 1 || result.Usage.TotalTokens == 0 {
		t.Fatalf("result=%+v failures=%s", result, detail(result.Outcomes))
	}
	if _, err := os.Stat(filepath.Join(home, ".mino")); !os.IsNotExist(err) {
		t.Fatalf("eval touched the user home: %v", err)
	}
}

func TestReplayDeniedCallIsGraded(t *testing.T) {
	isolateHome(t)
	c := Case{Name: "demo", Prompt: "Run it", Approve: Approval{}, Script: []Step{call("bash", `{"command":"echo hi"}`), {Text: "I was not allowed."}}, Graders: parseGraders(t, `
  - tool_called: bash
  - result_status: {call: 1, status: denied}
  - max_tool_calls: 0
`)}
	result, err := RunCase(context.Background(), c, Options{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || result.ToolCalls != 1 {
		t.Fatalf("result=%+v", result)
	}
	for _, o := range result.Outcomes {
		switch {
		case strings.HasPrefix(o.Grader, "tool_called"), strings.HasPrefix(o.Grader, "result_status"):
			if !o.Passed {
				t.Errorf("%s failed: %s", o.Grader, o.Detail)
			}
		case strings.HasPrefix(o.Grader, "max_tool_calls"):
			if o.Passed || o.Detail != "1 tool calls" {
				t.Errorf("max_tool_calls: %+v", o)
			}
		}
	}
}

func TestReplayToolOrderAndScriptErrors(t *testing.T) {
	isolateHome(t)
	c := Case{Name: "demo", Prompt: "Write then list", Approve: Approval{All: true}, Script: []Step{
		call("write_file", `{"path":"out.txt","content":"ok\n","mode":"create"}`),
		{Call: &ScriptCall{Tool: "bash", Arguments: json.RawMessage(`{"command":"cat out.txt"}`)}, ExpectInputContains: `\"status\":\"completed\"`},
		{Text: "done"},
	}, Graders: parseGraders(t, `
  - tool_order: [bash, write_file]
  - tool_order: [write_file, bash]
  - file_exists: out.txt
  - file_contains: {path: out.txt, text: ok}
  - result_status: {call: 2, status: completed}
  - result_status: {call: 3, status: completed}
  - final_text_not_contains: done
`)}
	result, err := RunCase(context.Background(), c, Options{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed {
		t.Fatal("expected failures")
	}
	want := map[string]string{
		`final_text_not_contains "done"`:   `final text "done"`,
		"tool_order [bash write_file]":     "calls: [write_file bash]",
		"tool_order [write_file bash]":     "",
		"file_exists out.txt":              "",
		`file_contains out.txt "ok"`:       "",
		"result_status call 2 = completed": "",
		"result_status call 3 = completed": "only 2 tool calls recorded",
	}
	for _, o := range result.Outcomes {
		if o.Detail != want[o.Grader] {
			t.Errorf("%s: %q", o.Grader, o.Detail)
		}
	}

	// A script that ends before the model finishes fails the run with the reason.
	short := Case{Name: "demo", Prompt: "Loop", Approve: Approval{All: true}, Script: []Step{call("bash", `{"command":"echo hi"}`)}, Graders: parseGraders(t, "  - tool_called: bash\n")}
	result, err = RunCase(context.Background(), short, Options{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || !strings.Contains(detail(result.Outcomes), "Model script exhausted") {
		t.Fatalf("exhausted: %+v", result.Outcomes)
	}
	if _, err := RunCase(context.Background(), Case{Name: "demo", Prompt: "x"}, Options{}, 1); err == nil {
		t.Fatal("replay without a script accepted")
	}
}

func TestReplaySeedsWorkspaceAndSkills(t *testing.T) {
	isolateHome(t)
	out := t.TempDir()
	c := Case{Name: "demo", Prompt: "Read the skill", Approve: Approval{}, Workspace: Workspace{Files: map[string]string{
		".agents/skills/demo/SKILL.md": "---\nname: demo\ndescription: Demo skill.\n---\nUse echo.\n",
		"notes/todo.md":                "remember\n",
	}}, Script: []Step{call("skill_view", `{"name":"demo"}`), call("read_file", `{"path":"notes/todo.md"}`), {Text: "The skill says to use echo."}}, Graders: parseGraders(t, `
  - result_status: {call: 1, status: completed}
  - result_status: {call: 2, status: completed}
  - tool_order: [skill_view, read_file]
`)}
	result, err := RunCase(context.Background(), c, Options{Out: out}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("failures=%s", detail(result.Outcomes))
	}
	data, err := os.ReadFile(filepath.Join(out, "demo", "run-2.jsonl"))
	if err != nil || !strings.Contains(string(data), `"kind":"session_skills"`) || !strings.Contains(string(data), `"kind":"turn_end"`) {
		t.Fatalf("run log: %v %q", err, data)
	}
}

func TestRunTimeoutFailsRun(t *testing.T) {
	isolateHome(t)
	c := Case{Name: "demo", Prompt: "Wait", Approve: Approval{All: true}, Script: []Step{call("bash", `{"command":"sleep 5"}`), {Text: "done"}}, Graders: parseGraders(t, "  - tool_called: bash\n")}
	start := time.Now()
	result, err := RunCase(context.Background(), c, Options{Timeout: 300 * time.Millisecond}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || time.Since(start) > 4*time.Second || !strings.Contains(detail(result.Outcomes), "turn failed") {
		t.Fatalf("result=%+v elapsed=%s", result.Outcomes, time.Since(start))
	}
}
