package mino

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	if err := saveConfig(config{server.URL, "test-key", "test-model"}); err != nil {
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

func TestRespondStreamsRefusalWithoutRepeatingDoneText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamEvent(w, `{"type":"response.refusal.delta","delta":"无法帮助"}`)
		streamEvent(w, `{"type":"response.refusal.delta","delta":"完成此请求。"}`)
		streamEvent(w, `{"type":"response.refusal.done","refusal":"无法帮助完成此请求。"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "")
	got, err := collectResponse(client, context.Background(), "Hello")
	if err != nil || got != "Model refused: 无法帮助完成此请求。" {
		t.Fatalf("refusal = %q, error = %v", got, err)
	}
}

func TestRespondStopsWhileStreamIsOpen(t *testing.T) {
	for _, mode := range []string{"completed", "cancel", "timeout", "output failure"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				streamEvent(w, `{"type":"response.output_text.delta","delta":"partial"}`)
				if mode == "completed" {
					streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
				}
				<-r.Context().Done()
			}))
			defer server.Close()
			client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if mode == "timeout" {
				client.httpClient.Timeout = 100 * time.Millisecond
			}
			outputError := errors.New("output unavailable")
			var got string
			err := client.respond(ctx, "Hello", func(delta string) error {
				got += delta
				if mode == "cancel" {
					cancel()
				}
				if mode == "output failure" {
					return outputError
				}
				return nil
			})
			wantError := map[string]error{"cancel": context.Canceled, "timeout": context.DeadlineExceeded, "output failure": outputError}[mode]
			if got != "partial" || !errors.Is(err, wantError) {
				t.Fatalf("output = %q, error = %v, want %v", got, err, wantError)
			}
		})
	}
}

func TestRespondRejectsNonStreamingEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"completed","output":[],"private":"test-key"}`)
	}))
	defer server.Close()
	client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "")
	_, err := collectResponse(client, context.Background(), "Hello")
	if err == nil || !strings.Contains(err.Error(), "supports streaming") || strings.Contains(err.Error(), "test-key") {
		t.Fatalf("non-streaming endpoint error = %v", err)
	}
}

func TestTerminalKeepsPartialAnswerAndContinuesAfterStreamFailure(t *testing.T) {
	var output, errorOutput bytes.Buffer
	err := runTerminal(context.Background(), strings.NewReader("first\nsecond\n/exit\n"), &output, &errorOutput,
		func(_ context.Context, prompt string, emit func(string) error) error {
			if prompt == "first" {
				if err := emit("partial\x1b"); err != nil {
					return err
				}
				return errors.New("Model response is incomplete")
			}
			return emit("recovered")
		})
	if err != nil || strings.Contains(output.String(), "\x1b") || !strings.Contains(output.String(), "Assistant> partial\n") || !strings.Contains(output.String(), "Assistant> recovered\n") || !strings.Contains(errorOutput.String(), "incomplete") {
		t.Fatalf("output = %q, stderr = %q, error = %v", output.String(), errorOutput.String(), err)
	}
}

func TestResponseBodyLimit(t *testing.T) {
	for _, size := range []int{maxResponseBytes - 1, maxResponseBytes, maxResponseBytes + 1} {
		body := &limitedResponseBody{ReadCloser: io.NopCloser(strings.NewReader(strings.Repeat("x", size))), remaining: maxResponseBytes}
		n, err := io.Copy(io.Discard, body)
		if size > maxResponseBytes {
			if !errors.Is(err, errResponseTooLarge) || n > maxResponseBytes {
				t.Fatalf("oversize read = %d, error = %v", n, err)
			}
		} else if err != nil || n != int64(size) {
			t.Fatalf("size %d: read = %d, error = %v", size, n, err)
		}
	}
}
