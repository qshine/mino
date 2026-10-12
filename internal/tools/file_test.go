package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func newTestFiles(t *testing.T) (*Files, string) {
	t.Helper()
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	files, err := NewFiles(project, filepath.Join(t.TempDir(), ".mino"))
	if err != nil {
		t.Fatal(err)
	}
	return files, project
}

func writeTestFile(t *testing.T, path, text string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), perm); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// run follows the Agent's order: Prepare, optional Inspect, then Execute.
func run(t *testing.T, tool Tool, raw string) (Call, Result) {
	t.Helper()
	call, err := tool.Prepare(raw)
	if err != nil {
		t.Fatalf("prepare %s: %v", raw, err)
	}
	if inspector, ok := tool.(Inspector); ok {
		if call, err = inspector.Inspect(call); err != nil {
			return call, Result{Status: "invalid_arguments", Output: err.Error()}
		}
	}
	return call, tool.Execute(context.Background(), call)
}

func TestFileToolArguments(t *testing.T) {
	files, _ := newTestFiles(t)
	cases := map[Tool][]string{
		files.Read(): {`{}`, `null`, `{"path":""}`, `{"path":"."}`, `{"path":"../x"}`, `{"path":"/etc/hosts"}`, `{"path":"a","path":"a"}`, `{"path":"a","extra":"x"}`, `{"path":1}`, `{"path":"a"} {}`, `{"path":"a\u0000b"}`},
		files.Write(): {`{"path":"a","content":"x"}`, `{"path":"a","content":"x","mode":"append"}`, `{"path":"../a","content":"x","mode":"create"}`, `{"path":"a","content":"x\u0000","mode":"create"}`,
			`{"path":"a","content":"` + strings.Repeat("x", maxFileBytes+1) + `","mode":"create"}`},
		files.Edit(): {`{"path":"a","old_string":"","new_string":"x"}`, `{"path":"a","old_string":"x","new_string":"x"}`, `{"path":"a","old_string":"x"}`},
	}
	for tool, inputs := range cases {
		for _, raw := range inputs {
			if _, err := tool.Prepare(raw); err == nil {
				t.Errorf("%s accepted %.80s", tool.Definition().Name, raw)
			}
		}
	}
	// Field order is free, paths are canonical, and preparation never touches the
	// filesystem, so history can validate calls for files that no longer exist.
	call, err := files.Edit().Prepare(`{"new_string":"b","path":"./src//missing.go","old_string":"a"}`)
	if err != nil || string(call.Arguments) != `{"path":"src/missing.go","old_string":"a","new_string":"b"}` {
		t.Fatalf("prepare: %s, %v", call.Arguments, err)
	}
	if call, err := files.Read().Prepare(`{"path":"missing"}`); err != nil || !call.ReadOnly {
		t.Fatalf("read prepare: %+v, %v", call, err)
	}
	if call, err := files.Write().Prepare(`{"path":"a","content":"x","mode":"create"}`); err != nil || call.ReadOnly {
		t.Fatalf("write must require approval: %+v, %v", call, err)
	}
}

