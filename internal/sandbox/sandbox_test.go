package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testSandbox(t *testing.T) (*Sandbox, string) {
	t.Helper()
	if !Available() {
		t.Skip("sandbox-exec is unavailable on this system")
	}
	root, home := t.TempDir(), t.TempDir()
	s, err := New(root, home)
	if err != nil {
		t.Fatal(err)
	}
	return s, s.Root()
}

// run executes a shell snippet inside the sandbox and returns its combined output.
func run(t *testing.T, s *Sandbox, dir, script string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/bash", "--noprofile", "--norc", "-c", script)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + os.Getenv("HOME")}
	s.Wrap(cmd)
	out, _ := cmd.CombinedOutput()
	return string(out), cmd.ProcessState.ExitCode()
}

func TestWrapRunsUnderSandboxExec(t *testing.T) {
	s, root := testSandbox(t)
	cmd := exec.Command("/bin/bash", "-c", "true")
	s.Wrap(cmd)
	if cmd.Path != executable || cmd.Args[0] != executable || cmd.Args[1] != "-p" || cmd.Args[2] != s.Profile() || cmd.Args[3] != "/bin/bash" {
		t.Fatalf("args=%q", cmd.Args)
	}
	if !strings.Contains(s.Profile(), `(subpath "`+root+`")`) || !strings.Contains(s.Profile(), "(deny network*)") {
		t.Fatalf("profile:\n%s", s.Profile())
	}
	if strings.Contains(s.WithNetwork().Profile(), "(deny network*)") || !strings.Contains(s.WithNetwork().Summary(), "network allowed") {
		t.Fatal("WithNetwork kept the network denial")
	}
}

func TestWritesStayInsideRoot(t *testing.T) {
	s, root := testSandbox(t)
	outside := t.TempDir() // a sibling temporary directory is writable: it is under os.TempDir
	home := os.Getenv("HOME")
	out, code := run(t, s, root, `echo in > inside.txt && echo "inside=$?"; mkdir -p sub && echo x > sub/y && echo "sub=$?"; touch "`+home+`/mino-sandbox-test" 2>/dev/null; echo "home=$?"; echo x > /dev/null; echo "devnull=$?"`)
	defer os.Remove(filepath.Join(home, "mino-sandbox-test"))
	for _, want := range []string{"inside=0", "sub=0", "home=1", "devnull=0"} {
		if !strings.Contains(out, want) {
			t.Errorf("want %s in output (exit %d):\n%s", want, code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(home, "mino-sandbox-test")); err == nil {
		t.Error("a file was created in the home directory")
	}
	if _, err := os.Stat(filepath.Join(root, "inside.txt")); err != nil {
		t.Error("the file inside the root was not created")
	}
	_ = outside
}

func TestProtectedPathsAreBlocked(t *testing.T) {
	s, root := testSandbox(t)
	for _, name := range []string{".env", "sub/.ENV", "conf/.env.local", "keys/server.pem", "keys/ID_RSA", ".ssh/config"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("secret"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "env.md"), []byte("public"), 0600); err != nil {
		t.Fatal(err)
	}
	out, _ := run(t, s, root, `for f in .env sub/.ENV conf/.env.local keys/server.pem keys/ID_RSA .ssh/config; do cat "$f" 2>/dev/null && echo "read $f"; echo new > "$f" 2>/dev/null && echo "wrote $f"; done; cat env.md`)
	if strings.Contains(out, "secret") || strings.Contains(out, "read ") || strings.Contains(out, "wrote ") {
		t.Fatalf("protected content escaped:\n%s", out)
	}
	if !strings.Contains(out, "public") {
		t.Fatalf("ordinary file unreadable:\n%s", out)
	}
}

func TestGitDirectoryIsReadableButNotWritable(t *testing.T) {
	s, root := testSandbox(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	init := exec.Command("git", "init", "-q", root)
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	out, _ := run(t, s, root, `git status --short >/dev/null && echo "status=ok"; echo x > f && git add f 2>/dev/null && echo "add=ok" || echo "add=blocked"`)
	if !strings.Contains(out, "status=ok") || !strings.Contains(out, "add=blocked") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestNetworkIsBlockedUnlessAllowed(t *testing.T) {
	s, root := testSandbox(t)
	// Connecting to a closed loopback port distinguishes a kernel denial (not
	// permitted) from a refused connection without needing the Internet.
	script := `exec 3<>/dev/tcp/127.0.0.1/9 2>&1; echo "rc=$?"`
	out, _ := run(t, s, root, script)
	if !strings.Contains(out, "Operation not permitted") {
		t.Fatalf("sandboxed connect was not denied by the kernel:\n%s", out)
	}
	out, _ = run(t, s.WithNetwork(), root, script)
	if strings.Contains(out, "Operation not permitted") {
		t.Fatalf("network variant still denied:\n%s", out)
	}
}

func TestComponentRegex(t *testing.T) {
	cases := map[string]string{".env": "[.][eE][nN][vV]", "*.pem": "[^/]*[.][pP][eE][mM]", "id_rsa*": "[iI][dD]_[rR][sS][aA][^/]*", ".git": "[.][gG][iI][tT]"}
	for pattern, want := range cases {
		if got := componentRegex(pattern); got != want {
			t.Errorf("%s: got %s, want %s", pattern, got, want)
		}
	}
	if got := quote(`/a "b"\c`); got != `"/a \"b\"\\c"` {
		t.Errorf("quote: %s", got)
	}
}

func TestNewRejectsBadInput(t *testing.T) {
	if !Available() {
		t.Skip("sandbox-exec is unavailable on this system")
	}
	if _, err := New(filepath.Join(t.TempDir(), "missing"), t.TempDir()); err == nil {
		t.Error("accepted a missing root")
	}
	if _, err := New(t.TempDir(), "relative"); err == nil {
		t.Error("accepted a relative home")
	}
}
