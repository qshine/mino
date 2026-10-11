package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/tools"
)

func functionCall(name, id string, arguments map[string]string) string {
	raw, _ := json.Marshal(arguments)
	item, _ := json.Marshal(map[string]string{"type": "function_call", "name": name, "call_id": id, "arguments": string(raw)})
	return `{"type":"response.completed","response":{"status":"completed","output":[` + string(item) + `]}}`
}

func lastOutput(t *testing.T, input []map[string]any, id string) string {
	t.Helper()
	last := input[len(input)-1]
	if last["call_id"] != id {
		t.Fatalf("last input is not the %s result: %v", id, last)
	}
	return last["output"].(string)
}

func TestFileToolsReadFreelyAndWriteOnlyWithApproval(t *testing.T) {
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(project, "greet.go")
	if err := os.WriteFile(path, []byte("package main\n\nconst greeting = \"hi\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := tools.NewFiles(project, filepath.Join(t.TempDir(), ".mino"))
	if err != nil {
		t.Fatal(err)
	}
	available := []tools.Tool{files.Read(), files.Write(), files.Edit()}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []map[string]any
			Tools []map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Tools) != 3 {
			t.Errorf("tools=%v", body.Tools)
		}
		requests++
		switch requests {
		case 1:
			streamEvent(w, functionCall("read_file", "read-1", map[string]string{"path": "greet.go"}))
		case 2:
			if out := lastOutput(t, body.Input, "read-1"); !strings.Contains(out, `const greeting = \"hi\"`) {
				t.Errorf("read result=%s", out)
			}
			streamEvent(w, functionCall("edit_file", "edit-1", map[string]string{"path": "greet.go", "old_string": `"hi"`, "new_string": `"hello"`}))
		case 3:
			if out := lastOutput(t, body.Input, "edit-1"); !strings.Contains(out, `"status":"completed"`) {
				t.Errorf("edit result=%s", out)
			}
			streamEvent(w, functionCall("edit_file", "edit-missing", map[string]string{"path": "greet.go", "old_string": "absent", "new_string": "x"}))
		case 4:
			if out := lastOutput(t, body.Input, "edit-missing"); !strings.Contains(out, "invalid_arguments") {
				t.Errorf("missing edit result=%s", out)
			}
			streamEvent(w, functionCall("write_file", "write-1", map[string]string{"path": "greet.go", "content": "package main\n", "mode": "overwrite"}))
		case 5:
			if out := lastOutput(t, body.Input, "write-1"); !strings.Contains(out, `"status":"denied"`) {
				t.Errorf("write result=%s", out)
			}
			streamEvent(w, `{"type":"response.output_text.delta","delta":"Done."}`)
			streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
		default:
			t.Errorf("unexpected request %d", requests)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	s, err := OpenSession(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test"}, s, available)
	if err != nil {
		t.Fatal(err)
	}
	var confirmed []Confirmation
	err = a.Handle(context.Background(), "Change the greeting", testInteraction{confirm: func(_ context.Context, c Confirmation) (bool, error) {
		confirmed = append(confirmed, c)
		return c.ToolName == "edit_file", nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	// Only the inspected edit and the overwrite reached approval: the read needed
	// none, and the edit whose old_string was absent failed before confirmation.
	if requests != 5 || len(confirmed) != 2 || confirmed[0].ToolName != "edit_file" || confirmed[1].ToolName != "write_file" {
		t.Fatalf("requests=%d confirmations=%+v", requests, confirmed)
	}
	fields := map[string]string{}
	for _, field := range confirmed[0].Fields {
		fields[field.Label] = field.Value
	}
	if fields["Remove"] != `"hi"` || fields["Insert"] != `"hello"` || fields["Starting line"] != "3" {
		t.Fatalf("edit confirmation=%+v", confirmed[0].Fields)
	}
	if data, _ := os.ReadFile(path); string(data) != "package main\n\nconst greeting = \"hello\"\n" {
		t.Fatalf("file=%q", data)
	}
	s.Close()

	// Replay validates recorded arguments without reading the file again, so a
	// later change or removal cannot make the saved session unreadable.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	s, err = OpenSession(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.state.pending.id != "" || len(s.state.replayTurns) != 1 {
		t.Fatal("file tool turn was not recovered")
	}
}
