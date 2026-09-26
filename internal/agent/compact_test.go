package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/tools"
)

func sendCompactText(w http.ResponseWriter, text string) {
	raw, _ := json.Marshal(text)
	streamEvent(w, `{"type":"response.output_text.delta","delta":`+string(raw)+`}`)
	streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
}

func TestCompactCommandUsesSummaryAndRecentTurns(t *testing.T) {
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		requests = append(requests, body)
		if len(requests) == 1 {
			if body["tool_choice"] != "none" || body["store"] != false || body["truncation"] != "disabled" {
				t.Errorf("unsafe summary request: %v", body)
			}
			if tools, ok := body["tools"].([]any); ok && len(tools) != 0 {
				t.Error("summary can call tools")
			}
			sendCompactText(w, "Goal: finish the task. Earlier work is complete.")
		} else {
			sendCompactText(w, "Continued.")
		}
	}))
	defer server.Close()
	store, err := OpenSessions(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	seedCompactTurns(t, store.current, 4, 300)
	before, _ := os.ReadFile(store.path(store.id))
	m, err := NewSessionManager(Options{BaseURL: server.URL, APIKey: "test", Model: "test", Instructions: "Original SOUL"}, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(context.Background(), "/compact", testInteraction{}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || len(store.current.state.input) != 4 {
		t.Fatal("wrong retention or request count")
	}
	if err := m.Handle(context.Background(), "Continue", testInteraction{}); err != nil {
		t.Fatal(err)
	}
	input := requests[1]["input"].([]any)
	if len(input) != 6 || input[0].(map[string]any)["role"] != "assistant" || !strings.Contains(input[0].(map[string]any)["content"].(string), "Earlier work") {
		t.Fatalf("bad context: %v", input)
	}
	if requests[1]["instructions"] != "Original SOUL" {
		t.Fatal("summary was promoted to instructions")
	}
	after, _ := os.ReadFile(store.path(store.id))
	if !bytes.HasPrefix(after, before) || bytes.Contains(after, []byte(`"text":"/compact"`)) {
		t.Fatal("command stored as user input or history rewritten")
	}
}

func TestCompactFailureKeepsOldContext(t *testing.T) {
	for _, mode := range []string{"http", "empty", "refusal", "incomplete", "tool", "large", "not smaller", "cancel", "sync"} {
		t.Run(mode, func(t *testing.T) {
			s, err := OpenSession(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			seedCompactTurns(t, s, 3, 200)
			before, _ := os.ReadFile(s.file.Name())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				switch mode {
				case "http":
					http.Error(w, "private error", 500)
				case "empty":
					sendCompactText(w, " ")
				case "refusal":
					streamEvent(w, `{"type":"response.refusal.delta","delta":"No"}`)
				case "incomplete":
					streamEvent(w, `{"type":"response.incomplete","response":{"status":"incomplete"}}`)
				case "tool":
					streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":[{"type":"function_call","call_id":"a","name":"bash","arguments":"{}"}]}}`)
				case "large":
					sendCompactText(w, strings.Repeat("x", 9000))
				case "not smaller":
					sendCompactText(w, strings.Repeat("x", 1000))
				case "cancel":
					cancel()
					<-r.Context().Done()
				default:
					sendCompactText(w, "Summary.")
				}
			}))
			defer server.Close()
			a, err := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test"}, s, nil)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "sync" {
				s.writer = failingHistoryWriter{s.file, "sync"}
			}
			err = a.Compact(ctx, testInteraction{})
			if err == nil || s.state.summary != "" || len(s.state.input) != 6 {
				t.Fatalf("err=%v summary=%q", err, s.state.summary)
			}
			if strings.Contains(err.Error(), "private error") {
				t.Fatal("API error body leaked")
			}
			var storage *StorageError
			if (mode == "sync") != errors.As(err, &storage) {
				t.Fatal("wrong storage error classification")
			}
			after, _ := os.ReadFile(s.file.Name())
			if mode != "sync" && !bytes.Equal(before, after) {
				t.Fatal("failed compaction changed log")
			}
		})
	}
}

