package agent

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/qshine/mino/internal/tools"
)

const maxHistoryRecordBytes = 16 << 20
const maxHistoryBytes = 64 << 20

// Only the write/sync boundary is replaceable, to test short writes and disk failures.
type historyWriter interface {
	Write([]byte) (int, error)
	Sync() error
}

type history struct {
	file, lock *os.File
	writer     historyWriter
	size       int64
	broken     bool
	notice     string
}

func newHistoryID() string {
	var id [16]byte
	rand.Read(id[:])
	return hex.EncodeToString(id[:])
}

func validHistoryID(id string) bool {
	if len(id) != 32 || strings.ToLower(id) != id {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

// OpenSession restores one JSONL conversation without executing tools.
func OpenSession(dir string, available []tools.Tool) (_ *Session, err error) {
	registry, err := toolRegistry(available)
	if err != nil {
		return nil, err
	}
	if err := secureSessionDirectory(dir); err != nil {
		return nil, err
	}
	return openSessionFile(filepath.Join(dir, "history.jsonl"), filepath.Join(dir, "history.lock"), registry, true)
}

func secureSessionDirectory(dir string) error {
	if !filepath.IsAbs(dir) {
		return errors.New("Session directory must be absolute")
	}
	if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("Session directory must not be a file or symbolic link")
	}
	return os.Chmod(dir, 0700)
}

// A session store holds its own lock. The legacy opener supplies history.lock.
func openSessionFile(path, lockPath string, registry map[string]tools.Tool, create bool) (_ *Session, err error) {
	h := &Session{history: &history{}, state: historyState{turns: make(map[string]bool), tools: registry}}
	defer func() {
		if err != nil {
			h.Close()
		}
	}()
	if lockPath != "" {
		h.lock, err = lockHistory(lockPath)
		if err != nil {
			return nil, err
		}
	}
	flags := 0
	if create {
		flags = syscall.O_CREAT
	}
	h.file, err = openPrivateFile(path, flags)
	if err != nil {
		return nil, err
	}
	h.writer = h.file
	if err = syncHistoryDirectory(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("Failed to sync history directory: %w", err)
	}
	info, err := h.file.Stat()
	if err != nil {
		return nil, fmt.Errorf("Failed to inspect history: %w", err)
	}
	if info.Size() > maxHistoryBytes {
		return nil, errors.New("History exceeds the 64 MiB limit. Back it up and move it aside before restarting.")
	}
	data, err := io.ReadAll(io.LimitReader(h.file, maxHistoryBytes+1))
	if err != nil {
		return nil, fmt.Errorf("Failed to read history: %w", err)
	}
	if len(data) > maxHistoryBytes {
		return nil, errors.New("History exceeds the 64 MiB limit")
	}
	if err = h.load(data); err != nil {
		return nil, err
	}
	if h.state.pending.id != "" {
		hadTools := len(h.state.pending.calls) > 0
		if err = h.finishTurn("interrupted"); err != nil {
			return nil, &StorageError{err}
		}
		if hadTools {
			h.notice += "An interrupted tool turn was recovered without executing commands.\n"
		} else {
			h.notice += "An interrupted turn was retained but excluded from context. It was not retried.\n"
		}
	}
	return h, nil
}

func lockHistory(path string) (*os.File, error) {
	lock, err := openPrivateHistoryFile(path)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("History is in use by another Mino process. Close it and try again.")
	}
	return lock, nil
}

func openPrivateHistoryFile(path string) (*os.File, error) {
	return openPrivateFile(path, syscall.O_CREAT)
}

func openPrivateFile(path string, flags int) (*os.File, error) {
	// O_NOFOLLOW closes the check/open symlink race; NONBLOCK avoids blocking on a FIFO.
	fd, err := syscall.Open(path, syscall.O_RDWR|flags|syscall.O_APPEND|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return nil, fmt.Errorf("Failed to open %s safely: %w", filepath.Base(path), err)
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = errors.New("must be a regular file")
	}
	if err == nil {
		err = f.Chmod(0600)
	}
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("Failed to secure %s: %w", filepath.Base(path), err)
	}
	return f, nil
}

func (h *history) Close() error {
	var err error
	if h.file != nil {
		err = h.file.Close()
		h.file = nil
	}
	// Closing releases flock; never unlink the stable lock file.
	if h.lock != nil {
		err = errors.Join(err, h.lock.Close())
		h.lock = nil
	}
	return err
}

func validHistoryText(text string) bool {
	return utf8.ValidString(text) && strings.TrimSpace(text) != ""
}

func (h *Session) load(data []byte) error {
	for offset, lineNo := 0, 1; offset < len(data); lineNo++ {
		end := bytes.IndexByte(data[offset:], '\n')
		if end < 0 {
			return h.recoverTail(data, offset)
		}
		end += offset
		line := data[offset:end]
		if len(line)+1 > maxHistoryRecordBytes {
			return fmt.Errorf("History line %d exceeds the 16 MiB limit", lineNo)
		}
		if !json.Valid(line) {
			if end+1 == len(data) {
				return h.recoverTail(data, offset)
			}
			return fmt.Errorf("Invalid JSON in history at line %d. Restore a valid backup before restarting.", lineNo)
		}
		var record historyRecord
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if !utf8.Valid(line) {
			return fmt.Errorf("Invalid UTF-8 in history at line %d", lineNo)
		}
		if err := decoder.Decode(&record); err != nil {
			return fmt.Errorf("Invalid history fields at line %d", lineNo)
		}
		if err := h.state.apply(record); err != nil {
			return fmt.Errorf("Invalid history at line %d: %w", lineNo, err)
		}
		offset = end + 1
	}
	h.size = int64(len(data))
	return nil
}

func (h *Session) recoverTail(data []byte, offset int) (err error) {
	defer func() {
		if err != nil {
			err = &StorageError{err}
		}
	}()
	backup, err := os.CreateTemp(filepath.Dir(h.file.Name()), strings.TrimSuffix(filepath.Base(h.file.Name()), ".jsonl")+"-recovery-*.jsonl")
	if err != nil {
		return fmt.Errorf("Failed to create history recovery backup: %w", err)
	}
	name := backup.Name()
	n, err := backup.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = backup.Sync()
	}
	err = errors.Join(err, backup.Close())
	if err == nil {
		err = syncHistoryDirectory(filepath.Dir(name))
	}
	if err != nil {
		return fmt.Errorf("Failed to save recovery backup; history was not changed: %w", err)
	}
	if err := h.file.Truncate(int64(offset)); err != nil {
		return fmt.Errorf("Failed to repair history; backup at %s: %w", name, err)
	}
	if err := h.file.Sync(); err != nil {
		return fmt.Errorf("Failed to sync repaired history; backup at %s: %w", name, err)
	}
	h.size = int64(offset)
	h.notice = fmt.Sprintf("Recovered an incomplete history tail. Original saved to %s.\n", name)
	return nil
}

func syncHistoryDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
