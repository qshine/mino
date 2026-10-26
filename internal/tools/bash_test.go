package tools

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/qshine/mino/internal/sandbox"
)

func TestParseBashArguments(t *testing.T) {
	for _, raw := range []string{`{}`, `null`, `[]`, `{"command":""}`, `{"command":12}`, `{"command":"ok","cwd":"/"}`, `{"command":"ok","command":"bad"}`, `{"command":"ok"} {}`, `{"command":"\u0000"}`} {
		if _, err := ParseBashArguments(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	a, err := ParseBashArguments(`{"command":"printf 'hello\\n'"}`)
	if err != nil || a.Command != "printf 'hello\\n'" {
		t.Fatalf("args=%#v error=%v", a, err)
	}
}

func testBash(t *testing.T) *Bash {
	t.Helper()
	b, err := NewBash(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBashOutputExitAndEnvironment(t *testing.T) {
	b := testBash(t)
	injected := filepath.Join(t.TempDir(), "inject.sh")
	if err := os.WriteFile(injected, []byte("printf injected"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BASH_ENV", injected)
	t.Setenv("OPENAI_API_KEY", "must-not-inherit")
	r := b.execute(context.Background(), BashArgs{Command: `printf 'out'; printf 'err' >&2; printf '%s' "$OPENAI_API_KEY"; pwd; exit 7`}, false, BashTimeout, BashOutputLimit)
	if r.Status != "failed" || r.ExitCode == nil || *r.ExitCode != 7 || !strings.Contains(r.Output, "out") || !strings.Contains(r.Output, "err") || !strings.Contains(r.Output, b.Directory()) || strings.Contains(r.Output, "injected") || strings.Contains(r.Output, "must-not-inherit") {
		t.Fatalf("result=%#v", r)
	}
	r = b.execute(context.Background(), BashArgs{Command: "command -v sh"}, false, BashTimeout, BashOutputLimit)
	if r.Status != "completed" || !strings.Contains(r.Output, "/sh") {
		t.Fatalf("System commands unavailable in documented PATH: %#v", r)
	}
}

func TestBashLimitsAndCancellation(t *testing.T) {
	for _, mode := range []string{"timeout", "overflow", "cancel", "background"} {
		t.Run(mode, func(t *testing.T) {
			b := testBash(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			command, want := "sleep 10", "timed_out"
			switch mode {
			case "overflow":
				command, want = "while :; do printf '1234567890'; printf 'abcdefghij' >&2; done", "output_limit"
			case "cancel":
				cancel()
				want = "cancelled"
			case "background":
				command, want = "sleep 10 & echo $! > child.pid", "failed"
			}
			start := time.Now()
			r := b.execute(ctx, BashArgs{Command: command}, false, 700*time.Millisecond, 128)
			if r.Status != want || len(r.Output) > 128 || (mode == "overflow" && !r.Truncated) || time.Since(start) > 3*time.Second {
				t.Fatalf("status=%s bytes=%d truncated=%v elapsed=%v", r.Status, len(r.Output), r.Truncated, time.Since(start))
			}
			if mode == "background" {
				data, err := os.ReadFile(filepath.Join(b.Directory(), "child.pid"))
				if err != nil {
					t.Fatal(err)
				}
				pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
				deadline := time.Now().Add(time.Second)
				for syscall.Kill(pid, 0) == nil && time.Now().Before(deadline) {
					time.Sleep(10 * time.Millisecond)
				}
				if syscall.Kill(pid, 0) == nil {
					t.Fatalf("child %d survived", pid)
				}
			}
		})
	}
}

func TestBashSandboxArgumentsAndConfinement(t *testing.T) {
	plain := testBash(t)
	call, err := plain.Prepare(`{"command":"true","sandbox":false}`)
	if err != nil || call.Sandboxed || !strings.Contains(fieldValue(call, "Sandbox"), "unavailable") || string(call.Arguments) != `{"command":"true","sandbox":false}` {
		t.Fatalf("without a sandbox: %+v %v", call, err)
	}
	for _, raw := range []string{`{"command":"x","sandbox":"yes"}`, `{"command":"x","sandbox":1}`, `{"command":"x","sandbox":false,"sandbox":true}`, `{"sandbox":true}`} {
		if _, err := ParseBashArguments(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if !sandbox.Available() {
		t.Skip("sandbox-exec is unavailable on this system")
	}
	b := testBash(t)
	confined, err := sandbox.New(b.Directory(), filepath.Dir(b.Directory()))
	if err != nil {
		t.Fatal(err)
	}
	b.SetSandbox(confined)
	if !strings.Contains(b.Definition().Description, "OS sandbox") {
		t.Fatal("definition does not describe the sandbox")
	}
	for _, raw := range []string{`{"command":"cat .env"}`, `{"command":"cat .env","sandbox":null}`, `{"command":"cat .env","sandbox":true}`} {
		call, err := b.Prepare(raw)
		if err != nil || !call.Sandboxed || string(call.Arguments) != `{"command":"cat .env"}` || !strings.HasPrefix(fieldValue(call, "Sandbox"), "on: writes only inside "+b.Directory()) || strings.Contains(call.Warning, "not a sandbox") {
			t.Fatalf("%s: %+v %v", raw, call, err)
		}
	}
	call, err = b.Prepare(`{"command":"cat .env","sandbox":false}`)
	if err != nil || call.Sandboxed || !strings.HasPrefix(fieldValue(call, "Sandbox"), "off") || !strings.Contains(call.Warning, "not a sandbox") || string(call.Arguments) != `{"command":"cat .env","sandbox":false}` {
		t.Fatalf("opt-out: %+v %v", call, err)
	}
	if err := os.WriteFile(filepath.Join(b.Directory(), ".env"), []byte("TOKEN=secret-value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// Temporary directories stay writable, so the escape attempt targets the home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	escaped := filepath.Join(home, "mino-bash-sandbox-escape.txt")
	defer os.Remove(escaped)
	script := `cat .env; echo "cat=$?"; echo x > "` + escaped + `"; echo "escape=$?"; echo y > inside.txt; echo "inside=$?"`
	call, _ = b.Prepare(`{"command":` + strconv.Quote(script) + `}`)
	r := b.Execute(context.Background(), call)
	if r.Status != "completed" || strings.Contains(r.Output, "secret-value") || !strings.Contains(r.Output, "cat=1") || !strings.Contains(r.Output, "escape=1") || !strings.Contains(r.Output, "inside=0") || !strings.Contains(r.Output, "Operation not permitted") {
		t.Fatalf("sandboxed result=%#v", r)
	}
	if _, err := os.Stat(escaped); err == nil {
		t.Fatal("the sandboxed command wrote outside the project")
	}
	// The approved opt-out is the same instance without the wrapper.
	call, _ = b.Prepare(`{"command":"cat .env","sandbox":false}`)
	if r := b.Execute(context.Background(), call); r.Status != "completed" || !strings.Contains(r.Output, "secret-value") {
		t.Fatalf("unconfined result=%#v", r)
	}
}

func fieldValue(call Call, label string) string {
	for _, f := range call.Fields {
		if f.Label == label {
			return f.Value
		}
	}
	return ""
}
