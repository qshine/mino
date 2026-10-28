package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/qshine/mino/internal/memory"
	"github.com/qshine/mino/internal/policy"
	"github.com/qshine/mino/internal/tools"
)

// subagentModel answers the parent and each child from its own script. A child
// is recognised by its instructions and keyed by its task, the first user message.
type subagentModel struct {
	t        *testing.T
	mu       sync.Mutex
	scripts  map[string][]string
	requests map[string][]subagentRequest
	// before runs outside the lock with the agent's key and request number (from 1).
	before func(r *http.Request, key string, n int)
}

type subagentRequest struct {
	Instructions string
	Input        []map[string]any
	Tools        []string
}

func (m *subagentModel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body struct {
		Instructions string
		Input        []map[string]any
		Tools        []struct{ Name string }
	}
	json.Unmarshal(raw, &body)
	request := subagentRequest{Instructions: body.Instructions, Input: body.Input}
	for _, tool := range body.Tools {
		request.Tools = append(request.Tools, tool.Name)
	}
	key := "parent"
	if strings.Contains(body.Instructions, "You are a subagent") {
		for _, item := range body.Input {
			if text, ok := item["content"].(string); ok && item["role"] == "user" {
				key = text
				break
			}
		}
	}
	m.mu.Lock()
	m.requests[key] = append(m.requests[key], request)
	n := len(m.requests[key])
	step := ""
	if n <= len(m.scripts[key]) {
		step = m.scripts[key][n-1]
	}
	m.mu.Unlock()
	if m.before != nil {
		m.before(r, key, n)
	}
	if step != "" {
		streamEvent(w, step)
		return
	}
	text := "done: " + key
	if key == "parent" {
		text = "parent done"
	}
	streamEvent(w, `{"type":"response.output_text.delta","delta":`+quoteJSON(text)+`}`)
	streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
}

func (m *subagentModel) get(key string) []subagentRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.requests[key])
}

func quoteJSON(text string) string {
	raw, _ := json.Marshal(text)
	return string(raw)
}

func spawnCall(id string, tasks ...string) string {
	raw, _ := json.Marshal(map[string][]string{"tasks": tasks})
	item, _ := json.Marshal(map[string]string{"type": "function_call", "name": "spawn_agent", "call_id": id, "arguments": string(raw)})
	return `{"type":"response.completed","response":{"status":"completed","output":[` + string(item) + `]}}`
}

// probeTool stands in for any tool that needs approval, without a sandbox.
type probeTool struct{ readOnly bool }

func (probeTool) Definition() tools.Definition {
	return tools.Definition{Name: "probe", Description: "Probe.", Parameters: map[string]any{
		"type": "object", "properties": map[string]any{"value": map[string]any{"type": "string"}}, "required": []string{"value"}, "additionalProperties": false,
	}, Strict: true}
}
func (p probeTool) Prepare(raw string) (tools.Call, error) {
	var args struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil || args.Value == "" {
		return tools.Call{}, errors.New("probe requires a value")
	}
	encoded, _ := json.Marshal(args)
	return tools.Call{Arguments: encoded, ReadOnly: p.readOnly, Fields: []tools.Field{{Label: "Value", Value: args.Value}}}, nil
}
func (probeTool) Execute(_ context.Context, c tools.Call) tools.Result {
	return tools.Result{Status: "completed", Output: "probed " + string(c.Arguments)}
}

// barrier releases the first request of every child only once n children have
// sent one, which can happen only if they run at the same time.
func barrier(t *testing.T, n int) func(*http.Request, string, int) {
	var mu sync.Mutex
	arrived, all := 0, make(chan struct{})
	return func(_ *http.Request, key string, request int) {
		if key == "parent" || request != 1 {
			return
		}
		mu.Lock()
		arrived++
		if arrived == n {
			close(all)
		}
		mu.Unlock()
		select {
		case <-all:
		case <-time.After(5 * time.Second):
			t.Errorf("only %d of %d subagents were running at once", arrived, n)
		}
	}
}

