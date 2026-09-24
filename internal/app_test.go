package mino

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUsesSoulInsteadOfProjectInstructions(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprintf("custom=%t", custom), func(t *testing.T) {
			path := isolateConfig(t)
			const instructions = "You are Mino. 请使用中文，回答不超过两句话。\n"
			for _, name := range []string{"AGENTS.md", "SOUL.md"} {
				if err := os.WriteFile(name, []byte("private development instructions"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				got, _ := body["instructions"].(string)
				if !strings.Contains(got, "You are Mino") || strings.Contains(got, "private development") {
					t.Errorf("request did not use Mino identity: %q", got)
				}
				if custom && got != instructions {
					t.Errorf("request did not preserve the user's SOUL.md: %q", got)
				}
				if body["input"] != "你好" {
					t.Errorf("input = %v", body["input"])
				}
				requests++
				fmt.Fprint(w, `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"你好！"}]}]}`)
			}))
			defer server.Close()
			if err := saveConfig(config{server.URL + "/v1", "test-key", "test-model"}); err != nil {
				t.Fatal(err)
			}
			if custom {
				if err := os.WriteFile(filepath.Join(filepath.Dir(path), "SOUL.md"), []byte(instructions), 0600); err != nil {
					t.Fatal(err)
				}
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
			if err != nil || len(files) != 2 || files[0].Name() != "AGENTS.md" || files[1].Name() != "SOUL.md" {
				t.Fatalf("program unexpectedly persisted files in the working directory: %v, %v", files, err)
			}
		})
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
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "SOUL.md")); !os.IsNotExist(err) {
		t.Fatal("incomplete setup created SOUL.md")
	}
}

func TestRunWithDefaultSoulOutsideProject(t *testing.T) {
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
