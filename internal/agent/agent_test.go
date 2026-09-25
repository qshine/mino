package agent

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/qshine/mino/internal/tools"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAgentApprovalResultsAndReplay(t *testing.T) {
	for _, mode := range []string{"approved", "denied", "invalid", "unknown", "eof", "duplicate", "reused", "limit"} {
		t.Run(mode, func(t *testing.T) {
			isolateConfig(t)
			h, err := openHistory()
			if err != nil {
				t.Fatal(err)
			}
			defer h.Close()
			requests, executions, approvals := 0, 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if requests == 1 || mode == "reused" || mode == "limit" {
					reply := callReply("a")
					switch mode {
					case "invalid":
						reply.Output = []json.RawMessage{json.RawMessage(`{"type":"function_call","call_id":"a","name":"bash","arguments":"{\"command\":\"echo ok\",\"cwd\":\"/\"}"}`)}
					case "unknown":
						reply.Output = []json.RawMessage{json.RawMessage(`{"type":"function_call","call_id":"a","name":"other","arguments":"{}"}`)}
					case "duplicate":
						reply = callReply("a", "a")
					case "limit":
						reply = callReply(strings.Repeat("x", requests))
					}
					raw, _ := json.Marshal(reply.Output)
					streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":`+string(raw)+`}}`)
					return
				}
				input := body["input"].([]any)
				last := input[len(input)-1].(map[string]any)
				if last["type"] != "function_call_output" || last["call_id"] != "a" {
					t.Errorf("missing paired result: %v", last)
				}
				want := map[string]string{"approved": "completed", "denied": "denied", "invalid": "invalid_arguments", "unknown": "unknown_tool"}[mode]
				if !strings.Contains(last["output"].(string), `"status":"`+want+`"`) {
					t.Errorf("result=%v", last)
				}
				streamEvent(w, `{"type":"response.output_text.delta","delta":"Done."}`)
				streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
			}))
			defer server.Close()
			bash, err := tools.NewBash(t.TempDir(), t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			chat := turnFixture{history: h, client: newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, ""), actions: &toolFixtureActions{
				bash: bash,
				approve: func(context.Context, tools.BashArgs) (bool, error) {
					approvals++
					if mode == "eof" {
						return false, io.EOF
					}
					return mode != "denied", nil
				},
				execute: func(context.Context, tools.BashArgs) tools.Result {
					executions++
					code := 0
					return tools.Result{Status: "completed", Output: "go version example", ExitCode: &code}
				},
			}}
			err = chat.respond(context.Background(), "Go version?", func(string) error { return nil })
			switch mode {
			case "approved":
				if err != nil || executions != 1 || requests != 2 {
					t.Fatalf("err=%v executions=%d requests=%d", err, executions, requests)
				}
			case "denied", "invalid", "unknown":
				if err != nil || executions != 0 || requests != 2 {
					t.Fatalf("err=%v executions=%d requests=%d", err, executions, requests)
				}
			case "eof":
				if !errors.Is(err, io.EOF) || executions != 0 || requests != 1 {
					t.Fatalf("err=%v executions=%d", err, executions)
				}
			case "duplicate":
				if err == nil || executions != 0 || approvals != 0 {
					t.Fatal("invalid batch was executed")
				}
			case "reused":
				if err == nil || executions != 1 || requests != 2 {
					t.Fatalf("reused call: err=%v executions=%d requests=%d", err, executions, requests)
				}
			case "limit":
				if err == nil || requests != 8 || executions != 7 {
					t.Fatalf("limit: err=%v executions=%d requests=%d", err, executions, requests)
				}
			}
			if h.state.pending.id != "" {
				t.Fatal("turn not closed")
			}
			data, _ := os.ReadFile(h.file.Name())
			if mode == "duplicate" && strings.Contains(string(data), "function_call") {
				t.Fatal("invalid batch saved")
			}
		})
	}
}

func TestAgentStopsWhenToolResultCannotBeSaved(t *testing.T) {
	isolateConfig(t)
	h, err := openHistory()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	requests, executions := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		raw, _ := json.Marshal(callReply("a").Output)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":`+string(raw)+`}}`)
	}))
	defer server.Close()
	bash, _ := tools.NewBash(t.TempDir(), t.TempDir())
	chat := turnFixture{history: h, client: newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, ""), actions: &toolFixtureActions{bash: bash, approve: func(context.Context, tools.BashArgs) (bool, error) { return true, nil }, execute: func(context.Context, tools.BashArgs) tools.Result {
		executions++
		h.writer = failingHistoryWriter{h.file, "write"}
		code := 0
		return tools.Result{Status: "completed", ExitCode: &code}
	}}}
	err = chat.respond(context.Background(), "test", func(string) error { return nil })
	var storage *StorageError
	if !errors.As(err, &storage) || executions != 1 || requests != 1 {
		t.Fatalf("err=%v executions=%d requests=%d", err, executions, requests)
	}
}
