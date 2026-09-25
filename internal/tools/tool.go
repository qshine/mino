// Package tools defines local operations independently of their callers and model transport.
package tools

import (
	"context"
	"encoding/json"
)

const OutputLimit = 64 << 10

// Tool prepares an operation without executing it. The caller must obtain approval
// and durably record the start before Execute, and save its result before continuing.
type Tool interface {
	Definition() Definition
	Prepare(arguments string) (Call, error)
	Execute(context.Context, Call) Result
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
