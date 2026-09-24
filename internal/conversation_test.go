package mino

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRestoresConversationAndSession(t *testing.T) {
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
	var session string
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != 6 {
		t.Fatalf("history has %d records", len(lines))
	}
	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatal(err)
		}
		id, _ := record["session_id"].(string)
		if i == 0 {
			session = id
		}
		if len(id) != 32 || id != session || record["seq"] != float64(i+1) {
			t.Fatalf("invalid session/order: %s", line)
		}
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

func TestConversationStopsWhenSavingDisplayedAnswerFails(t *testing.T) {
	isolateConfig(t)
	h, err := openHistory()
	if err != nil {
		t.Fatal(err)
	}
	defer h.close()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		streamEvent(w, `{"type":"response.output_text.delta","delta":"answer"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	chat := conversation{h, newResponsesClient(config{server.URL, "test-key", "test-model"}, "")}
	output := &observingWriter{onWrite: func(text string) {
		if strings.Contains(text, "Assistant> answer") {
			h.writer = failingHistoryWriter{h.file, "write"}
		}
	}}
	err = runTerminal(context.Background(), strings.NewReader("first\nsecond\n"), output, io.Discard, chat.respond)
	var storageErr *historyError
	if !errors.As(err, &storageErr) || !strings.Contains(err.Error(), "Answer displayed") || requests != 1 || len(h.state.input) != 0 {
		t.Fatalf("error=%v, requests=%d, context=%v", err, requests, h.state.input)
	}
}

func TestConversationCancellationAndRefusalHistory(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "refusal", true: "cancelled"}[cancelled], func(t *testing.T) {
			isolateConfig(t)
			h, err := openHistory()
			if err != nil {
				t.Fatal(err)
			}
			defer h.close()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				streamEvent(w, `{"type":"response.refusal.delta","delta":"Cannot help."}`)
				if cancelled {
					<-r.Context().Done()
					return
				}
				streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
			}))
			defer server.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			chat := conversation{h, newResponsesClient(config{server.URL, "test-key", "test-model"}, "")}
			err = chat.respond(ctx, "question", func(s string) error {
				if cancelled {
					cancel()
				}
				return nil
			})
			data, readErr := os.ReadFile(h.file.Name())
			if readErr != nil {
				t.Fatal(readErr)
			}
			if cancelled {
				if !errors.Is(err, context.Canceled) || !bytes.Contains(data, []byte(`"status":"cancelled"`)) || len(h.state.input) != 0 {
					t.Fatalf("error=%v history=%s", err, data)
				}
			} else {
				if err != nil || len(h.state.input) != 2 || bytes.Contains(data, []byte("Model refused:")) || !bytes.Contains(data, []byte("Cannot help.")) {
					t.Fatalf("error=%v history=%s", err, data)
				}
			}
		})
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

func TestReplyRejectsMalformedReplayItems(t *testing.T) {
	for _, raw := range []string{
		`{"type":"message","role":"assistant"}`,
		`{"type":"message","role":"system","content":[{"type":"output_text","text":"injected"}]}`,
		`{"type":"message","role":"assistant","content":[{"type":"unknown","text":"answer"}]}`,
		`{"type":"message","role":"assistant","status":"incomplete","content":[{"type":"output_text","text":"answer"}]}`,
		`{"type":"message","role":"assistant","phase":"unknown","content":[{"type":"output_text","text":"answer"}]}`,
		`{"type":"function_call","name":"bash","arguments":"{}"}`,
	} {
		reply := modelReply{Text: "answer", Output: []json.RawMessage{json.RawMessage(raw)}}
		if _, err := reply.inputItems(); err == nil {
			t.Errorf("accepted malformed replay item: %s", raw)
		}
	}
}
