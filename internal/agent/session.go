package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"sync"
	"unicode/utf8"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/qshine/mino/internal/tools"
)

// Session owns the in-memory replay state. append commits memory only after the
// complete JSONL write and Sync succeed. Gateways never modify this state.
type Session struct {
	*history
	state historyState
	busy  sync.Mutex
}

type historyRecord struct {
	Version   int               `json:"v"`
	Seq       int               `json:"seq"`
	TurnID    string            `json:"turn_id"`
	Kind      string            `json:"kind"`
	Text      string            `json:"text,omitempty"`
	Output    []json.RawMessage `json:"output,omitempty"`
	Status    string            `json:"status,omitempty"`
	CallID    string            `json:"call_id,omitempty"`
	Name      string            `json:"name,omitempty"`
	Arguments json.RawMessage   `json:"arguments,omitempty"`
	CWD       string            `json:"cwd,omitempty"`
	Result    *tools.Result     `json:"result,omitempty"`
}

type pendingTurn struct {
	id, prompt string
	reply      *modelReply
	items      responses.ResponseInputParam
	calls      []pendingCall
	final      bool
	modelSeen  bool
}

type historyState struct {
	tools     map[string]tools.Tool
	seq       int
	pending   pendingTurn
	turns     map[string]bool
	input     responses.ResponseInputParam
	uncertain map[string]bool
}

func (h *Session) append(records ...historyRecord) error {
	if h.broken {
		return errors.New("History is unavailable after a write failure. Restart Mino.")
	}
	var data bytes.Buffer
	next := h.state.clone()
	for i := range records {
		r := &records[i]
		r.Version, r.Seq = 2, h.state.seq+i+1
		if err := next.apply(*r); err != nil {
			return fmt.Errorf("Invalid history transition: %w", err)
		}
		line, err := json.Marshal(r)
		if err != nil {
			return errors.New("Failed to encode history record")
		}
		if len(line)+1 > maxHistoryRecordBytes {
			return errors.New("History record exceeds the 16 MiB limit")
		}
		data.Write(line)
		data.WriteByte('\n')
	}
	if h.size+int64(data.Len()) > maxHistoryBytes {
		return errors.New("History reached the 64 MiB limit. Back it up and move it aside before restarting.")
	}
	n, err := h.writer.Write(data.Bytes())
	if err == nil && n != data.Len() {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = h.writer.Sync()
	}
	if err != nil {
		h.broken = true
		return fmt.Errorf("Failed to save history: %w", err)
	}
	h.size += int64(n)
	h.state = next
	return nil
}

type pendingCall struct {
	call              toolCall
	started, resolved bool
}

func (s historyState) clone() historyState {
	s.turns = maps.Clone(s.turns)
	s.uncertain = maps.Clone(s.uncertain)
	s.input = slices.Clone(s.input)
	s.pending.items = slices.Clone(s.pending.items)
	s.pending.calls = slices.Clone(s.pending.calls)
	return s
}

func (p *pendingTurn) unresolved() *pendingCall {
	for i := range p.calls {
		if !p.calls[i].resolved {
			return &p.calls[i]
		}
	}
	return nil
}

