package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/skills"
	"github.com/qshine/mino/internal/tools"
)

func projectSkill(t *testing.T, root, description string) string {
	t.Helper()
	dir := filepath.Join(root, ".agents", "skills", "review")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte("---\nname: review\ndescription: "+description+"\n---\nBODY_ONLY_VIA_TOOL"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSkillCatalogStableAcrossTurnsSwitchClearAndRestart(t *testing.T) {
	project, dir := t.TempDir(), t.TempDir()
	projectSkill(t, project, "Original description")
	var prompts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Instructions string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		prompts = append(prompts, body.Instructions)
		streamEvent(w, `{"type":"response.output_text.delta","delta":"OK"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer server.Close()
	available := []tools.Tool{tools.NewSkillView(nil)}
	options := Options{BaseURL: server.URL, APIKey: "test", Model: "test", Instructions: "SOUL", ProjectDirectory: project}
	store, err := OpenSessions(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { store.Close() }()
	m, err := NewSessionManager(options, store, available)
	if err != nil {
		t.Fatal(err)
	}
	do := func(message string) {
		t.Helper()
		if err := m.Handle(context.Background(), message, testInteraction{confirm: func(context.Context, Confirmation) (bool, error) { return true, nil }}); err != nil {
			t.Fatal(err)
		}
	}
	first := store.id
	do("First")
	projectSkill(t, project, "Changed description")
	do("Second")
	do("/new")
	do("Third")
	do("/resume " + first)
	do("Fourth")
	do("/clear")
	do("Fifth")
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	// Restoring a session must not rescan, even if the project no longer exists.
	if err := os.RemoveAll(filepath.Join(project, ".agents")); err != nil {
		t.Fatal(err)
	}
	store, err = OpenSessions(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	options.ProjectDirectory = t.TempDir()
	projectSkill(t, options.ProjectDirectory, "Different project")
	m, err = NewSessionManager(options, store, available)
	if err != nil {
		t.Fatal(err)
	}
	do("Sixth")
	if len(prompts) != 6 {
		t.Fatalf("prompts=%d", len(prompts))
	}
	for _, index := range []int{1, 3, 4, 5} {
		if prompts[index] != prompts[0] {
			t.Fatalf("prompt changed at %d", index)
		}
	}
	if !strings.Contains(prompts[0], "Original description") || strings.Contains(prompts[0], "BODY_ONLY") || !strings.HasPrefix(prompts[0], "SOUL") || !strings.Contains(prompts[2], "Changed description") {
		t.Fatalf("prompts=%v", prompts)
	}
}

func TestSkillViewLoopWithoutApprovalAndDurableReplay(t *testing.T) {
	project, dir := t.TempDir(), t.TempDir()
	path := projectSkill(t, project, "Review code")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Instructions string
			Input        []map[string]any
			Tools        []map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		requests++
		if strings.Contains(body.Instructions, "BODY_ONLY") || !strings.Contains(body.Instructions, "Review code") {
			t.Errorf("instructions=%s", body.Instructions)
		}
		if len(body.Tools) != 1 || body.Tools[0]["name"] != "skill_view" {
			t.Errorf("tools=%v", body.Tools)
		}
		if requests == 1 {
			streamEvent(w, `{"type":"response.completed","response":{"status":"completed","output":[{"type":"function_call","name":"skill_view","call_id":"view-1","arguments":"{\"name\":\"review\"}"}]}}`)
		} else {
			if len(body.Input) != 3 || body.Input[2]["call_id"] != "view-1" || !strings.Contains(body.Input[2]["output"].(string), "BODY_ONLY_VIA_TOOL") {
				t.Errorf("input=%v", body.Input)
			}
			streamEvent(w, `{"type":"response.output_text.delta","delta":"Read the skill."}`)
			streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
		}
	}))
	defer server.Close()
	available := []tools.Tool{tools.NewSkillView(nil)}
	s, err := OpenSession(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test", ProjectDirectory: project}, s, available)
	if err != nil {
		t.Fatal(err)
	}
	err = a.Handle(context.Background(), "Review this", testInteraction{confirm: func(context.Context, Confirmation) (bool, error) {
		t.Error("skill_view requested approval")
		return false, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests=%d", requests)
	}
	s.Close()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	s, err = OpenSession(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if len(s.state.input) != 4 || s.state.skillCatalog == nil {
		t.Fatal("skill tool result/catalog not recovered")
	}
}

func TestSkillSnapshotSyncFailureDoesNotCommitMemory(t *testing.T) {
	s, err := OpenSession(t.TempDir(), []tools.Tool{tools.NewSkillView(nil)})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.writer = &skillFailWriter{}
	_, err = New(Options{ProjectDirectory: t.TempDir()}, s, []tools.Tool{tools.NewSkillView(nil)})
	var storage *StorageError
	if !errors.As(err, &storage) || s.state.skillCatalog != nil || s.state.seq != 0 {
		t.Fatalf("err=%v, state=%+v", err, s.state)
	}
}

type skillFailWriter struct{}

func (*skillFailWriter) Write(p []byte) (int, error) { return len(p), nil }
func (*skillFailWriter) Sync() error                 { return errors.New("sync failed") }

func TestSkillEmptySnapshotAndInvalidNewKeepSelection(t *testing.T) {
	project := t.TempDir()
	available := []tools.Tool{tools.NewSkillView(nil)}
	store, err := OpenSessions(t.TempDir(), available)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	m, err := NewSessionManager(Options{ProjectDirectory: project}, store, available)
	if err != nil {
		t.Fatal(err)
	}
	first, prompt := store.id, m.runner.options.Instructions
	path := projectSkill(t, project, "Appeared later")
	m, err = NewSessionManager(Options{ProjectDirectory: project}, store, available)
	if err != nil || m.runner.options.Instructions != prompt {
		t.Fatalf("empty catalog rescanned: %v", err)
	}
	if err := os.WriteFile(path, []byte("broken metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(context.Background(), "/new", testInteraction{}); err == nil {
		t.Fatal("invalid skill accepted")
	}
	if store.id != first || m.runner.session != store.current || m.runner.options.Instructions != prompt {
		t.Fatal("failed /new changed selection")
	}
}

func TestSkillCatalogSurvivesCompactionAndCountsInBudget(t *testing.T) {
	project := t.TempDir()
	projectSkill(t, project, strings.Repeat("Description ", 80))
	available := []tools.Tool{tools.NewSkillView(nil)}
	s, err := OpenSession(t.TempDir(), available)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var prompt string
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Instructions string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		requests++
		if body.Instructions != summaryInstructions && body.Instructions != prompt {
			t.Error("compaction changed skill instructions")
		}
		sendCompactText(w, "Retain the goal.")
	}))
	defer server.Close()
	a, err := New(Options{BaseURL: server.URL, APIKey: "test", Model: "test", ProjectDirectory: project, ContextWindow: 512}, s, available)
	if err != nil {
		t.Fatal(err)
	}
	prompt = a.options.Instructions
	if err := a.Handle(context.Background(), "Hello", testInteraction{}); err == nil || requests != 0 {
		t.Fatalf("catalog ignored in budget: %v, requests=%d", err, requests)
	}
	a.options.ContextWindow = DefaultContextWindow
	seedCompactTurns(t, s, 4, 400)
	if err := a.Compact(context.Background(), testInteraction{}); err != nil {
		t.Fatal(err)
	}
	if err := a.Handle(context.Background(), "Continue", testInteraction{}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || s.state.summary == "" || a.options.Instructions != prompt {
		t.Fatal("compaction lost fixed catalog")
	}
}

func TestSkillHistoryRejectsInvalidSnapshotTransitions(t *testing.T) {
	for _, mode := range []string{"old version", "bad catalog", "duplicate", "during turn", "wrong kind", "extra text"} {
		t.Run(mode, func(t *testing.T) {
			s, err := OpenSession(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			c := &skills.Catalog{Root: t.TempDir()}
			if mode == "duplicate" {
				if err := s.append(historyRecord{TurnID: newHistoryID(), Kind: "session_skills", Skills: c}); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "during turn" {
				if err := s.append(historyRecord{TurnID: newHistoryID(), Kind: "user_message", Text: "Pending"}); err != nil {
					t.Fatal(err)
				}
			}
			r := historyRecord{Version: 4, Seq: s.state.seq + 1, TurnID: newHistoryID(), Kind: "session_skills", Skills: c}
			switch mode {
			case "old version":
				r.Version = 3
			case "bad catalog":
				c.Root = "relative"
			case "wrong kind":
				r.Kind = "user_message"
				r.Text = "Hello"
			case "extra text":
				r.Text = "extra"
			}
			if err := s.state.apply(r); err == nil {
				t.Fatal("accepted invalid snapshot transition")
			}
		})
	}
}
