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
	r := b.execute(context.Background(), BashArgs{`printf 'out'; printf 'err' >&2; printf '%s' "$OPENAI_API_KEY"; pwd; exit 7`}, BashTimeout, BashOutputLimit)
	if r.Status != "failed" || r.ExitCode == nil || *r.ExitCode != 7 || !strings.Contains(r.Output, "out") || !strings.Contains(r.Output, "err") || !strings.Contains(r.Output, b.Directory()) || strings.Contains(r.Output, "injected") || strings.Contains(r.Output, "must-not-inherit") {
		t.Fatalf("result=%#v", r)
	}
	r = b.execute(context.Background(), BashArgs{"command -v sh"}, BashTimeout, BashOutputLimit)
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
			r := b.execute(ctx, BashArgs{command}, 700*time.Millisecond, 128)
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
