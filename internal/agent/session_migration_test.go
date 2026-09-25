package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionsMigrateLegacyHistoryOnce(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "history.jsonl")
	original := historyFixture(t)
	if err := os.WriteFile(legacy, original, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	first := store.id
	if len(store.current.state.input) != 2 || !strings.Contains(store.notice, "archive") {
		t.Fatal("legacy history was not restored or archive not explained")
	}
	migrated, _ := os.ReadFile(store.path(first))
	preserved, _ := os.ReadFile(legacy)
	if !bytes.Equal(migrated, original) || !bytes.Equal(preserved, original) {
		t.Fatal("migration changed completed records")
	}
	store.Close()
	store, err = OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	entries, _ := store.list()
	if store.id != first || len(entries) != 1 {
		t.Fatal("restart repeated migration")
	}
	if _, err := os.Stat(filepath.Join(dir, "history-migration.json")); !os.IsNotExist(err) {
		t.Fatal("migration marker not cleaned up")
	}
}

func TestMigrationResumesAfterPointerPublicationFailure(t *testing.T) {
	for _, failure := range []string{"rename", "directory sync"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			os.Mkdir(filepath.Join(dir, "sessions"), 0700)
			original := historyFixture(t)
			os.WriteFile(filepath.Join(dir, "history.jsonl"), original, 0600)
			store := &Sessions{dir: dir, rename: os.Rename, syncDir: syncHistoryDirectory}
			if failure == "rename" {
				store.rename = func(from, to string) error {
					if filepath.Base(to) == "active-session.json" {
						return errors.New("failed pointer rename")
					}
					return os.Rename(from, to)
				}
			} else {
				store.syncDir = func(path string) error {
					if _, err := os.Stat(filepath.Join(dir, "active-session.json")); err == nil {
						return errors.New("failed pointer directory sync")
					}
					return syncHistoryDirectory(path)
				}
			}
			if err := store.migrate(); err == nil {
				t.Fatal("expected publication failure")
			}
			marker, err := readSessionPointer(filepath.Join(dir, "history-migration.json"))
			if err != nil {
				t.Fatal(err)
			}
			store.Close()
			recovered, err := OpenSessions(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer recovered.Close()
			if _, err := os.Stat(filepath.Join(dir, "history-migration.json")); !os.IsNotExist(err) {
				t.Fatal("completed migration marker survived restart")
			}
			entries, _ := recovered.list()
			if len(entries) != 1 || recovered.id != marker.ID || len(recovered.current.state.input) != 2 {
				t.Fatalf("duplicate/lost import: %v %s", entries, recovered.id)
			}
		})
	}
}

func TestMigrationDoesNotOverwriteConflictingTarget(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "sessions"), 0700)
	id := newHistoryID()
	pointer, _ := json.Marshal(sessionPointer{Version: 1, ID: id})
	os.WriteFile(filepath.Join(dir, "history-migration.json"), pointer, 0600)
	os.WriteFile(filepath.Join(dir, "history.jsonl"), historyFixture(t), 0600)
	target := filepath.Join(dir, "sessions", id+".jsonl")
	different := bytes.ReplaceAll(historyFixture(t), []byte("blue"), []byte("pine"))
	os.WriteFile(target, different, 0600)
	store, err := OpenSessions(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.current != nil {
		t.Fatal("silently accepted a changed migration target")
	}
	after, _ := os.ReadFile(target)
	if !bytes.Equal(after, different) {
		t.Fatal("migration overwrote target")
	}
	if err := store.resume(id); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationHonorsLegacyLockAndRejectsCorruption(t *testing.T) {
	dir := t.TempDir()
	old, err := OpenSession(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if store, err := OpenSessions(dir, nil); err == nil {
		store.Close()
		t.Fatal("migrated while old process holds lock")
	}
	old.Close()
	os.WriteFile(filepath.Join(dir, "history.jsonl"), []byte("bad\n{}\n"), 0600)
	if store, err := OpenSessions(dir, nil); err == nil {
		store.Close()
		t.Fatal("migrated corrupt history")
	}
}
