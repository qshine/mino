package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qshine/mino/internal/mcp"
	"github.com/qshine/mino/internal/tools"
)

func TestMCPToolCallsNeedApprovalAndReplayOffline(t *testing.T) {
	var calls atomic.Int32
	docs := sdk.NewServer(&sdk.Implementation{Name: "docs", Version: "1"}, nil)
	docs.AddTool(&sdk.Tool{Name: "query-docs", Description: "Read docs.", InputSchema: map[string]any{"type": "object"}}, func(_ context.Context, r *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		calls.Add(1)
		return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: "useEffect runs after render. " + string(r.Params.Arguments)}}}, nil
	})
	service := httptest.NewServer(sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return docs }, nil))
	defer service.Close()
	manager, statuses := mcp.Connect(context.Background(), mcp.Config{Servers: map[string]mcp.Server{"docs": {Type: "http", URL: service.URL}}}, t.TempDir())
	defer manager.Close()
	if statuses[0].Err != nil {
		t.Fatal(statuses[0].Err)
	}
	available := manager.Tools()

	requests := 0
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []map[string]any
			Tools []map[string]any
		}
		json.NewDecoder(r.Body).Decode(&body)
		requests++
		switch requests {
		case 1:
			if len(body.Tools) != 1 || body.Tools[0]["name"] != "mcp_docs_query-docs" || body.Tools[0]["strict"] != false {
				t.Errorf("tools=%v", body.Tools)
			}
			streamEvent(w, functionCall("mcp_docs_query-docs", "q-1", map[string]string{"q": "useEffect"}))
		case 2:
			if out := lastOutput(t, body.Input, "q-1"); !strings.Contains(out, "useEffect runs after render") {
				t.Errorf("result=%s", out)
			}
			streamEvent(w, functionCall("mcp_docs_query-docs", "q-2", map[string]string{"q": "again"}))
		case 3:
			if out := lastOutput(t, body.Input, "q-2"); !strings.Contains(out, `"status":"denied"`) {
				t.Errorf("denied result=%s", out)
			}
			streamEvent(w, `{"type":"response.output_text.delta","delta":"It runs after render."}`)
			streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
		}
	}))
	defer model.Close()

	dir := t.TempDir()
	s, err := OpenSession(dir, available)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(Options{BaseURL: model.URL, APIKey: "test", Model: "test"}, s, available)
	if err != nil {
		t.Fatal(err)
	}
	var confirmations []Confirmation
	err = a.Handle(context.Background(), "When does useEffect run?", testInteraction{confirm: func(_ context.Context, c Confirmation) (bool, error) {
		confirmations = append(confirmations, c)
		return len(confirmations) == 1, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 3 || len(confirmations) != 2 || calls.Load() != 1 {
		t.Fatalf("requests=%d confirmations=%d server calls=%d", requests, len(confirmations), calls.Load())
	}
	fields := map[string]string{}
	for _, f := range confirmations[0].Fields {
		fields[f.Label] = f.Value
	}
	if fields["Server"] != "docs" || fields["Destination"] != service.URL || fields["Tool"] != "query-docs" || !strings.Contains(fields["Arguments"], "useEffect") {
		t.Fatalf("confirmation=%+v", confirmations[0])
	}
	s.Close()

	// With the server gone, the saved turn still loads; nothing is called again.
	s, err = OpenSession(dir, []tools.Tool{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if len(s.state.replayTurns) != 1 || calls.Load() != 1 {
		t.Fatal("MCP turn not recovered offline")
	}
}

type unknownTool struct{}

func (unknownTool) Definition() tools.Definition {
	return tools.Definition{Name: "mcp_remote_act", Parameters: map[string]any{"type": "object"}}
}
func (unknownTool) Prepare(raw string) (tools.Call, error) {
	return tools.Call{Arguments: json.RawMessage(raw)}, nil
}
func (unknownTool) Execute(context.Context, tools.Call) tools.Result {
	return tools.Result{Status: "unknown", Output: "The call may have reached the server."}
}

// A live unknown result is acknowledged before the next message, without a restart.
func TestLiveUnknownResultIsAcknowledgedBeforeNextMessage(t *testing.T) {
	requests := 0
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			streamEvent(w, functionCall("mcp_remote_act", "u-1", map[string]string{}))
			return
		}
		streamEvent(w, `{"type":"response.output_text.delta","delta":"ok"}`)
		streamEvent(w, `{"type":"response.completed","response":{"status":"completed"}}`)
	}))
	defer model.Close()
	available := []tools.Tool{unknownTool{}}
	s, err := OpenSession(t.TempDir(), available)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, err := New(Options{BaseURL: model.URL, APIKey: "test", Model: "test"}, s, available)
	if err != nil {
		t.Fatal(err)
	}
	approve := func(_ context.Context, c Confirmation) (bool, error) { return c.Kind == "tool", nil }
	if err := a.Handle(context.Background(), "Act", testInteraction{confirm: approve}); err != nil {
		t.Fatal(err)
	}
	if len(s.state.uncertain) != 1 || requests != 2 {
		t.Fatalf("uncertain=%v requests=%d", s.state.uncertain, requests)
	}
	seq := s.state.seq
	if err := a.Handle(context.Background(), "Next", testInteraction{}); err == nil || s.state.seq != seq || requests != 2 {
		t.Fatalf("declined acknowledgement sent the message: %v", err)
	}
	var kinds []string
	err = a.Handle(context.Background(), "Next", testInteraction{confirm: func(_ context.Context, c Confirmation) (bool, error) {
		kinds = append(kinds, c.Kind)
		return true, nil
	}})
	if err != nil || len(kinds) != 1 || kinds[0] != "recovery" || len(s.state.uncertain) != 0 || requests != 3 {
		t.Fatalf("err=%v kinds=%v requests=%d", err, kinds, requests)
	}
}
