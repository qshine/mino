package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/qshine/mino/internal/gateway"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func historyFixture(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	for i, r := range []map[string]any{
		{"kind": "user_message", "text": "Remember blue."},
		{"kind": "assistant_message", "text": "Blue."},
		{"kind": "turn_end", "status": "completed"},
	} {
		r["v"], r["turn_id"], r["seq"] = 1, strings.Repeat("b", 32), i+1
		if err := json.NewEncoder(&b).Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	return b.Bytes()
}

func writeHistoryFixture(t *testing.T, data []byte) string {
	t.Helper()
	isolateConfig(t)
	dir, err := userDirectory()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "history.jsonl")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHistoryRecoversEveryTruncatedWrite(t *testing.T) {
	data := historyFixture(t)
	// Every byte boundary includes complete lines, incomplete JSON, and JSON without a newline.
	for cut := 0; cut <= len(data); cut++ {
		t.Run(strconv.Itoa(cut), func(t *testing.T) {
			path := writeHistoryFixture(t, data[:cut])
			h, err := openHistory()
			if err != nil {
				t.Fatalf("cut=%d: %v", cut, err)
			}
			if cut == len(data) {
				if len(h.state.input) != 2 {
					t.Fatal("complete turn lost")
				}
			} else if len(h.state.input) != 0 {
				t.Fatalf("cut=%d replayed an incomplete turn", cut)
			}
			if h.state.pending.id != "" {
				t.Fatal("pending turn was not closed")
			}
			if err := h.Close(); err != nil {
				t.Fatal(err)
			}
			backups, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "history-recovery-*.jsonl"))
			wantBackup := cut > 0 && data[cut-1] != '\n'
			if len(backups) != boolInt(wantBackup) {
				t.Fatalf("cut=%d backups=%v", cut, backups)
			}
			if wantBackup {
				got, err := os.ReadFile(backups[0])
				if err != nil || !bytes.Equal(got, data[:cut]) {
					t.Fatal("backup changed original bytes")
				}
				info, _ := os.Stat(backups[0])
				if info.Mode().Perm() != 0600 {
					t.Fatal("backup is not private")
				}
			}
			// Recovery is idempotent and never retries a pending model request.
			reopened, err := openHistory()
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			if reopened.notice != "" {
				t.Fatalf("repeated recovery: %q", reopened.notice)
			}
		})
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestHistoryRejectsCorruptionWithoutChangingFile(t *testing.T) {
	good := string(historyFixture(t))
	for name, data := range map[string]string{
		"middle JSON":           strings.Replace(good, "\n", "\n{bad}\n", 1),
		"removed session field": strings.ReplaceAll(good, `"v":1`, `"session_id":"`+strings.Repeat("a", 32)+`","v":1`),
		"turn mismatch":         strings.Replace(good, `"turn_id":"`+strings.Repeat("b", 32)+`"`, `"turn_id":"`+strings.Repeat("c", 32)+`"`, 1),
		"invalid turn ID":       strings.Replace(good, `"turn_id":"`+strings.Repeat("b", 32)+`"`, `"turn_id":"invalid"`, 1),
		"sequence":              strings.Replace(good, `"seq":2`, `"seq":1`, 1),
		"version":               strings.Replace(good, `"v":1`, `"v":9`, 1),
		"unknown field":         strings.Replace(good, `"v":1`, `"future":true,"v":1`, 1),
		"unknown kind":          strings.Replace(good, `"turn_end"`, `"future_kind"`, 1),
		"missing answer":        strings.Split(good, "\n")[0] + "\n" + strings.Replace(strings.Split(good, "\n")[2], `"seq":3`, `"seq":2`, 1) + "\n",
		"invalid UTF-8":         strings.Replace(good, "Blue.", "\xff", 1),
	} {
		t.Run(name, func(t *testing.T) {
			path := writeHistoryFixture(t, []byte(data))
			h, err := openHistory()
			if err == nil {
				h.Close()
				t.Fatal("accepted corrupt history")
			}
			got, _ := os.ReadFile(path)
			if string(got) != data {
				t.Fatal("changed corrupt history")
			}
			if !strings.Contains(err.Error(), "line") {
				t.Fatalf("missing failure location: %v", err)
			}
		})
	}
}

