package mino

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

	"github.com/openai/openai-go/v3/responses"
)

const historyLocation = "~/.mino/history.jsonl"
const maxHistoryRecordBytes = 16 << 20
const maxHistoryBytes = 64 << 20

type historyRecord struct {
	Version int               `json:"v"`
	Seq     int               `json:"seq"`
	TurnID  string            `json:"turn_id"`
	Kind    string            `json:"kind"`
	Text    string            `json:"text,omitempty"`
	Output  []json.RawMessage `json:"output,omitempty"`
	Status  string            `json:"status,omitempty"`
}

type pendingTurn struct {
	id, prompt string
	reply      *modelReply
}

type historyState struct {
	seq     int
	pending pendingTurn
	turns   map[string]bool
	input   responses.ResponseInputParam
}

// Only the write/sync boundary is replaceable, to test short writes and disk failures.
type historyWriter interface {
	Write([]byte) (int, error)
	Sync() error
}

type history struct {
	file, lock *os.File
	writer     historyWriter
	state      historyState
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

func openHistory() (_ *history, err error) {
	dir, err := userDirectory()
	if err != nil {
		return nil, err
	}
	h := &history{state: historyState{turns: make(map[string]bool)}}
	defer func() {
		if err != nil {
			h.close()
		}
	}()
	h.lock, err = openPrivateHistoryFile(filepath.Join(dir, "history.lock"))
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(h.lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, errors.New("History is in use by another Mino process. Close it and try again.")
	}
	h.file, err = openPrivateHistoryFile(filepath.Join(dir, "history.jsonl"))
	if err != nil {
		return nil, err
	}
	h.writer = h.file
	if err = syncHistoryDirectory(dir); err != nil {
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
		if err = h.append(historyRecord{TurnID: h.state.pending.id, Kind: "turn_end", Status: "interrupted"}); err != nil {
			return nil, err
		}
		h.notice += "An interrupted turn was retained but excluded from context. It was not retried.\n"
	}
	return h, nil
}

func openPrivateHistoryFile(path string) (*os.File, error) {
	// O_NOFOLLOW closes the check/open symlink race; NONBLOCK avoids blocking on a FIFO.
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_APPEND|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0600)
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

func (h *history) close() error {
	var err error
	if h.file != nil {
		err = h.file.Close()
	}
	// Closing releases flock; never unlink the stable lock file.
	if h.lock != nil {
		err = errors.Join(err, h.lock.Close())
	}
	return err
}

func (h *history) append(records ...historyRecord) error {
	if h.broken {
		return errors.New("History is unavailable after a write failure. Restart Mino.")
	}
	var data bytes.Buffer
	for i := range records {
		r := &records[i]
		r.Version, r.Seq = 1, h.state.seq+i+1
		line, err := json.Marshal(r)
		if err != nil {
			return errors.New("Failed to encode history record")
		}
		if len(line)+1 > maxHistoryRecordBytes {
			return errors.New("History record exceeds the 16 MiB limit")
		}
		data.Write(line)
		data.WriteByte('\n')
	}
	if h.size+int64(data.Len()) > maxHistoryBytes {
		return errors.New("History reached the 64 MiB limit. Back it up and move it aside before restarting.")
	}
	n, err := h.writer.Write(data.Bytes())
	if err == nil && n != data.Len() {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = h.writer.Sync()
	}
	if err != nil {
		h.broken = true
		return fmt.Errorf("Failed to save history: %w", err)
	}
	h.size += int64(n)
	for _, record := range records {
		if err := h.state.apply(record); err != nil {
			h.broken = true
			return fmt.Errorf("Invalid history transition: %w", err)
		}
	}
	return nil
}

func (s *historyState) apply(r historyRecord) error {
	if r.Version != 1 || r.Seq != s.seq+1 || !validHistoryID(r.TurnID) {
		return errors.New("invalid version, sequence, or identifier")
	}
	switch r.Kind {
	case "user_message":
		if s.pending.id != "" || s.turns[r.TurnID] || !validHistoryText(r.Text) || r.Status != "" || len(r.Output) != 0 {
			return errors.New("invalid user message or turn order")
		}
		s.pending = pendingTurn{id: r.TurnID, prompt: r.Text}
		s.turns[r.TurnID] = true
	case "assistant_message":
		if s.pending.id != r.TurnID || s.pending.reply != nil || !validHistoryText(r.Text) || r.Status != "" {
			return errors.New("invalid assistant message or turn order")
		}
		reply := modelReply{Text: r.Text, Output: r.Output}
		if _, err := reply.inputItems(); err != nil {
			return err
		}
		s.pending.reply = &reply
	case "turn_end":
		if s.pending.id != r.TurnID || r.Text != "" || len(r.Output) != 0 {
			return errors.New("invalid turn ending")
		}
		switch r.Status {
		case "completed":
			if s.pending.reply == nil {
				return errors.New("completed turn has no answer")
			}
			items, err := s.pending.reply.inputItems()
			if err != nil {
				return err
			}
			s.input = append(s.input, userInput(s.pending.prompt))
			s.input = append(s.input, items...)
		case "failed", "cancelled", "interrupted":
		default:
			return errors.New("unknown turn status")
		}
		s.pending = pendingTurn{}
	default:
		return errors.New("unknown record kind")
	}
	s.seq = r.Seq
	return nil
}

func validHistoryText(text string) bool {
	return utf8.ValidString(text) && strings.TrimSpace(text) != ""
}

func (h *history) load(data []byte) error {
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

func (h *history) recoverTail(data []byte, offset int) error {
	backup, err := os.CreateTemp(filepath.Dir(h.file.Name()), "history-recovery-*.jsonl")
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
