package agent

import (
	"context"
	"errors"
	"fmt"
	"github.com/openai/openai-go/v3/responses"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRespondStreamsRefusalWithoutRepeatingDoneText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamEvent(w, `{"type":"response.refusal.delta","delta":"无法帮助"}`)
		streamEvent(w, `{"type":"response.refusal.delta","delta":"完成此请求。"}`)
		streamEvent(w, `{"type":"response.refusal.done","refusal":"无法帮助完成此请求。"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	client := newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, "")
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
			client := newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, "")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if mode == "timeout" {
				client.httpClient.Timeout = 100 * time.Millisecond
			}
			outputError := errors.New("output unavailable")
			var got string
			_, err := client.respond(ctx, responses.ResponseInputParam{userInput("Hello")}, func(delta string) error {
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
	client := newModelFixture(t, testConfig{server.URL, "test-key", "test-model"}, "")
	_, err := collectResponse(client, context.Background(), "Hello")
	if err == nil || !strings.Contains(err.Error(), "supports streaming") || strings.Contains(err.Error(), "test-key") {
		t.Fatalf("non-streaming endpoint error = %v", err)
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
