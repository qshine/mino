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

func TestRunRestoresConversationHistory(t *testing.T) {
	path := filepath.Join(filepath.Dir(isolateConfig(t)), "history.jsonl")
	var inputs [][]map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input   []map[string]any `json:"input"`
			Store   bool             `json:"store"`
			Include []string         `json:"include"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		inputs = append(inputs, body.Input)
		if body.Store {
			t.Error("response storage must stay disabled")
		}
		if len(body.Include) != 1 || body.Include[0] != "reasoning.encrypted_content" {
			t.Error("missing stateless reasoning request")
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"Blue."}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":[{"type":"reasoning","id":"rs_1","summary":[],"encrypted_content":"opaque-state"},{"type":"message","id":"msg_1","role":"assistant","status":"completed","phase":"final_answer","content":[{"type":"output_text","text":"Blue.","annotations":[]}]}]}}`)
	}))
	defer server.Close()
	if err := saveConfig(config{server.URL, "test-key", "test-model"}); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"My favorite color is blue.\n/exit\n", "What color?\n/exit\n"} {
		var output, stderr bytes.Buffer
		if err := run(context.Background(), strings.NewReader(input), &output, &stderr); err != nil || stderr.Len() != 0 {
			t.Fatalf("run: %v, stderr: %s", err, &stderr)
		}
		if !strings.Contains(output.String(), "Assistant> Blue.") {
			t.Fatal("answer not displayed")
		}
	}
	if len(inputs) != 2 || len(inputs[0]) != 1 || len(inputs[1]) != 4 {
		t.Fatalf("request history = %#v", inputs)
	}
	if inputs[1][0]["content"] != "My favorite color is blue." || inputs[1][1]["encrypted_content"] != "opaque-state" || inputs[1][2]["phase"] != "final_answer" || inputs[1][3]["content"] != "What color?" {
		t.Fatalf("history lost content or protocol fields: %#v", inputs[1])
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var turnIDs [2]string
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != 6 {
		t.Fatalf("history has %d records", len(lines))
	}
	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatal(err)
		}
		if _, ok := record["session_id"]; ok {
			t.Fatalf("record contains removed session_id: %s", line)
		}
		id, _ := record["turn_id"].(string)
		if i%3 == 0 {
			turnIDs[i/3] = id
		}
		if (len(id) != 32 || strings.Trim(id, "0123456789abcdef") != "") || id != turnIDs[i/3] || record["seq"] != float64(i+1) {
			t.Fatalf("invalid turn/order: %s", line)
		}
	}
	if turnIDs[0] == turnIDs[1] {
		t.Fatal("different turns reused the same identifier")
	}
	if strings.Contains(string(data), "test-key") {
		t.Fatal("configuration key was saved in history")
	}
	for _, name := range []string{"history.jsonl", "history.lock"} {
		info, err := os.Stat(filepath.Join(filepath.Dir(path), name))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("permissions for %s: %v, %v", name, info, err)
		}
	}
}

func TestRunFailedTurnDoesNotEnterContext(t *testing.T) {
	isolateConfig(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if requests == 1 {
			streamEvent(w, `{"type":"response.output_text.delta","delta":"partial"}`)
			return
		}
		input, _ := body["input"].([]any)
		if len(input) != 1 {
			t.Errorf("failed turn entered context: %#v", body["input"])
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"Recovered."}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	if err := saveConfig(config{server.URL, "test-key", "test-model"}); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if err := run(context.Background(), strings.NewReader("first\nsecond\n/exit\n"), io.Discard, &stderr); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !strings.Contains(stderr.String(), "incomplete") {
		t.Fatalf("requests=%d, errors=%s", requests, &stderr)
	}
}
