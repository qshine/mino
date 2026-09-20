package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Stdin, os.Stdout, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: "+terminalText(err.Error()))
		os.Exit(1)
	}
}

func run(ctx context.Context, input io.Reader, output, errorOutput io.Writer) error {
	instructions, err := loadInstructions()
	if err != nil {
		return err
	}
	// 配置与聊天共用缓冲区，避免首次配置吞掉已经读入的第一条问题。
	reader := bufio.NewReader(input)
	setup := false
	cfg, err := loadConfig(func(label, fallback string, secret bool) (string, error) {
		file, ok := input.(*os.File)
		if !ok {
			return "", fmt.Errorf("Config is incomplete. Run go run . in a terminal to finish setup.")
		}
		if !setup {
			fmt.Fprintln(output, "Complete the missing settings. Press Enter to accept a value in brackets, or Ctrl+C to cancel. Settings will be saved to "+configLocation+".")
			setup = true
		}
		return promptConfigValue(ctx, file, reader, output, label, fallback, secret)
	})
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
		fmt.Fprintln(output, "\nSetup cancelled.")
		return nil
	}
	if err != nil {
		return err
	}
	if setup {
		fmt.Fprintln(output, "Settings saved. Next time, chat will start immediately.")
	}
	client := newResponsesClient(cfg, instructions)
	return runTerminal(ctx, reader, output, errorOutput, client.respond)
}
