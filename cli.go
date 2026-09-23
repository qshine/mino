package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// Release builds set this from the Git tag with -ldflags; source builds say dev.
var version = "dev"

//go:embed install.sh
var installScript string

func runCLI(ctx context.Context, args []string, input io.Reader, output, errorOutput io.Writer) error {
	if len(args) == 0 {
		return run(ctx, input, output, errorOutput)
	}
	if args[0] == "update" && len(args) <= 2 {
		updateCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		commandArgs := append([]string{"-s", "--"}, args[1:]...)
		command := exec.CommandContext(updateCtx, "/bin/bash", commandArgs...)
		command.Stdin = strings.NewReader(installScript)
		command.Stdout = output
		command.Stderr = errorOutput
		if err := command.Run(); err != nil {
			return fmt.Errorf("Update failed: %w", err)
		}
		return nil
	}
	if len(args) == 1 {
		switch args[0] {
		case "version", "--version", "-v":
			_, err := fmt.Fprintln(output, "mino "+version)
			return err
		case "help", "--help", "-h":
			_, err := fmt.Fprintln(output, "Usage: mino [version | update [VERSION] | help]\n\nRun without arguments to start a terminal chat.\nSettings: "+configLocation)
			return err
		}
	}
	return fmt.Errorf("Unknown command or unexpected arguments. Run mino help for usage.")
}
