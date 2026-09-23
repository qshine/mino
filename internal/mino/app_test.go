package mino

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRunFromProjectRoot(t *testing.T) {
	path := isolateConfig(t)
	const instructions = "请使用中文，回答不超过两句话。\n"
	if err := os.WriteFile("AGENTS.md", []byte(instructions), 0600); err != nil {
		t.Fatal(err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["instructions"] != instructions || body["input"] != "你好" {
			t.Errorf("request did not include project instructions and question: %#v", body)
		}
		requests++
		fmt.Fprint(w, `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"你好！"}]}]}`)
	}))
	defer server.Close()
	if err := saveConfig(config{server.URL + "/v1", "test-key", "test-model"}); err != nil {
		t.Fatal(err)
	}
	var output, errorOutput bytes.Buffer
	if err := run(context.Background(), strings.NewReader("你好\n/exit\n"), &output, &errorOutput); err != nil {
		t.Fatal(err)
	}
	if requests != 1 || !strings.Contains(output.String(), "Assistant> 你好！") || errorOutput.Len() != 0 {
		t.Fatalf("requests = %d, output = %q, stderr = %q", requests, output.String(), errorOutput.String())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("configuration was not saved in the home directory")
	}
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name() != "AGENTS.md" {
		t.Fatalf("program unexpectedly persisted files: %v", files)
	}
}

func TestRunReportsStartupErrors(t *testing.T) {
	path := isolateConfig(t)
	var output bytes.Buffer
	if err := run(context.Background(), strings.NewReader("这是一条聊天输入\n"), &output, &output); err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("missing config with piped input error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("piped chat input was saved as configuration")
	}
}

func TestRunWithoutProjectInstructions(t *testing.T) {
	isolateConfig(t)
	if err := saveConfig(config{"https://api.openai.com/v1", "fake-key", "test-model"}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run(context.Background(), strings.NewReader("/exit\n"), &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Goodbye.") {
		t.Fatal("chat did not start outside a project")
	}
}
