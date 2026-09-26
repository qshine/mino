// Package gateway adapts terminal input and output to the Agent contract.
package gateway

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"

	assets "github.com/qshine/mino"
	"github.com/qshine/mino/internal/agent"
)

var errTerminalOutput = errors.New("Failed to write terminal output")

type CLI struct {
	input               io.Reader
	reader              *bufio.Reader
	output, errorOutput io.Writer
	lines               <-chan inputLine
	interactive         bool
}

func NewCLI(input io.Reader, output, errorOutput io.Writer) *CLI {
	c := &CLI{input: input, reader: bufio.NewReader(input), output: output, errorOutput: errorOutput}
	if file, ok := input.(*os.File); ok {
		_, err := stty(file, "-g")
		c.interactive = err == nil
	}
	return c
}

// Command handles non-chat commands before the application opens configuration or history.
func (c *CLI) Command(ctx context.Context, version string, args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if args[0] == "update" && len(args) <= 2 {
		updateCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		command := exec.CommandContext(updateCtx, "/bin/bash", append([]string{"-s", "--"}, args[1:]...)...)
		command.Stdin = strings.NewReader(assets.InstallerScript)
		command.Stdout, command.Stderr = c.output, c.errorOutput
		if err := command.Run(); err != nil {
			return true, fmt.Errorf("Update failed: %w", err)
		}
		return true, nil
	}
	if len(args) == 1 {
		switch args[0] {
		case "version", "--version", "-v":
			_, err := fmt.Fprintln(c.output, "mino "+version)
			return true, err
		case "help", "--help", "-h":
			_, err := fmt.Fprintln(c.output, "Usage: mino [version | update [VERSION] | help]\n\nRun without arguments to start a terminal chat.\nSettings: ~/.mino/config.json\nSessions: ~/.mino/sessions/<id>.jsonl\nChat commands: /new, /sessions, /resume <id>, /clear, /compact, /help, /exit")
			return true, err
		}
	}
	return true, errors.New("Unknown command or unexpected arguments. Run mino help for usage.")
}

// Run shares one input stream between chat and confirmations. Agent sees neither
// the terminal nor the stream, so another gateway can implement Interaction itself.
func (c *CLI) Run(ctx context.Context, handler agent.Handler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.lines = scanLines(ctx, c.reader)
	return c.runLines(ctx, handler)
}

func (c *CLI) runLines(ctx context.Context, handler agent.Handler) error {
	if err := handler.Start(ctx, c); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	fmt.Fprintln(c.output, "Mino - Chapter 05: Context Compaction")
	fmt.Fprintln(c.output, "The last session is restored on startup. Context compacts automatically near its budget; use /compact to compact manually or /help for commands. Bash commands require approval. Use /exit, Ctrl+D, or Ctrl+C to quit.")
	for {
		fmt.Fprint(c.output, "\nYou> ")
		select {
		case <-ctx.Done():
			fmt.Fprintln(c.output)
			return nil
		case line, ok := <-c.lines:
			if !ok {
				fmt.Fprintln(c.output)
				return nil
			}
			if line.err != nil {
				return fmt.Errorf("Failed to read input (each line must be smaller than 1 MiB): %w", line.err)
			}
			message := strings.TrimSpace(line.text)
			if message == "/exit" {
				fmt.Fprintln(c.output, "Goodbye.")
				return nil
			}
			if message == "" {
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			err := handler.Handle(ctx, message, c)
			fmt.Fprintln(c.output)
			var storage *agent.StorageError
			if errors.As(err, &storage) || errors.Is(err, errTerminalOutput) {
				return err
			}
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				return nil
			}
			if err != nil {
				fmt.Fprintln(c.errorOutput, "Error: "+Text(err.Error()))
			}
		}
	}
}

