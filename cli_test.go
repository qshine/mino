package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInformationDoesNotRequireSetup(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v", "help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			path := isolateConfig(t)
			var output bytes.Buffer
			if err := runCLI(context.Background(), []string{arg}, strings.NewReader(""), &output, io.Discard); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "mino") {
				t.Fatal("missing CLI information")
			}
			if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
				t.Fatal("informational command created user configuration")
			}
		})
	}
}

func TestCLIVersionMatchesBuildVersion(t *testing.T) {
	previous := version
	version = "0.1.1"
	t.Cleanup(func() { version = previous })
	var output bytes.Buffer
	if err := runCLI(context.Background(), []string{"version"}, nil, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	if output.String() != "mino 0.1.1\n" {
		t.Fatalf("version output = %q", output.String())
	}
}

func TestCLIRejectsUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"version", "extra"}, {"help", "extra"}} {
		if err := runCLI(context.Background(), args, nil, io.Discard, io.Discard); err == nil {
			t.Fatalf("accepted arguments %q", args)
		}
	}
}
