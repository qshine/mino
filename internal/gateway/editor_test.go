package gateway

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"
)

func typeKeys(e *editor, keys string) (lines []string, eof bool) {
	for _, r := range keys {
		line, done, end := e.key(r)
		if end {
			return lines, true
		}
		if done {
			lines = append(lines, line)
		}
	}
	return lines, false
}

func TestEditorErasesWideCharactersByWidth(t *testing.T) {
	var out bytes.Buffer
	e := newEditor(&out, func() int { return 80 })
	e.prompt("\nYou> ")
	out.Reset()
	typeKeys(e, "我试\x7f")
	// 试 began at column 7, after the five-column prompt and the two-column 我.
	if got := out.String(); got != "我试\r\x1b[7C\x1b[J" {
		t.Fatalf("output = %q", got)
	}
	if lines, _ := typeKeys(e, "们\n"); len(lines) != 1 || lines[0] != "我们" {
		t.Fatalf("lines = %q", lines)
	}
}

func TestEditorErasesAcrossWrappedRows(t *testing.T) {
	var out bytes.Buffer
	e := newEditor(&out, func() int { return 10 })
	e.prompt("You> ")
	typeKeys(e, "我我我")
	// The third 我 does not fit in column 9, so it starts the next row.
	if e.starts[2] != (cell{1, 0}) || e.at != (cell{1, 2}) {
		t.Fatalf("starts = %v, cursor = %v", e.starts, e.at)
	}
	out.Reset()
	typeKeys(e, "\x7f\x7f")
	if got := out.String(); got != "\r\x1b[J\x1b[1A\r\x1b[7C\x1b[J" {
		t.Fatalf("output = %q", got)
	}
	if lines, _ := typeKeys(e, "\n"); lines[0] != "我" {
		t.Fatalf("lines = %q", lines)
	}
}

func TestEditorLineKeys(t *testing.T) {
	var out bytes.Buffer
	e := newEditor(&out, func() int { return 0 })
	e.prompt("> ")
	lines, eof := typeKeys(e, "hello wörld 中文\x17\x17x\n丢掉\x15ok\x04\n\x04")
	if !eof || strings.Join(lines, "|") != "hello x|ok" {
		t.Fatalf("lines = %q, eof = %v", lines, eof)
	}
	// Other control characters are ignored and never echoed.
	lines, _ = typeKeys(e, "a\x01\x07\tb\n")
	if lines[0] != "a\tb" || strings.ContainsAny(out.String(), "\x01\x07") {
		t.Fatalf("line = %q, output = %q", lines[0], out.String())
	}
}

func TestEditorShowsTypeAheadAfterPrompt(t *testing.T) {
	var out bytes.Buffer
	e := newEditor(&out, func() int { return 80 })
	typeKeys(e, "下一")
	out.Reset()
	e.prompt("\nYou> ")
	if out.String() != "\nYou> 下一" || e.starts[1] != (cell{0, 7}) {
		t.Fatalf("output = %q, starts = %v", out.String(), e.starts)
	}
}

func TestEditLinesDropsEscapeSequences(t *testing.T) {
	var out bytes.Buffer
	e := newEditor(&out, func() int { return 80 })
	input := bufio.NewReader(strings.NewReader("ab\x1b[D\x1bOA\x7fc\nlast\n\x04ignored\n"))
	var got []string
	for line := range editLines(context.Background(), input, e) {
		if line.err != nil {
			t.Fatal(line.err)
		}
		got = append(got, line.text)
	}
	if strings.Join(got, "|") != "ac|last" || strings.Contains(out.String(), "[D") {
		t.Fatalf("lines = %q, output = %q", got, out.String())
	}
}

func TestRuneWidth(t *testing.T) {
	for text, want := range map[string]int{"abc": 3, "中文": 4, "，。": 4, "ｱ": 1, "Ａ": 2, "é": 1, "é": 1, "한": 2, "🙂": 2} {
		if got := displayWidth(text); got != want {
			t.Errorf("displayWidth(%q) = %d, want %d", text, got, want)
		}
	}
}
