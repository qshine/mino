// Package sandbox confines a child process with the macOS Seatbelt facility,
// invoked through /usr/bin/sandbox-exec. The policy package decides whether a
// call runs; this package decides what the running process may touch, and the
// kernel enforces it: reads are allowed except for protected files, writes stay
// inside the writable roots, and no network connection opens.
package sandbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/qshine/mino/internal/policy"
)

const executable = "/usr/bin/sandbox-exec"

// Available reports whether this system can confine processes. Seatbelt exists
// only on macOS; elsewhere commands run unconfined and the caller must say so.
func Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	info, err := os.Stat(executable)
	return err == nil && info.Mode().IsRegular()
}

// Sandbox is one rendered profile. Two instances differ only in their writable
// root and whether the network is allowed; the protected paths and the
// temporary and cache directories are the same for every process.
type Sandbox struct {
	root     string
	writable []string
	private  []string // Mino's own data directory: neither readable nor writable
	network  bool
	profile  string
}

// New builds a profile whose writes are confined to root, the temporary
// directories, and the per-user cache directories under home. Caches stay
// writable so that compilers and package managers keep working; nothing else in
// the home directory does.
func New(root, home string) (*Sandbox, error) {
	if !Available() {
		return nil, errors.New("The sandbox is unavailable on this system")
	}
	root, err := realDirectory(root)
	if err != nil {
		return nil, errors.New("The sandbox root must be an existing directory")
	}
	if !filepath.IsAbs(home) || strings.ContainsRune(home, 0) {
		return nil, errors.New("The sandbox requires an absolute home directory")
	}
	writable := []string{root}
	for _, candidate := range []string{os.TempDir(), "/private/tmp", filepath.Join(home, "Library", "Caches"), filepath.Join(home, ".cache")} {
		if dir, err := realDirectory(candidate); err == nil {
			writable = append(writable, dir)
		}
	}
	// ~/.mino holds the API key, conversations, and memories. Commands never need
	// it, and a command that runs unasked must not be able to read it.
	private := []string{filepath.Join(home, ".mino")}
	if dir, err := realDirectory(private[0]); err == nil && dir != private[0] {
		private = append(private, dir)
	}
	s := &Sandbox{root: root, writable: writable, private: private}
	s.profile = s.render()
	return s, nil
}

// WithNetwork returns a copy that may open network connections. MCP stdio
// servers usually proxy a remote service, so they get this variant.
func (s *Sandbox) WithNetwork() *Sandbox {
	copied := *s
	copied.network = true
	copied.profile = copied.render()
	return &copied
}

func (s *Sandbox) Root() string    { return s.root }
func (s *Sandbox) Profile() string { return s.profile }

// Summary is the confirmation text: what the kernel will and will not allow.
func (s *Sandbox) Summary() string {
	network := "no network"
	if s.network {
		network = "network allowed"
	}
	return fmt.Sprintf("on: writes only inside %s, temporary and cache directories; protected paths and ~/.mino blocked; %s.", s.root, network)
}

// Wrap makes cmd run under sandbox-exec with this profile. The profile is passed
// inline, so no file needs to exist at execution time.
func (s *Sandbox) Wrap(cmd *exec.Cmd) {
	cmd.Args = append([]string{executable, "-p", s.profile}, cmd.Args...)
	cmd.Path = executable
}

// render writes the Seatbelt profile. Later rules win, so the protected-path
// denials and the network denial come last.
func (s *Sandbox) render() string {
	var b strings.Builder
	b.WriteString("(version 1)\n(deny default)\n")
	// Processes, signals, and the kernel services that ordinary command-line
	// tools need, none of which reaches files outside the rules below.
	b.WriteString("(allow process*)\n(allow signal)\n(allow sysctl-read)\n(allow mach-lookup)\n(allow ipc-posix*)\n(allow file-ioctl (subpath \"/dev\"))\n")
	b.WriteString("(allow file-read*)\n")
	b.WriteString("(allow file-write*")
	for _, dir := range s.writable {
		fmt.Fprintf(&b, " (subpath %s)", quote(dir))
	}
	b.WriteString(` (literal "/dev/null") (regex #"^/dev/ttys[0-9]+$"))` + "\n")
	for _, pattern := range policy.Protected {
		name, dir := strings.CutSuffix(pattern, "/")
		expression := "/" + componentRegex(name) + "$"
		if dir {
			expression = "/" + componentRegex(name) + "(/|$)"
		}
		// .git/ is protected from writes because of hooks and config, not because
		// it holds secrets; git itself cannot run without reading it.
		operations := "file-read* file-write*"
		if pattern == ".git/" {
			operations = "file-write*"
		}
		fmt.Fprintf(&b, "(deny %s (regex #\"%s\"))\n", operations, expression)
	}
	for _, dir := range s.private {
		fmt.Fprintf(&b, "(deny file-read* file-write* (subpath %s))\n", quote(dir))
	}
	if s.network {
		b.WriteString("(allow network*)\n")
	} else {
		b.WriteString("(deny network*)\n")
	}
	return b.String()
}

// componentRegex turns one protected pattern into a Seatbelt regular expression
// for a single path component. Letters match either case because the default
// macOS file system does; "*" matches within the component only.
func componentRegex(pattern string) string {
	var b strings.Builder
	for _, r := range pattern {
		switch {
		case r == '*':
			b.WriteString("[^/]*")
		case r == '?':
			b.WriteString("[^/]")
		case r >= 'a' && r <= 'z':
			fmt.Fprintf(&b, "[%c%c]", r, r-'a'+'A')
		case r >= 'A' && r <= 'Z':
			fmt.Fprintf(&b, "[%c%c]", r-'A'+'a', r)
		case r >= '0' && r <= '9' || r == '_' || r == '-' || r == '/':
			b.WriteRune(r)
		default:
			fmt.Fprintf(&b, "[%c]", r)
		}
	}
	return b.String()
}

// quote renders a profile string literal; paths may contain quotes or backslashes.
func quote(s string) string {
	replaced := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
	return `"` + replaced + `"`
}

func realDirectory(path string) (string, error) {
	if path == "" || strings.ContainsRune(path, 0) {
		return "", errors.New("empty path")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil || !info.IsDir() {
		return "", errors.New("not a directory")
	}
	return real, nil
}