func TestReadFileStaysInsideProject(t *testing.T) {
	files, project := newTestFiles(t)
	outside := filepath.Join(t.TempDir(), "secret.txt")
	writeTestFile(t, outside, "SECRET", 0600)
	writeTestFile(t, filepath.Join(project, "src", "main.go"), "package main\n", 0600)
	writeTestFile(t, filepath.Join(project, "big.txt"), strings.Repeat("x", maxFileBytes+1), 0600)
	writeTestFile(t, filepath.Join(project, "binary"), "\xff\xfe", 0600)
	if err := os.Symlink(outside, filepath.Join(project, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(outside), filepath.Join(project, "outside")); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(outside, filepath.Join(project, "hard.txt")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(project, "fifo"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, result := run(t, files.Read(), `{"path":"src/main.go"}`); result.Status != "completed" || result.Output != "package main\n" {
		t.Fatalf("read: %+v", result)
	}
	for _, path := range []string{"link.txt", "outside/secret.txt", "hard.txt", "fifo", "src", "big.txt", "binary", "missing"} {
		_, result := run(t, files.Read(), `{"path":"`+path+`"}`)
		if result.Status != "failed" || strings.Contains(result.Output, "SECRET") {
			t.Errorf("%s: %+v", path, result)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	call, _ := files.Read().Prepare(`{"path":"src/main.go"}`)
	if result := files.Read().Execute(ctx, call); result.Status != "cancelled" {
		t.Fatalf("cancelled: %+v", result)
	}
}

func TestFileToolsDisabledWhenProjectContainsData(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(home, ".mino", "config.json"), `{"api_key":"KEY"}`, 0600)
	for _, project := range []string{home, filepath.Join(home, ".mino")} {
		files, err := NewFiles(project, filepath.Join(home, ".mino"))
		if err != nil {
			t.Fatal(err)
		}
		_, result := run(t, files.Read(), `{"path":".mino/config.json"}`)
		if result.Status != "failed" || strings.Contains(result.Output, "KEY") || !strings.Contains(result.Output, "unavailable") {
			t.Fatalf("%s: %+v", project, result)
		}
		if _, result := run(t, files.Write(), `{"path":"new.txt","content":"x","mode":"create"}`); result.Status != "invalid_arguments" {
			t.Fatalf("write: %+v", result)
		}
	}
}

func TestWriteFileModesAndApprovalBinding(t *testing.T) {
	files, project := newTestFiles(t)
	path := filepath.Join(project, "notes.txt")
	call, result := run(t, files.Write(), `{"path":"notes.txt","content":"one\ntwo\n","mode":"create"}`)
	if result.Status != "completed" || readTestFile(t, path) != "one\ntwo\n" {
		t.Fatalf("create: %+v", result)
	}
	if !strings.Contains(call.Fields[2].Value, "Create") || call.Fields[3].Value != "8 bytes, 2 lines" || call.Warning == "" {
		t.Fatalf("fields: %+v", call.Fields)
	}
	if _, result := run(t, files.Write(), `{"path":"notes.txt","content":"x","mode":"create"}`); result.Status != "invalid_arguments" || readTestFile(t, path) != "one\ntwo\n" {
		t.Fatalf("create over existing: %+v", result)
	}
	if _, result := run(t, files.Write(), `{"path":"missing/notes.txt","content":"x","mode":"create"}`); result.Status != "invalid_arguments" {
		t.Fatalf("missing parent: %+v", result)
	}
	if _, result := run(t, files.Write(), `{"path":"absent.txt","content":"x","mode":"overwrite"}`); result.Status != "invalid_arguments" {
		t.Fatalf("overwrite absent: %+v", result)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(project, "outside")); err != nil {
		t.Fatal(err)
	}
	if _, result := run(t, files.Write(), `{"path":"outside/escape.txt","content":"x","mode":"create"}`); result.Status != "invalid_arguments" {
		t.Fatalf("escape: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(outside, "escape.txt")); !os.IsNotExist(err) {
		t.Fatal("wrote outside the project")
	}

	if err := os.Chmod(path, 0640); err != nil {
		t.Fatal(err)
	}
	if _, result := run(t, files.Write(), `{"path":"notes.txt","content":"three\n","mode":"overwrite"}`); result.Status != "completed" || readTestFile(t, path) != "three\n" {
		t.Fatalf("overwrite: %+v", result)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("mode not preserved: %v %v", info.Mode(), err)
	}

	// Approval covers the inspected file. A change after inspection, or a target
	// created after inspecting a create request, aborts without writing.
	write := files.Write().(*WriteFile)
	call, _ = write.Prepare(`{"path":"notes.txt","content":"approved\n","mode":"overwrite"}`)
	call, err := write.Inspect(call)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, "changed\n", 0640)
	if result := write.Execute(context.Background(), call); result.Status != "failed" || readTestFile(t, path) != "changed\n" {
		t.Fatalf("changed after approval: %+v", result)
	}
	call, _ = write.Prepare(`{"path":"late.txt","content":"approved\n","mode":"create"}`)
	if call, err = write.Inspect(call); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(project, "late.txt"), "someone else\n", 0600)
	if result := write.Execute(context.Background(), call); result.Status != "failed" || readTestFile(t, filepath.Join(project, "late.txt")) != "someone else\n" {
		t.Fatalf("created after approval: %+v", result)
	}
	uninspected, _ := write.Prepare(`{"path":"late.txt","content":"x","mode":"overwrite"}`)
	if result := write.Execute(context.Background(), uninspected); result.Status != "failed" {
		t.Fatalf("uninspected: %+v", result)
	}
	entries, _ := os.ReadDir(project)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".mino-") {
			t.Fatalf("temporary file left behind: %s", entry.Name())
		}
	}
}

func TestEditFileReplacesOneUniqueFragment(t *testing.T) {
	files, project := newTestFiles(t)
	path := filepath.Join(project, "main.go")
	writeTestFile(t, path, "package main\n\nfunc a() {}\nfunc b() {}\nfunc b() {}\n", 0600)
	call, result := run(t, files.Edit(), `{"path":"main.go","old_string":"func a() {}","new_string":"func a() { println() }"}`)
	if result.Status != "completed" || readTestFile(t, path) != "package main\n\nfunc a() { println() }\nfunc b() {}\nfunc b() {}\n" {
		t.Fatalf("edit: %+v", result)
	}
	if call.Fields[2].Value != "3" || call.Fields[3].Value != "func a() {}" {
		t.Fatalf("fields: %+v", call.Fields)
	}
	for raw, want := range map[string]string{
		`{"path":"main.go","old_string":"func c() {}","new_string":"x"}`: "not found",
		`{"path":"main.go","old_string":"func b() {}","new_string":"x"}`: "more than once",
		`{"path":"gone.go","old_string":"a","new_string":"b"}`:           "does not exist",
	} {
		if _, result := run(t, files.Edit(), raw); result.Status != "invalid_arguments" || !strings.Contains(result.Output, want) {
			t.Errorf("%s: %+v", raw, result)
		}
	}
	edit := files.Edit().(*EditFile)
	call, _ = edit.Prepare(`{"path":"main.go","old_string":"package main","new_string":"package app"}`)
	if call, err := edit.Inspect(call); err != nil {
		t.Fatal(err)
	} else {
		writeTestFile(t, path, "package main\n// changed\n", 0600)
		if result := edit.Execute(context.Background(), call); result.Status != "failed" || readTestFile(t, path) != "package main\n// changed\n" {
			t.Fatalf("changed after approval: %+v", result)
		}
	}
}

func TestPreviewIsBounded(t *testing.T) {
	text := strings.Repeat("line\n", 100)
	shown := preview(text)
	if strings.Count(shown, "line") != previewLines || !strings.Contains(shown, "more bytes not shown") {
		t.Fatalf("preview: %q", shown)
	}
	if shown := preview(strings.Repeat("界", previewBytes)); len(shown) > previewBytes+64 || !strings.Contains(shown, "more bytes") {
		t.Fatalf("byte bound: %d", len(shown))
	}
}
