package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/qshine/mino/internal/tools"
)

const sessionHelp = "Commands: /new, /sessions, /resume <id>, /clear, /compact, /help, /exit.\n"

// SessionManager serializes selection commands and whole turns. Agent remains
// responsible for model/tool execution and recovery acknowledgement.
type SessionManager struct {
	store     *Sessions
	runner    *Agent
	options   Options
	available []tools.Tool
	busy      sync.Mutex
	broken    bool
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
	m := &SessionManager{store: store, options: options, available: available}
	if store.current != nil {
		var err error
		m.runner, err = New(options, store.current, available)
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
	return i.Emit(Event{Kind: "info", Text: "Session: " + m.store.id + "\n"})
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
		if err := m.store.create(); err != nil {
			return err
		}
		m.runner, err = New(m.options, m.store.current, m.available)
		if err != nil {
			return err
		}
		return m.showCurrent(i)
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
		runner, err := New(m.options, next, m.available)
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
		approved, err := i.Confirm(ctx, Confirmation{Kind: "clear", SessionID: m.store.id, Warning: "Clear this session's saved history? This does not undo executed commands or delete migration archives, recovery copies, or manual backups."})
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
		m.runner, err = New(m.options, m.store.current, m.available)
		if err != nil {
			return err
		}
		return i.Emit(Event{Kind: "info", Text: "Cleared session: " + m.store.id + "\n"})
	}
	return errors.New("Unknown command or unexpected arguments. " + strings.TrimSpace(sessionHelp))
}

func (s *Sessions) clear() error {
	if err := s.writeFile(s.path(s.id), nil, true); err != nil {
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
	return nil
}
