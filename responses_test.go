package main

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
)

func TestRespondSendsIndependentRequests(t *testing.T) {
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
		if body["model"] != "test-model" || body["instructions"] != "请用中文回答。\n" || body["store"] != false {
			t.Errorf("unexpected request: %#v", body)
		}
		for _, field := range []string{"previous_response_id", "conversation", "messages", "tools"} {
			if _, ok := body[field]; ok {
				t.Errorf("single-turn request contains %s", field)
			}
		}
		input, ok := body["input"].(string)
		if !ok {
			t.Error("input must be the current question only")
		}
		inputs = append(inputs, input)
		fmt.Fprint(w, `{"status":"completed","output":[
			{"type":"reasoning","summary":[]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"你好，"},{"type":"output_text","text":"世界！"}]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"\n第二段。"}]}
		]}`)
	}))
	defer server.Close()
	client := newResponsesClient(config{server.URL + "/custom/v1", "test-key", "test-model"}, "请用中文回答。\n")
	for _, prompt := range []string{"我叫小明。", "我叫什么？"} {
		got, err := client.respond(context.Background(), prompt)
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
		{"bad JSON", 200, `<html>test-key</html>`, "", "JSON"},
		{"oversized response", 200, strings.Repeat("x", maxResponseBytes+1), "", "8 MiB"},
		{"empty output", 200, `{"status":"completed","output":[]}`, "", "no text"},
		{"failed", 200, `{"status":"failed","error":{"message":"test-key"}}`, "", "generation failed"},
		{"incomplete", 200, `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}`, "", "incomplete"},
		{"not completed", 200, `{"status":"queued"}`, "", "incomplete"},
		{"refusal", 200, `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"refusal","refusal":"无法帮助完成此请求。"}]}]}`, "Model refused: 无法帮助完成此请求。", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			client := newResponsesClient(config{server.URL, "test-key", "test-model"}, "instructions")
			got, err := client.respond(context.Background(), "你好")
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
	if _, err := client.respond(context.Background(), "你好"); err == nil || !strings.Contains(err.Error(), "307") {
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
		if _, err := client.respond(ctx, "你好"); !errors.Is(err, context.Canceled) {
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
		if _, err := client.respond(context.Background(), "你好"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout error = %v", err)
		}
	})
}
