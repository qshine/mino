package eval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
)

const maxRequestBytes = 8 << 20

// FakeModel is a loopback Responses API that answers from a script. The Agent
// talks to it exactly as it would talk to a real service, so replay evaluates
// Mino's own loop rather than a mock of it.
type FakeModel struct {
	server   *http.Server
	listener net.Listener
	mu       sync.Mutex
	script   []Step
	requests int
	err      error
}

func StartFake(script []Step) (*FakeModel, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("Cannot start the scripted model: %w", err)
	}
	f := &FakeModel{listener: listener, script: script}
	f.server = &http.Server{Handler: f}
	go f.server.Serve(listener)
	return f, nil
}

// URL is the base_url the Agent is configured with.
func (f *FakeModel) URL() string { return "http://" + f.listener.Addr().String() + "/v1" }

// Err reports the first script violation: a request beyond the script or a
// request body that did not contain what a step expected.
func (f *FakeModel) Err() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err
}

func (f *FakeModel) Requests() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

func (f *FakeModel) Close() error { return f.server.Close() }

func (f *FakeModel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
		http.NotFound(w, r)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes+1))
	if err != nil || len(body) > maxRequestBytes {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++
	index := f.requests
	w.Header().Set("Content-Type", "text/event-stream")
	if index > len(f.script) {
		f.record(fmt.Errorf("Model script exhausted: request %d has no scripted response", index))
		writeEvent(w, map[string]any{"type": "response.failed", "response": map[string]any{"id": fmt.Sprintf("resp_%d", index), "status": "failed"}})
		return
	}
	step := f.script[index-1]
	if step.ExpectInputContains != "" && !bytes.Contains(body, []byte(step.ExpectInputContains)) {
		f.record(fmt.Errorf("Request %d did not contain %q", index, step.ExpectInputContains))
	}
	var output []any
	if step.Call != nil {
		output = []any{map[string]any{
			"type": "function_call", "id": fmt.Sprintf("fc_%d", index), "call_id": fmt.Sprintf("call_%d", index),
			"name": step.Call.Tool, "arguments": string(step.Call.Arguments), "status": "completed",
		}}
	} else {
		writeEvent(w, map[string]any{"type": "response.output_text.delta", "delta": step.Text})
		output = []any{map[string]any{
			"type": "message", "id": fmt.Sprintf("msg_%d", index), "role": "assistant", "status": "completed",
			"content": []any{map[string]any{"type": "output_text", "text": step.Text, "annotations": []any{}}},
		}}
	}
	encoded, _ := json.Marshal(output)
	// Token counts are a stated approximation (four bytes per token) so the usage
	// path is exercised without pretending to be a tokenizer.
	input, generated := int64(len(body)/4), int64(len(encoded)/4)
	writeEvent(w, map[string]any{"type": "response.completed", "response": map[string]any{
		"id": fmt.Sprintf("resp_%d", index), "status": "completed", "output": output,
		"usage": map[string]any{"input_tokens": input, "output_tokens": generated, "total_tokens": input + generated},
	}})
}

func (f *FakeModel) record(err error) {
	if f.err == nil {
		f.err = err
	}
}

func writeEvent(w http.ResponseWriter, event map[string]any) {
	data, err := json.Marshal(event)
	if err != nil {
		data = []byte(`{"type":"error"}`)
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

var errScriptRequired = errors.New("replay mode requires a model script")
