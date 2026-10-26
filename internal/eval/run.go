package eval

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qshine/mino/internal/agent"
	"github.com/qshine/mino/internal/sandbox"
	"github.com/qshine/mino/internal/tools"
)

// Options configure every run. Live mode supplies the configured service;
// replay mode ignores BaseURL, APIKey, and Model and starts a FakeModel.
type Options struct {
	Live                   bool
	BaseURL, APIKey, Model string
	Instructions           string // Default when a case has none.
	Timeout                time.Duration
	Out                    string // Optional directory for report.json and run logs.
}

// RecordedCall is one tool request the model made, paired with what came back.
// Denied and unknown calls appear too: they have a result but never started.
type RecordedCall struct {
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Started   bool            `json:"started"`
	Result    tools.Result    `json:"result"`
}

// Transcript is what graders see. It is rebuilt from the run's session log, the
// same JSONL chat would write, so the eval grades the durable record.
type Transcript struct {
	FinalText string
	Requests  int
	Calls     []RecordedCall
	Usage     agent.Usage
	Workspace string
	Err       error
}

// RunCase executes one run of a case in fresh temporary directories and grades
// it before removing them. The returned error reports a harness failure; a
// failing turn is graded, not returned.
func RunCase(ctx context.Context, c Case, o Options, run int) (RunResult, error) {
	if !o.Live && len(c.Script) == 0 {
		return RunResult{}, fmt.Errorf("Case %s: %w", c.Name, errScriptRequired)
	}
	root, err := os.MkdirTemp("", "mino-eval-")
	if err != nil {
		return RunResult{}, fmt.Errorf("Cannot create the eval workspace: %w", err)
	}
	defer os.RemoveAll(root)
	workspace, home, sessionDir := filepath.Join(root, "workspace"), filepath.Join(root, "home"), filepath.Join(root, "session")
	for _, dir := range []string{workspace, filepath.Join(home, ".mino")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return RunResult{}, fmt.Errorf("Cannot create the eval workspace: %w", err)
		}
	}
	if err := seed(workspace, c.Workspace.Files); err != nil {
		return RunResult{}, err
	}
	baseURL, apiKey, model := o.BaseURL, o.APIKey, o.Model
	var fake *FakeModel
	if !o.Live {
		fake, err = StartFake(c.Script)
		if err != nil {
			return RunResult{}, err
		}
		defer fake.Close()
		baseURL, apiKey, model = fake.URL(), "eval", "scripted"
	}
	instructions := o.Instructions
	if c.Instructions != "" {
		instructions = c.Instructions
	}
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	transcript := Transcript{}
	recorder := &interaction{approve: c.Approve}
	transcript.Err = func() error {
		bash, err := tools.NewBash(workspace, home)
		if err != nil {
			return err
		}
		transcript.Workspace = bash.Directory()
		// Cases run as chat does: confined when the system can confine.
		if sandbox.Available() {
			confined, err := sandbox.New(bash.Directory(), home)
			if err != nil {
				return err
			}
			bash.SetSandbox(confined)
		}
		files, err := tools.NewFiles(bash.Directory(), filepath.Join(home, ".mino"))
		if err != nil {
			return err
		}
		available := []tools.Tool{bash, tools.NewSkillView(nil), files.Read(), files.Write(), files.Edit()}
		session, err := agent.OpenSession(sessionDir, available)
		if err != nil {
			return err
		}
		defer session.Close()
		a, err := agent.New(agent.Options{BaseURL: baseURL, APIKey: apiKey, Model: model, Instructions: instructions, ProjectDirectory: bash.Directory()}, session, available)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return a.Handle(ctx, c.Prompt, recorder)
	}()
	if transcript.Workspace == "" {
		transcript.Workspace = workspace
	}
	transcript.FinalText, transcript.Requests, transcript.Usage = recorder.final, recorder.requests, recorder.usage
	logPath, err := sessionLog(sessionDir)
	if err != nil && transcript.Err == nil {
		transcript.Err = err
	}
	calls, err := readCalls(logPath)
	if err != nil && transcript.Err == nil {
		transcript.Err = err
	}
	transcript.Calls = calls
	// A script violation explains a failed turn better than the generic model error.
	if fake != nil && fake.Err() != nil {
		transcript.Err = fake.Err()
	}
	if o.Out != "" {
		if err := copyLog(logPath, filepath.Join(o.Out, c.Name, fmt.Sprintf("run-%d.jsonl", run))); err != nil {
			return RunResult{}, err
		}
	}
	outcomes := Grade(c, transcript)
	return RunResult{Run: run, Passed: passed(outcomes), Requests: transcript.Requests, ToolCalls: len(transcript.Calls), Usage: transcript.Usage, Outcomes: outcomes}, nil
}

