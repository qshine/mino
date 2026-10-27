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

	"github.com/qshine/mino/internal/memory"
	"github.com/qshine/mino/internal/policy"
	"github.com/qshine/mino/internal/tools"
)

type memoryRun struct {
	inputs  [][]map[string]any // request inputs in order
	asked   []string
	notices []string
	infos   []string
	session string // session log text
}

// runMemoryTurns drives prompts through one Agent backed by a real memory store.
// script supplies the stream for each model request; approve answers confirmations.
func runMemoryTurns(t *testing.T, mode policy.Mode, store *memory.Store, script []string, approve bool, prompts ...string) memoryRun {
	t.Helper()
	var run memoryRun
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct{ Input []map[string]any }
		json.Unmarshal(raw, &body)
		run.inputs = append(run.inputs, body.Input)
		if len(run.inputs) <= len(script) && script[len(run.inputs)-1] != "" {
			streamEvent(w, script[len(run.inputs)-1])
			return
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"ok"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer model.Close()
	available := []tools.Tool{tools.NewMemoryView(store), tools.NewMemorySave(store)}
	dir := t.TempDir()
	s, err := OpenSession(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rules := policy.New()
	rules.SetMode(mode)
	a, err := New(Options{BaseURL: model.URL, APIKey: "test", Model: "test", Policy: rules, Memory: store}, s, available)
	if err != nil {
		t.Fatal(err)
	}
	interaction := testInteraction{emit: func(e Event) error {
		if e.Kind == "notice" {
			run.notices = append(run.notices, e.Text)
		}
		if e.Kind == "info" {
			run.infos = append(run.infos, e.Text)
		}
		return nil
	}, confirm: func(_ context.Context, c Confirmation) (bool, error) {
		run.asked = append(run.asked, c.ToolName)
		return approve, nil
	}}
	for _, prompt := range prompts {
		if err := a.Handle(context.Background(), prompt, interaction); err != nil {
			t.Fatal(err)
		}
	}
	logs, _ := filepath.Glob(filepath.Join(dir, "sessions", "*.jsonl"))
	if len(logs) == 1 {
		data, _ := os.ReadFile(logs[0])
		run.session = string(data)
	}
	return run
}

func openTestStore(t *testing.T, memories ...memory.Memory) *memory.Store {
	t.Helper()
	store, err := memory.Open(filepath.Join(t.TempDir(), "memory"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range memories {
		if err := os.WriteFile(filepath.Join(store.Dir(), memory.FileName(m.Name)), []byte(memory.Format(m)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

func frontText(t *testing.T, input []map[string]any) string {
	t.Helper()
	if len(input) < 2 || input[0]["role"] != "assistant" || input[1]["role"] != "user" {
		t.Fatalf("input does not start with one assistant memory message: %v", input)
	}
	text, _ := input[0]["content"].(string)
	return text
}

func TestMemoryIndexIsRebuiltEachTurnAndNeverStored(t *testing.T) {
	store := openTestStore(t, memory.Memory{Name: "prefers-chinese", Type: "feedback", Summary: "Always answer in Chinese"})
	script := []string{functionCall("memory_save", "c1", map[string]string{"name": "uses-pnpm", "type": "project", "summary": "Install with pnpm, not npm", "details": "", "mode": "create"})}
	run := runMemoryTurns(t, policy.Manual, store, script, true, "Remember that we use pnpm", "Install the dependencies")
	if len(run.inputs) != 3 || strings.Join(run.asked, ",") != "memory_save" {
		t.Fatalf("requests=%d asked=%v", len(run.inputs), run.asked)
	}
	for i, input := range run.inputs[:2] {
		text := frontText(t, input)
		if !strings.HasPrefix(text, memory.Label) || !strings.Contains(text, "- [feedback] prefers-chinese: Always answer in Chinese") || strings.Contains(text, "uses-pnpm") {
			t.Fatalf("turn 1 request %d memory = %q", i+1, text)
		}
	}
	if text := frontText(t, run.inputs[2]); !strings.Contains(text, "- [project] uses-pnpm: Install with pnpm, not npm") {
		t.Fatalf("turn 2 memory = %q", text)
	}
	if strings.Contains(run.session, "Mino user memory") {
		t.Fatal("the memory index was written to session history")
	}
}

func TestMemorySaveAsksUnlessBypass(t *testing.T) {
	for _, tc := range []struct {
		mode  policy.Mode
		asked string
	}{{policy.AcceptEdits, "memory_save"}, {policy.Bypass, ""}} {
		store := openTestStore(t)
		script := []string{functionCall("memory_save", "c1", map[string]string{"name": "fact", "type": "user", "summary": "始终用中文回答", "details": "", "mode": "create"})}
		run := runMemoryTurns(t, tc.mode, store, script, false, "Remember this")
		if strings.Join(run.asked, ",") != tc.asked {
			t.Fatalf("%s asked %v", tc.mode, run.asked)
		}
		// An unasked save is still shown, with its summary readable.
		if tc.mode == policy.Bypass && !strings.Contains(strings.Join(run.infos, ""), `Summary: "始终用中文回答"`) {
			t.Fatalf("bypass display = %q", run.infos)
		}
		_, err := os.Stat(filepath.Join(store.Dir(), "fact.md"))
		if saved := err == nil; saved != (tc.mode == policy.Bypass) {
			t.Fatalf("%s saved=%v", tc.mode, saved)
		}
		// An empty store adds nothing, so requests look exactly as before Chapter 13.
		if first := run.inputs[0][0]; first["role"] != "user" {
			t.Fatalf("empty memory still sent a front message: %v", first)
		}
	}
}

func TestMemoryViewNeedsNoApprovalAndProblemsAreShownOnce(t *testing.T) {
	store := openTestStore(t, memory.Memory{Name: "release", Type: "reference", Summary: "Release with check.sh", Details: "Step one: build."})
	os.WriteFile(filepath.Join(store.Dir(), "broken.md"), []byte("not a memory"), 0600)
	script := []string{functionCall("memory_view", "c1", map[string]string{"name": "release"})}
	run := runMemoryTurns(t, policy.Manual, store, script, false, "How do I release?", "Thanks")
	if len(run.asked) != 0 {
		t.Fatalf("memory_view asked for approval: %v", run.asked)
	}
	if out := lastOutput(t, run.inputs[1], "c1"); !strings.Contains(out, "Step one: build.") {
		t.Fatalf("memory_view result = %q", out)
	}
	if text := frontText(t, run.inputs[0]); !strings.Contains(text, "release: Release with check.sh (+details)") {
		t.Fatalf("memory = %q", text)
	}
	if len(run.notices) != 1 || !strings.Contains(run.notices[0], "broken.md") {
		t.Fatalf("notices = %q", run.notices)
	}
}

func TestFrontInputMergesMemoryAndSummary(t *testing.T) {
	if frontInput("", "") != nil {
		t.Fatal("no memory and no summary must add no message")
	}
	raw, _ := json.Marshal(frontInput(memory.Label+"- [user] a: b\n", "earlier work"))
	var items []map[string]any
	json.Unmarshal(raw, &items)
	if len(items) != 1 || items[0]["role"] != "assistant" {
		t.Fatalf("front input = %s", raw)
	}
	text := items[0]["content"].(string)
	if !strings.HasPrefix(text, memory.Label) || !strings.Contains(text, "\n"+summaryLabel+"earlier work") {
		t.Fatalf("front text = %q", text)
	}
}
