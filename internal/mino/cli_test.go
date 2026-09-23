package mino

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
			if err := runCLI(context.Background(), "dev", []string{arg}, strings.NewReader(""), &output, io.Discard); err != nil {
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
	var output bytes.Buffer
	if err := runCLI(context.Background(), "0.1.0", []string{"version"}, nil, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	if output.String() != "mino 0.1.0\n" {
		t.Fatalf("version output = %q", output.String())
	}
}

func TestCLIRejectsUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"version", "extra"}, {"help", "extra"}, {"update", "v0.1.0", "extra"}} {
		if err := runCLI(context.Background(), "dev", args, nil, io.Discard, io.Discard); err == nil {
			t.Fatalf("accepted arguments %q", args)
		}
	}
}

func TestCLIUpdateRunsOutsideProjectWithoutChangingSettings(t *testing.T) {
	_, home, _ := installerFixture(t)
	path := filepath.Join(home, ".mino", "config.json")
	writeTestConfig(t, path, `{"api_key":"keep-settings"}`)
	t.Chdir(t.TempDir())
	var output bytes.Buffer
	if err := runCLI(context.Background(), "dev", []string{"update", "v0.1.0"}, nil, &output, &output); err != nil {
		t.Fatalf("update: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "Installed mino 0.1.0") {
		t.Fatal("CLI did not use the release installer")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != `{"api_key":"keep-settings"}` {
		t.Fatal("update changed settings")
	}
}