func seed(workspace string, files map[string]string) error {
	for path, content := range files {
		if !validRelativePath(path) {
			return fmt.Errorf("Workspace file %q must stay inside the workspace", path)
		}
		full := filepath.Join(workspace, path)
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			return fmt.Errorf("Cannot seed %s: %w", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0600); err != nil {
			return fmt.Errorf("Cannot seed %s: %w", path, err)
		}
	}
	return nil
}

// interaction answers the Agent without a terminal. Approval follows the case;
// recovery and clear confirmations never apply to a fresh session and are denied.
type interaction struct {
	approve  Approval
	current  strings.Builder
	final    string
	requests int
	usage    agent.Usage
}

func (i *interaction) Emit(e agent.Event) error {
	switch e.Kind {
	case "response_started":
		i.requests++
		i.current.Reset()
	case "text":
		i.current.WriteString(e.Text)
	case "response_completed":
		i.final = i.current.String()
		i.usage.InputTokens += e.Usage.InputTokens
		i.usage.OutputTokens += e.Usage.OutputTokens
		i.usage.TotalTokens += e.Usage.TotalTokens
	}
	return nil
}

func (i *interaction) Confirm(_ context.Context, c agent.Confirmation) (bool, error) {
	if c.Kind != "tool" {
		return false, nil
	}
	return i.approve.Allows(c.ToolName), nil
}

// sessionLog locates the one conversation OpenSession wrote under sessions/.
func sessionLog(dir string) (string, error) {
	logs, err := filepath.Glob(filepath.Join(dir, "sessions", "*.jsonl"))
	if err != nil || len(logs) != 1 {
		return "", fmt.Errorf("Cannot locate the run log in %s", dir)
	}
	return logs[0], nil
}

// readCalls rebuilds the tool requests from the session log: every function_call
// the model produced, whether it started, and the result that was saved.
func readCalls(path string) ([]RecordedCall, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read the run log: %w", err)
	}
	defer file.Close()
	var calls []RecordedCall
	index := map[string]int{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), 16<<20)
	for scanner.Scan() {
		var record struct {
			Kind   string            `json:"kind"`
			Output []json.RawMessage `json:"output"`
			CallID string            `json:"call_id"`
			Result *tools.Result     `json:"result"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("Cannot read the run log: %w", err)
		}
		switch record.Kind {
		case "model_response":
			for _, raw := range record.Output {
				var item struct {
					Type      string `json:"type"`
					CallID    string `json:"call_id"`
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}
				if json.Unmarshal(raw, &item) == nil && item.Type == "function_call" {
					index[item.CallID] = len(calls)
					calls = append(calls, RecordedCall{CallID: item.CallID, Name: item.Name, Arguments: json.RawMessage(item.Arguments)})
				}
			}
		case "tool_start":
			if i, ok := index[record.CallID]; ok {
				calls[i].Started = true
			}
		case "tool_result":
			if i, ok := index[record.CallID]; ok && record.Result != nil {
				calls[i].Result = *record.Result
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Cannot read the run log: %w", err)
	}
	return calls, nil
}

func copyLog(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return fmt.Errorf("Cannot write eval output: %w", err)
	}
	in, err := os.Open(source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("Cannot write eval output: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Cannot write eval output: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("Cannot write eval output: %w", err)
	}
	return out.Close()
}
