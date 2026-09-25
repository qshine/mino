package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qshine/mino/internal/tools"
)

// Sessions owns one store-wide lock, the durable selection, and its open log.
// SessionManager serializes commands and complete Agent turns using this store.
type Sessions struct {
	dir        string
	lock       *os.File
	current    *Session
	id, notice string
	registry   map[string]tools.Tool
	// Only filesystem publication boundaries are replaceable for fault tests.
	rename  func(string, string) error
	syncDir func(string) error
}

type sessionPointer struct {
	Version int    `json:"v"`
	ID      string `json:"session_id"`
}

type sessionEntry struct {
	id       string
	modified time.Time
}

func OpenSessions(dir string, available []tools.Tool) (_ *Sessions, err error) {
	registry, err := toolRegistry(available)
	if err != nil {
		return nil, err
	}
	if err = secureSessionDirectory(dir); err != nil {
		return nil, err
	}
	s := &Sessions{dir: dir, registry: registry, rename: os.Rename, syncDir: syncHistoryDirectory}
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	s.lock, err = lockHistory(filepath.Join(dir, "sessions.lock"))
	if err != nil {
		return nil, err
	}
	if err = secureSessionDirectory(filepath.Join(dir, "sessions")); err != nil {
		return nil, err
	}
	if err = s.syncDir(dir); err != nil {
		return nil, err
	}
	pointer, readErr := readSessionPointer(filepath.Join(dir, "active-session.json"))
	if readErr == nil {
		s.current, err = s.open(pointer.ID)
		if err != nil {
			var storage *StorageError
			if errors.As(err, &storage) {
				return nil, err
			}
			s.selectionNotice(err)
			return s, nil
		}
		s.id = pointer.ID
		// Publication may have succeeded just before a crash. Once a matching
		// active pointer is readable, finish cleanup without importing again.
		marker, markerErr := readSessionPointer(filepath.Join(dir, "history-migration.json"))
		if markerErr == nil && marker.ID == s.id {
			if err = s.finishMigration(); err != nil {
				return nil, err
			}
		}
		return s, nil
	}
	if !errors.Is(readErr, os.ErrNotExist) {
		s.selectionNotice(readErr)
		return s, nil
	}
	// A pending import takes precedence over a missing pointer: retry its fixed
	// destination rather than creating another copy of the old conversation.
	_, markerErr := readSessionPointer(filepath.Join(dir, "history-migration.json"))
	if markerErr == nil {
		if err = s.migrate(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if !errors.Is(markerErr, os.ErrNotExist) {
		s.selectionNotice(markerErr)
		return s, nil
	}
	entries, err := s.list()
	if err != nil {
		return nil, err
	}
	if len(entries) > 0 {
		s.selectionNotice(errors.New("Active session pointer is missing"))
		return s, nil
	}
	if _, legacyErr := os.Lstat(filepath.Join(dir, "history.jsonl")); legacyErr == nil {
		if err = s.migrate(); err != nil {
			return nil, err
		}
		return s, nil
	} else if !errors.Is(legacyErr, os.ErrNotExist) {
		return nil, legacyErr
	}
	if err = s.create(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Sessions) selectionNotice(err error) {
	s.notice = fmt.Sprintf("Cannot restore the active session: %v. Use /sessions and /resume <id> to select a session, or /new to start one.\n", err)
}

func readSessionPointer(path string) (sessionPointer, error) {
	var pointer sessionPointer
	f, err := openPrivateFile(path, 0)
	if err != nil {
		return pointer, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 1025))
	if err != nil {
		return pointer, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if len(data) > 1024 || decoder.Decode(&pointer) != nil || pointer.Version != 1 || !validHistoryID(pointer.ID) {
		return pointer, errors.New("Invalid session pointer")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return pointer, errors.New("Invalid session pointer")
	}
	return pointer, nil
}

func (s *Sessions) path(id string) string { return filepath.Join(s.dir, "sessions", id+".jsonl") }

func (s *Sessions) open(id string) (*Session, error) {
	if !validHistoryID(id) {
		return nil, errors.New("Use a complete 32-character lowercase session ID from /sessions")
	}
	return openSessionFile(s.path(id), "", s.registry, false)
}

func (s *Sessions) create() error {
	id := newHistoryID()
	if err := s.writeFile(s.path(id), nil, false); err != nil {
		return &StorageError{err}
	}
	return s.resume(id)
}

func (s *Sessions) resume(id string) error {
	if id == s.id && s.current != nil {
		return nil
	}
	next, err := s.open(id)
	if err != nil {
		return err
	}
	if err = s.activate(id, next); err != nil {
		next.Close()
		return err
	}
	return nil
}

func (s *Sessions) activate(id string, next *Session) error {
	data, _ := json.Marshal(sessionPointer{Version: 1, ID: id})
	if err := s.writeFile(filepath.Join(s.dir, "active-session.json"), append(data, '\n'), true); err != nil {
		return &StorageError{err}
	}
	old := s.current
	s.current, s.id = next, id
	if old != nil {
		if err := old.Close(); err != nil {
			return &StorageError{err}
		}
	}
	return nil
}

func (s *Sessions) list() ([]sessionEntry, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, "sessions"))
	if err != nil {
		return nil, err
	}
	var result []sessionEntry
	for _, entry := range entries {
		id := strings.TrimSuffix(entry.Name(), ".jsonl")
		if entry.Name() != id+".jsonl" || !validHistoryID(id) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		result = append(result, sessionEntry{id, info.ModTime()})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].modified.Equal(result[j].modified) {
			return result[i].id < result[j].id
		}
		return result[i].modified.After(result[j].modified)
	})
	return result, nil
}

// writeFile publishes only fully written and synced private files. A failed
// directory sync after publication has uncertain durability: callers stop chat.
func (s *Sessions) writeFile(path string, data []byte, replace bool) error {
	if info, err := os.Lstat(path); err == nil {
		if !replace {
			return os.ErrExist
		}
		if !info.Mode().IsRegular() {
			return errors.New("Refusing to replace a non-regular session file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".session-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	n, err := temp.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = temp.Sync()
	}
	err = errors.Join(err, temp.Close())
	if err != nil {
		return err
	}
	if replace {
		err = s.rename(temp.Name(), path)
	} else {
		// A hard link publishes a new file without ever replacing an existing ID.
		err = os.Link(temp.Name(), path)
		if err == nil {
			err = os.Remove(temp.Name())
		}
	}
	if err != nil {
		return err
	}
	return s.syncDir(filepath.Dir(path))
}

func (s *Sessions) Close() error {
	var err error
	if s.current != nil {
		err = s.current.Close()
		s.current = nil
	}
	if s.lock != nil {
		err = errors.Join(err, s.lock.Close())
		s.lock = nil
	}
	return err
}