func TestCompactRepeatedSummarySessionIsolationAndClear(t *testing.T) {
	var summaries [][]map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Input []map[string]any }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		summaries = append(summaries, body.Input)
		// A completed output item, without deltas, is also a valid summary.
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Retain goal, constraints, and completed work."}]}]}}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	store, err := OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	options := Options{BaseURL: server.URL, APIKey: "test", Model: "test"}
	m, _ := NewSessionManager(options, store, nil)
	do := func(command string) {
		t.Helper()
		if err := m.Handle(context.Background(), command, testInteraction{confirm: func(context.Context, Confirmation) (bool, error) { return true, nil }}); err != nil {
			t.Fatal(err)
		}
	}
	first := store.id
	seedCompactTurns(t, store.current, 4, 300)
	do("/compact")
	seedCompactTurns(t, store.current, 1, 300)
	do("/compact")
	if len(summaries) != 2 || len(summaries[1]) != 4 || summaries[1][0]["role"] != "assistant" || !strings.Contains(summaries[1][0]["content"].(string), "Retain goal") {
		t.Fatalf("repeated summary input=%v", summaries)
	}
	do("/new")
	second := store.id
	if store.current.state.summary != "" || len(store.current.state.input) != 0 {
		t.Fatal("summary leaked to new session")
	}
	do("/compact")
	if len(summaries) != 2 {
		t.Fatal("empty session caused summary request")
	}
	do("/resume " + first)
	if store.current.state.summary == "" || len(store.current.state.input) != 4 {
		t.Fatal("resume lost summary or retained turns")
	}
	store.Close()
	store, err = OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	m, _ = NewSessionManager(options, store, nil)
	if store.id != first || store.current.state.summary == "" {
		t.Fatal("restart lost selection or summary")
	}
	do("/clear")
	if store.current.state.summary != "" || len(store.current.state.input) != 0 || store.current.state.coveredSeq != 0 {
		t.Fatal("clear retained summary")
	}
	do("/resume " + second)
	if store.current.state.summary != "" {
		t.Fatal("clear contaminated another session")
	}
}

func TestCompactPreservesHistoricalToolPairsWithoutExecuting(t *testing.T) {
	s, err := OpenSession(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := newHistoryID()
	if err := s.append(
		historyRecord{TurnID: id, Kind: "user_message", Text: "Inspect environment."},
		historyRecord{TurnID: id, Kind: "model_response", Output: callReply("a", "b").Output},
		historyRecord{TurnID: id, Kind: "tool_result", CallID: "a", Result: &tools.Result{Status: "denied", Output: "Denied. " + strings.Repeat("x", 300)}},
		historyRecord{TurnID: id, Kind: "tool_result", CallID: "b", Result: &tools.Result{Status: "denied", Output: "Denied."}},
		historyRecord{TurnID: id, Kind: "turn_end", Status: "cancelled"},
	); err != nil {
		t.Fatal(err)
	}
	seedCompactTurns(t, s, 2, 300)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Input []map[string]any }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		for n, id := range []string{"a", "b"} {
			if body.Input[1+n]["call_id"] != id || body.Input[3+n]["call_id"] != id || body.Input[3+n]["type"] != "function_call_output" {
				t.Error("split historical tool pair")
			}
		}
		sendCompactText(w, "Environment inspection denied; no command ran.")
	}))
	defer server.Close()
	a, _ := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test"}, s, nil)
	i := testInteraction{confirm: func(context.Context, Confirmation) (bool, error) {
		t.Error("compaction requested tool approval")
		return false, nil
	}}
	if err := a.Compact(context.Background(), i); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s.state.summary, "denied") || len(s.state.input) != 4 {
		t.Fatal("bad summary context")
	}
	s.busy.Lock()
	if err := a.Compact(context.Background(), i); !errors.Is(err, ErrBusy) {
		t.Errorf("busy error=%v", err)
	}
	s.busy.Unlock()
}
