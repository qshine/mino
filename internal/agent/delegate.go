package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/qshine/mino/internal/tools"
)

// Subagent limits. One spawn_agent call is one tool call in the parent's turn:
// the children run inside it, so the parent's history keeps one start and one
// result per call and its format does not change.
const (
	maxSubagentTasks      = 10
	maxSubagentsPerTurn   = 16
	maxSubagentTaskBytes  = 8 << 10
	maxSubagentTasksBytes = 32 << 10
	// The children's final texts share this budget, below the 64 KiB tool output
	// limit so the headers always fit and the parent's context stays small.
	subagentResultBudget = 48 << 10
	maxFailureBytes      = 256

	DefaultSubagentConcurrency = 3
	MaxSubagentConcurrency     = 10
)

const spawnAgentName = "spawn_agent"

// SpawnTool delegates tasks to child Agents. Prepare is pure, so recorded calls
// validate during replay; the Agent runs the call itself (see delegate), because
// a child's storage failure must stop chat and Tool.Execute cannot report one.
type SpawnTool struct{}

func NewSpawnTool() *SpawnTool { return &SpawnTool{} }

func (*SpawnTool) Definition() tools.Definition {
	return tools.Definition{Name: spawnAgentName, Description: "Delegate 1 to 10 tasks to subagents that run in parallel. Each subagent is a separate Agent with its own context: it cannot see this conversation, so write every task as a complete, self-contained instruction. Subagents use the same tools and permission mode as you, except that they cannot spawn subagents or save memories; their tool calls that need approval ask the user one at a time. Only each subagent's final text comes back, as one result. Use this for broad searches or investigations whose intermediate output would fill your context. Do not give parallel tasks work that modifies the same files.", Parameters: map[string]any{
		"type": "object", "properties": map[string]any{
			"tasks": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "1 to 10 self-contained task instructions, each at most 8 KiB."},
		}, "required": []string{"tasks"}, "additionalProperties": false,
	}, Strict: true}
}

type spawnArgs struct {
	Tasks []string `json:"tasks"`
}

func parseSpawnArguments(raw string) (spawnArgs, error) {
	invalid := errors.New("spawn_agent requires a tasks array of 1 to 10 non-empty UTF-8 strings, each at most 8 KiB and 32 KiB in total")
	if !utf8.ValidString(raw) {
		return spawnArgs{}, invalid
	}
	d := json.NewDecoder(strings.NewReader(raw))
	if first, err := d.Token(); err != nil || first != json.Delim('{') {
		return spawnArgs{}, invalid
	}
	var args spawnArgs
	seen := false
	for d.More() {
		key, err := d.Token()
		if err != nil || key != "tasks" || seen {
			return spawnArgs{}, invalid
		}
		seen = true
		if err := d.Decode(&args.Tasks); err != nil {
			return spawnArgs{}, invalid
		}
	}
	if last, err := d.Token(); err != nil || last != json.Delim('}') {
		return spawnArgs{}, invalid
	}
	if _, err := d.Token(); err != io.EOF {
		return spawnArgs{}, invalid
	}
	if len(args.Tasks) == 0 || len(args.Tasks) > maxSubagentTasks {
		return spawnArgs{}, invalid
	}
	total := 0
	for _, task := range args.Tasks {
		total += len(task)
		if !validHistoryText(task) || len(task) > maxSubagentTaskBytes {
			return spawnArgs{}, invalid
		}
	}
	if total > maxSubagentTasksBytes {
		return spawnArgs{}, invalid
	}
	return args, nil
}

func (*SpawnTool) Prepare(raw string) (tools.Call, error) {
	args, err := parseSpawnArguments(raw)
	if err != nil {
		return tools.Call{}, err
	}
	encoded, _ := json.Marshal(args)
	call := tools.Call{Arguments: encoded, Fields: []tools.Field{{Label: "Subagents", Value: strconv.Itoa(len(args.Tasks))}},
		Warning: "Each subagent works in its own context with this session's permission mode; its tool calls that need approval will ask one at a time."}
	for i, task := range args.Tasks {
		call.Fields = append(call.Fields, tools.Field{Label: fmt.Sprintf("Task %d", i+1), Value: task})
	}
	return call, nil
}

