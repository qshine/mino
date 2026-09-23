package main

import (
	"context"
	"fmt"
	"io"
)

// Release builds set this from the Git tag with -ldflags; source builds say dev.
var version = "dev"

func runCLI(ctx context.Context, args []string, input io.Reader, output, errorOutput io.Writer) error {
	if len(args) == 0 {
		return run(ctx, input, output, errorOutput)
	}
	if len(args) == 1 {
		switch args[0] {
		case "version", "--version", "-v":
			_, err := fmt.Fprintln(output, "mino "+version)
			return err
		case "help", "--help", "-h":
			_, err := fmt.Fprintln(output, "Usage: mino [version | help]\n\nRun without arguments to start a terminal chat.\nSettings: "+configLocation)
			return err
		}
	}
	return fmt.Errorf("Unknown command or unexpected arguments. Run mino help for usage.")
}
