package eval

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/agent"
)

func TestPassAtKAndPassPowK(t *testing.T) {
	for _, tc := range []struct {
		n, c, k   int
		atK, powK float64
	}{
		{3, 0, 3, 0, 0}, {3, 1, 3, 1, 0}, {3, 3, 3, 1, 1}, {5, 3, 2, 0.9, 0.3}, {1, 1, 1, 1, 1}, {1, 0, 1, 0, 0}, {3, 1, 4, 0, 0}, {3, 1, 0, 0, 0},
	} {
		if got := PassAtK(tc.n, tc.c, tc.k); math.Abs(got-tc.atK) > 1e-9 {
			t.Errorf("PassAtK(%d,%d,%d)=%v want %v", tc.n, tc.c, tc.k, got, tc.atK)
		}
		if got := PassPowK(tc.n, tc.c, tc.k); math.Abs(got-tc.powK) > 1e-9 {
			t.Errorf("PassPowK(%d,%d,%d)=%v want %v", tc.n, tc.c, tc.k, got, tc.powK)
		}
	}
}

func TestSummarizeAndReportOutput(t *testing.T) {
	results := []RunResult{
		{Run: 1, Passed: true, Requests: 2, ToolCalls: 1, Usage: agent.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}, Outcomes: []Outcome{{Grader: "tool_called bash", Passed: true}}},
		{Run: 2, Passed: false, Requests: 3, ToolCalls: 2, Usage: agent.Usage{InputTokens: 20, OutputTokens: 5, TotalTokens: 25}, Outcomes: []Outcome{{Grader: "tool_order [bash write_file]", Detail: "calls: [bash]"}}},
	}
	c := Summarize("demo", results)
	if c.Runs != 2 || c.Passed != 1 || c.PassAtK != 1 || c.PassPowK != 0 || c.MeanRequests != 2.5 || c.MeanToolCalls != 1.5 || c.Usage.InputTokens != 30 || c.Usage.TotalTokens != 40 {
		t.Fatalf("summary=%+v", c)
	}
	report := Report{Mode: "live", Model: "test-model", Cases: []CaseReport{c, Summarize("other", results[:1])}}
	report.AllPassed = report.Failed() == 0
	var out bytes.Buffer
	if err := report.WriteText(&out); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"Eval live mode: 2 cases, 2 runs each, model test-model.", "CASE", "demo", "FAIL", "other", "pass", "demo run 2: tool_order [bash write_file]: calls: [bash]", "1 of 2 cases failed."} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	if report.AllPassed || report.Failed() != 1 {
		t.Fatalf("failed=%d", report.Failed())
	}
	path := filepath.Join(t.TempDir(), "report.json")
	if err := report.WriteJSON(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("report file: %v %v", info, err)
	}
	data, _ := os.ReadFile(path)
	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil || len(decoded.Cases) != 2 || decoded.Cases[0].Results[1].Outcomes[0].Detail != "calls: [bash]" {
		t.Fatalf("round trip: %+v %v", decoded, err)
	}
	replay := Report{Mode: "replay", Cases: []CaseReport{Summarize("solo", results[:1])}}
	out.Reset()
	if err := replay.WriteText(&out); err != nil || !strings.Contains(out.String(), "scripted model (synthetic token counts)") || !strings.Contains(out.String(), "All cases passed.") {
		t.Fatalf("replay header: %q %v", out.String(), err)
	}
}
