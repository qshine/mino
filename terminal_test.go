package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestTerminalReadsQuestionsAndExits(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"command", " \n你好\n第二个问题\n/exit\n不应发送\n"},
		{"EOF", "你好\n第二个问题"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output, errorOutput bytes.Buffer
			var prompts []string
			respond := func(ctx context.Context, prompt string) (string, error) {
				prompts = append(prompts, prompt)
				return "中文回答", nil
			}
			err := runTerminal(context.Background(), strings.NewReader(tc.input), &output, &errorOutput, respond)
			if err != nil || errorOutput.Len() != 0 {
				t.Fatalf("error = %v, stderr = %q", err, errorOutput.String())
			}
			if strings.Join(prompts, "|") != "你好|第二个问题" || strings.Count(output.String(), "Assistant> 中文回答") != 2 {
				t.Fatalf("prompts = %q, output = %q", prompts, output.String())
			}
		})
	}
}

func TestTerminalContinuesAfterRequestError(t *testing.T) {
	var output, errorOutput bytes.Buffer
	respond := func(ctx context.Context, prompt string) (string, error) {
		if prompt == "第一次" {
			return "", errors.New("HTTP 429")
		}
		return "恢复正常", nil
	}
	err := runTerminal(context.Background(), strings.NewReader("第一次\n第二次\n/exit\n"), &output, &errorOutput, respond)
	if err != nil || !strings.Contains(errorOutput.String(), "HTTP 429") || !strings.Contains(output.String(), "恢复正常") {
		t.Fatalf("error = %v, output = %q, stderr = %q", err, output.String(), errorOutput.String())
	}
}

func TestTerminalCancellation(t *testing.T) {
	for _, waitingForResponse := range []bool{false, true} {
		t.Run(map[bool]string{false: "waiting for input", true: "waiting for response"}[waitingForResponse], func(t *testing.T) {
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := make(chan struct{})
			done := make(chan error, 1)
			respond := func(ctx context.Context, prompt string) (string, error) {
				close(started)
				<-ctx.Done()
				return "", ctx.Err()
			}
			go func() { done <- runTerminal(ctx, reader, io.Discard, io.Discard, respond) }()
			if waitingForResponse {
				if _, err := io.WriteString(writer, "你好\n"); err != nil {
					t.Fatal(err)
				}
				<-started
			}
			cancel()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("terminal did not stop after cancellation")
			}
		})
	}
}

func TestTerminalRejectsOversizedInput(t *testing.T) {
	err := runTerminal(context.Background(), strings.NewReader(strings.Repeat("a", 1<<20)), io.Discard, io.Discard,
		func(context.Context, string) (string, error) { t.Fatal("oversized input was sent"); return "", nil })
	if err == nil || !strings.Contains(err.Error(), "read input") {
		t.Fatalf("input error = %v", err)
	}
}

func TestTerminalTreatsControlSequencesAsText(t *testing.T) {
	var output bytes.Buffer
	err := runTerminal(context.Background(), strings.NewReader("你好\n/exit\n"), &output, io.Discard,
		func(context.Context, string) (string, error) { return "你好\x1b[2J\x00\r\n世界\t！", nil })
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(output.String(), "\x1b\x00\r") || !strings.Contains(output.String(), "世界\t！") {
		t.Fatalf("unsafe terminal output: %q", output.String())
	}
}
