package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/qshine/mino/internal/policy"
	"github.com/qshine/mino/internal/skills"
	"github.com/qshine/mino/internal/tools"
)

const sessionHelp = "Commands: /new, /sessions, /resume <id>, /clear, /compact, /mode [manual|accept-edits|bypass], /help, /exit.\n"

// SessionManager serializes selection commands and whole turns. Agent remains
// responsible for model/tool execution and recovery acknowledgement.
type SessionManager struct {
	store     *Sessions
	runner    *Agent
	options   Options
	available []tools.Tool
	busy      sync.Mutex
	broken    bool
	// Each session keeps its own permission mode until Mino exits; modes are
	// never saved, so every session starts in manual mode after a restart.
	modes map[string]*policy.Policy
}

func NewSessionManager(options Options, store *Sessions, available []tools.Tool) (*SessionManager, error) {
	if store == nil {
		return nil, errors.New("Session manager requires a store")
	}
	if _, err := toolRegistry(available); err != nil {
		return nil, err
	}
	if options.ContextWindow < 0 {
		return nil, errors.New("context_window must be a non-negative integer")
	}
	m := &SessionManager{store: store, options: options, available: available, modes: map[string]*policy.Policy{}}
	if store.current != nil {
		var err error
		m.runner, err = New(m.optionsFor(store.id), store.current, available)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (m *SessionManager) Start(ctx context.Context, i Interaction) error {
	if !m.busy.TryLock() {
		return ErrBusy
	}
	defer m.busy.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	window, source := m.options.ContextWindow, "config"
	if window == 0 {
		window, source = DefaultContextWindow, "default"
	}
	if err := i.Emit(Event{Kind: "info", Text: fmt.Sprintf("Context window: %d tokens (%s).\n", window, source)}); err != nil {
		return err
	}
	if m.store.notice != "" {
		if err := i.Emit(Event{Kind: "notice", Text: m.store.notice}); err != nil {
			return err
		}
	}
	if m.runner == nil {
		return nil
	}
	if err := m.runner.Start(ctx, i); err != nil {
		return err
	}
	return m.showCurrent(i)
}

func (m *SessionManager) showCurrent(i Interaction) error {
	return i.Emit(Event{Kind: "info", Text: fmt.Sprintf("Session: %s\nPermission mode: %s. Use /mode to change it.\n", m.store.id, m.runner.options.Policy.Mode())})
}

// optionsFor binds a session's own policy, creating a manual-mode one on first use.
func (m *SessionManager) optionsFor(id string) Options {
	p := m.modes[id]
	if p == nil {
		p = policy.New()
		m.modes[id] = p
	}
	options := m.options
	options.Policy = p
	return options
}

func (m *SessionManager) Handle(ctx context.Context, message string, i Interaction) (err error) {
	if !m.busy.TryLock() {
		return ErrBusy
	}
	defer m.busy.Unlock()
	if m.broken {
		return &StorageError{errors.New("Session storage is unavailable. Restart Mino.")}
	}
	defer func() {
		var storage *StorageError
		if errors.As(err, &storage) {
			m.broken = true
		}
	}()
	if err = ctx.Err(); err != nil {
		return err
	}
	fields := strings.Fields(message)
	if len(fields) == 0 {
		return errors.New("Input must be non-empty UTF-8 text")
	}
	if !strings.HasPrefix(fields[0], "/") {
		if m.runner == nil {
			return errors.New("Select a session with /resume <id> or /new before chatting")
		}
		return m.runner.Handle(ctx, message, i)
	}
	switch fields[0] {
	case "/mode":
		if len(fields) > 2 {
			break
		}
		return m.setMode(ctx, fields[1:], i)
	case "/compact":
		if len(fields) != 1 {
			break
		}
		if m.runner == nil {
			return errors.New("Select a session before compacting it")
		}
		return m.runner.Compact(ctx, i)
	case "/help":
		if len(fields) != 1 {
			break
		}
		return i.Emit(Event{Kind: "info", Text: sessionHelp})
	case "/sessions":
		if len(fields) != 1 {
			break
		}
		entries, err := m.store.list()
		if err != nil {
			return &StorageError{err}
		}
		var text strings.Builder
		text.WriteString("Sessions (most recently updated first):\n")
		for _, entry := range entries {
			marker := " "
			if entry.id == m.store.id {
				marker = "*"
			}
			fmt.Fprintf(&text, "%s %s  %s\n", marker, entry.id, entry.modified.UTC().Format(time.RFC3339))
		}
		if len(entries) == 0 {
			text.WriteString("No sessions. Use /new to create one.\n")
		}
		return i.Emit(Event{Kind: "info", Text: text.String()})
	case "/new":
		if len(fields) != 1 {
			break
		}
		var catalog *skills.Catalog
		var problems []string
		if m.options.ProjectDirectory != "" {
			catalog, problems, err = skills.Scan(m.options.ProjectDirectory)
			if err != nil {
				return err
			}
		}
		if err := m.store.createWithSkills(catalog); err != nil {
			return err
		}
		m.runner, err = New(m.optionsFor(m.store.id), m.store.current, m.available)
		if err != nil {
			return err
		}
		// New scanned the subagent definitions itself; keep its notice too.
		m.runner.skipped = skillNotice(m.options.ProjectDirectory, problems) + m.runner.skipped
		if err := m.showCurrent(i); err != nil {
			return err
		}
		return m.runner.showSkipped(i)
	case "/resume":
		if len(fields) != 2 {
			break
		}
		if fields[1] == m.store.id && m.runner != nil {
			return m.showCurrent(i)
		}
		next, err := m.store.open(fields[1])
		if err != nil {
			return err
		}
		runner, err := New(m.optionsFor(fields[1]), next, m.available)
		if err == nil {
			err = runner.Start(ctx, i)
		}
		// A rejected recovery never replaces the current selection.
		if err == nil {
			err = m.store.activate(fields[1], next)
		}
		if err != nil {
			next.Close()
			return err
		}
		m.runner = runner
		return m.showCurrent(i)
	case "/clear":
		if len(fields) != 1 {
			break
		}
		if m.runner == nil {
			return errors.New("Select a session before clearing it")
		}
		approved, err := i.Confirm(ctx, Confirmation{Kind: "clear", SessionID: m.store.id, Warning: "Clear this session's saved history, including its subagent logs? This does not undo executed commands or delete recovery copies or manual backups."})
		if err != nil {
			return err
		}
		if !approved {
			return i.Emit(Event{Kind: "info", Text: "Session was not cleared.\n"})
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = m.store.clear(); err != nil {
			return err
		}
		m.runner, err = New(m.optionsFor(m.store.id), m.store.current, m.available)
		if err != nil {
			return err
		}
		return i.Emit(Event{Kind: "info", Text: "Cleared session: " + m.store.id + "\n"})
	}
	return errors.New("Unknown command or unexpected arguments. " + strings.TrimSpace(sessionHelp))
}

var modeWarnings = map[policy.Mode]string{
	policy.AcceptEdits: "In this session, accept-edits mode writes and edits project files without asking. Bash and MCP calls still ask; protected files stay refused.",
	policy.Bypass:      "In this session, bypass mode runs every tool call without asking, including Bash commands with your account permissions and MCP calls that send data to their servers. Only protected files stay refused.",
}

// setMode shows or changes the current session's permission mode. Loosening it
// needs a separate confirmation in an interactive terminal; tightening never does.
func (m *SessionManager) setMode(ctx context.Context, args []string, i Interaction) error {
	if m.runner == nil {
		return errors.New("Select a session before changing its permission mode")
	}
	rules := m.runner.options.Policy
	current := rules.Mode()
	if len(args) == 0 {
		return i.Emit(Event{Kind: "info", Text: fmt.Sprintf("Permission mode for this session: %s. Modes: manual, accept-edits, bypass.\n", current)})
	}
	mode, ok := policy.ParseMode(args[0])
	if !ok {
		return errors.New("Unknown mode. Use /mode manual, /mode accept-edits, or /mode bypass.")
	}
	if mode > current {
		approved, err := i.Confirm(ctx, Confirmation{Kind: "mode", Warning: modeWarnings[mode]})
		if err != nil {
			return err
		}
		if !approved {
			return i.Emit(Event{Kind: "info", Text: fmt.Sprintf("Permission mode for this session unchanged: %s.\n", current)})
		}
	}
	rules.SetMode(mode)
	return i.Emit(Event{Kind: "info", Text: fmt.Sprintf("Permission mode for this session: %s.\n", mode)})
}

func (s *Sessions) clear() error {
	// Clearing conversation history preserves the session's discovery snapshot.
	data, err := initialRecords(s.current.state.skillCatalog, s.current.state.agentCatalog)
	if err != nil {
		return err
	}
	if err := s.writeFile(s.path(s.id), data, true); err != nil {
		s.current.broken = true
		return &StorageError{err}
	}
	next, err := s.open(s.id)
	if err != nil {
		s.current.broken = true
		return &StorageError{err}
	}
	old := s.current
	s.current = next
	if err := old.Close(); err != nil {
		return &StorageError{err}
	}
	// Subagent logs belong to the cleared conversation and go with it.
	if err := os.RemoveAll(strings.TrimSuffix(s.path(s.id), ".jsonl") + ".agents"); err != nil {
		return &StorageError{err}
	}
	if err := s.syncDir(filepath.Dir(s.path(s.id))); err != nil {
		return &StorageError{err}
	}
	return nil
}
