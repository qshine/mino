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
				if custom && !strings.HasPrefix(got, instructions+"\n\nAvailable project skills") {
					t.Errorf("request did not preserve the user's SOUL.md: %q", got)
				}
				if input, ok := body["input"].([]any); !ok || len(input) != 1 || input[0].(map[string]any)["content"] != "你好" {
					t.Errorf("input = %v", body["input"])
				}
				requests++
				streamEvent(w, `{"type":"response.output_text.delta","delta":"你好！"}`)
				streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
			}))
			defer server.Close()
			if err := saveConfig(config{server.URL + "/v1", "test-key", "test-model", 0}); err != nil {
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
	if err := saveConfig(config{"https://api.openai.com/v1", "fake-key", "test-model", 0}); err != nil {
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

func TestRunSendsMemoryIndexAndMemoryTools(t *testing.T) {
	path := isolateConfig(t)
	memories := filepath.Join(filepath.Dir(path), "memory")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []map[string]any
			Tools []map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		var names []string
		for _, tool := range body.Tools {
			names = append(names, fmt.Sprint(tool["name"]))
		}
		if !strings.Contains(strings.Join(names, ","), "memory_view,memory_save") {
			t.Errorf("tools = %v", names)
		}
		if len(body.Input) != 2 || body.Input[0]["role"] != "assistant" || !strings.Contains(fmt.Sprint(body.Input[0]["content"]), "- [user] name: The user's name is Lin.") {
			t.Errorf("input = %v", body.Input)
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"Hi Lin"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	if err := saveConfig(config{server.URL + "/v1", "test-key", "test-model", 0}); err != nil {
		t.Fatal(err)
	}
	// The first start creates the private directory; a memory saved later is read.
	var output bytes.Buffer
	if err := run(context.Background(), strings.NewReader("/exit\n"), &output, &output); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(memories); err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("memory directory = %v, %v", info, err)
	}
	if err := os.WriteFile(filepath.Join(memories, "name.md"), []byte("---\nname: name\ntype: user\nsummary: The user's name is Lin.\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := run(context.Background(), strings.NewReader("hello\n/exit\n"), &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Hi Lin") {
		t.Fatalf("output = %q", output.String())
	}
}