func (*SpawnTool) Execute(context.Context, tools.Call) tools.Result {
	return tools.Result{Status: "failed", Output: "spawn_agent runs only inside an Agent turn."}
}

// subagentInstructions follows the startup SOUL in every child's instructions.
func subagentInstructions(budget int) string {
	return fmt.Sprintf("\n\nYou are a subagent. Another Mino agent delegated the task in the user message to you. You cannot see its conversation and cannot ask the user questions; work only on this task. Other subagents may be working in the same project at the same time. Your final message is returned to that agent as your entire result: state what you found or did, with file paths where they help, in at most %d bytes. Longer text is cut off.", budget)
}

type subagentResult struct {
	session, status, failure, text string
	requests, calls                int
}

// delegate runs one batch: it creates every child session first, then runs the
// children through a pool of at most the configured concurrency. One child's
// failure does not stop the others; a storage failure stops them all and chat.
func (a *Agent) delegate(ctx context.Context, call tools.Call, interaction Interaction) (tools.Result, error) {
	args, err := parseSpawnArguments(string(call.Arguments))
	if err != nil {
		return tools.Result{Status: "failed", Output: err.Error()}, nil
	}
	a.spawned += len(args.Tasks)
	budget := subagentResultBudget / len(args.Tasks)
	dir := strings.TrimSuffix(a.session.file.Name(), ".jsonl") + ".agents"
	if err := secureSessionDirectory(dir); err != nil {
		return tools.Result{}, &StorageError{fmt.Errorf("Failed to create the subagent session directory: %w", err)}
	}
	if err := syncHistoryDirectory(filepath.Dir(dir)); err != nil {
		return tools.Result{}, &StorageError{err}
	}
	var available []tools.Tool
	for _, tool := range a.available {
		if name := tool.Definition().Name; name != spawnAgentName && name != "memory_save" {
			available = append(available, tool)
		}
	}
	registry, err := toolRegistry(available)
	if err != nil {
		return tools.Result{}, err
	}
	options := a.options
	options.Instructions = a.instructions + subagentInstructions(budget)
	options.ProjectDirectory, options.Memory = "", nil
	children := make([]*Agent, len(args.Tasks))
	results := make([]subagentResult, len(args.Tasks))
	defer func() {
		for _, child := range children {
			if child != nil {
				child.session.Close()
			}
		}
	}()
	for i := range args.Tasks {
		id := newHistoryID()
		session, err := openSessionFile(filepath.Join(dir, id+".jsonl"), "", registry, true)
		if err != nil {
			return tools.Result{}, &StorageError{err}
		}
		// Children read the parent's frozen skill catalog, never a fresh scan.
		if catalog := a.session.state.skillCatalog; catalog != nil {
			if err := session.append(historyRecord{TurnID: newHistoryID(), Kind: "session_skills", Skills: catalog}); err != nil {
				session.Close()
				return tools.Result{}, &StorageError{err}
			}
		}
		child, err := New(options, session, available)
		if err != nil {
			session.Close()
			return tools.Result{}, err
		}
		// The memory index this turn rendered: children never rescan the directory.
		child.memory = a.memory
		child.httpClient.Timeout = a.httpClient.Timeout
		children[i], results[i].session = child, id
	}

	limit := a.options.Subagents
	if limit == 0 {
		limit = DefaultSubagentConcurrency
	}
	batch, cancel := context.WithCancel(ctx)
	defer cancel()
	gate := make(chan struct{}, 1)
	slots := make(chan struct{}, limit)
	var wait sync.WaitGroup
	var fatal error
	var failed sync.Once
	for i, child := range children {
		select {
		case slots <- struct{}{}:
		case <-batch.Done():
			results[i].status = "cancelled"
			continue
		}
		wait.Add(1)
		go func(i int, child *Agent) {
			defer wait.Done()
			defer func() { <-slots }()
			view := &childInteraction{parent: interaction, gate: gate, label: fmt.Sprintf("subagent %d/%d", i+1, len(children)), task: args.Tasks[i]}
			err := child.Handle(batch, args.Tasks[i], view)
			result := &results[i]
			result.text, result.requests, result.calls = view.text.String(), view.requests, view.calls
			var storage *StorageError
			switch {
			case err == nil:
				result.status = "completed"
			case errors.As(err, &storage):
				failed.Do(func() { fatal = err })
				cancel()
				result.status = "cancelled"
			case batch.Err() != nil:
				result.status = "cancelled"
			default:
				result.status, result.failure = "failed", err.Error()
			}
		}(i, child)
	}
	wait.Wait()
	if fatal != nil {
		return tools.Result{}, fatal
	}
	return formatSubagentResults(results, budget, ctx.Err() != nil), nil
}

