package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/qshine/mino/internal/tools"
	"io"
	"os"
	"testing"
)

func callReply(ids ...string) modelReply {
	r := modelReply{}
	for _, id := range ids {
		raw, _ := json.Marshal(map[string]any{"type": "function_call", "call_id": id, "name": "bash", "arguments": `{"command":"go version"}`, "status": "completed"})
		r.Output = append(r.Output, raw)
	}
	return r
}

func TestToolHistoryRecoveryPreservesPairsAndAcknowledgement(t *testing.T) {
	for _, stage := range []string{"response", "start", "result"} {
		t.Run(stage, func(t *testing.T) {
			isolateConfig(t)
			h, err := openHistory()
			if err != nil {
				t.Fatal(err)
			}
			id := newHistoryID()
			records := []historyRecord{{TurnID: id, Kind: "user_message", Text: "Go version?"}, {TurnID: id, Kind: "model_response", Output: callReply("a", "b").Output}}
			if stage != "response" {
				records = append(records, historyRecord{TurnID: id, Kind: "tool_start", CallID: "a", Name: "bash", Arguments: json.RawMessage(`{"command":"go version"}`), CWD: t.TempDir()})
			}
			if stage == "result" {
				code := 0
				records = append(records, historyRecord{TurnID: id, Kind: "tool_result", CallID: "a", Result: &tools.Result{Status: "completed", Output: "go version example", ExitCode: &code}})
			}
			if err := h.append(records...); err != nil {
				t.Fatal(err)
			}
			h.Close()
			h, err = openHistory()
			if err != nil {
				t.Fatal(err)
			}
			if len(h.state.input) != 6 || h.state.pending.id != "" {
				t.Fatalf("recovery input=%v pending=%v", h.state.input, h.state.pending)
			}
			if (len(h.state.uncertain) > 0) != (stage == "start") {
				t.Fatalf("uncertainty=%v", h.state.uncertain)
			}
			data, _ := os.ReadFile(h.file.Name())
			if !bytes.Contains(data, []byte(`"status":"not_executed"`)) || (stage == "start" && !bytes.Contains(data, []byte(`"status":"unknown"`))) {
				t.Fatalf("history=%s", data)
			}
			h.Close()
			h, err = openHistory()
			if err != nil {
				t.Fatal(err)
			}
			if (len(h.state.uncertain) > 0) != (stage == "start") {
				t.Fatal("uncertainty lost across restart")
			}
			if err := h.acknowledgeRecovery(); err != nil {
				t.Fatal(err)
			}
			h.Close()
			h, err = openHistory()
			if err != nil {
				t.Fatal(err)
			}
			defer h.Close()
			if len(h.state.uncertain) != 0 {
				t.Fatal("acknowledgement was not persisted")
			}
		})
	}
}

func TestInvalidToolHistoryDoesNotReachDisk(t *testing.T) {
	for _, mode := range []string{"duplicate response", "unpaired result", "wrong arguments", "unfinished turn"} {
		t.Run(mode, func(t *testing.T) {
			isolateConfig(t)
			h, err := openHistory()
			if err != nil {
				t.Fatal(err)
			}
			defer h.Close()
			id := newHistoryID()
			if err := h.append(historyRecord{TurnID: id, Kind: "user_message", Text: "test"}, historyRecord{TurnID: id, Kind: "model_response", Output: callReply("a").Output}); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(h.file.Name())
			var r historyRecord
			switch mode {
			case "duplicate response":
				r = historyRecord{TurnID: id, Kind: "model_response", Output: callReply("a").Output}
			case "unpaired result":
				r = historyRecord{TurnID: id, Kind: "tool_result", CallID: "b", Result: &tools.Result{Status: "denied"}}
			case "wrong arguments":
				r = historyRecord{TurnID: id, Kind: "tool_start", CallID: "a", Name: "bash", Arguments: json.RawMessage(`{"command":"other"}`), CWD: t.TempDir()}
			case "unfinished turn":
				r = historyRecord{TurnID: id, Kind: "turn_end", Status: "failed"}
			}
			if err := h.append(r); err == nil {
				t.Fatal("accepted invalid transition")
			}
			after, _ := os.ReadFile(h.file.Name())
			if !bytes.Equal(before, after) {
				t.Fatal("invalid transition reached disk")
			}
		})
	}
}

func TestUnknownResultGatesChatUntilAcknowledged(t *testing.T) {
	isolateConfig(t)
	h, err := openHistory()
	if err != nil {
		t.Fatal(err)
	}
	id := newHistoryID()
	if err := h.append(
		historyRecord{TurnID: id, Kind: "user_message", Text: "Go version?"},
		historyRecord{TurnID: id, Kind: "model_response", Output: callReply("a").Output},
		historyRecord{TurnID: id, Kind: "tool_start", CallID: "a", Name: "bash", Arguments: json.RawMessage(`{"command":"go version"}`), CWD: t.TempDir()},
	); err != nil {
		t.Fatal(err)
	}
	h.Close()
	h, err = openHistory()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	// The interrupted tool turn is replayed with its paired unknown result and a
	// runtime notice, so the model can see that the command may have run.
	input, _ := json.Marshal(h.state.input)
	if !bytes.Contains(input, []byte(`"call_id":"a"`)) || !bytes.Contains(input, []byte("unknown")) || !bytes.Contains(input, []byte("Mino runtime notice")) {
		t.Fatalf("input=%s", input)
	}
	a, err := New(Options{}, h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Handle(context.Background(), "continue", testInteraction{}); err == nil {
		t.Fatal("accepted a question before acknowledgement")
	}
	var kinds []string
	confirm := func(answer bool) testInteraction {
		return testInteraction{confirm: func(_ context.Context, c Confirmation) (bool, error) {
			kinds = append(kinds, c.Kind)
			return answer, nil
		}}
	}
	if err := a.Start(context.Background(), confirm(false)); !errors.Is(err, io.EOF) {
		t.Fatalf("declined acknowledgement: %v", err)
	}
	if err := a.Start(context.Background(), confirm(true)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(h.file.Name())
	if len(kinds) != 2 || kinds[0] != "recovery" || len(h.state.uncertain) != 0 || bytes.Count(data, []byte(`"kind":"recovery_ack"`)) != 1 {
		t.Fatalf("kinds=%v history=%s", kinds, data)
	}
}
