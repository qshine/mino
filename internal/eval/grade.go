package eval

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Outcome is one grader's verdict for one run.
type Outcome struct {
	Grader string `json:"grader"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

// Grade applies every grader to what the run produced: the final answer, the
// calls the model requested with their results, and the workspace files.
func Grade(c Case, t Transcript) []Outcome {
	outcomes := make([]Outcome, 0, len(c.Graders)+1)
	if t.Err != nil {
		outcomes = append(outcomes, Outcome{Grader: "run", Detail: "turn failed: " + t.Err.Error()})
	}
	for _, g := range c.Graders {
		detail := grade(g, t)
		outcomes = append(outcomes, Outcome{Grader: g.String(), Passed: detail == "", Detail: detail})
	}
	return outcomes
}

func passed(outcomes []Outcome) bool {
	for _, o := range outcomes {
		if !o.Passed {
			return false
		}
	}
	return true
}

// grade returns an empty string on success or the reason for failure.
func grade(g Grader, t Transcript) string {
	names := make([]string, len(t.Calls))
	for i, call := range t.Calls {
		names[i] = call.Name
	}
	switch g.Kind {
	case "final_text_contains":
		if !strings.Contains(t.FinalText, g.Text) {
			return fmt.Sprintf("final text %q", truncate(t.FinalText))
		}
	case "final_text_matches":
		if !g.Pattern.MatchString(t.FinalText) {
			return fmt.Sprintf("final text %q", truncate(t.FinalText))
		}
	case "tool_called":
		if !contains(names, g.Tool) {
			return "calls: " + list(names)
		}
	case "tool_not_called":
		if contains(names, g.Tool) {
			return "calls: " + list(names)
		}
	case "tool_order":
		if !subsequence(g.Tools, names) {
			return "calls: " + list(names)
		}
	case "result_status":
		if g.Call > len(t.Calls) {
			return fmt.Sprintf("only %d tool calls recorded", len(t.Calls))
		}
		if status := t.Calls[g.Call-1].Result.Status; status != g.Status {
			if status == "" {
				status = "no result"
			}
			return fmt.Sprintf("call %d (%s) was %s", g.Call, t.Calls[g.Call-1].Name, status)
		}
	case "file_exists", "file_contains":
		return gradeFile(g, t.Workspace)
	case "max_requests":
		if t.Requests > g.Limit {
			return fmt.Sprintf("%d requests", t.Requests)
		}
	case "max_tool_calls":
		if len(t.Calls) > g.Limit {
			return fmt.Sprintf("%d tool calls", len(t.Calls))
		}
	}
	return ""
}

func gradeFile(g Grader, workspace string) string {
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return "workspace unavailable: " + err.Error()
	}
	defer root.Close()
	info, err := root.Lstat(g.Path)
	if err != nil {
		return "file not found"
	}
	if !info.Mode().IsRegular() {
		return "not a regular file"
	}
	if g.Kind == "file_exists" {
		return ""
	}
	file, err := root.Open(g.Path)
	if err != nil {
		return "cannot open file: " + err.Error()
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSeedBytes+1))
	if err != nil {
		return "cannot read file: " + err.Error()
	}
	if len(data) > maxSeedBytes {
		return "file exceeds 64 KiB"
	}
	if !strings.Contains(string(data), g.Text) {
		return fmt.Sprintf("file content %q", truncate(string(data)))
	}
	return ""
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// subsequence reports whether want appears in order within got, possibly with
// other calls in between.
func subsequence(want, got []string) bool {
	i := 0
	for _, name := range got {
		if i < len(want) && name == want[i] {
			i++
		}
	}
	return i == len(want)
}

func list(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return "[" + strings.Join(names, " ") + "]"
}

func truncate(text string) string {
	const limit = 120
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