func (s *historyState) apply(r historyRecord) error {
	invalid := errors.New("invalid history fields or turn order")
	if (r.Version != 1 && r.Version != 2) || r.Seq != s.seq+1 || !validHistoryID(r.TurnID) {
		return invalid
	}
	toolFields := r.CallID != "" || r.Name != "" || r.Arguments != nil || r.CWD != "" || r.Result != nil
	if toolFields && (r.Version != 2 || (r.Kind != "tool_start" && r.Kind != "tool_result")) {
		return invalid
	}
	if r.Version == 1 && r.Kind != "user_message" && r.Kind != "assistant_message" && r.Kind != "turn_end" {
		return invalid
	}
	p := &s.pending
	switch r.Kind {
	case "user_message":
		if p.id != "" || s.turns[r.TurnID] || !validHistoryText(r.Text) || r.Status != "" || len(r.Output) != 0 {
			return invalid
		}
		*p = pendingTurn{id: r.TurnID, prompt: r.Text}
		s.turns[r.TurnID] = true
	case "assistant_message": // Chapter 02 records remain readable.
		if p.id != r.TurnID || p.reply != nil || p.modelSeen || !validHistoryText(r.Text) || r.Status != "" {
			return invalid
		}
		reply := modelReply{Text: r.Text, Output: r.Output}
		if _, err := reply.inputItems(); err != nil {
			return err
		}
		if len(reply.calls()) != 0 {
			return invalid
		}
		p.reply = &reply
	case "model_response":
		if p.id != r.TurnID || p.reply != nil || p.final || p.unresolved() != nil || r.Status != "" {
			return invalid
		}
		reply := modelReply{Text: r.Text, Output: r.Output}
		items, err := reply.inputItems()
		if err != nil {
			return err
		}
		calls := reply.calls()
		for _, call := range calls {
			for _, previous := range p.calls {
				if previous.call.CallID == call.CallID {
					return errors.New("reused tool call identifier")
				}
			}
			p.calls = append(p.calls, pendingCall{call: call})
		}
		p.items = append(p.items, items...)
		p.modelSeen, p.final = true, len(calls) == 0
	case "tool_start":
		call := p.unresolved()
		if p.id != r.TurnID || call == nil || call.started || call.call.CallID != r.CallID || call.call.Name != r.Name || r.Arguments == nil || (r.CWD != "" && !filepath.IsAbs(r.CWD)) || r.Result != nil || r.Text != "" || len(r.Output) != 0 || r.Status != "" {
			return invalid
		}
		tool := s.tools[r.Name]
		if tool == nil {
			return errors.New("History contains a started call for an unavailable tool")
		}
		prepared, err := tool.Prepare(call.call.Arguments)
		if err != nil || !sameArguments(prepared.Arguments, r.Arguments) {
			return invalid
		}
		if prepared.Directory != "" && !filepath.IsAbs(r.CWD) {
			return invalid
		}
		call.started = true
	case "tool_result":
		call := p.unresolved()
		if p.id != r.TurnID || call == nil || call.call.CallID != r.CallID || r.Result == nil || r.Text != "" || len(r.Output) != 0 || r.Status != "" || r.Name != "" || r.Arguments != nil || r.CWD != "" {
			return invalid
		}
		if err := validateToolResult(*r.Result, call.started); err != nil {
			return err
		}
		p.items = append(p.items, toolResultInput(r.CallID, *r.Result))
		call.resolved = true
		if r.Result.Status == "unknown" {
			if s.uncertain == nil {
				s.uncertain = make(map[string]bool)
			}
			s.uncertain[r.TurnID] = true
		}
	case "turn_end":
		if p.id != r.TurnID || r.Text != "" || len(r.Output) != 0 || p.unresolved() != nil {
			return invalid
		}
		switch r.Status {
		case "completed":
			if p.reply != nil {
				items, err := p.reply.inputItems()
				if err != nil {
					return err
				}
				p.items = items
			} else if !p.final {
				return errors.New("completed turn has no answer")
			}
		case "failed", "cancelled", "interrupted":
		default:
			return invalid
		}
		// Failed tool turns still contain real effects. Replay all paired items,
		// then an explicit local interruption notice, never partial streamed text.
		if r.Status == "completed" || len(p.calls) > 0 {
			s.input = append(s.input, userInput(p.prompt))
			s.input = append(s.input, p.items...)
			if r.Status != "completed" {
				s.input = append(s.input, responses.ResponseInputItemParamOfMessage("Mino runtime notice: the previous turn ended with status "+r.Status+". Tool results above are retained; no command was automatically retried.", responses.EasyInputMessageRoleAssistant))
			}
		}
		*p = pendingTurn{}
	case "recovery_ack":
		if p.id != "" || !s.uncertain[r.TurnID] || r.Text != "" || r.Status != "" || len(r.Output) != 0 {
			return invalid
		}
		delete(s.uncertain, r.TurnID)
	default:
		return errors.New("unknown record kind")
	}
	s.seq = r.Seq
	return nil
}

func toolResultInput(id string, result tools.Result) responses.ResponseInputItemUnionParam {
	raw, _ := json.Marshal(result)
	item := responses.ResponseInputItemParamOfFunctionCallOutput(string(raw))
	item.OfFunctionCallOutput.CallID = openai.String(id)
	return item
}

func validateToolResult(r tools.Result, started bool) error {
	invalid := errors.New("invalid tool result")
	if !utf8.ValidString(r.Output) || len(r.Output) > tools.OutputLimit {
		return invalid
	}
	switch r.Status {
	case "completed", "failed", "timed_out", "output_limit", "cancelled":
		if !started {
			return invalid
		}
	case "unknown":
		if !started || r.ExitCode != nil || r.Truncated {
			return invalid
		}
	case "denied", "invalid_arguments", "unknown_tool", "not_executed":
		if started || r.ExitCode != nil || r.Truncated {
			return invalid
		}
	default:
		return invalid
	}
	if r.ExitCode != nil && *r.ExitCode < 0 {
		return invalid
	}
	if r.Status == "completed" && r.ExitCode != nil && *r.ExitCode != 0 {
		return invalid
	}
	if r.Status == "output_limit" && !r.Truncated {
		return invalid
	}
	return nil
}

// finishTurn closes every outstanding call without ever running it. A start
// without a result is unknown, including a crash just before process creation.
func (h *Session) finishTurn(status string) error {
	var records []historyRecord
	for _, call := range h.state.pending.calls {
		if call.resolved {
			continue
		}
		result := tools.Result{Status: "not_executed", Output: "Command was not executed; the turn stopped."}
		if call.started {
			result = tools.Result{Status: "unknown", Output: "Execution was started but its result was not saved. The operation may have happened. Do not automatically retry it."}
		}
		records = append(records, historyRecord{TurnID: h.state.pending.id, Kind: "tool_result", CallID: call.call.CallID, Result: &result})
	}
	records = append(records, historyRecord{TurnID: h.state.pending.id, Kind: "turn_end", Status: status})
	return h.append(records...)
}

func (h *Session) acknowledgeRecovery() error {
	var records []historyRecord
	for _, id := range slices.Sorted(maps.Keys(h.state.uncertain)) {
		records = append(records, historyRecord{TurnID: id, Kind: "recovery_ack"})
	}
	if len(records) == 0 {
		return nil
	}
	return h.append(records...)
}

// Compare complete JSON values, including whitespace-insensitive object fields.
func sameArguments(a, b json.RawMessage) bool {
	var left, right any
	if !json.Valid(a) || !json.Valid(b) {
		return false
	}
	// Tool arguments may contain integers beyond float64's exact range.
	ldecoder, rdecoder := json.NewDecoder(bytes.NewReader(a)), json.NewDecoder(bytes.NewReader(b))
	ldecoder.UseNumber()
	rdecoder.UseNumber()
	if ldecoder.Decode(&left) != nil || rdecoder.Decode(&right) != nil {
		return false
	}
	l, _ := json.Marshal(left)
	r, _ := json.Marshal(right)
	return bytes.Equal(l, r)
}
