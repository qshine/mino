// Package policy makes one allow, ask, or deny decision for every live tool call.
// Rules come from code and the permission mode the user selects in the terminal,
// never from the model, a project, tool output, or conversation history.
package policy

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
)

type Decision int

const (
	Ask Decision = iota
	Allow
	Deny
)

func (d Decision) String() string {
	return [...]string{"ask", "allow", "deny"}[d]
}

// Mode chooses which calls run without a confirmation. No mode can override a
// denial: protected paths are refused in every mode.
type Mode int32

const (
	Manual      Mode = iota // ask before every call that is not a confined read
	AcceptEdits             // also run project file writes and edits unasked
	Bypass                  // run every call unasked
)

var modeNames = [...]string{"manual", "accept-edits", "bypass"}

func (m Mode) String() string { return modeNames[m] }

// ParseMode accepts the names shown by String.
func ParseMode(name string) (Mode, bool) {
	for i, n := range modeNames {
		if n == name {
			return Mode(i), true
		}
	}
	return Manual, false
}

// Protected always applies. A pattern without "/" matches a file name; one ending
// in "/" matches any directory name.
var Protected = []string{".env", ".env.*", "*.pem", "*.key", "*.p12", "id_rsa*", "id_ecdsa*", "id_ed25519*", ".netrc", ".npmrc", ".pypirc", ".git/", ".ssh/", ".aws/", ".gnupg/"}

// Policy holds the current mode. Each process starts in Manual; the mode changes
// only through SetMode, called by the session manager between whole turns.
type Policy struct{ mode atomic.Int32 }

func New() *Policy { return &Policy{} }

func (p *Policy) Mode() Mode        { return Mode(p.mode.Load()) }
func (p *Policy) SetMode(mode Mode) { p.mode.Store(int32(mode)) }

// Call is what the program knows about a prepared call. Every field is set by
// trusted code; none is taken from model text.
type Call struct {
	Tool      string
	ReadOnly  bool   // a confined read marked by the tool itself
	Target    string // canonical project-relative path, if any
	Sandboxed bool   // the process runs under the OS sandbox, marked by the tool itself
}

// Decide applies the rules in a fixed order: protected paths first, so no mode
// can override them. A call with a Target that is not read-only is a confined
// project file change; a Sandboxed call is a command the kernel confines to the
// same project. Those two are the only kinds AcceptEdits runs unasked.
func (p *Policy) Decide(c Call) (Decision, string) {
	mode := p.Mode()
	switch {
	case c.Target != "" && IsProtected(c.Target):
		return Deny, fmt.Sprintf("%s is a protected path; file tools never read or write it.", c.Target)
	case c.ReadOnly:
		return Allow, "Confined read."
	case mode == Bypass:
		return Allow, "Allowed without confirmation in bypass mode."
	case mode == AcceptEdits && (c.Target != "" || c.Sandboxed):
		return Allow, "Allowed without confirmation in accept-edits mode."
	}
	return Ask, ""
}

// IsProtected matches every path component, so .git/hooks/x and a/.env both
// match. Matching ignores case: the default macOS file system opens .env for ".ENV".
func IsProtected(path string) bool {
	parts := strings.Split(strings.ToLower(filepath.Clean(path)), string(filepath.Separator))
	for i, part := range parts {
		last := i == len(parts)-1
		for _, pattern := range Protected {
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
