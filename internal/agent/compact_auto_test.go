package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/tools"
)

func TestAutomaticCompactionBeforeRequest(t *testing.T) {
	for _, window := range []int{6000, 128000} {
		t.Run(strconv.Itoa(window), func(t *testing.T) {
			s, err := OpenSession(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			seedCompactTurns(t, s, 4, 500)
			requests, summaries := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["instructions"] == summaryInstructions {
					summaries++
					sendCompactText(w, "Goal: finish. Earlier work complete.")
					return
				}
				if window == 6000 && (s.state.summary == "" || len(body["input"].([]any)) != 6) {
					t.Error("chat request preceded durable summary or lost recent context")
				}
				if body["max_output_tokens"] != float64(min(8192, window/8)) || body["truncation"] != "disabled" {
					t.Error("output reservation or truncation policy missing")
				}
				sendCompactText(w, "Done.")
			}))
			defer server.Close()
			a, _ := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test", ContextWindow: window}, s, nil)
			if err := a.Handle(context.Background(), "Continue", testInteraction{}); err != nil {
				t.Fatal(err)
			}
			want := 0
			if window == 6000 {
				want = 1
			}
			if summaries != want || requests != want+1 {
				t.Fatalf("summaries=%d requests=%d", summaries, requests)
			}
		})
	}
}

func TestAutomaticCompactionCountsTowardTurnLimit(t *testing.T) {
	executions, requests, summaries := 0, 0, 0
	echo := echoTool{execute: func(context.Context, tools.Call) tools.Result {
		executions++
		return tools.Result{Status: "completed", Output: "ok"}
	}}
	available := []tools.Tool{echo}
	s, err := OpenSession(t.TempDir(), available)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedCompactTurns(t, s, 4, 500)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["instructions"] == summaryInstructions {
			summaries++
			sendCompactText(w, "Previous goal and work.")
			return
		}
		streamEvent(w, fmt.Sprintf(`{"type":"response.completed","response":{"status":"completed","output":[{"type":"function_call","call_id":"call-%d","name":"echo","arguments":"{\"text\":\"hello\"}"}]}}`, requests))
	}))
	defer server.Close()
	a, _ := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test", ContextWindow: 6000}, s, available)
	err = a.Handle(context.Background(), "Keep checking", testInteraction{confirm: func(context.Context, Confirmation) (bool, error) { return true, nil }})
	if err == nil || !strings.Contains(err.Error(), "limit") || requests != 8 || summaries != 1 || executions != 6 {
		t.Fatalf("err=%v requests=%d summaries=%d executions=%d", err, requests, summaries, executions)
	}
}

func TestAutomaticCompactionFailureStopsBeforeChat(t *testing.T) {
	s, err := OpenSession(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedCompactTurns(t, s, 4, 500)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; http.Error(w, "private", 500) }))
	defer server.Close()
	a, _ := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test", ContextWindow: 6000}, s, nil)
	err = a.Handle(context.Background(), "Continue", testInteraction{})
	if err == nil || requests != 1 || s.state.summary != "" || len(s.state.input) != 8 || s.state.pending.id != "" {
		t.Fatalf("err=%v requests=%d", err, requests)
	}
}

func TestOversizedInputDoesNotCallModel(t *testing.T) {
	s, err := OpenSession(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedCompactTurns(t, s, 4, 300)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("oversized input sent to model")
		http.Error(w, "bad", 400)
	}))
	defer server.Close()
	a, _ := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test", ContextWindow: 6000}, s, nil)
	if err := a.Handle(context.Background(), strings.Repeat("x", 6000), testInteraction{}); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("error=%v", err)
	}
	if s.state.pending.id != "" || s.state.summary != "" || len(s.state.input) != 8 {
		t.Fatal("oversized turn changed prior context or was left open")
	}
}

func TestAutomaticCompactionAfterToolResult(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		t.Run(map[bool]string{false: "compact", true: "too large"}[oversized], func(t *testing.T) {
			executions, requests, summaries := 0, 0, 0
			echo := echoTool{execute: func(context.Context, tools.Call) tools.Result {
				executions++
				size := 2200
				if oversized {
					size = 10000
				}
				return tools.Result{Status: "completed", Output: strings.Repeat("z", size)}
			}}
			available := []tools.Tool{echo}
			s, err := OpenSession(t.TempDir(), available)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			seedCompactTurns(t, s, 4, 250)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["instructions"] == summaryInstructions {
					summaries++
					sendCompactText(w, "Earlier goals and work.")
					return
				}
				if requests == 1 {
					streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":[{"type":"function_call","call_id":"one","name":"echo","arguments":"{\"text\":\"hello\"}"}]}}`)
					return
				}
				items := body["input"].([]any)
				last := items[len(items)-1].(map[string]any)
				if last["call_id"] != "one" || last["type"] != "function_call_output" {
					t.Error("active tool pair was lost")
				}
				sendCompactText(w, "Done.")
			}))
			defer server.Close()
			a, _ := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test", ContextWindow: 6000}, s, available)
			err = a.Handle(context.Background(), "Run the tool.", testInteraction{confirm: func(context.Context, Confirmation) (bool, error) { return true, nil }})
			if executions != 1 {
				t.Fatal("command rerun")
			}
			if oversized {
				if err == nil || requests != 1 || summaries != 0 {
					t.Fatalf("err=%v requests=%d summaries=%d", err, requests, summaries)
				}
			} else if err != nil || requests != 3 || summaries != 1 {
				t.Fatalf("err=%v requests=%d summaries=%d", err, requests, summaries)
			}
		})
	}
}
