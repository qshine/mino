package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/memory"
)

func newTestMemory(t *testing.T) (*memory.Store, *MemoryView, *MemorySave) {
	t.Helper()
	store, err := memory.Open(filepath.Join(t.TempDir(), "memory"))
	if err != nil {
		t.Fatal(err)
	}
	return store, NewMemoryView(store), NewMemorySave(store)
}

func saveArguments(name, mode, summary, details string) string {
	raw, _ := json.Marshal(map[string]string{"name": name, "type": "feedback", "summary": summary, "details": details, "mode": mode})
	return string(raw)
}

// approveSave runs the live path the Agent uses: Prepare, Inspect, Execute.
func approveSave(t *testing.T, save *MemorySave, arguments string) (Call, Result) {
	t.Helper()
	call, err := save.Prepare(arguments)
	if err != nil {
		t.Fatal(err)
	}
	if call.ReadOnly || call.Target != "" || call.Sandboxed {
		t.Fatalf("memory_save must ask outside bypass mode: %+v", call)
	}
	call, err = save.Inspect(call)
	if err != nil {
		return Call{}, Result{Status: "inspect_failed", Output: err.Error()}
	}
	return call, save.Execute(context.Background(), call)
}

func TestMemorySaveCreatesAndOverwrites(t *testing.T) {
	store, view, save := newTestMemory(t)
	call, result := approveSave(t, save, saveArguments("prefers-chinese", "create", "始终用中文回答", ""))
	if result.Status != "completed" || call.Expect != "absent" {
		t.Fatalf("create = %+v, expect %q", result, call.Expect)
	}
	path := filepath.Join(store.Dir(), "prefers-chinese.md")
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("memory file mode = %v, %v", info, err)
	}
	var fields []string
	for _, f := range call.Fields {
		fields = append(fields, f.Label+"="+f.Value)
	}
	if !strings.Contains(strings.Join(fields, ";"), "Summary=始终用中文回答") || strings.Contains(strings.Join(fields, ";"), "Details") || !strings.Contains(call.Warning, "every later session") {
		t.Fatalf("confirmation = %q, %q", fields, call.Warning)
	}

	if _, result := approveSave(t, save, saveArguments("prefers-chinese", "create", "again", "")); !strings.Contains(result.Output, "already exists") {
		t.Fatalf("create over an existing memory = %+v", result)
	}
	if _, result := approveSave(t, save, saveArguments("missing", "overwrite", "s", "")); !strings.Contains(result.Output, "use mode create") {
		t.Fatalf("overwrite of a missing memory = %+v", result)
	}

	call, result = approveSave(t, save, saveArguments("prefers-chinese", "overwrite", "Answer in Chinese", "Code stays in English."))
	if result.Status != "completed" || call.Expect == "absent" {
		t.Fatalf("overwrite = %+v", result)
	}
	viewCall, err := view.Prepare(`{"name":"prefers-chinese"}`)
	if err != nil || !viewCall.ReadOnly {
		t.Fatalf("memory_view must be read-only: %+v, %v", viewCall, err)
	}
	if result := view.Execute(context.Background(), viewCall); result.Status != "completed" || !strings.Contains(result.Output, "Code stays in English.") {
		t.Fatalf("view = %+v", result)
	}
	if entries, _, _ := store.Scan(); len(entries) != 1 || !entries[0].Details || entries[0].Summary != "Answer in Chinese" {
		t.Fatalf("index = %+v", entries)
	}
}

func TestMemorySaveRechecksAfterApproval(t *testing.T) {
	store, _, save := newTestMemory(t)
	approveSave(t, save, saveArguments("fact", "create", "one", ""))
	call, _ := save.Prepare(saveArguments("fact", "overwrite", "two", ""))
	call, err := save.Inspect(call)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir(), "fact.md")
	os.WriteFile(path, []byte(memory.Format(memory.Memory{Name: "fact", Type: "user", Summary: "edited by the user"})), 0600)
	if result := save.Execute(context.Background(), call); result.Status != "failed" || !strings.Contains(result.Output, "changed after approval") {
		t.Fatalf("execute after a change = %+v", result)
	}
	if !strings.Contains(readTestFile(t, path), "edited by the user") {
		t.Fatal("the user's edit was overwritten")
	}

	create, _ := save.Prepare(saveArguments("race", "create", "s", ""))
	create, err = save.Inspect(create)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(store.Dir(), "race.md"), []byte("written meanwhile"), 0600)
	if result := save.Execute(context.Background(), create); result.Status != "failed" {
		t.Fatalf("create over a file written after approval = %+v", result)
	}
	if result := save.Execute(context.Background(), Call{Arguments: create.Arguments}); !strings.Contains(result.Output, "not inspected") {
		t.Fatalf("uninspected execute = %+v", result)
	}
}

func TestMemorySaveRefusesFullIndexBeforeAsking(t *testing.T) {
	store, _, save := newTestMemory(t)
	for i := range memory.MaxEntries {
		name := fmt.Sprintf("m%03d", i)
		os.WriteFile(filepath.Join(store.Dir(), name+".md"), []byte(memory.Format(memory.Memory{Name: name, Type: "user", Summary: "s"})), 0600)
	}
	if _, result := approveSave(t, save, saveArguments("one-more", "create", "s", "")); result.Status != "inspect_failed" || !strings.Contains(result.Output, "index is full") {
		t.Fatalf("save into a full index = %+v", result)
	}
}

func TestMemoryArgumentsAreValidated(t *testing.T) {
	_, view, save := newTestMemory(t)
	for _, raw := range []string{
		saveArguments("Bad_Name", "create", "s", ""),
		saveArguments("ok", "append", "s", ""),
		saveArguments("ok", "create", "two\nlines", ""),
		saveArguments("ok", "create", strings.Repeat("x", memory.MaxSummaryBytes+1), ""),
		saveArguments("ok", "create", " ", ""),
		`{"name":"ok","type":"system","summary":"s","details":"","mode":"create"}`,
		`{"name":"ok","type":"user","summary":"s","mode":"create"}`,
		`{"name":"ok","type":"user","summary":"s","details":"","mode":"create","extra":""}`,
	} {
		if _, err := save.Prepare(raw); err == nil {
			t.Errorf("memory_save accepted %s", raw)
		}
	}
	for _, raw := range []string{`{}`, `{"name":"../x"}`, `{"name":"a","path":"b"}`} {
		if _, err := view.Prepare(raw); err == nil {
			t.Errorf("memory_view accepted %s", raw)
		}
	}
	call, _ := view.Prepare(`{"name":"absent"}`)
	if result := view.Execute(context.Background(), call); result.Status != "failed" {
		t.Fatalf("view of a missing memory = %+v", result)
	}
}
