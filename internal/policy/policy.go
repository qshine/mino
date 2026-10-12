// Package policy makes one allow, ask, or deny decision for every live tool call.
// Rules come from code defaults and the user's own ~/.mino/policy.json, never from
// the model, a project, tool output, or conversation history.
package policy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unicode/utf8"
)

const maxPolicyBytes = 64 << 10

type Decision int

const (
	Ask Decision = iota
	Allow
	Deny
)

func (d Decision) String() string {
	return [...]string{"ask", "allow", "deny"}[d]
}

// DefaultProtected always applies; a policy file can only add patterns. A pattern
// without "/" matches a file name; one ending in "/" matches any directory name.
var DefaultProtected = []string{".env", ".env.*", "*.pem", "*.key", "*.p12", "id_rsa*", "id_ecdsa*", "id_ed25519*", ".netrc", ".npmrc", ".pypirc", ".git/", ".ssh/", ".aws/", ".gnupg/"}

// File is the user's optional policy file.
type File struct {
	Bash      string   `json:"bash,omitempty"` // "ask" (default) or "deny"
	Protected []string `json:"protected_paths,omitempty"`
	Allow     []string `json:"allow_tools,omitempty"`
	Deny      []string `json:"deny_tools,omitempty"`
}

// Policy is fixed for the life of the process, so an approval can never outlive
// the rules under which it was given.
type Policy struct {
	bashDenied bool
	protected  []string
	allow      []string
	deny       []string
}

// Default asks before every non-read-only call and protects the default paths.
func Default() *Policy { return &Policy{protected: slices.Clone(DefaultProtected)} }

func validPattern(pattern string) bool {
	name := strings.TrimSuffix(pattern, "/")
	if name == "" || len(pattern) > 256 || !utf8.ValidString(pattern) || strings.ContainsAny(name, "/\x00") || name == "." || name == ".." {
		return false
	}
	_, err := filepath.Match(name, "x")
	return err == nil
}

func (f File) validate() error {
	if f.Bash != "" && f.Bash != "ask" && f.Bash != "deny" {
		return errors.New(`"bash" must be "ask" or "deny"`)
	}
	if len(f.Protected)+len(f.Allow)+len(f.Deny) > 256 {
		return errors.New("The policy has too many entries")
	}
	for _, pattern := range f.Protected {
		if !validPattern(pattern) {
			return fmt.Errorf("Invalid protected path pattern %q: use a file name pattern or a directory name ending in /", pattern)
		}
	}
	for _, name := range f.Allow {
		// Only external MCP tools can skip confirmation by name. Bash and file
		// writes stay behind approval, because their effects are local and broad.
		if !strings.HasPrefix(name, "mcp_") || len(name) > 64 {
			return fmt.Errorf("allow_tools accepts only MCP tool names such as mcp_context7_query-docs, not %q", name)
		}
	}
	for _, name := range f.Deny {
		if name == "" || len(name) > 64 || !utf8.ValidString(name) {
			return fmt.Errorf("Invalid deny_tools entry %q", name)
		}
	}
	return nil
}

// Load reads the policy file. A missing file means the default policy. The file
// must not be writable by other users, because it decides what runs unasked.
func Load(path string) (*Policy, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return nil, errors.New("Cannot open ~/.mino/policy.json; it must be a regular file, not a symbolic link")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("~/.mino/policy.json must be a regular file")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return nil, errors.New("~/.mino/policy.json must not be writable by other users (chmod 600)")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPolicyBytes+1))
	if err != nil || len(data) > maxPolicyBytes {
		return nil, errors.New("~/.mino/policy.json must be at most 64 KiB")
	}
	var f File
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&f); err != nil || decoder.More() {
		return nil, errors.New("~/.mino/policy.json is not valid policy JSON")
	}
	if err := f.validate(); err != nil {
		return nil, fmt.Errorf("~/.mino/policy.json: %w", err)
	}
	p := Default()
	p.bashDenied = f.Bash == "deny"
	p.protected = append(p.protected, f.Protected...)
	p.allow, p.deny = slices.Clone(f.Allow), slices.Clone(f.Deny)
	return p, nil
}

// Call is what the program knows about a prepared call. Every field is set by
// trusted code; none is taken from model text.
type Call struct {
	Tool     string
	ReadOnly bool   // a confined read marked by the tool itself
	Target   string // canonical project-relative path, if any
}

// Decide applies the rules in a fixed order: explicit denials and protected paths
// first, so no allowance can override them.
func (p *Policy) Decide(c Call) (Decision, string) {
	switch {
	case slices.Contains(p.deny, c.Tool):
		return Deny, "This tool is denied by ~/.mino/policy.json."
	case c.Target != "" && p.Protected(c.Target):
		return Deny, fmt.Sprintf("%s is a protected path; file tools never read or write it.", c.Target)
	case c.Tool == "bash" && p.bashDenied:
		return Deny, "Bash is disabled by ~/.mino/policy.json."
	case c.ReadOnly:
		return Allow, "Confined read."
	case slices.Contains(p.allow, c.Tool):
		return Allow, "Allowed without confirmation by ~/.mino/policy.json."
	}
	return Ask, ""
}

// Offered reports whether a tool should be offered to the model at all.
func (p *Policy) Offered(tool string) bool {
	return !slices.Contains(p.deny, tool) && !(tool == "bash" && p.bashDenied)
}

// Protected matches every path component, so .git/hooks/x and a/.env both match.
// Matching ignores case: the default macOS file system opens .env for ".ENV".
func (p *Policy) Protected(path string) bool {
	parts := strings.Split(strings.ToLower(filepath.Clean(path)), string(filepath.Separator))
	for i, part := range parts {
		last := i == len(parts)-1
		for _, pattern := range p.protected {
			// File patterns match the final component; directory patterns match
			// every component before it.
			name, dir := strings.CutSuffix(pattern, "/")
			if dir == last {
				continue
			}
			if ok, _ := filepath.Match(strings.ToLower(name), part); ok {
				return true
			}
		}
	}
	return false
}

// Unknown lists allow and deny names that match no available tool, usually a
// typo or an MCP server that is not connected now.
func (p *Policy) Unknown(available []string) []string {
	var unknown []string
	for _, name := range append(slices.Clone(p.allow), p.deny...) {
		if !slices.Contains(available, name) && !slices.Contains(unknown, name) {
			unknown = append(unknown, name)
		}
	}
	return unknown
}

// Summary describes the effective policy without listing every default.
func (p *Policy) Summary() string {
	bash := "ask"
	if p.bashDenied {
		bash = "denied"
	}
	return fmt.Sprintf("Policy: Bash %s; %d protected path patterns; %d MCP tools allowed without confirmation; %d tools denied.", bash, len(p.protected), len(p.allow), len(p.deny))
}
