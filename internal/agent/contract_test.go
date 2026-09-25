package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/tools"
)

type echoTool struct {
	execute func(context.Context, tools.Call) tools.Result
}

func (e echoTool) Definition() tools.Definition {
	return tools.Definition{Name: "echo", Description: "Return the supplied text.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}, "required": []string{"text"}, "additionalProperties": false}, Strict: true}
}
func (e echoTool) Prepare(raw string) (tools.Call, error) {
	var args struct {
		Text string `json:"text"`
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&args); err != nil {
		return tools.Call{}, err
	}
	data, _ := json.Marshal(args)
	return tools.Call{Arguments: data, Fields: []tools.Field{{Label: "Text", Value: args.Text}}}, nil
}
func (e echoTool) Execute(ctx context.Context, call tools.Call) tools.Result {
	return e.execute(ctx, call)
}

func TestHandlePersistsEachBoundaryAndUsesInjectedTool(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "history.jsonl")
	readRecords := func() []historyRecord {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var records []historyRecord
		for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
			var record historyRecord
			if err := json.Unmarshal(line, &record); err != nil {
				t.Fatal(err)
			}
			records = append(records, record)
		}
		return records
	}
	executions, approvals, requests := 0, 0, 0
	echo := echoTool{execute: func(_ context.Context, call tools.Call) tools.Result {
		executions++
		records := readRecords()
		if records[len(records)-1].Kind != "tool_start" {
			t.Error("execution preceded durable start")
		}
		if string(call.Arguments) != `{"text":"hello"}` {
			t.Error("approved arguments changed")
		}
		return tools.Result{Status: "completed", Output: "hello"}
	}}
	available := []tools.Tool{echo}
	session, err := OpenSession(directory, available)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		records := readRecords()
		var body struct {
			Input []map[string]any
			Tools []map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Tools) != 1 || body.Tools[0]["name"] != "echo" {
			t.Error("injected tool missing from model request")
		}
		if requests == 1 {
			if len(records) != 1 || records[0].Kind != "user_message" || len(body.Input) != 1 {
				t.Error("request preceded durable user message or duplicated it")
			}
			streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":[{"type":"function_call","call_id":"one","name":"echo","arguments":"{\"text\":\"hello\"}"}]}}`)
		} else {
			if records[len(records)-1].Kind != "tool_result" || len(body.Input) != 3 || body.Input[2]["call_id"] != "one" {
				t.Error("request preceded paired durable result")
			}
			streamEvent(w, `{"type":"response.output_text.delta","delta":"Done."}`)
			streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
		}
	}))
	defer server.Close()
	a, err := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test"}, session, available)
	if err != nil {
		t.Fatal(err)
	}
	interaction := testInteraction{confirm: func(_ context.Context, request Confirmation) (bool, error) {
		approvals++
		records := readRecords()
		if records[len(records)-1].Kind != "model_response" {
			t.Error("approval preceded durable model response")
		}
		// A gateway may format or modify its display copy without changing execution.
		request.Fields[0].Value = "changed display"
		return true, nil
	}}
	if err := a.Handle(context.Background(), "echo hello", interaction); err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, record := range readRecords() {
		kinds = append(kinds, record.Kind)
	}
	if !reflect.DeepEqual(kinds, []string{"user_message", "model_response", "tool_start", "tool_result", "model_response", "turn_end"}) {
		t.Fatalf("records=%v", kinds)
	}
	if approvals != 1 || executions != 1 || requests != 2 || len(session.state.input) != 4 {
		t.Fatalf("approvals=%d executions=%d requests=%d memory=%d", approvals, executions, requests, len(session.state.input))
	}
}

func TestSessionRejectsConcurrentTurnsBeforeWriting(t *testing.T) {
	session, err := OpenSession(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		streamEvent(w, `{"type":"response.output_text.delta","delta":"done"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	a, _ := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test"}, session, nil)
	done := make(chan error, 1)
	go func() { done <- a.Handle(context.Background(), "first", testInteraction{}) }()
	<-started
	err = a.Handle(context.Background(), "second", testInteraction{})
	close(release)
	firstErr := <-done
	if !errors.Is(err, ErrBusy) || firstErr != nil {
		t.Fatalf("concurrent=%v first=%v", err, firstErr)
	}
	data, _ := os.ReadFile(session.file.Name())
	if bytes.Contains(data, []byte("second")) {
		t.Fatal("busy turn was persisted")
	}
}

func TestSessionDoesNotCommitMemoryAfterSyncFailure(t *testing.T) {
	session, err := OpenSession(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	session.writer = failingHistoryWriter{session.file, "sync"}
	a, _ := New(Options{BaseURL: "http://127.0.0.1:1", APIKey: "test", Model: "test"}, session, nil)
	err = a.Handle(context.Background(), "hello", testInteraction{})
	var storage *StorageError
	if !errors.As(err, &storage) || session.state.seq != 0 || session.state.pending.id != "" || len(session.state.input) != 0 {
		t.Fatalf("err=%v state=%+v", err, session.state)
	}
}

func TestAgentRejectsDuplicateToolNames(t *testing.T) {
	session, err := OpenSession(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if _, err := New(Options{}, session, []tools.Tool{echoTool{}, echoTool{}}); err == nil {
		t.Fatal("duplicate tool names accepted")
	}
}

func TestStoredArgumentsKeepLargeIntegersDistinct(t *testing.T) {
	if sameArguments(json.RawMessage(`{"value":9007199254740992}`), json.RawMessage(`{"value":9007199254740993}`)) {
		t.Fatal("different tool arguments were treated as equal")
	}
}
