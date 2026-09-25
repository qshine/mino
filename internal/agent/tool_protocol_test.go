package agent

import (
	"context"
	"encoding/json"
	"github.com/openai/openai-go/v3/responses"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestToolOnlyResponseAndStatelessReplay(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		definitions, _ := body["tools"].([]any)
		if len(definitions) != 1 || definitions[0].(map[string]any)["name"] != "bash" {
			t.Error("missing Bash definition")
		}
		if requests == 1 {
			streamEvent(w, `{"type":"response.function_call_arguments.delta","delta":"incomplete"}`)
			streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":[{"type":"reasoning","id":"rs_1","summary":[],"encrypted_content":"opaque"},{"type":"function_call","id":"fc_1","call_id":"call_1","name":"bash","arguments":"{\"command\":\"go version\"}","status":"completed"}]}}`)
			return
		}
		items := body["input"].([]any)
		if len(items) != 4 || items[1].(map[string]any)["encrypted_content"] != "opaque" || items[2].(map[string]any)["call_id"] != "call_1" || items[3].(map[string]any)["call_id"] != "call_1" {
			t.Errorf("invalid replay: %#v", items)
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"Done."}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	client := newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, "")
	got, err := collectResponse(client, context.Background(), "Go version?")
	if err != nil || got != "Done." || requests != 2 {
		t.Fatalf("got=%q requests=%d err=%v", got, requests, err)
	}

}

func TestToolResponseRequiresCompleteValidBatch(t *testing.T) {
	for _, output := range []string{
		`[{"type":"function_call","call_id":"same","name":"bash","arguments":"{}"},{"type":"function_call","call_id":"same","name":"bash","arguments":"{}"}]`,
		`[{"type":"function_call","name":"bash","arguments":"{}"}]`,
		`[{"type":"function_call","call_id":"a","name":"bash","arguments":"{}","status":"in_progress"}]`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":`+output+`}}`)
		}))
		client := newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, "")
		_, err := client.respond(context.Background(), responses.ResponseInputParam{userInput("test")}, func(s string) error { _, e := io.WriteString(io.Discard, s); return e })
		server.Close()
		if err == nil {
			t.Errorf("accepted malformed call batch: %s", output)
		}
	}
}
