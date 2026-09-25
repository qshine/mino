package mino

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/qshine/mino/internal/agent"
	"github.com/qshine/mino/internal/gateway"
	"github.com/qshine/mino/internal/tools"
)

// Main is the composition root: configuration, Session, tools, Agent, then CLI.
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

func runCLI(ctx context.Context, version string, args []string, input io.Reader, output, errorOutput io.Writer) error {
	cli := gateway.NewCLI(input, output, errorOutput)
	if handled, err := cli.Command(ctx, version, args); handled {
		return err
	}
	return runChat(ctx, cli, output)
}

func run(ctx context.Context, input io.Reader, output, errorOutput io.Writer) error {
	return runChat(ctx, gateway.NewCLI(input, output, errorOutput), output)
}

func runChat(ctx context.Context, cli *gateway.CLI, output io.Writer) (err error) {
	setup := false
	cfg, err := loadConfig(func(label, fallback string, secret bool) (string, error) {
		if !setup {
			fmt.Fprintln(output, "Complete the missing settings. Press Enter to accept a value in brackets, or Ctrl+C to cancel. Settings will be saved to "+configLocation+".")
			setup = true
		}
		var validate func(string) error
		if label == "API URL" {
			validate = validateBaseURL
		}
		return cli.PromptConfig(ctx, label, fallback, secret, validate)
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
	cwd, err := os.Getwd()
	if err != nil {
		return errors.New("Cannot determine the Bash working directory")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.New("Cannot determine the user home directory")
	}
	bash, err := tools.NewBash(cwd, home)
	if err != nil {
		return err
	}
	available := []tools.Tool{bash}
	session, err := agent.OpenSessions(directory, available)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, session.Close()) }()
	runner, err := agent.NewSessionManager(agent.Options{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model, Instructions: instructions}, session, available)
	if err != nil {
		return err
	}
	return cli.Run(ctx, runner)
}