type subagentRun struct {
	agent    *Agent
	session  *Session
	model    *subagentModel
	server   *httptest.Server
	dir      string
	mu       sync.Mutex
	asked    []Confirmation
	infos    []string
	texts    []string
	approve  func(Confirmation) bool
	readOnly bool
}

func newSubagentRun(t *testing.T, mode policy.Mode, concurrency int, scripts map[string][]string, options Options, available ...tools.Tool) *subagentRun {
	t.Helper()
	run := &subagentRun{model: &subagentModel{t: t, scripts: scripts, requests: map[string][]subagentRequest{}}, dir: t.TempDir()}
	run.server = httptest.NewServer(run.model)
	t.Cleanup(run.server.Close)
	available = append(available, NewSpawnTool())
	session, err := OpenSession(run.dir, available)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	rules := policy.New()
	rules.SetMode(mode)
	options.BaseURL, options.APIKey, options.Model, options.Policy, options.Subagents = run.server.URL, "test", "test", rules, concurrency
	if options.Instructions == "" {
		options.Instructions = "SOUL-TEXT"
	}
	run.agent, err = New(options, session, available)
	if err != nil {
		t.Fatal(err)
	}
	run.session = session
	return run
}

func (r *subagentRun) interaction() Interaction {
	return testInteraction{emit: func(e Event) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		switch e.Kind {
		case "info", "notice":
			r.infos = append(r.infos, e.Text)
		case "text":
			r.texts = append(r.texts, e.Text)
		}
		return nil
	}, confirm: func(_ context.Context, c Confirmation) (bool, error) {
		r.mu.Lock()
		r.asked = append(r.asked, c)
		r.mu.Unlock()
		if r.approve != nil {
			return r.approve(c), nil
		}
		return true, nil
	}}
}

func (r *subagentRun) logPath(t *testing.T) string {
	t.Helper()
	logs, _ := filepath.Glob(filepath.Join(r.dir, "sessions", "*.jsonl"))
	if len(logs) != 1 {
		t.Fatalf("logs=%v", logs)
	}
	return logs[0]
}

func (r *subagentRun) childLogs(t *testing.T) []string {
	t.Helper()
	logs, _ := filepath.Glob(filepath.Join(strings.TrimSuffix(r.logPath(t), ".jsonl")+".agents", "*.jsonl"))
	return logs
}

func toolOutput(t *testing.T, input []map[string]any, id string) string {
	t.Helper()
	for _, item := range input {
		if item["type"] == "function_call_output" && item["call_id"] == id {
			var result tools.Result
			if err := json.Unmarshal([]byte(item["output"].(string)), &result); err != nil {
				t.Fatal(err)
			}
			return result.Output
		}
	}
	t.Fatalf("no output for %s in %v", id, input)
	return ""
}

