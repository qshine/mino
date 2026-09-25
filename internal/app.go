package mino

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/qshine/mino/internal/agent"
	"github.com/qshine/mino/internal/gateway"
	"io"
	"os"
	"os/signal"
)

// Main runs the terminal application and returns its process exit code.
func Main(version string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := runCLI(ctx, version, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: "+gateway.Text(err.Error()))
		return 1
	}
	return 0
}

func run(ctx context.Context, input io.Reader, output, errorOutput io.Writer) (err error) {
	// 配置与聊天共用缓冲区，避免首次配置吞掉已经读入的第一条问题。
	reader := bufio.NewReader(input)
	setup := false
	cfg, err := loadConfig(func(label, fallback string, secret bool) (string, error) {
		file, ok := input.(*os.File)
		if !ok {
			return "", fmt.Errorf("Config is incomplete. Start Mino in a terminal to finish setup.")
		}
		if !setup {
			fmt.Fprintln(output, "Complete the missing settings. Press Enter to accept a value in brackets, or Ctrl+C to cancel. Settings will be saved to "+configLocation+".")
			setup = true
		}
		var validate func(string) error
		if label == "API URL" {
			validate = validateBaseURL
		}
		return gateway.PromptConfigValue(ctx, file, reader, output, label, fallback, secret, validate)
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
	instructions, err := loadInstructions()
	if err != nil {
		return err
	}
	directory, err := userDirectory()
	if err != nil {
		return err
	}
	history, err := agent.OpenSession(directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, history.Close()) }()
	if history.Notice() != "" {
		if _, err := fmt.Fprint(errorOutput, gateway.Text(history.Notice())); err != nil {
			return err
		}
	}
	chat := agent.New(agent.Options{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model}, instructions, history)
	return gateway.Run(ctx, reader, output, errorOutput, chat.Handle)
}

func runCLI(ctx context.Context, version string, args []string, input io.Reader, output, errorOutput io.Writer) error {
	if handled, err := gateway.Command(ctx, version, args, output, errorOutput); handled {
		return err
	}
	return run(ctx, input, output, errorOutput)
}
