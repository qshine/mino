package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"
)

func runTerminal(ctx context.Context, input io.Reader, output, errorOutput io.Writer, respond func(context.Context, string) (string, error)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	fmt.Fprintln(output, "Miniagent - Chapter 01: Terminal Chat")
	fmt.Fprintln(output, "Each question is independent. Use /exit, Ctrl+D, or Ctrl+C to quit.")
	lines := scanLines(ctx, input)
	for {
		fmt.Fprint(output, "\nYou> ")
		select {
		case <-ctx.Done():
			fmt.Fprintln(output)
			return nil
		case line, ok := <-lines:
			if !ok {
				fmt.Fprintln(output)
				return nil
			}
			if line.err != nil {
				return fmt.Errorf("Failed to read input (each line must be smaller than 1 MiB): %w", line.err)
			}
			prompt := strings.TrimSpace(line.text)
			if prompt == "/exit" {
				fmt.Fprintln(output, "Goodbye.")
				return nil
			}
			if prompt == "" {
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			answer, err := respond(ctx, prompt)
			if ctx.Err() != nil {
				fmt.Fprintln(output)
				return nil
			}
			if err != nil {
				fmt.Fprintln(errorOutput, "Error: "+terminalText(err.Error()))
				continue
			}
			fmt.Fprintln(output, "\nAssistant> "+terminalText(answer))
		}
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
func terminalText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, text)
}