func TestSubagentsRunInParallelAndReturnOneResult(t *testing.T) {
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(project, ".agents", "skills", "lookup")
	os.MkdirAll(skill, 0700)
	os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: lookup\ndescription: Find things\n---\nBody"), 0600)
	store := openTestStore(t, memory.Memory{Name: "uses-pnpm", Type: "project", Summary: "Install with pnpm"})
	tasks := []string{"task A", "task B", "task C"}
	run := newSubagentRun(t, policy.Manual, 3, map[string][]string{"parent": {spawnCall("s1", tasks...)}},
		Options{ProjectDirectory: project, Memory: store}, tools.NewSkillView(nil), tools.NewMemoryView(store), tools.NewMemorySave(store))
	run.model.before = barrier(t, 3)
	if err := run.agent.Handle(context.Background(), "Investigate three things", run.interaction()); err != nil {
		t.Fatal(err)
	}
	if len(run.asked) != 1 || run.asked[0].ToolName != "spawn_agent" || run.asked[0].Fields[0].Value != "3" {
		t.Fatalf("asked=%+v", run.asked)
	}
	parent := run.model.get("parent")
	if len(parent) != 2 {
		t.Fatalf("parent requests=%d", len(parent))
	}
	output := toolOutput(t, parent[1].Input, "s1")
	a, b, c := strings.Index(output, "[1/3] completed"), strings.Index(output, "[2/3] completed"), strings.Index(output, "[3/3] completed")
	if !strings.HasPrefix(output, "Subagent results: 3 tasks, 3 completed, 0 failed, 0 cancelled.") || a < 0 || b < a || c < b ||
		!strings.Contains(output[a:b], "done: task A") || !strings.Contains(output[c:], "done: task C") {
		t.Fatalf("output:\n%s", output)
	}
	for _, task := range tasks {
		child := run.model.get(task)
		if len(child) != 1 {
			t.Fatalf("%s requests=%d", task, len(child))
		}
		request := child[0]
		if !strings.HasPrefix(request.Instructions, "SOUL-TEXT") || !strings.Contains(request.Instructions, "at most 16384 bytes") || !strings.Contains(request.Instructions, `"lookup"`) {
			t.Fatalf("instructions: %s", request.Instructions)
		}
		if slices.Contains(request.Tools, "spawn_agent") || slices.Contains(request.Tools, "memory_save") || !slices.Contains(request.Tools, "memory_view") || !slices.Contains(request.Tools, "skill_view") {
			t.Fatalf("child tools=%v", request.Tools)
		}
		if !strings.Contains(frontText(t, request.Input), "Install with pnpm") {
			t.Fatalf("child did not get the memory index: %v", request.Input)
		}
	}
	for _, text := range run.texts {
		if strings.Contains(text, "done: task") {
			t.Fatalf("a subagent's text was streamed to the terminal: %q", text)
		}
	}
	children := run.childLogs(t)
	if len(children) != 3 {
		t.Fatalf("child logs=%v", children)
	}
	for _, path := range children {
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), `"kind":"session_skills"`) || !strings.Contains(string(data), `"kind":"turn_end"`) {
			t.Fatalf("child log:\n%s", data)
		}
	}
	data, _ := os.ReadFile(run.logPath(t))
	if strings.Count(string(data), `"kind":"tool_start"`) != 1 || strings.Count(string(data), `"kind":"tool_result"`) != 1 {
		t.Fatalf("parent log:\n%s", data)
	}
}

// Children share the parent's policy and terminal: questions come one at a time,
// labelled with the subagent, and a mode that skips questions skips theirs too.
func TestSubagentConfirmationsAskOneAtATime(t *testing.T) {
	for _, mode := range []policy.Mode{policy.Manual, policy.Bypass} {
		t.Run(mode.String(), func(t *testing.T) {
			scripts := map[string][]string{
				"parent":    {spawnCall("s1", "probe one", "probe two")},
				"probe one": {functionCall("probe", "p1", map[string]string{"value": "one"})},
				"probe two": {functionCall("probe", "p2", map[string]string{"value": "two"})},
			}
			run := newSubagentRun(t, mode, 2, scripts, Options{}, probeTool{})
			run.model.before = barrier(t, 2)
			var active, overlaps int
			var mu sync.Mutex
			run.approve = func(c Confirmation) bool {
				mu.Lock()
				active++
				overlaps += active - 1
				mu.Unlock()
				time.Sleep(30 * time.Millisecond)
				mu.Lock()
				active--
				mu.Unlock()
				return c.ToolName == "spawn_agent" || c.Fields[1].Value == "one"
			}
			if err := run.agent.Handle(context.Background(), "Probe twice", run.interaction()); err != nil {
				t.Fatal(err)
			}
			if overlaps != 0 {
				t.Fatalf("%d confirmations overlapped", overlaps)
			}
			infos := strings.Join(run.infos, "")
			if mode == policy.Bypass {
				if len(run.asked) != 0 || !strings.Contains(infos, "[subagent 1/2] [Tool: probe, allowed without confirmation in bypass mode]") {
					t.Fatalf("asked=%+v infos=%s", run.asked, infos)
				}
				return
			}
			var labels []string
			for _, c := range run.asked[1:] {
				if c.Fields[0].Label != "Subagent" {
					t.Fatalf("confirmation without a subagent label: %+v", c)
				}
				labels = append(labels, c.Fields[0].Value)
			}
			slices.Sort(labels)
			if len(run.asked) != 3 || strings.Join(labels, ",") != "1/2: probe one,2/2: probe two" {
				t.Fatalf("asked=%+v", run.asked)
			}
			if !strings.Contains(infos, "[subagent 1/2] probe: completed") || !strings.Contains(infos, "[subagent 2/2] probe: denied") {
				t.Fatalf("infos=%s", infos)
			}
		})
	}
}

