package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatParseRoundTrip(t *testing.T) {
	for _, m := range []Memory{
		{Name: "prefers-chinese", Type: "feedback", Summary: "始终用中文回答；代码和命令保持英文"},
		{Name: "release-process", Type: "reference", Summary: "Run check.sh: then tag chapter-NN", Details: "1. Build.\n2. Tag.\n"},
		{Name: "yes", Type: "user", Summary: "true"},
	} {
		text := Format(m)
		got, err := Parse(text)
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		want := m
		want.Details = strings.TrimSpace(m.Details)
		if got != want {
			t.Fatalf("got %+v, want %+v from %q", got, want, text)
		}
	}
}

func TestParseRejectsInvalidFiles(t *testing.T) {
	for name, text := range map[string]string{
		"no front matter":  "name: a\n",
		"unclosed":         "---\nname: a\ntype: user\nsummary: s\n",
		"missing field":    "---\nname: a\ntype: user\n---\n",
		"extra field":      "---\nname: a\ntype: user\nsummary: s\nrole: admin\n---\n",
		"non-string":       "---\nname: a\ntype: user\nsummary: 12\n---\n",
		"unknown type":     "---\nname: a\ntype: system\nsummary: s\n---\n",
		"multi-line":       "---\nname: a\ntype: user\nsummary: \"one\\ntwo\"\n---\n",
		"long summary":     "---\nname: a\ntype: user\nsummary: " + strings.Repeat("x", MaxSummaryBytes+1) + "\n---\n",
		"invalid name":     "---\nname: A_b\ntype: user\nsummary: s\n---\n",
		"duplicate field":  "---\nname: a\nname: b\ntype: user\nsummary: s\n---\n",
		"oversize details": "---\nname: a\ntype: user\nsummary: s\n---\n" + strings.Repeat("x", MaxDetailsBytes+1),
	} {
		if _, err := Parse(text); err == nil {
			t.Errorf("%s: accepted %q", name, text)
		}
	}
}

func TestRenderMarksDetailsAndIsEmptyWithoutEntries(t *testing.T) {
	if Render(nil) != "" {
		t.Fatal("an empty index must render nothing")
	}
	text := Render([]Entry{{"a", "user", "short", false}, {"b", "reference", "long", true}})
	if !strings.HasPrefix(text, Label) || !strings.Contains(text, "- [user] a: short\n") || !strings.Contains(text, "- [reference] b: long (+details)\n") {
		t.Fatalf("render = %q", text)
	}
}

func write(t *testing.T, dir, file, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestScanSortsEntriesAndSkipsUnusableFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "memory")
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("memory directory mode = %v, %v", info, err)
	}
	write(t, dir, "zeta.md", Format(Memory{Name: "zeta", Type: "project", Summary: "last", Details: "more"}))
	write(t, dir, "alpha.md", Format(Memory{Name: "alpha", Type: "user", Summary: "first"}))
	write(t, dir, "wrong.md", Format(Memory{Name: "other", Type: "user", Summary: "mismatch"}))
	write(t, dir, "broken.md", "not a memory\n")
	write(t, dir, ".alpha.md.mino-1.tmp", "ignored temporary file")
	write(t, dir, "notes.txt", "ignored")
	os.WriteFile(filepath.Join(t.TempDir(), "target.md"), []byte(Format(Memory{Name: "link", Type: "user", Summary: "outside"})), 0600)
	if err := os.Symlink(filepath.Join(t.TempDir(), "target.md"), filepath.Join(dir, "link.md")); err != nil {
		t.Fatal(err)
	}
	entries, problems, err := store.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name != "alpha" || entries[1].Name != "zeta" || !entries[1].Details || entries[0].Details {
		t.Fatalf("entries = %+v", entries)
	}
	joined := strings.Join(problems, "\n")
	for _, file := range []string{"broken.md", "link.md", "wrong.md"} {
		if !strings.Contains(joined, file) {
			t.Fatalf("problems %q do not mention %s", joined, file)
		}
	}
	if len(problems) != 3 {
		t.Fatalf("problems = %q", problems)
	}
	if text, err := store.Read("zeta"); err != nil || !strings.Contains(text, "more") {
		t.Fatalf("read = %q, %v", text, err)
	}
	if _, err := store.Read("missing"); err == nil || !strings.Contains(err.Error(), "No memory") {
		t.Fatalf("missing read error = %v", err)
	}
	if _, err := store.Read("link"); err == nil {
		t.Fatal("read followed a symbolic link")
	}
}

func TestIndexLimits(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "memory")
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := range MaxEntries {
		name := fmt.Sprintf("m%03d", i)
		write(t, dir, name+".md", Format(Memory{Name: name, Type: "user", Summary: "s"}))
	}
	if err := store.Fits(Memory{Name: "m000", Type: "user", Summary: "replaced"}); err != nil {
		t.Fatalf("replacing an entry in a full index: %v", err)
	}
	if err := store.Fits(Memory{Name: "extra", Type: "user", Summary: "s"}); err == nil {
		t.Fatal("a full index accepted another memory")
	}
	write(t, dir, "zz.md", Format(Memory{Name: "zz", Type: "user", Summary: "s"}))
	entries, problems, err := store.Scan()
	if err != nil || len(entries) != MaxEntries || len(problems) != 1 || !strings.Contains(problems[0], "zz.md") {
		t.Fatalf("entries=%d problems=%q err=%v", len(entries), problems, err)
	}

	// The byte limit applies before the count limit with long summaries.
	big := filepath.Join(t.TempDir(), "memory")
	store, _ = Open(big)
	for i := range 100 {
		name := fmt.Sprintf("b%d", i)
		write(t, big, name+".md", Format(Memory{Name: name, Type: "user", Summary: strings.Repeat("x", MaxSummaryBytes)}))
	}
	entries, problems, _ = store.Scan()
	if len(Render(entries)) > MaxIndexBytes || len(problems) == 0 {
		t.Fatalf("index of %d bytes with %d problems", len(Render(entries)), len(problems))
	}
}

func TestOpenRefusesSymbolicLink(t *testing.T) {
	base := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(base, "memory")); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(filepath.Join(base, "memory")); err == nil {
		t.Fatal("Open accepted a symbolic link")
	}
	if _, err := Open("relative"); err == nil {
		t.Fatal("Open accepted a relative directory")
	}
}
