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
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionCommandsIsolateModelRequestsAndClear(t *testing.T) {
	var requests [][]map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Input []map[string]any }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		requests = append(requests, body.Input)
		streamEvent(w, `{"type":"response.output_text.delta","delta":"Noted."}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	store, err := OpenSessions(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := NewSessionManager(Options{BaseURL: server.URL, APIKey: "test", Model: "test"}, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	interaction := testInteraction{}
	do := func(message string) {
		t.Helper()
		if err := manager.Handle(ctx, message, interaction); err != nil {
			t.Fatal(err)
		}
	}
	first := store.id
	do("Remember pine.")
	do("/new")
	second := store.id
	do("What word?")
	do("/sessions")
	do("/help")
	if err := manager.Handle(ctx, "/resume ../bad", interaction); err == nil {
		t.Fatal("accepted path")
	}
	do("/resume " + first)
	do("What word?")
	if len(requests) != 3 || len(requests[1]) != 1 || len(requests[2]) != 3 || requests[2][0]["content"] != "Remember pine." {
		t.Fatalf("requests=%v", requests)
	}
	do("/clear") // Denied: state stays intact.
	if len(store.current.state.input) != 4 {
		t.Fatal("denied clear changed state")
	}
	interaction.confirm = func(_ context.Context, c Confirmation) (bool, error) {
		if c.Kind != "clear" || c.SessionID != first {
			t.Fatalf("confirmation=%+v", c)
		}
		return true, nil
	}
	do("/clear")
	do("After clear")
	if len(requests[3]) != 1 || store.id != first {
		t.Fatal("clear retained context or changed ID")
	}
	do("/resume " + second)
	if len(store.current.state.input) != 2 {
		t.Fatal("clear affected other session")
	}
	data, _ := os.ReadFile(store.path(first))
	if strings.Contains(string(data), "Remember pine.") || strings.Contains(string(data), "/clear") {
		t.Fatal("clear retained history or command was persisted")
	}
}

func TestSessionSelectionModeDoesNotCallModel(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := store.id
	store.Close()
	os.WriteFile(filepath.Join(dir, "active-session.json"), []byte("bad"), 0600)
	store, err = OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := NewSessionManager(Options{}, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background(), testInteraction{}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Handle(context.Background(), "hello", testInteraction{}); err == nil || !strings.Contains(err.Error(), "session") {
		t.Fatalf("err=%v", err)
	}
	if err := manager.Handle(context.Background(), "/resume "+id, testInteraction{}); err != nil {
		t.Fatal(err)
	}
}

func TestSessionMutationFailuresKeepMemoryAndStopChat(t *testing.T) {
	for _, operation := range []string{"clear", "switch"} {
		for _, failure := range []string{"rename", "directory sync"} {
			t.Run(operation+failure, func(t *testing.T) {
				store, err := OpenSessions(t.TempDir(), nil)
				if err != nil {
					t.Fatal(err)
				}
				defer store.Close()
				target := store.id
				if err := store.create(); err != nil {
					t.Fatal(err)
				}
				current := store.current
				if err := current.append(historyRecord{TurnID: newHistoryID(), Kind: "user_message", Text: "keep"}); err != nil {
					t.Fatal(err)
				}
				if err := current.finishTurn("cancelled"); err != nil {
					t.Fatal(err)
				}
				if failure == "rename" {
					store.rename = func(string, string) error { return errors.New("rename failed") }
				} else {
					store.syncDir = func(string) error { return errors.New("sync failed") }
				}
				manager, _ := NewSessionManager(Options{}, store, nil)
				command := "/resume " + target
				if operation == "clear" {
					command = "/clear"
				}
				err = manager.Handle(context.Background(), command, testInteraction{confirm: func(context.Context, Confirmation) (bool, error) { return true, nil }})
				var storage *StorageError
				if !errors.As(err, &storage) {
					t.Fatalf("err=%v", err)
				}
				if store.current != current || store.current.state.seq != 2 {
					t.Fatal("memory committed after publication failure")
				}
				if err = manager.Handle(context.Background(), "must not be saved", testInteraction{}); !errors.As(err, &storage) {
					t.Fatalf("manager continued after failure: %v", err)
				}
			})
		}
	}
}

func TestResumeRetainsToolPairsAndRequiresAcknowledgement(t *testing.T) {
	dir := t.TempDir()
	bash, err := tools.NewBash(dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	executions := 0
	wrapped := fixtureTool{Tool: bash, execute: func(context.Context, tools.Call) tools.Result { executions++; return tools.Result{Status: "completed"} }}
	available := []tools.Tool{wrapped}
	store, err := OpenSessions(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	first := store.id
	turn := newHistoryID()
	if err := store.current.append(
		historyRecord{TurnID: turn, Kind: "user_message", Text: "Run a command"},
		historyRecord{TurnID: turn, Kind: "model_response", Output: callReply("one").Output},
		historyRecord{TurnID: turn, Kind: "tool_start", CallID: "one", Name: "bash", Arguments: json.RawMessage(`{"command":"go version"}`), CWD: dir},
	); err != nil {
		t.Fatal(err)
	}
	// Simulate interruption by closing a log containing a durable tool_start.
	store.Close()
	// Select a different clean session, then resume the interrupted one.
	store = &Sessions{dir: dir, registry: map[string]tools.Tool{"bash": wrapped}, rename: os.Rename, syncDir: syncHistoryDirectory}
	if err := store.create(); err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	second := store.id
	manager, _ := NewSessionManager(Options{}, store, available)
	confirms := 0
	interaction := testInteraction{confirm: func(_ context.Context, c Confirmation) (bool, error) {
		confirms++
		if c.Kind != "recovery" {
			t.Fatal(c.Kind)
		}
		return false, nil
	}}
	if err := manager.Handle(context.Background(), "/resume "+first, interaction); !errors.Is(err, io.EOF) {
		t.Fatalf("err=%v", err)
	}
	if store.id != second {
		t.Fatal("declined recovery changed selection")
	}
	interaction.confirm = func(context.Context, Confirmation) (bool, error) { confirms++; return true, nil }
	if err := manager.Handle(context.Background(), "/resume "+first, interaction); err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(store.current.state.input)
	if executions != 0 || confirms != 2 || !strings.Contains(string(input), `"call_id":"one"`) || !strings.Contains(string(input), `unknown`) || len(store.current.state.input) != 4 {
		t.Fatalf("executions=%d confirms=%d input=%s", executions, confirms, input)
	}
	data, _ := os.ReadFile(store.path(first))
	if !strings.Contains(string(data), `"kind":"recovery_ack"`) {
		t.Fatal("acknowledgement not durable")
	}
}

func TestSessionCommandsCannotInterleaveWithRunningTurn(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		streamEvent(w, `{"type":"response.output_text.delta","delta":"Done."}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	store, err := OpenSessions(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, _ := NewSessionManager(Options{BaseURL: server.URL, APIKey: "test", Model: "test"}, store, nil)
	first := store.id
	done := make(chan error, 1)
	go func() { done <- manager.Handle(context.Background(), "first", testInteraction{}) }()
	<-started
	for _, command := range []string{"/new", "/clear", "/resume " + first, "/sessions"} {
		if err := manager.Handle(context.Background(), command, testInteraction{}); !errors.Is(err, ErrBusy) {
			t.Errorf("%s: %v", command, err)
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if store.id != first {
		t.Fatal("running session was replaced")
	}
}