// formatSubagentResults keeps the tasks in their requested order. Each final text
// gets an equal share of the budget, so ten tasks cannot fill the parent's context.
func formatSubagentResults(results []subagentResult, budget int, cancelled bool) tools.Result {
	counts := map[string]int{}
	for _, r := range results {
		counts[r.status]++
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Subagent results: %d tasks, %d completed, %d failed, %d cancelled.\n", len(results), counts["completed"], counts["failed"], counts["cancelled"])
	for i, r := range results {
		status := r.status
		if r.failure != "" {
			status += ": " + truncateText(r.failure, maxFailureBytes)
		}
		fmt.Fprintf(&b, "\n[%d/%d] %s · session %s · %s, %s\n", i+1, len(results), status, r.session, count(r.requests, "request"), count(r.calls, "tool call"))
		text := strings.TrimSpace(r.text)
		if text == "" {
			text = "(no text)"
		}
		if len(text) > budget {
			omitted := len(text)
			text = truncateText(text, budget)
			text += fmt.Sprintf("\n[truncated %d bytes]", omitted-len(text))
		}
		b.WriteString(text + "\n")
	}
	status := "completed"
	switch {
	case cancelled:
		status = "cancelled"
	case counts["completed"] != len(results):
		status = "failed"
	}
	return tools.Result{Status: status, Output: b.String()}
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// truncateText cuts at most limit bytes without splitting a UTF-8 sequence.
func truncateText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}

// childInteraction is one child's view of the parent's gateway. All children of
// a batch share gate, so the terminal shows one question at a time and output
// pauses while it is open. The child's streamed text is kept, not shown.
type childInteraction struct {
	parent      Interaction
	gate        chan struct{}
	label, task string
	text        strings.Builder
	requests    int
	calls       int
}

func (c *childInteraction) Emit(e Event) error {
	switch e.Kind {
	case "response_started":
		c.requests++
		c.text.Reset()
		return nil
	case "text":
		c.text.WriteString(e.Text)
		return nil
	case "response_completed":
		return c.forward(Event{Kind: "subagent_usage", Usage: e.Usage})
	case "tool_result":
		c.calls++
		return c.forward(Event{Kind: "info", Text: fmt.Sprintf("[%s] %s: %s\n", c.label, e.ToolName, e.Result.Status)})
	case "info", "notice":
		lines := strings.SplitAfter(e.Text, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				lines[i] = "[" + c.label + "] " + line
			}
		}
		return c.forward(Event{Kind: e.Kind, Text: strings.Join(lines, "")})
	}
	return c.forward(e)
}

func (c *childInteraction) forward(e Event) error {
	c.gate <- struct{}{}
	defer func() { <-c.gate }()
	return c.parent.Emit(e)
}

func (c *childInteraction) Confirm(ctx context.Context, r Confirmation) (bool, error) {
	select {
	case c.gate <- struct{}{}:
	case <-ctx.Done():
		return false, ctx.Err()
	}
	defer func() { <-c.gate }()
	summary, _, _ := strings.Cut(strings.TrimSpace(c.task), "\n")
	if runes := []rune(summary); len(runes) > 60 {
		summary = string(runes[:60]) + "…"
	}
	r.Fields = append([]tools.Field{{Label: "Subagent", Value: strings.TrimPrefix(c.label, "subagent ") + ": " + summary}}, r.Fields...)
	return c.parent.Confirm(ctx, r)
}
