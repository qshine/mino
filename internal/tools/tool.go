// Package tools defines local operations independently of their callers and model transport.
package tools

import (
	"context"
	"encoding/json"
)

const OutputLimit = 64 << 10

// Tool prepares an operation without executing it. The caller must obtain approval
// unless Prepare marks a confined read as ReadOnly, durably record the start before
// Execute, and save its result before continuing.
type Tool interface {
	Definition() Definition
	Prepare(arguments string) (Call, error)
	Execute(context.Context, Call) Result
}

// Inspector is implemented by tools whose confirmation depends on current local
// state. Prepare must stay independent of that state because history replay calls
// it to validate recorded arguments; the caller runs Inspect only for live calls,
// after Prepare and before approval.
type Inspector interface {
	Inspect(Call) (Call, error)
}

type Definition struct {
	Name, Description string
	Parameters        map[string]any
	Strict            bool
}

// Fields are data for the gateway to escape and present, not terminal markup.
type Field struct{ Label, Value string }
type Call struct {
	Arguments json.RawMessage
	ReadOnly  bool   // Set by trusted tool code, never decoded from model arguments.
	Expect    string // State captured by Inspect and rechecked by Execute; never persisted.
	Target    string // Canonical project-relative path for policy checks, set by trusted tool code.
	Sandboxed bool   // The process will run under the OS sandbox; set by trusted tool code.
	Directory string
	Fields    []Field
	Warning   string
}

// Result records facts available after execution, denial, or recovery. ExitCode is
// absent when no process exited with a known code, especially after a crash.
type Result struct {
	Status    string `json:"status"`
	Output    string `json:"output"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}
