package mino

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func streamEvent(w http.ResponseWriter, event string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprintf(w, "data: %s\n\n", event)
	w.(http.Flusher).Flush()
}

type observingWriter struct {
	bytes.Buffer
	onWrite func(string)
}

func (w *observingWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	w.onWrite(w.String())
	return n, err
}

func TestRunStreamsBeforeResponseCompletes(t *testing.T) {
	isolateConfig(t)
	firstDisplayed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body["stream"] != true {
			t.Error("request must enable streaming")
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"你好，"}`)
		// The server cannot finish until the terminal has displayed the first delta.
		select {
		case <-firstDisplayed:
		case <-r.Context().Done():
			return
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"世界！"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	if err := saveConfig(config{server.URL, "test-key", "test-model", 0}); err != nil {
		t.Fatal(err)
	}
	output := &observingWriter{onWrite: func(text string) {
		if strings.Contains(text, "Assistant> 你好，") {
			select {
			case <-firstDisplayed:
			default:
				close(firstDisplayed)
			}
		}
	}}
	var errorOutput bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := run(ctx, strings.NewReader("你好\n/exit\n"), output, &errorOutput)
	if err != nil || errorOutput.Len() != 0 || !strings.Contains(output.String(), "Assistant> 你好，世界！\n") {
		t.Fatalf("streamed terminal output = %q, stderr = %q, error = %v", output.String(), errorOutput.String(), err)
	}
}
