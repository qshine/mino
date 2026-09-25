package gateway

import (
	"bytes"
	"context"
	"errors"
	"github.com/qshine/mino/internal/agent"
	"github.com/qshine/mino/internal/tools"
	"io"
	"strings"
	"testing"
)

func TestApprovalDefaultsAndExactDisplay(t *testing.T) {
	bash, _ := tools.NewBash(t.TempDir(), t.TempDir())
	for _, tc := range []struct {
		answer            string
		interactive, want bool
	}{{"y", true, true}, {"yes", true, true}, {"", true, false}, {"n", true, false}, {"y", false, false}} {
		var output bytes.Buffer
		lines := make(chan inputLine, 2)
		lines <- inputLine{text: tc.answer}
		lines <- inputLine{text: "next question"}
		close(lines)
		terminal := CLI{lines: lines, output: &output, interactive: tc.interactive}
		call, _ := bash.Prepare(`{"command":"printf '\u001b[2J'\n#\u202e"}`)
		approved, err := terminal.Confirm(context.Background(), agent.Confirmation{Kind: "tool", ToolName: "bash", Fields: call.Fields, Warning: call.Warning})
		if err != nil || approved != tc.want {
			t.Fatalf("answer=%q approved=%v err=%v", tc.answer, approved, err)
		}
		if strings.Contains(output.String(), "\x1b") || strings.Contains(output.String(), "\u202e") || !strings.Contains(output.String(), `\x1b`) {
			t.Fatalf("unsafe display=%q", output.String())
		}
		next := <-lines
		want := "next question"
		if !tc.interactive {
			want = tc.answer
		}
		if next.text != want {
			t.Fatal("approval consumed the wrong input")
		}
	}
}

func TestApprovalEOFAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		lines := make(chan inputLine)
		if !cancelled {
			close(lines)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if cancelled {
			cancel()
		}
		terminal := CLI{lines: lines, output: io.Discard, interactive: true}
		_, err := terminal.confirm(ctx, "Approve?")
		want := io.EOF
		if cancelled {
			want = context.Canceled
		}
		if !errors.Is(err, want) {
			t.Fatalf("err=%v", err)
		}
	}
}

func TestChatAndApprovalShareInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lines := scanLines(ctx, strings.NewReader("first\ny\nsecond\nn\n/exit\n"))
	var output bytes.Buffer
	terminal := CLI{lines: lines, output: &output, interactive: true}
	var prompts []string
	var approvals []bool
	err := runTerminalLines(ctx, lines, &output, io.Discard, func(ctx context.Context, prompt string, emit func(string) error) error {
		prompts = append(prompts, prompt)
		approved, err := terminal.confirm(ctx, "Approve?")
		approvals = append(approvals, approved)
		return err
	})
	if err != nil || strings.Join(prompts, ",") != "first,second" || len(approvals) != 2 || !approvals[0] || approvals[1] {
		t.Fatalf("prompts=%v approvals=%v err=%v", prompts, approvals, err)
	}
}

func TestClearRequiresExactSessionIDInInteractiveTerminal(t *testing.T) {
	const id = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, tc := range []struct {
		answer            string
		interactive, want bool
	}{
		{id, true, true}, {"yes", true, false}, {strings.ToUpper(id), true, false}, {"", true, false}, {id, false, false},
	} {
		var output bytes.Buffer
		lines := make(chan inputLine, 2)
		lines <- inputLine{text: tc.answer}
		lines <- inputLine{text: "next"}
		close(lines)
		cli := CLI{lines: lines, output: &output, interactive: tc.interactive}
		approved, err := cli.Confirm(context.Background(), agent.Confirmation{Kind: "clear", SessionID: id, Warning: "Clear saved history?"})
		if err != nil || approved != tc.want {
			t.Fatalf("answer=%q approved=%v err=%v", tc.answer, approved, err)
		}
		next := <-lines
		want := "next"
		if !tc.interactive {
			want = tc.answer
		}
		if next.text != want {
			t.Fatal("clear consumed the wrong input")
		}
		if !strings.Contains(output.String(), id) {
			t.Fatal("clear did not identify current session")
		}
	}
}
