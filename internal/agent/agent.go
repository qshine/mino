// Package agent owns the synchronous Agent loop and its durable session.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/qshine/mino/internal/tools"
)

const maxModelRequests = 8
const maxToolCalls = 16
const maxResponseBytes = 8 << 20

var errResponseTooLarge = errors.New("Response exceeds the 8 MiB limit")
var ErrBusy = errors.New("The session is already processing a message")

// Handler is the gateway's only access to the Agent and its session.
type Handler interface {
	Start(context.Context, Interaction) error
	Handle(context.Context, string, Interaction) error
}

type Interaction interface {
	Emit(Event) error
	Confirm(context.Context, Confirmation) (bool, error)
}

type Event struct {
	Kind   string // response_started, text, tool_result, notice
	Text   string
	Result tools.Result
}

type Confirmation struct {
	Kind     string // tool or recovery
	ToolName string
	Fields   []tools.Field
	Warning  string
}

type Options struct{ BaseURL, APIKey, Model, Instructions string }

type Agent struct {
	session     *Session
	options     Options
	api         responses.ResponseService
	httpClient  *http.Client
	tools       map[string]tools.Tool
	definitions []responses.ToolUnionParam
}

func New(options Options, session *Session, available []tools.Tool) (*Agent, error) {
	if session == nil {
		return nil, errors.New("Agent requires a session")
	}
	registry, err := toolRegistry(available)
	if err != nil {
		return nil, err
	}
	a := &Agent{session: session, options: options, tools: registry, httpClient: &http.Client{
		Timeout:       2 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
	for _, tool := range available {
		d := tool.Definition()
		definition := responses.ToolParamOfFunction(d.Name, d.Parameters, d.Strict)
		definition.OfFunction.Description = openai.String(d.Description)
		a.definitions = append(a.definitions, definition)
	}
	// The service constructor does not load OPENAI_* environment defaults.
	a.api = responses.NewResponseService(option.WithBaseURL(options.BaseURL), option.WithAPIKey(options.APIKey), option.WithHTTPClient(a.httpClient), option.WithMaxRetries(0), option.WithMiddleware(checkResponse))
	return a, nil
}

func toolRegistry(available []tools.Tool) (map[string]tools.Tool, error) {
	registry := make(map[string]tools.Tool, len(available))
	for _, tool := range available {
		if tool == nil {
			return nil, errors.New("Tool must not be nil")
		}
		name := tool.Definition().Name
		if strings.TrimSpace(name) == "" || registry[name] != nil {
			return nil, errors.New("Tool names must be non-empty and unique")
		}
		registry[name] = tool
	}
	return registry, nil
}

func (a *Agent) Start(ctx context.Context, interaction Interaction) error {
	if !a.session.busy.TryLock() {
		return ErrBusy
	}
	defer a.session.busy.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.session.notice != "" {
		if err := interaction.Emit(Event{Kind: "notice", Text: a.session.notice}); err != nil {
			return err
		}
	}
	if len(a.session.state.uncertain) == 0 {
		return nil
	}
	approved, err := interaction.Confirm(ctx, Confirmation{Kind: "recovery", Warning: "A previous command started, but its result is unknown. It may have changed files or external systems. Mino has not rerun it. Review those effects before continuing."})
	if err != nil {
		return err
	}
	if !approved {
		return io.EOF
	}
	if err := a.session.acknowledgeRecovery(); err != nil {
		return &StorageError{err}
	}
	return nil
}

// Handle makes the user message durable before entering runLoop. Session serializes
// the complete turn, including approvals, so different gateways cannot interleave it.
func (a *Agent) Handle(ctx context.Context, message string, interaction Interaction) error {
	if !a.session.busy.TryLock() {
		return ErrBusy
	}
	defer a.session.busy.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validHistoryText(message) {
		return errors.New("Input must be non-empty UTF-8 text")
	}
	if len(a.session.state.uncertain) > 0 {
		return errors.New("Acknowledge unknown tool results before continuing")
	}
	if err := a.session.append(historyRecord{TurnID: newHistoryID(), Kind: "user_message", Text: message}); err != nil {
		return &StorageError{err}
	}
	return a.runLoop(ctx, interaction)
}

// runLoop keeps each model request, completed response, and tool step in one place.
func (a *Agent) runLoop(ctx context.Context, interaction Interaction) error {
	s := a.session
	id := s.state.pending.id
	for request := 1; request <= maxModelRequests; request++ {
		if err := ctx.Err(); err != nil {
			return a.stopTurn(err)
		}
		if err := interaction.Emit(Event{Kind: "response_started"}); err != nil {
			return a.stopTurn(err)
		}
		input := append(append(responses.ResponseInputParam{}, s.state.input...), userInput(s.state.pending.prompt))
		input = append(input, s.state.pending.items...)

		var response *http.Response
		stream := a.api.NewStreaming(ctx, responses.ResponseNewParams{
			Model: a.options.Model, Instructions: openai.String(a.options.Instructions),
			Input: responses.ResponseNewParamsInputUnion{OfInputItemList: input},
			Store: openai.Bool(false), Include: []responses.ResponseIncludable{responses.ResponseIncludableReasoningEncryptedContent},
			Tools: a.definitions, ParallelToolCalls: openai.Bool(false),
		}, option.WithResponseInto(&response))
		var reply modelReply
		var text strings.Builder
		var responseErr error
		completed, refusing := false, false
		for stream.Next() {
			event := stream.Current()
			switch event.Type {
			case "response.output_text.delta", "response.refusal.delta":
				text.WriteString(event.Delta)
				delta := event.Delta
				if event.Type == "response.refusal.delta" && !refusing {
					delta = "Model refused: " + delta
					refusing = true
				}
				responseErr = interaction.Emit(Event{Kind: "text", Text: delta})
			case "response.completed":
				result := event.Response
				rawError := result.JSON.Error.Raw()
				if result.Status == "failed" || (rawError != "" && rawError != "null") {
					responseErr = errors.New("Model generation failed. Check the service status and model settings.")
				} else if result.Status != "completed" {
					responseErr = errors.New("Model response is incomplete. Try again or shorten your question.")
				} else {
					reply.Text = text.String()
					for _, item := range result.Output {
						reply.Output = append(reply.Output, json.RawMessage(item.RawJSON()))
					}
					_, responseErr = reply.inputItems()
					completed = responseErr == nil
				}
			case "response.failed", "error":
				responseErr = errors.New("Model generation failed. Check the service status and model settings.")
			case "response.incomplete":
				responseErr = errors.New("Model response is incomplete. Try again or shorten your question.")
			}
			if responseErr != nil || completed {
				break
			}
		}
		if responseErr == nil && !completed {
			responseErr = streamFailure(stream.Err(), response)
		}
		stream.Close()
		if responseErr != nil {
			return a.stopTurn(responseErr)
		}

		calls := reply.calls()
		for _, call := range calls {
			for _, old := range s.state.pending.calls {
				if old.call.CallID == call.CallID {
					return a.stopTurn(errors.New("Model reused a tool call identifier; no calls from this response were executed"))
				}
			}
		}
		// Sync before asking for approval or acting on any tool request.
		if err := s.append(historyRecord{TurnID: id, Kind: "model_response", Text: reply.Text, Output: reply.Output}); err != nil {
			return &StorageError{fmt.Errorf("Answer displayed or response received, but saving history could not be confirmed: %w", err)}
		}
		if len(calls) == 0 {
			if err := s.finishTurn("completed"); err != nil {
				return &StorageError{err}
			}
			return nil
		}
		if request == maxModelRequests || len(s.state.pending.calls) > maxToolCalls {
			return a.stopTurn(errors.New("Tool limit reached."))
		}
		for _, call := range calls {
			if err := ctx.Err(); err != nil {
				return a.stopTurn(err)
			}
			result, err := a.runCall(ctx, id, call, interaction)
			if err != nil {
				var storage *StorageError
				if errors.As(err, &storage) {
					return err
				}
				return a.stopTurn(err)
			}
			if err := s.append(historyRecord{TurnID: id, Kind: "tool_result", CallID: call.CallID, Result: &result}); err != nil {
				return &StorageError{fmt.Errorf("Tool result was not reliably saved. The command will not be retried: %w", err)}
			}
			if err := interaction.Emit(Event{Kind: "tool_result", Result: result}); err != nil {
				return a.stopTurn(err)
			}
			if ctx.Err() != nil || result.Status == "cancelled" {
				return a.stopTurn(context.Canceled)
			}
		}
	}
	return a.stopTurn(errors.New("Tool limit reached."))
}

func (a *Agent) runCall(ctx context.Context, id string, call toolCall, interaction Interaction) (tools.Result, error) {
	tool, ok := a.tools[call.Name]
	if !ok {
		return tools.Result{Status: "unknown_tool", Output: "Unknown tool. Use a tool from the supplied definitions."}, nil
	}
	prepared, err := tool.Prepare(call.Arguments)
	if err != nil {
		return tools.Result{Status: "invalid_arguments", Output: err.Error()}, nil
	}
	// Give the gateway a copy of display data, never the executable arguments.
	approved, err := interaction.Confirm(ctx, Confirmation{Kind: "tool", ToolName: call.Name, Fields: append([]tools.Field(nil), prepared.Fields...), Warning: prepared.Warning})
	if err != nil {
		return tools.Result{}, err
	}
	if !approved {
		return tools.Result{Status: "denied", Output: "The user did not approve this command. Do not retry without a new user request."}, nil
	}
	if err := ctx.Err(); err != nil {
		return tools.Result{}, err
	}
	if err := a.session.append(historyRecord{TurnID: id, Kind: "tool_start", CallID: call.CallID, Name: call.Name, Arguments: prepared.Arguments, CWD: prepared.Directory}); err != nil {
		return tools.Result{}, &StorageError{err}
	}
	return tool.Execute(ctx, prepared), nil
}

func (a *Agent) stopTurn(cause error) error {
	status := "failed"
	if errors.Is(cause, context.Canceled) || errors.Is(cause, io.EOF) {
		status = "cancelled"
	}
	if err := a.session.finishTurn(status); err != nil {
		return &StorageError{err}
	}
	return cause
}

// StorageError tells any gateway that continuing could lose durable conversation state.
type StorageError struct{ err error }

func (e *StorageError) Error() string { return e.err.Error() }
func (e *StorageError) Unwrap() error { return e.err }

func streamFailure(err error, response *http.Response) error {
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("Request timed out: %w", err)
	case errors.Is(err, errResponseTooLarge):
		return errResponseTooLarge
	case response != nil && (response.StatusCode < 200 || response.StatusCode >= 300):
		return fmt.Errorf("Responses API returned HTTP %d (%s). Check the URL, API key, model access, or quota.", response.StatusCode, http.StatusText(response.StatusCode))
	case err != nil && response != nil:
		return errors.New("The server returned an invalid or unreadable Responses API stream. Check that your endpoint supports streaming (text/event-stream).")
	case err != nil:
		return errors.New("Request failed. Check your connection and base_url in ~/.mino/config.json.")
	default:
		return errors.New("Model response is incomplete: the stream ended before completion. Try again.")
	}
}

type modelReply struct {
	Text   string
	Output []json.RawMessage
}

func userInput(text string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemParamOfMessage(text, responses.EasyInputMessageRoleUser)
}

func (r modelReply) inputItems() (responses.ResponseInputParam, error) {
	if len(r.Output) == 0 {
		if !validHistoryText(r.Text) {
			return nil, errors.New("The model returned no text or tool calls")
		}
		return responses.ResponseInputParam{responses.ResponseInputItemParamOfMessage(r.Text, responses.EasyInputMessageRoleAssistant)}, nil
	}
	items := make(responses.ResponseInputParam, 0, len(r.Output))
	hasMessage, hasCall := false, false
	seen := make(map[string]bool)
	for _, raw := range r.Output {
		var item struct {
			Type, Role, Status, Phase string
			Content                   []struct{ Type, Text, Refusal string }
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, errors.New("Invalid model output in history")
		}
		switch item.Type {
		case "message":
			if item.Role != "assistant" {
				return nil, errors.New("History output must contain only assistant messages")
			}
			if (item.Status != "" && item.Status != "completed") || (item.Phase != "" && item.Phase != "commentary" && item.Phase != "final_answer") {
				return nil, errors.New("Invalid assistant status or phase in history")
			}
			hasContent := false
			for _, part := range item.Content {
				switch part.Type {
				case "output_text":
					hasContent = hasContent || validHistoryText(part.Text)
				case "refusal":
					hasContent = hasContent || validHistoryText(part.Refusal)
				default:
					return nil, errors.New("Unsupported assistant content in history")
				}
			}
			if !hasContent {
				return nil, errors.New("Assistant message in history has no text")
			}
			hasMessage = true
		case "reasoning":
			if item.Role != "" {
				return nil, errors.New("Invalid reasoning item in history")
			}
		case "function_call":
			var call toolCall
			if err := json.Unmarshal(raw, &call); err != nil || !validHistoryText(call.CallID) || len(call.CallID) > 512 || !validHistoryText(call.Name) || seen[call.CallID] || (item.Status != "" && item.Status != "completed") {
				return nil, errors.New("Invalid or duplicate tool call identifier, name, or status")
			}
			seen[call.CallID], hasCall = true, true
		default:
			return nil, errors.New("Unsupported model output; this endpoint must support Responses function calls and stateless replay")
		}
		// Preserve the full output item, including phase and encrypted reasoning state.
		// https://developers.openai.com/api/docs/guides/conversation-state
		items = append(items, param.Override[responses.ResponseInputItemUnionParam](raw))
	}
	if !hasMessage && !hasCall {
		return nil, errors.New("The model returned no text or tool calls")
	}
	return items, nil
}

type toolCall struct {
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Calls are extracted only from the validated, completed response.
func (r modelReply) calls() []toolCall {
	var calls []toolCall
	for _, raw := range r.Output {
		var item struct {
			Type string
			toolCall
		}
		if json.Unmarshal(raw, &item) == nil && item.Type == "function_call" {
			calls = append(calls, item.toolCall)
		}
	}
	return calls
}

// checkResponse enforces HTTP limits without buffering the stream before display.
func checkResponse(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	response, err := next(req)
	if err != nil {
		return response, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Do not read or display a server error body, which may contain private input.
		response.Body.Close()
		return response, errors.New("Unexpected HTTP status")
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/event-stream" {
		response.Body.Close()
		return response, errors.New("Expected a Responses API stream")
	}
	response.Body = &limitedResponseBody{ReadCloser: response.Body, remaining: maxResponseBytes}
	return response, nil
}

type limitedResponseBody struct {
	io.ReadCloser
	remaining int
}

func (b *limitedResponseBody) Read(p []byte) (int, error) {
	// Read at most one byte past the limit to distinguish exact size from overflow.
	if len(p) > b.remaining+1 {
		p = p[:b.remaining+1]
	}
	n, err := b.ReadCloser.Read(p)
	if n > b.remaining {
		return 0, errResponseTooLarge
	}
	b.remaining -= n
	return n, err
}
