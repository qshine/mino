package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openai/openai-go/v3/responses"
	"github.com/qshine/mino/internal/tools"
)

// Fixtures exercise the public Agent entry, using HTTP mocks and real temporary
// sessions. No fixture implements model parsing or the Agent loop itself.
type testConfig struct{ BaseURL, APIKey, Model string }
type modelFixture struct {
	t            *testing.T
	config       testConfig
	instructions string
	httpClient   *http.Client
}

func newModelFixture(t *testing.T, config testConfig, instructions string) *modelFixture {
	return &modelFixture{t, config, instructions, &http.Client{Timeout: 2 * time.Minute}}
}
func (m *modelFixture) agent(session *Session, available []tools.Tool) *Agent {
	a, err := New(Options{m.config.BaseURL, m.config.APIKey, m.config.Model, m.instructions}, session, available)
	if err != nil {
		m.t.Fatal(err)
	}
	a.httpClient.Timeout = m.httpClient.Timeout
	return a
}
func (m *modelFixture) respond(ctx context.Context, input responses.ResponseInputParam, emit func(string) error) (modelReply, error) {
	directory := m.t.TempDir()
	bash, err := tools.NewBash(directory, directory)
	if err != nil {
		m.t.Fatal(err)
	}
	available := []tools.Tool{bash}
	session, err := OpenSession(filepath.Join(directory, "session"), available)
	if err != nil {
		m.t.Fatal(err)
	}
	defer session.Close()
	// These transport cases provide exactly one user message; history cases use turnFixture.
	raw, _ := json.Marshal(input)
	var messages []struct{ Content string }
	if err := json.Unmarshal(raw, &messages); err != nil || len(messages) != 1 {
		m.t.Fatalf("unexpected test input: %s", raw)
	}
	a := m.agent(session, available)
	return modelReply{}, a.Handle(ctx, messages[0].Content, testInteraction{emit: func(e Event) error {
		if e.Kind == "text" {
			return emit(e.Text)
		}
		return nil
	}})
}

type testInteraction struct {
	emit    func(Event) error
	confirm func(context.Context, Confirmation) (bool, error)
}

func (i testInteraction) Emit(e Event) error {
	if i.emit != nil {
		return i.emit(e)
	}
	return nil
}
func (i testInteraction) Confirm(ctx context.Context, r Confirmation) (bool, error) {
	if i.confirm != nil {
		return i.confirm(ctx, r)
	}
	return false, nil
}

type toolFixtureActions struct {
	bash    *tools.Bash
	approve func(context.Context, tools.BashArgs) (bool, error)
	execute func(context.Context, tools.BashArgs) tools.Result
	report  func(tools.Result) error
}
type fixtureTool struct {
	tools.Tool
	execute func(context.Context, tools.Call) tools.Result
}

func (t fixtureTool) Execute(ctx context.Context, c tools.Call) tools.Result {
	if t.execute != nil {
		return t.execute(ctx, c)
	}
	return t.Tool.Execute(ctx, c)
}

type turnFixture struct {
	history *Session
	client  *modelFixture
	actions *toolFixtureActions
}

func (f turnFixture) respond(ctx context.Context, prompt string, emit func(string) error) error {
	var available []tools.Tool
	interaction := testInteraction{emit: func(e Event) error {
		if e.Kind == "text" {
			return emit(e.Text)
		}
		if e.Kind == "tool_result" && f.actions != nil && f.actions.report != nil {
			return f.actions.report(e.Result)
		}
		return nil
	}}
	if f.actions != nil {
		action := f.actions
		wrapped := fixtureTool{Tool: action.bash}
		if action.execute != nil {
			wrapped.execute = func(ctx context.Context, c tools.Call) tools.Result {
				args, _ := tools.ParseBashArguments(string(c.Arguments))
				return action.execute(ctx, args)
			}
		}
		available = []tools.Tool{wrapped}
		interaction.confirm = func(ctx context.Context, c Confirmation) (bool, error) {
			if action.approve == nil {
				return false, nil
			}
			var command string
			for _, field := range c.Fields {
				if field.Label == "Command" {
					command = field.Value
				}
			}
			return action.approve(ctx, tools.BashArgs{Command: command})
		}
	}
	a := f.client.agent(f.history, available)
	return a.Handle(ctx, prompt, interaction)
}

func isolateConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return filepath.Join(home, ".mino", "config.json")
}
func userDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".mino")
	err = os.MkdirAll(dir, 0700)
	return dir, err
}
func openHistory() (*Session, error) {
	dir, err := userDirectory()
	if err != nil {
		return nil, err
	}
	bash, err := tools.NewBash(dir, filepath.Dir(dir))
	if err != nil {
		return nil, err
	}
	return OpenSession(dir, []tools.Tool{bash})
}
func streamEvent(w http.ResponseWriter, event string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprintf(w, "data: %s\n\n", event)
	w.(http.Flusher).Flush()
}

type observingWriter struct {
	bytes.Buffer
	onWrite func(string)
}

func (w *observingWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	w.onWrite(w.String())
	return n, err
}

// A minimal driver tests whether a fatal turn error permits another request.
// Terminal behavior itself is covered in the gateway package and application tests.
func runTerminal(ctx context.Context, input io.Reader, output, errorOutput io.Writer, respond func(context.Context, string, func(string) error) error) error {
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		prompt := strings.TrimSpace(scanner.Text())
		if prompt == "/exit" {
			return nil
		}
		fmt.Fprint(output, "Assistant> ")
		err := respond(ctx, prompt, func(text string) error { _, err := fmt.Fprint(output, text); return err })
		var storage *StorageError
		if errors.As(err, &storage) {
			return err
		}
		if err != nil {
			fmt.Fprintln(errorOutput, err)
		}
	}
	return scanner.Err()
}
