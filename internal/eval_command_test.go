package mino

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func evalCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var output bytes.Buffer
	err := runCLI(context.Background(), "dev", append([]string{"eval"}, args...), strings.NewReader(""), &output, io.Discard)
	return output.String(), err
}

func TestEvalCommandRunsReplayCasesWithoutUserData(t *testing.T) {
	cases, err := filepath.Abs(filepath.Join("eval", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	path := isolateConfig(t)
	out, err := evalCommand(t, cases)
	if err == nil || !strings.Contains(err.Error(), "1 of 2 cases") {
		t.Fatalf("mixed suite: %v\n%s", err, out)
	}
	for _, want := range []string{"Eval replay mode: 2 cases, 1 runs each", "echo-approved", "write-denied", "FAIL", `final_text_contains "not allowed": final text "I created notes.md."`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if out, err := evalCommand(t, "--filter", "echo-approved", cases); err != nil || !strings.Contains(out, "All cases passed.") {
		t.Fatalf("passing case: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("replay touched ~/.mino: %v", err)
	}
	if out, err := evalCommand(t, "help"); err != nil || !strings.Contains(out, "mino eval [--live]") {
		t.Fatalf("help: %v %q", err, out)
	}
	for _, args := range [][]string{
		{t.TempDir()},                  // no cases
		{"--filter", "missing", cases}, // unknown case
		{"--runs", "0", cases},         // bad runs
		{"--runs", "33", cases},        // too many runs
		{"--timeout", "10ms", cases},   // bad timeout
		{"--bogus", cases},             // unknown flag
		{cases, "extra"},               // too many arguments
		{"--out", filepath.Join(filepath.Dir(path), "evals"), cases}, // inside ~/.mino
	} {
		if _, err := evalCommand(t, args...); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("rejected --out created ~/.mino: %v", err)
	}
}

func TestEvalCommandWritesReportAndLogs(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plain.yaml"), []byte("name: plain\nprompt: Say ok.\nruns: 2\nmodel:\n  - text: ok\ngraders:\n  - final_text_contains: ok\n  - max_tool_calls: 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "results")
	text, err := evalCommand(t, "--out", out, dir)
	if err != nil || !strings.Contains(text, "Report written to") {
		t.Fatalf("eval: %v\n%s", err, text)
	}
	data, err := os.ReadFile(filepath.Join(out, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Mode  string `json:"mode"`
		Cases []struct {
			Name    string  `json:"name"`
			Runs    int     `json:"runs"`
			Passed  int     `json:"passed"`
			PassPow float64 `json:"pass_pow_k"`
		} `json:"cases"`
		AllPassed bool `json:"all_passed"`
	}
	if err := json.Unmarshal(data, &report); err != nil || report.Mode != "replay" || !report.AllPassed || len(report.Cases) != 1 || report.Cases[0].Runs != 2 || report.Cases[0].Passed != 2 || report.Cases[0].PassPow != 1 {
		t.Fatalf("report: %+v %v", report, err)
	}
	for _, run := range []string{"run-1.jsonl", "run-2.jsonl"} {
		log, err := os.ReadFile(filepath.Join(out, "plain", run))
		if err != nil || !strings.Contains(string(log), `"kind":"user_message"`) || !strings.Contains(string(log), `"status":"completed"`) {
			t.Fatalf("%s: %v %q", run, err, log)
		}
	}
	if _, err := evalCommand(t, "--live", "--runs", "1", dir); err == nil || !strings.Contains(err.Error(), "--live requires") {
		t.Fatalf("live without config: %v", err)
	}
}

func TestEvalCommandLiveUsesConfiguredModel(t *testing.T) {
	isolateConfig(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["model"] != "test-model" || !strings.Contains(body["instructions"].(string), "Mino") {
			t.Errorf("request: model=%v instructions=%v", body["model"], body["instructions"])
		}
		requests++
		streamEvent(w, `{"type":"response.output_text.delta","delta":"Hello from the service."}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":40,"output_tokens":6,"total_tokens":46}}}`)
	}))
	defer server.Close()
	if err := saveConfig(config{server.URL + "/v1", "test-key", "test-model", 0, 0}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "greet.yaml"), []byte("name: greet\nprompt: Say hello.\ngraders:\n  - final_text_contains: Hello\n  - max_requests: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := evalCommand(t, "--live", "--runs", "2", dir)
	if err != nil {
		t.Fatalf("live eval: %v\n%s", err, out)
	}
	if requests != 2 || !strings.Contains(out, "Eval live mode: 1 cases, 2 runs each, model test-model.") || !strings.Contains(out, "80") || !strings.Contains(out, "12") {
		t.Fatalf("requests=%d output:\n%s", requests, out)
	}
	// Live mode defaults to three runs when --runs is omitted.
	requests = 0
	if _, err := evalCommand(t, "--live", dir); err != nil || requests != 3 {
		t.Fatalf("default runs: %v requests=%d", err, requests)
	}
	// A replay-only case is skipped live and named; a suite of only such cases is an error.
	if err := os.WriteFile(filepath.Join(dir, "only.yaml"), []byte("name: only\nprompt: Say hello.\nlive: false\nmodel:\n  - text: hi\ngraders:\n  - max_requests: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	requests = 0
	out, err = evalCommand(t, "--live", "--runs", "1", dir)
	if err != nil || requests != 1 || !strings.Contains(out, "Skipped replay-only cases (live: false): only") || strings.Contains(out, "\nonly ") {
		t.Fatalf("skip: %v requests=%d\n%s", err, requests, out)
	}
	if _, err := evalCommand(t, "--live", "--filter", "only", dir); err == nil || !strings.Contains(err.Error(), "replay-only") {
		t.Fatalf("all skipped: %v", err)
	}
}