func TestSubagentLimitPerTurnIsDeniedBeforeAsking(t *testing.T) {
	ten := make([]string, 10)
	for i := range ten {
		ten[i] = fmt.Sprintf("task %d", i+1)
	}
	run := newSubagentRun(t, policy.Manual, 0, map[string][]string{"parent": {spawnCall("s1", ten...), spawnCall("s2", ten[:7]...)}}, Options{})
	if err := run.agent.Handle(context.Background(), "Many tasks", run.interaction()); err != nil {
		t.Fatal(err)
	}
	parent := run.model.get("parent")
	if len(run.asked) != 1 || len(parent) != 3 || len(run.childLogs(t)) != 10 {
		t.Fatalf("asked=%d requests=%d children=%d", len(run.asked), len(parent), len(run.childLogs(t)))
	}
	if output := toolOutput(t, parent[2].Input, "s2"); !strings.Contains(output, "already started 10 of at most 16 subagents") {
		t.Fatalf("output=%s", output)
	}
}

func TestSubagentCancellationStopsTheBatch(t *testing.T) {
	run := newSubagentRun(t, policy.Bypass, 2, map[string][]string{"parent": {spawnCall("s1", "slow one", "slow two")}}, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	waiting := 0
	run.model.before = func(r *http.Request, key string, _ int) {
		if key == "parent" {
			return
		}
		mu.Lock()
		waiting++
		if waiting == 2 {
			cancel()
		}
		mu.Unlock()
		<-r.Context().Done()
	}
	err := run.agent.Handle(ctx, "Slow work", run.interaction())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	data, _ := os.ReadFile(run.logPath(t))
	if !strings.Contains(string(data), `"status":"cancelled","output":"Subagent results: 2 tasks, 0 completed, 0 failed, 2 cancelled.`) {
		t.Fatalf("parent log:\n%s", data)
	}
}

func TestSubagentStorageFailureStopsChat(t *testing.T) {
	run := newSubagentRun(t, policy.Bypass, 0, map[string][]string{"parent": {spawnCall("s1", "task")}}, Options{})
	// A file where the subagent directory belongs makes child storage unusable.
	if err := os.WriteFile(strings.TrimSuffix(run.logPath(t), ".jsonl")+".agents", nil, 0600); err != nil {
		t.Fatal(err)
	}
	var storage *StorageError
	if err := run.agent.Handle(context.Background(), "Delegate", run.interaction()); !errors.As(err, &storage) {
		t.Fatalf("err=%v", err)
	}
}

type failingResultWriter struct{ historyWriter }

func (w failingResultWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), `"kind":"tool_result"`) {
		return 0, errors.New("disk full")
	}
	return w.historyWriter.Write(p)
}

// A crash during a batch leaves the parent's spawn_agent started without a
// result. Restarting closes it as unknown, as for any tool, and runs nothing.
func TestInterruptedSubagentBatchIsUnknownAfterRestart(t *testing.T) {
	run := newSubagentRun(t, policy.Bypass, 0, map[string][]string{"parent": {spawnCall("s1", "task")}}, Options{})
	run.session.writer = failingResultWriter{run.session.writer}
	var storage *StorageError
	if err := run.agent.Handle(context.Background(), "Delegate", run.interaction()); !errors.As(err, &storage) {
		t.Fatalf("err=%v", err)
	}
	run.session.Close()
	available := []tools.Tool{NewSpawnTool()}
	reopened, err := OpenSession(run.dir, available)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if len(reopened.state.uncertain) != 1 || !strings.Contains(reopened.notice, "interrupted tool turn") || len(run.model.get("task")) != 1 {
		t.Fatalf("uncertain=%v notice=%q", reopened.state.uncertain, reopened.notice)
	}
	reopened.Close()
	if _, err := OpenSession(run.dir, nil); err == nil || !strings.Contains(err.Error(), "unavailable tool") {
		t.Fatalf("history validated without spawn_agent: %v", err)
	}
}

