package mino

import (
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

	"github.com/openai/openai-go/v3/responses"
)

func TestRespondSendsOnlySuppliedInput(t *testing.T) {
	var inputs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/custom/v1/responses" {
			t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing authentication or JSON header")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body["model"] != "test-model" || body["instructions"] != "请用中文回答。\n" || body["store"] != false || body["stream"] != true {
			t.Errorf("unexpected request: %#v", body)
		}
		for _, field := range []string{"previous_response_id", "conversation", "messages", "tools"} {
			if _, ok := body[field]; ok {
				t.Errorf("single-turn request contains %s", field)
			}
		}
		items, ok := body["input"].([]any)
		if !ok || len(items) != 1 {
			t.Errorf("unexpected explicit input: %#v", body["input"])
			return
		}
		input := items[0].(map[string]any)["content"].(string)
		inputs = append(inputs, input)
		streamEvent(w, `{"type":"response.reasoning_text.delta","delta":"private reasoning"}`)
		streamEvent(w, `{"type":"response.output_text.delta","delta":"你好，"}`)
		streamEvent(w, `{"type":"response.output_text.delta","delta":"世界！"}`)
		streamEvent(w, `{"type":"response.output_text.delta","delta":"\n第二段。"}`)
		streamEvent(w, `{"type":"response.output_text.done","text":"你好，世界！\n第二段。"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"你好，世界！\n第二段。"}]}]}}`)
	}))
	defer server.Close()
	client := newResponsesClient(config{server.URL + "/custom/v1", "test-key", "test-model"}, "请用中文回答。\n")
	for _, prompt := range []string{"我叫小明。", "我叫什么？"} {
		got, err := collectResponse(client, context.Background(), prompt)
		if err != nil || got != "你好，世界！\n第二段。" {
			t.Fatalf("response = %q, error = %v", got, err)
		}
	}
	if strings.Join(inputs, "|") != "我叫小明。|我叫什么？" {
		t.Fatalf("requests mixed conversation history: %q", inputs)
	}
}

func TestRespondHandlesFailuresAndRefusal(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		status                int
		body, want, wantError string
	}{
		{"unauthorized", 401, `{"error":{"message":"invalid test-key"}}`, "", "HTTP 401"},
		{"rate limited", 429, `{"error":{"message":"slow down"}}`, "", "HTTP 429"},
		{"server error", 502, `<html>test-key</html>`, "", "HTTP 502"},
		{"bad JSON", 200, `<html>test-key</html>`, "", "invalid"},
		{"trailing JSON", 200, `{"type":"response.completed"} private-data`, "", "invalid"},
		{"null response", 200, `null`, "", "incomplete"},
		{"oversized response", 200, strings.Repeat("x", maxResponseBytes+1), "", "8 MiB"},
		{"empty output", 200, `{"type":"response.completed","response":{"status":"completed"}}`, "", "no text"},
		{"failed", 200, `{"type":"response.failed","response":{"status":"failed","error":{"message":"test-key"}}}`, "", "generation failed"},
		{"error event", 200, `{"type":"error","message":"test-key"}`, "", "generation failed"},
		{"nested error", 200, `{"error":{"message":"test-key"}}`, "", "invalid"},
		{"incomplete", 200, `{"type":"response.incomplete","response":{"status":"incomplete"}}`, "", "incomplete"},
		{"not completed", 200, `{"type":"response.completed","response":{"status":"queued"}}`, "", "incomplete"},
		{"completed with error", 200, `{"type":"response.completed","response":{"status":"completed","error":{"message":"test-key"}}}`, "", "generation failed"},
		{"EOF", 200, `{"type":"response.output_text.delta","delta":"partial"}`, "", "incomplete"},
		{"DONE", 200, `[DONE]`, "", "incomplete"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(tc.status)
				fmt.Fprintf(w, "data: %s\n\n", tc.body)
			}))
			defer server.Close()
			client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "instructions")
			got, err := collectResponse(client, context.Background(), "你好")
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				if strings.Contains(err.Error(), "test-key") {
					t.Fatalf("error leaks API key: %v", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("response = %q, error = %v", got, err)
			}
		})
	}
}

func TestRespondIgnoresEnvironmentConfiguration(t *testing.T) {
	for name, value := range map[string]string{
		"OPENAI_BASE_URL":       "http://127.0.0.1:1/unwanted",
		"OPENAI_API_KEY":        "environment-key",
		"OPENAI_ADMIN_KEY":      "environment-admin-key",
		"OPENAI_ORG_ID":         "environment-org",
		"OPENAI_PROJECT_ID":     "environment-project",
		"OPENAI_MODEL":          "environment-model",
		"OPENAI_WEBHOOK_SECRET": "environment-webhook-secret",
		"OPENAI_CUSTOM_HEADERS": "X-Unwanted: environment-header",
	} {
		t.Setenv(name, value)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/v1/responses" || r.Header.Get("Authorization") != "Bearer configured-key" {
			t.Error("request did not use explicit settings")
		}
		for _, header := range []string{"OpenAI-Organization", "OpenAI-Project", "X-Unwanted"} {
			if r.Header.Get(header) != "" {
				t.Errorf("request inherited %s from the environment", header)
			}
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body["model"] != "configured-model" || body["instructions"] != "" || body["store"] != false || body["stream"] != true {
			t.Errorf("unexpected request settings: %#v", body)
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"Hello"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed","error":null}}`)
	}))
	defer server.Close()
	client := newResponsesClient(config{server.URL + "/custom/v1", "configured-key", "configured-model"}, "")
	if got, err := collectResponse(client, context.Background(), "Hello"); err != nil || got != "Hello" {
		t.Fatalf("response = %q, error = %v", got, err)
	}
}

func TestRespondDoesNotRetry(t *testing.T) {
	for _, status := range []int{408, 409, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(status)
				fmt.Fprint(w, `{"error":{"message":"private-data"}}`)
			}))
			defer server.Close()
			client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "")
			_, err := collectResponse(client, context.Background(), "Hello")
			if err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) || strings.Contains(err.Error(), "private-data") {
				t.Fatalf("unexpected API error: %v", err)
			}
			if requests != 1 {
				t.Fatalf("sent %d requests, want one without retries", requests)
			}
		})
	}
}

func TestRespondDoesNotFollowRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("redirect destination must not receive the request or API key")
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "instructions")
	if _, err := collectResponse(client, context.Background(), "你好"); err == nil || !strings.Contains(err.Error(), "307") {
		t.Fatalf("redirect error = %v", err)
	}
}

func TestRespondHonorsCancellationAndTimeout(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		started := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 先消费请求体，HTTP/1 服务端才会在后台监测连接关闭。
			io.Copy(io.Discard, r.Body)
			close(started)
			<-r.Context().Done()
		}))
		defer server.Close()
		client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "instructions")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() { <-started; cancel() }()
		if _, err := collectResponse(client, ctx, "你好"); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation error = %v", err)
		}
	})
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.Copy(io.Discard, r.Body)
			<-r.Context().Done()
		}))
		defer server.Close()
		client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "instructions")
		client.httpClient.Timeout = 20 * time.Millisecond
		if _, err := collectResponse(client, context.Background(), "你好"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout error = %v", err)
		}
	})
}

// Collecting is test-only; the terminal receives deltas without waiting for completion.
func collectResponse(client *responsesClient, ctx context.Context, prompt string) (string, error) {
	var output strings.Builder
	_, err := client.respond(ctx, responses.ResponseInputParam{userInput(prompt)}, func(delta string) error {
		output.WriteString(delta)
		return nil
	})
	return output.String(), err
}
