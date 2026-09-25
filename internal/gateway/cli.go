package gateway

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	assets "github.com/qshine/mino"
)

// Command handles version, help, and update; false selects interactive chat.
func Command(ctx context.Context, version string, args []string, output, errorOutput io.Writer) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if args[0] == "update" && len(args) <= 2 {
		updateCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		commandArgs := append([]string{"-s", "--"}, args[1:]...)
		command := exec.CommandContext(updateCtx, "/bin/bash", commandArgs...)
		command.Stdin = strings.NewReader(assets.InstallerScript)
		command.Stdout = output
		command.Stderr = errorOutput
		if err := command.Run(); err != nil {
			return true, fmt.Errorf("Update failed: %w", err)
		}
		return true, nil
	}
	if len(args) == 1 {
		switch args[0] {
		case "version", "--version", "-v":
			_, err := fmt.Fprintln(output, "mino "+version)
			return true, err
		case "help", "--help", "-h":
			_, err := fmt.Fprintln(output, "Usage: mino [version | update [VERSION] | help]\n\nRun without arguments to start a terminal chat.\nSettings: ~/.mino/config.json"+"\nHistory: ~/.mino/history.jsonl")
			return true, err
		}
	}
	return true, fmt.Errorf("Unknown command or unexpected arguments. Run mino help for usage.")
}
