package agent

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestSessionsCreateResumeAndIsolate(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	first := store.id
	if !validHistoryID(first) {
		t.Fatalf("invalid session ID %q", first)
	}
	if err := store.current.append(historyRecord{TurnID: newHistoryID(), Kind: "user_message", Text: "pine"}); err != nil {
		t.Fatal(err)
	}
	if err := store.current.finishTurn("cancelled"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store.current.file.Name())
	if err := store.create(); err != nil {
		t.Fatal(err)
	}
	second := store.id
	if first == second || store.current.state.seq != 0 {
		t.Fatal("new session inherited state")
	}
	if err := store.resume(first); err != nil {
		t.Fatal(err)
	}
	if store.current.state.seq != 2 {
		t.Fatal("previous session not restored")
	}
	got, _ := os.ReadFile(store.current.file.Name())
	if !bytes.Equal(got, before) {
		t.Fatal("switch modified history")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.id != first {
		t.Fatal("restart lost active session")
	}
	entries, err := store.list()
	if err != nil || len(entries) != 2 {
		t.Fatalf("list=%v err=%v", entries, err)
	}
	for _, path := range []string{dir, filepath.Join(dir, "sessions"), filepath.Join(dir, "sessions.lock"), filepath.Join(dir, "active-session.json"), store.current.file.Name()} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0600)
		if info.IsDir() {
			mode = 0700
		}
		if info.Mode().Perm() != mode {
			t.Fatalf("%s mode=%o", path, info.Mode().Perm())
		}
	}
}

func TestSessionsRejectPathsAndMissingIDsWithoutCreatingFiles(t *testing.T) {
	store, err := OpenSessions(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first := store.current
	for _, id := range []string{"../history", "/tmp/history", "", "abc", strings.Repeat("A", 32), strings.Repeat("a", 32)} {
		if err := store.resume(id); err == nil {
			t.Fatalf("accepted %q", id)
		}
		if store.current != first {
			t.Fatal("failed resume replaced current")
		}
	}
	entries, _ := store.list()
	if len(entries) != 1 {
		t.Fatal("failed resume created a session")
	}
}

func TestSessionsDamagedPointerRequiresSelection(t *testing.T) {
	for _, data := range []string{"", "{bad}", `{"v":1,"session_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`} {
		t.Run(data, func(t *testing.T) {
			dir := t.TempDir()
			store, err := OpenSessions(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			id := store.id
			store.Close()
			pointer := filepath.Join(dir, "active-session.json")
			if data == "" {
				err = os.Remove(pointer)
			} else {
				err = os.WriteFile(pointer, []byte(data), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			store, err = OpenSessions(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if store.current != nil || store.notice == "" {
				t.Fatal("damaged pointer silently selected a session")
			}
			if err := store.resume(id); err != nil {
				t.Fatal(err)
			}
			if store.id != id {
				t.Fatal("could not recover selection")
			}
		})
	}
}

func TestSessionsRejectUnsafeFiles(t *testing.T) {
	for _, name := range []string{"sessions", "sessions.lock", "active-session.json", "session"} {
		for _, kind := range []string{"symlink", "directory", "fifo"} {
			if name == "sessions" && kind == "directory" {
				continue
			}
			t.Run(name+"/"+kind, func(t *testing.T) {
				dir := t.TempDir()
				target := filepath.Join(t.TempDir(), "target")
				os.WriteFile(target, []byte("unchanged"), 0644)
				path := filepath.Join(dir, name)
				id := strings.Repeat("a", 32)
				if name == "session" {
					os.Mkdir(filepath.Join(dir, "sessions"), 0700)
					path = filepath.Join(dir, "sessions", id+".jsonl")
					os.WriteFile(filepath.Join(dir, "active-session.json"), []byte(`{"v":1,"session_id":"`+id+`"}`), 0600)
				}
				var err error
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
				store, err := OpenSessions(dir, nil)
				if err == nil {
					defer store.Close()
					if store.current != nil || store.notice == "" {
						t.Fatal("unsafe file was accepted")
					}
					if name == "session" {
						if err := store.resume(id); err == nil {
							t.Fatal("unsafe session resumed")
						}
					}
				}
				data, _ := os.ReadFile(target)
				info, _ := os.Stat(target)
				if string(data) != "unchanged" || info.Mode().Perm() != 0644 {
					t.Fatal("unsafe target was changed")
				}
			})
		}
	}
}

func TestSessionsExclusiveLockAcrossProcesses(t *testing.T) {
	if dir := os.Getenv("MINO_TEST_SESSIONS_CHILD"); dir != "" {
		store, err := OpenSessions(dir, nil)
		if err == nil {
			store.Close()
			t.Fatal("second process acquired store")
		}
		if !strings.Contains(err.Error(), "in use") {
			t.Fatal(err)
		}
		return
	}
	dir := t.TempDir()
	store, err := OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSessionsExclusiveLockAcrossProcesses$")
	cmd.Env = append(os.Environ(), "MINO_TEST_SESSIONS_CHILD="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		store.Close()
		t.Fatalf("child: %v %s", err, out)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
}
