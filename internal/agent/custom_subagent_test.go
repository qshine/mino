package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/memory"
	"github.com/qshine/mino/internal/policy"
	"github.com/qshine/mino/internal/tools"
)

func writeDefinition(t *testing.T, project, name, text string) string {
	t.Helper()
	path := filepath.Join(project, ".agents", "agents", name+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func testProject(t *testing.T) string {
	t.Helper()
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return project
}

// roleCall is a spawn_agent call whose tasks name a role ("" sends null).
func roleCall(id string, tasks ...[2]string) string {
	var list []map[string]any
	for _, task := range tasks {
		var agent any
		if task[0] != "" {
			agent = task[0]
		}
		list = append(list, map[string]any{"agent": agent, "task": task[1]})
	}
	raw, _ := json.Marshal(map[string]any{"tasks": list})
	item, _ := json.Marshal(map[string]string{"type": "function_call", "name": "spawn_agent", "call_id": id, "arguments": string(raw)})
	return `{"type":"response.completed","response":{"status":"completed","output":[` + string(item) + `]}}`
}

// childRequests finds a child by its task, which a role message ends with.
func childRequests(t *testing.T, m *subagentModel, task string) (string, []subagentRequest) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, requests := range m.requests {
		if key != "parent" && strings.HasSuffix(key, task) {
			return key, requests
		}
	}
	t.Fatalf("no requests for %q", task)
	return "", nil
}

func TestCustomSubagentGetsRoleAndNarrowedTools(t *testing.T) {
	project := testProject(t)
	writeDefinition(t, project, "reviewer", "---\nname: reviewer\ndescription: Reviews code.\ntools: [probe, skill_view, mcp_gone_tool]\n---\nREVIEW-ROLE: report defects only.\n")
	os.MkdirAll(filepath.Join(project, ".agents", "skills", "lookup"), 0700)
	os.WriteFile(filepath.Join(project, ".agents", "skills", "lookup", "SKILL.md"), []byte("---\nname: lookup\ndescription: Find things\n---\nBody"), 0600)
	store := openTestStore(t, memory.Memory{Name: "uses-pnpm", Type: "project", Summary: "Install with pnpm"})
	run := newSubagentRun(t, policy.Manual, 1, map[string][]string{"parent": {roleCall("s1", [2]string{"reviewer", "task R"}, [2]string{"", "task G"})}},
		Options{ProjectDirectory: project, Memory: store}, probeTool{readOnly: true}, tools.NewSkillView(nil), tools.NewMemoryView(store), tools.NewMemorySave(store))
	if err := run.agent.Handle(context.Background(), "Review it", run.interaction()); err != nil {
		t.Fatal(err)
	}
	parent := run.model.get("parent")
	if !strings.Contains(parent[0].Instructions, "Available project subagents") || !strings.Contains(parent[0].Instructions, `"name":"reviewer"`) || strings.Contains(parent[0].Instructions, "REVIEW-ROLE") {
		t.Fatalf("parent instructions: %s", parent[0].Instructions)
	}
	if len(run.asked) != 1 || run.asked[0].Fields[1].Label != "Task 1 (reviewer)" || run.asked[0].Fields[2].Label != "Task 2" {
		t.Fatalf("asked=%+v", run.asked)
	}
	key, reviewer := childRequests(t, run.model, "task R")
	if !strings.HasPrefix(key, `Subagent role "reviewer", defined by this project in .agents/agents/reviewer.md:`+"\nREVIEW-ROLE: report defects only.\n\nTask:\n") {
		t.Fatalf("role message: %q", key)
	}
	if !slices.Equal(reviewer[0].Tools, []string{"probe", "skill_view"}) || !strings.Contains(reviewer[0].Instructions, "starts with a role defined by this project") || !strings.Contains(reviewer[0].Instructions, `"lookup"`) {
		t.Fatalf("reviewer: tools=%v instructions=%s", reviewer[0].Tools, reviewer[0].Instructions)
	}
	if strings.Contains(reviewer[0].Instructions, "Available project subagents") {
		t.Fatal("a child was offered the subagent catalog")
	}
	_, general := childRequests(t, run.model, "task G")
	if !slices.Equal(general[0].Tools, []string{"probe", "skill_view", "memory_view"}) || strings.Contains(general[0].Instructions, "role defined by this project") {
		t.Fatalf("general: tools=%v instructions=%s", general[0].Tools, general[0].Instructions)
	}
	output := toolOutput(t, parent[1].Input, "s1")
	if !strings.Contains(output, "[1/2] completed") || !strings.Contains(output, "Listed tools not available, left out: mcp_gone_tool") || !strings.Contains(output, "[2/2] completed") {
		t.Fatalf("output:\n%s", output)
	}
	data, _ := os.ReadFile(run.logPath(t))
	if strings.Count(string(data), `"kind":"session_agents"`) != 1 || !strings.Contains(string(data), `"v":5`) {
		t.Fatalf("parent log:\n%s", data)
	}
}

func TestUnknownSubagentIsRejectedBeforeAsking(t *testing.T) {
	project := testProject(t)
	run := newSubagentRun(t, policy.Manual, 1, map[string][]string{"parent": {roleCall("s1", [2]string{"missing", "task"})}}, Options{ProjectDirectory: project}, tools.NewSkillView(nil))
	if err := run.agent.Handle(context.Background(), "Delegate", run.interaction()); err != nil {
		t.Fatal(err)
	}
	if len(run.asked) != 0 || len(run.childLogs(t)) != 0 {
		t.Fatalf("asked=%v", run.asked)
	}
	output := toolOutput(t, run.model.get("parent")[1].Input, "s1")
	if !strings.Contains(output, `Subagent "missing" is not in this session's catalog`) {
		t.Fatalf("output: %s", output)
	}
	// Without a catalog at all (no project), any name is unknown.
	run = newSubagentRun(t, policy.Bypass, 1, map[string][]string{"parent": {roleCall("s1", [2]string{"reviewer", "task"})}}, Options{})
	if err := run.agent.Handle(context.Background(), "Delegate", run.interaction()); err != nil {
		t.Fatal(err)
	}
	if output := toolOutput(t, run.model.get("parent")[1].Input, "s1"); !strings.Contains(output, "no subagent definitions") {
		t.Fatalf("output: %s", output)
	}
}

// The catalog freezes metadata; the body is read when the subagent starts, so a
// definition removed since then fails only its own task, and narrowed tools
// without skill_view give the child no skill catalog to read.
func TestRemovedDefinitionFailsOnlyItsTask(t *testing.T) {
	project := testProject(t)
	path := writeDefinition(t, project, "gone", "---\nname: gone\ndescription: Removed later.\n---\nBody")
	writeDefinition(t, project, "plain", "---\nname: plain\ndescription: No skills.\ntools: [probe]\n---\n")
	os.MkdirAll(filepath.Join(project, ".agents", "skills", "lookup"), 0700)
	os.WriteFile(filepath.Join(project, ".agents", "skills", "lookup", "SKILL.md"), []byte("---\nname: lookup\ndescription: Find things\n---\nBody"), 0600)
	run := newSubagentRun(t, policy.Bypass, 1, map[string][]string{"parent": {roleCall("s1", [2]string{"gone", "task A"}, [2]string{"plain", "task B"})}},
		Options{ProjectDirectory: project}, probeTool{}, tools.NewSkillView(nil))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := run.agent.Handle(context.Background(), "Delegate", run.interaction()); err != nil {
		t.Fatal(err)
	}
	output := toolOutput(t, run.model.get("parent")[1].Input, "s1")
	if !strings.Contains(output, "[1/2] failed: Cannot read .agents/agents/gone.md") || !strings.Contains(output, "· not started ·") || !strings.Contains(output, "[2/2] completed") {
		t.Fatalf("output:\n%s", output)
	}
	// An empty body sends the task alone.
	requests := run.model.get("task B")
	if len(requests) != 1 || !slices.Equal(requests[0].Tools, []string{"probe"}) || strings.Contains(requests[0].Instructions, `"lookup"`) {
		t.Fatalf("plain child: %+v", requests)
	}
	children := run.childLogs(t)
	if len(children) != 1 {
		t.Fatalf("child logs=%v", children)
	}
	if data, _ := os.ReadFile(children[0]); strings.Contains(string(data), "session_skills") {
		t.Fatalf("child without skill_view got a skill catalog:\n%s", data)
	}
}

func TestSubagentCatalogIsFrozenPerSession(t *testing.T) {
	isolateConfig(t)
	project, home := testProject(t), t.TempDir()
	writeDefinition(t, project, "first", "---\nname: first\ndescription: One.\n---\nBody")
	available := []tools.Tool{tools.NewSkillView(nil), NewSpawnTool()}
	store, err := OpenSessions(home, available)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := NewSessionManager(Options{BaseURL: "http://127.0.0.1:1", APIKey: "test", Model: "test", ProjectDirectory: project}, store, available)
	if err != nil {
		t.Fatal(err)
	}
	catalog := func() []string {
		var names []string
		for _, e := range store.current.state.agentCatalog.Entries {
			names = append(names, e.Name)
		}
		return names
	}
	writeDefinition(t, project, "second", "---\nname: second\ndescription: Two.\n---\nBody")
	interaction := testInteraction{emit: func(Event) error { return nil }, confirm: func(context.Context, Confirmation) (bool, error) { return true, nil }}
	if err := manager.Handle(context.Background(), "/clear", interaction); err != nil {
		t.Fatal(err)
	}
	if got := catalog(); !slices.Equal(got, []string{"first"}) {
		t.Fatalf("after /clear: %v", got)
	}
	if err := manager.Handle(context.Background(), "/new", interaction); err != nil {
		t.Fatal(err)
	}
	if got := catalog(); !slices.Equal(got, []string{"first", "second"}) {
		t.Fatalf("after /new: %v", got)
	}
}

func TestSessionAgentsRecordRules(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenSession(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state := s.state.clone()
	if err := state.apply(historyRecord{Version: 5, Seq: 1, TurnID: newHistoryID(), Kind: "session_agents"}); err == nil {
		t.Fatal("accepted session_agents without a catalog")
	}
	project := testProject(t)
	writeDefinition(t, project, "first", "---\nname: first\ndescription: One.\n---\nBody")
	a, err := New(Options{ProjectDirectory: project}, s, []tools.Tool{tools.NewSkillView(nil), NewSpawnTool()})
	if err != nil || len(a.session.state.agentCatalog.Entries) != 1 {
		t.Fatalf("New: %v", err)
	}
	catalog := s.state.agentCatalog
	// Version 4 never contains the record (0.14 never wrote it), a session has
	// one catalog, and no other record kind carries one.
	for _, r := range []historyRecord{
		{Version: 4, Seq: s.state.seq + 1, TurnID: newHistoryID(), Kind: "session_agents", Agents: catalog},
		{Version: 5, Seq: s.state.seq + 1, TurnID: newHistoryID(), Kind: "session_agents", Agents: catalog},
		{Version: 5, Seq: s.state.seq + 1, TurnID: newHistoryID(), Kind: "user_message", Text: "hi", Agents: catalog},
	} {
		state := s.state.clone()
		if err := state.apply(r); err == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
	// Reopening restores the frozen catalog instead of scanning again.
	writeDefinition(t, project, "second", "---\nname: second\ndescription: Two.\n---\nBody")
	s.Close()
	s, err = OpenSession(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(Options{ProjectDirectory: project}, s, []tools.Tool{tools.NewSkillView(nil), NewSpawnTool()}); err != nil || len(s.state.agentCatalog.Entries) != 1 {
		t.Fatalf("reopened: %+v, %v", s.state.agentCatalog, err)
	}
}

func TestSpawnTaskForms(t *testing.T) {
	for raw, want := range map[string]string{
		`{"tasks":["a"]}`:                                 `{"tasks":["a"]}`,
		`{"tasks":[{"task":"a"}]}`:                        `{"tasks":[{"agent":null,"task":"a"}]}`,
		`{"tasks":[{"agent":null,"task":"a"}]}`:           `{"tasks":[{"agent":null,"task":"a"}]}`,
		`{"tasks":[{"task":"b","agent":"reviewer"},"c"]}`: `{"tasks":[{"agent":"reviewer","task":"b"},"c"]}`,
	} {
		call, err := NewSpawnTool().Prepare(raw)
		if err != nil || string(call.Arguments) != want {
			t.Fatalf("%s prepared as %s, %v", raw, call.Arguments, err)
		}
	}
	for _, raw := range []string{
		`{"tasks":[{}]}`,
		`{"tasks":[{"agent":"reviewer"}]}`,
		`{"tasks":[{"agent":"","task":"a"}]}`,
		`{"tasks":[{"agent":"Bad Name","task":"a"}]}`,
		`{"tasks":[{"agent":1,"task":"a"}]}`,
		`{"tasks":[{"task":"a","task":"b"}]}`,
		`{"tasks":[{"task":"a","agent":null,"agent":null}]}`,
		`{"tasks":[{"task":"a","tools":["bash"]}]}`,
		`{"tasks":[{"task":" "}]}`,
		`{"tasks":[null]}`,
		`{"tasks":[["a"]]}`,
	} {
		if _, err := NewSpawnTool().Prepare(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