func TestHistoryBacksUpMalformedFinalLine(t *testing.T) {
	data := append(historyFixture(t), []byte("{bad}\n")...)
	path := writeHistoryFixture(t, data)
	h, err := openHistory()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if len(h.state.input) != 2 || !strings.Contains(h.notice, "Recovered") {
		t.Fatal("valid context was not recovered")
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, historyFixture(t)) {
		t.Fatal("repair changed valid records")
	}
}

func TestHistoryRejectsUnsafeFiles(t *testing.T) {
	for _, name := range []string{"history.jsonl", "history.lock"} {
		for _, kind := range []string{"symlink", "directory", "fifo"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				isolateConfig(t)
				dir, err := userDirectory()
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(dir, name)
				target := filepath.Join(t.TempDir(), "private")
				if err := os.WriteFile(target, []byte("unchanged"), 0644); err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "symlink":
					err = os.Symlink(target, path)
				case "directory":
					err = os.Mkdir(path, 0700)
				case "fifo":
					err = syscall.Mkfifo(path, 0600)
				}
				if err != nil {
					t.Fatal(err)
				}
				h, err := openHistory()
				if err == nil {
					h.Close()
					t.Fatal("accepted unsafe file")
				}
				got, _ := os.ReadFile(target)
				info, _ := os.Stat(target)
				if string(got) != "unchanged" || info.Mode().Perm() != 0644 {
					t.Fatal("modified symlink target")
				}
			})
		}
	}
}

func TestHistoryExclusiveLockAcrossProcesses(t *testing.T) {
	if os.Getenv("MINO_TEST_LOCK_CHILD") == "1" {
		h, err := openHistory()
		if err == nil {
			h.Close()
			t.Fatal("second process acquired the lock")
		}
		if !strings.Contains(err.Error(), "in use") {
			t.Fatal(err)
		}
		return
	}
	isolateConfig(t)
	h, err := openHistory()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHistoryExclusiveLockAcrossProcesses$")
	cmd.Env = append(os.Environ(), "MINO_TEST_LOCK_CHILD=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		h.Close()
		t.Fatalf("child: %v: %s", err, out)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openHistory()
	if err != nil {
		t.Fatal("lock did not release:", err)
	}
	reopened.Close()
}

type failingHistoryWriter struct {
	target *os.File
	mode   string
}

func (w failingHistoryWriter) Write(p []byte) (int, error) {
	if w.mode == "short" {
		return w.target.Write(p[:len(p)/2])
	}
	if w.mode == "write" {
		return 0, errors.New("disk full")
	}
	return w.target.Write(p)
}
func (w failingHistoryWriter) Sync() error { return errors.New("sync failed") }

func TestHistoryFailureStopsBeforeFurtherRequests(t *testing.T) {
	for _, mode := range []string{"write", "short", "sync"} {
		t.Run(mode, func(t *testing.T) {
			isolateConfig(t)
			h, err := openHistory()
			if err != nil {
				t.Fatal(err)
			}
			defer h.Close()
			h.writer = failingHistoryWriter{h.file, mode}
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				t.Error("request sent after storage failure")
			}))
			defer server.Close()
			chat := Agent{h, newResponsesClient(Options{server.URL, "test-key", "test-model"}, "")}
			err = gateway.Run(context.Background(), strings.NewReader("first\nsecond\n"), io.Discard, io.Discard, chat.Handle)
			var storageErr *StorageError
			if !errors.As(err, &storageErr) || requests != 0 || len(h.state.input) != 0 {
				t.Fatalf("err=%v requests=%d", err, requests)
			}
		})
	}
}

func TestHistoryLimitsDoNotDiscardOldRecords(t *testing.T) {
	path := writeHistoryFixture(t, historyFixture(t))
	h, err := openHistory()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	before, _ := os.ReadFile(path)
	if err := h.append(historyRecord{Kind: "user_message", TurnID: newHistoryID(), Text: strings.Repeat("x", maxHistoryRecordBytes)}); err == nil {
		t.Fatal("accepted oversized record")
	}
	h.size = maxHistoryBytes
	if err := h.append(historyRecord{Kind: "user_message", TurnID: newHistoryID(), Text: "hello"}); err == nil {
		t.Fatal("accepted oversized file")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("limit failure altered history")
	}
}
