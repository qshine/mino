package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qshine/mino/internal/tools"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func emitCalls(w http.ResponseWriter, reply modelReply) {
	raw, _ := json.Marshal(reply.Output)
	streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":`+string(raw)+`}}`)
}

func TestAgentRunsRealBashAndRestoresWithoutExecution(t *testing.T) {
	isolateConfig(t)
	bash, err := tools.NewBash(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if requests == 1 {
			args, _ := json.Marshal(tools.BashArgs{Command: "printf 'chapter-three' > result.txt; cat result.txt"})
			call, _ := json.Marshal(map[string]any{"type": "function_call", "call_id": "real", "name": "bash", "arguments": string(args)})
			emitCalls(w, modelReply{Output: []json.RawMessage{call}})
			return
		}
		input, _ := json.Marshal(body["input"])
		if !bytes.Contains(input, []byte("chapter-three")) || !bytes.Contains(input, []byte("function_call_output")) {
			t.Errorf("missing replayed result: %s", input)
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"The file contains chapter-three."}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	h, err := openHistory()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	chat := turnFixture{history: h, client: newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, ""), actions: &toolFixtureActions{bash: bash, approve: func(context.Context, tools.BashArgs) (bool, error) { return true, nil }, report: func(result tools.Result) error {
		fmt.Fprintf(&output, "[Tool result: %s; exit code %d]", result.Status, *result.ExitCode)
		return nil
	}}}
	if err := chat.respond(context.Background(), "Write the marker.", func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(bash.Directory(), "result.txt"))
	if err != nil || string(data) != "chapter-three" {
		t.Fatalf("file=%q err=%v", data, err)
	}
	if !strings.Contains(output.String(), "[Tool result: completed; exit code 0]") || requests != 2 {
		t.Fatalf("requests=%d output=%s", requests, &output)
	}
	h.Close()
	h, err = openHistory()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	chat.history = h
	chat.actions.execute = func(context.Context, tools.BashArgs) tools.Result {
		t.Fatal("replayed command executed again")
		return tools.Result{}
	}
	if err := chat.respond(context.Background(), "What happened?", func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests=%d", requests)
	}
}

func TestAgentBatchBoundaries(t *testing.T) {
	for _, mode := range []string{"separate approvals", "call limit", "cancel first", "start write failure", "start sync failure", "incomplete response"} {
		t.Run(mode, func(t *testing.T) {
			isolateConfig(t)
			h, err := openHistory()
			if err != nil {
				t.Fatal(err)
			}
			defer h.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			requests, executions, approvals := 0, 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if requests > 1 {
					streamEvent(w, `{"type":"response.output_text.delta","delta":"Done."}`)
					streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
					return
				}
				if mode == "incomplete response" {
					streamEvent(w, `{"type":"response.output_item.done","item":{"type":"function_call","call_id":"a","name":"bash","arguments":"{\"command\":\"go version\"}"}}`)
					return
				}
				ids := []string{"a", "b"}
				if mode == "call limit" {
					for i := 2; i < 17; i++ {
						ids = append(ids, fmt.Sprint(i))
					}
				}
				emitCalls(w, callReply(ids...))
			}))
			defer server.Close()
			bash, _ := tools.NewBash(t.TempDir(), t.TempDir())
			chat := turnFixture{history: h, client: newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, ""), actions: &toolFixtureActions{bash: bash, approve: func(context.Context, tools.BashArgs) (bool, error) {
				approvals++
				if mode == "start write failure" {
					h.writer = failingHistoryWriter{h.file, "write"}
				}
				if mode == "start sync failure" {
					h.writer = failingHistoryWriter{h.file, "sync"}
				}
				return approvals == 1, nil
			}, execute: func(context.Context, tools.BashArgs) tools.Result {
				executions++
				if mode == "cancel first" {
					cancel()
					return tools.Result{Status: "cancelled"}
				}
				code := 0
				return tools.Result{Status: "completed", ExitCode: &code}
			}}}
			err = chat.respond(ctx, "test", func(string) error { return nil })
			if mode == "separate approvals" {
				if err != nil || executions != 1 || approvals != 2 || requests != 2 {
					t.Fatalf("err=%v executions=%d approvals=%d requests=%d", err, executions, approvals, requests)
				}
			} else {
				if err == nil || requests != 1 {
					t.Fatalf("err=%v requests=%d", err, requests)
				}
				want := 0
				if mode == "cancel first" {
					want = 1
				}
				if executions != want {
					t.Fatalf("executions=%d", executions)
				}
			}
			if strings.HasPrefix(mode, "start ") {
				var storage *StorageError
				if !errors.As(err, &storage) {
					t.Fatalf("not storage error: %v", err)
				}
				return
			}
			if h.state.pending.id != "" {
				t.Fatal("turn left open")
			}
			data, _ := os.ReadFile(h.file.Name())
			if mode == "call limit" && bytes.Count(data, []byte(`"status":"not_executed"`)) != 17 {
				t.Fatal("limit did not close each call")
			}
			if mode == "cancel first" && !bytes.Contains(data, []byte(`"status":"not_executed"`)) {
				t.Fatal("cancel did not close remaining call")
			}
		})
	}
}