func TestClearRemovesSubagentLogsAndListingIgnoresThem(t *testing.T) {
	home := t.TempDir()
	available := []tools.Tool{NewSpawnTool()}
	store, err := OpenSessions(home, available)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := NewSessionManager(Options{BaseURL: "http://127.0.0.1:1", APIKey: "test", Model: "test"}, store, available)
	if err != nil {
		t.Fatal(err)
	}
	children := filepath.Join(home, "sessions", store.id+".agents")
	os.Mkdir(children, 0700)
	os.WriteFile(filepath.Join(children, newHistoryID()+".jsonl"), nil, 0600)
	var infos []string
	interaction := testInteraction{emit: func(e Event) error { infos = append(infos, e.Text); return nil }, confirm: func(context.Context, Confirmation) (bool, error) { return true, nil }}
	if err := manager.Handle(context.Background(), "/sessions", interaction); err != nil {
		t.Fatal(err)
	}
	if strings.Count(infos[0], "\n") != 2 {
		t.Fatalf("listing=%q", infos[0])
	}
	if err := manager.Handle(context.Background(), "/clear", interaction); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(children); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("subagent logs survived /clear: %v", err)
	}
}

func TestSpawnArgumentsAreBounded(t *testing.T) {
	tasks := func(n, size int) string {
		list := make([]string, n)
		for i := range list {
			list[i] = strings.Repeat("x", size)
		}
		raw, _ := json.Marshal(map[string][]string{"tasks": list})
		return string(raw)
	}
	valid := []string{tasks(1, 1), tasks(10, 3000), tasks(4, 8<<10)}
	invalid := []string{`{}`, `{"tasks":[]}`, `{"tasks":["  "]}`, tasks(11, 1), tasks(1, 8<<10+1), tasks(5, 8<<10), `{"tasks":["a"],"tasks":["b"]}`, `{"tasks":["a"],"agent":"x"}`, `{"tasks":["a"]} {}`, `{"tasks":"a"}`}
	for _, raw := range valid {
		if _, err := NewSpawnTool().Prepare(raw); err != nil {
			t.Fatalf("rejected %.60s: %v", raw, err)
		}
	}
	for _, raw := range invalid {
		if _, err := NewSpawnTool().Prepare(raw); err == nil {
			t.Fatalf("accepted %.60s", raw)
		}
	}
}

func TestSubagentResultsShareTheBudget(t *testing.T) {
	long := strings.Repeat("界", 10000) // 30,000 bytes of three-byte characters
	results := make([]subagentResult, 10)
	for i := range results {
		results[i] = subagentResult{session: newHistoryID(), status: "completed", text: long}
	}
	results[3].status, results[3].failure = "failed", strings.Repeat("e", 1000)
	result := formatSubagentResults(results, subagentResultBudget/10, false)
	if result.Status != "failed" || !utf8.ValidString(result.Output) || len(result.Output) > tools.OutputLimit || strings.Count(result.Output, "[truncated ") != 10 {
		t.Fatalf("status=%s bytes=%d", result.Status, len(result.Output))
	}
	if err := validateToolResult(result, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Output, "[4/10] failed: "+strings.Repeat("e", maxFailureBytes)+" ·") {
		t.Fatal("failure reason was not bounded")
	}
	if got := formatSubagentResults([]subagentResult{{status: "completed"}}, 100, false); got.Status != "completed" || !strings.Contains(got.Output, "(no text)") {
		t.Fatalf("got=%+v", got)
	}
}
