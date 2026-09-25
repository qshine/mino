package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/qshine/mino/internal/gateway"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestConversationStopsWhenSavingDisplayedAnswerFails(t *testing.T) {
	isolateConfig(t)
	h, err := openHistory()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		streamEvent(w, `{"type":"response.output_text.delta","delta":"answer"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	chat := Agent{h, newResponsesClient(Options{server.URL, "test-key", "test-model"}, "")}
	output := &observingWriter{onWrite: func(text string) {
		if strings.Contains(text, "Assistant> answer") {
			h.writer = failingHistoryWriter{h.file, "write"}
		}
	}}
	err = gateway.Run(context.Background(), strings.NewReader("first\nsecond\n"), output, io.Discard, chat.Handle)
	var storageErr *StorageError
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
			defer h.Close()
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
			chat := Agent{h, newResponsesClient(Options{server.URL, "test-key", "test-model"}, "")}
			err = chat.Handle(ctx, "question", func(s string) error {
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