func (c *CLI) Emit(event agent.Event) error {
	var err error
	switch event.Kind {
	case "response_started":
		_, err = fmt.Fprint(c.output, "\nAssistant> ")
	case "text":
		_, err = fmt.Fprint(c.output, Text(event.Text))
	case "info":
		_, err = fmt.Fprint(c.output, Text(event.Text))
	case "notice":
		_, err = fmt.Fprint(c.errorOutput, Text(event.Text))
	case "tool_result":
		result := event.Result
		code, truncated := "", ""
		if result.ExitCode != nil {
			code = fmt.Sprintf("; exit code %d", *result.ExitCode)
		}
		if result.Truncated {
			truncated = "; output truncated"
		}
		_, err = fmt.Fprintf(c.output, "\n[Tool result: %s%s%s]\n%s\n", Text(result.Status), code, truncated, Text(result.Output))
	default:
		return errors.New("Unknown Agent event")
	}
	if err != nil {
		return errTerminalOutput
	}
	return nil
}

func (c *CLI) Confirm(ctx context.Context, request agent.Confirmation) (bool, error) {
	if request.Kind == "clear" {
		if _, err := fmt.Fprintf(c.output, "\nSession: %s\n%s\n", Text(request.SessionID), Text(request.Warning)); err != nil {
			return false, errTerminalOutput
		}
		if !c.interactive {
			if _, err := fmt.Fprintln(c.output, "Denied: clearing a session requires an interactive terminal."); err != nil {
				return false, errTerminalOutput
			}
			return false, nil
		}
		return c.confirmAnswer(ctx, "Type the complete session ID to clear it: ", request.SessionID)
	}
	prompt := "Approve this operation? [y/N] "
	if request.Kind == "recovery" {
		if _, err := fmt.Fprintln(c.output, Text(request.Warning)); err != nil {
			return false, errTerminalOutput
		}
		if !c.interactive {
			return false, errors.New("Unknown tool results require acknowledgement in an interactive terminal")
		}
		prompt = "Do you understand that these commands may already have run? [y/N] "
	} else {
		if _, err := fmt.Fprintf(c.output, "\n[Tool: %s]\n", Text(request.ToolName)); err != nil {
			return false, errTerminalOutput
		}
		for _, field := range request.Fields {
			if _, err := fmt.Fprintf(c.output, "%s: %s\n", Text(field.Label), strconv.QuoteToASCII(field.Value)); err != nil {
				return false, errTerminalOutput
			}
		}
		if _, err := fmt.Fprintln(c.output, Text(request.Warning)); err != nil {
			return false, errTerminalOutput
		}
		if !c.interactive {
			if _, err := fmt.Fprintln(c.output, "Denied: command approval requires an interactive terminal."); err != nil {
				return false, errTerminalOutput
			}
			return false, nil
		}
	}
	return c.confirm(ctx, prompt)
}

func (c *CLI) confirm(ctx context.Context, prompt string) (bool, error) {
	return c.confirmAnswer(ctx, prompt, "")
}

func (c *CLI) confirmAnswer(ctx context.Context, prompt, expected string) (bool, error) {
	if !c.interactive {
		return false, nil
	}
	if _, err := fmt.Fprint(c.output, prompt); err != nil {
		return false, errTerminalOutput
	}
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case line, ok := <-c.lines:
		if !ok {
			return false, io.EOF
		}
		if line.err != nil {
			return false, fmt.Errorf("Failed to read approval: %w", line.err)
		}
		answer := strings.TrimSpace(line.text)
		if answer == "/exit" {
			return false, io.EOF
		}
		if expected != "" {
			return answer == expected, nil
		}
		answer = strings.ToLower(answer)
		return answer == "y" || answer == "yes", nil
	}
}

type inputLine struct {
	text string
	err  error
}

// 扫描放在一个 goroutine 中，让主循环等待终端输入时也能响应 Ctrl+C。
// stdin 的阻塞读取由进程退出结束；调用者传入其他流时负责关闭该流。
func scanLines(ctx context.Context, input io.Reader) <-chan inputLine {
	lines := make(chan inputLine)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(input)
		scanner.Buffer(make([]byte, 4096), 1<<20)
		for scanner.Scan() {
			select {
			case lines <- inputLine{text: scanner.Text()}:
			case <-ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case lines <- inputLine{err: err}:
			case <-ctx.Done():
			}
		}
	}()
	return lines
}

// 模型输出是文本，不能让控制字符在终端执行清屏、改标题等操作。
func Text(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) || (unicode.IsControl(r) && r != '\n' && r != '\t') {
			return -1
		}
		return r
	}, text)
}
