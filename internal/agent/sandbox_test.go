package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/policy"
	"github.com/qshine/mino/internal/sandbox"
	"github.com/qshine/mino/internal/tools"
)

// A sandboxed command is the second kind of call accept-edits runs unasked. The
// opt-out is a different call: it asks in that mode, and the sandboxed command
// still asks in manual mode.
func TestSandboxedBashInModes(t *testing.T) {
	if !sandbox.Available() {
		t.Skip("sandbox-exec is unavailable on this system")
	}
	// functionCall takes string arguments; the sandbox field is a boolean or null.
	call := func(id, arguments string) string {
		item, _ := json.Marshal(map[string]string{"type": "function_call", "name": "bash", "call_id": id, "arguments": arguments})
		return `{"type":"response.completed","response":{"status":"completed","output":[` + string(item) + `]}}`
	}
	script := []string{
		call("c1", `{"command":"printf hi > out.txt && cat .env","sandbox":null}`),
		call("c2", `{"command":"echo x > ../escaped.txt","sandbox":false}`),
	}
	for _, mode := range []policy.Mode{policy.Manual, policy.AcceptEdits} {
		project, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(project, ".env"), []byte("API_KEY=SECRET-VALUE\n"), 0600)
		bash, err := tools.NewBash(project, filepath.Dir(project))
		if err != nil {
			t.Fatal(err)
		}
		confined, err := sandbox.New(project, filepath.Dir(project))
		if err != nil {
			t.Fatal(err)
		}
		bash.SetSandbox(confined)
		available := []tools.Tool{bash}
		var outputs, infos, asked []string
		requests := 0
		model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			if strings.Contains(string(raw), "SECRET-VALUE") {
				t.Error("protected content reached the model service")
			}
			var body struct{ Input []map[string]any }
			json.Unmarshal(raw, &body)
			if requests > 0 {
				outputs = append(outputs, body.Input[len(body.Input)-1]["output"].(string))
			}
			if requests < len(script) {
				streamEvent(w, script[requests])
			} else {
				streamEvent(w, `{"type":"response.output_text.delta","delta":"Done."}`)
				streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
			}
			requests++
		}))
		s, err := OpenSession(t.TempDir(), available)
		if err != nil {
			t.Fatal(err)
		}
		p := policy.New()
		p.SetMode(mode)
		a, err := New(Options{BaseURL: model.URL, APIKey: "test", Model: "test", Policy: p}, s, available)
		if err != nil {
			t.Fatal(err)
		}
		err = a.Handle(context.Background(), "Go", testInteraction{emit: func(e Event) error {
			if e.Kind == "info" {
				infos = append(infos, e.Text)
			}
			return nil
		}, confirm: func(_ context.Context, c Confirmation) (bool, error) {
			asked = append(asked, c.ToolName)
			for _, f := range c.Fields {
				if f.Label == "Sandbox" {
					asked[len(asked)-1] += ":" + strings.SplitN(f.Value, ":", 2)[0]
				}
			}
			return false, nil
		}})
		s.Close()
		model.Close()
		if err != nil {
			t.Fatal(err)
		}
		switch mode {
		case policy.Manual:
			if strings.Join(asked, ",") != "bash:on,bash:off" || len(infos) != 0 {
				t.Fatalf("manual asked=%v infos=%v", asked, infos)
			}
			if _, err := os.Stat(filepath.Join(project, "out.txt")); err == nil {
				t.Fatal("manual mode ran the command without approval")
			}
		case policy.AcceptEdits:
			if strings.Join(asked, ",") != "bash:off" || len(infos) != 1 || !strings.Contains(infos[0], "bash, allowed without confirmation in accept-edits mode") || !strings.Contains(infos[0], `Sandbox: "on: writes only inside `+project) {
				t.Fatalf("accept-edits asked=%v infos=%q", asked, infos)
			}
			// The command ran confined: the project write succeeded and the
			// kernel refused the protected read, which the model then sees.
			if data, err := os.ReadFile(filepath.Join(project, "out.txt")); err != nil || string(data) != "hi" {
				t.Fatalf("sandboxed command did not run: %q %v", data, err)
			}
			if len(outputs) != 2 || !strings.Contains(outputs[0], "Operation not permitted") || strings.Contains(outputs[0], "SECRET-VALUE") || !strings.Contains(outputs[1], "did not approve") {
				t.Fatalf("outputs=%q", outputs)
			}
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(project), "escaped.txt")); err == nil {
			t.Fatal("the opt-out command ran without approval")
		}
	}
}
