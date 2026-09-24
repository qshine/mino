package mino

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

type modelReply struct {
	Text   string
	Output []json.RawMessage
}

func userInput(text string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemParamOfMessage(text, responses.EasyInputMessageRoleUser)
}

func (r modelReply) inputItems() (responses.ResponseInputParam, error) {
	if !validHistoryText(r.Text) {
		return nil, errors.New("The model returned no text")
	}
	if len(r.Output) == 0 {
		return responses.ResponseInputParam{responses.ResponseInputItemParamOfMessage(r.Text, responses.EasyInputMessageRoleAssistant)}, nil
	}
	items := make(responses.ResponseInputParam, 0, len(r.Output))
	hasMessage := false
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
		default:
			return nil, errors.New("This chapter supports text replies only; unsupported model output was not saved")
		}
		// Preserve the full output item, including phase and encrypted reasoning state.
		// https://developers.openai.com/api/docs/guides/conversation-state
		items = append(items, param.Override[responses.ResponseInputItemUnionParam](raw))
	}
	if !hasMessage {
		return nil, errors.New("Model output contains no assistant message")
	}
	return items, nil
}

type conversation struct {
	history *history
	client  *responsesClient
}

// A storage failure must stop the terminal even when cancellation happens concurrently.
type historyError struct{ err error }

func (e *historyError) Error() string { return e.err.Error() }
func (e *historyError) Unwrap() error { return e.err }

func (c *conversation) respond(ctx context.Context, prompt string, emit func(string) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validHistoryText(prompt) {
		return errors.New("Input must be non-empty UTF-8 text")
	}
	h := c.history
	id := newHistoryID()
	if err := h.append(historyRecord{TurnID: id, Kind: "user_message", Text: prompt}); err != nil {
		return &historyError{err}
	}
	input := append(append(responses.ResponseInputParam{}, h.state.input...), userInput(prompt))
	reply, err := c.client.respond(ctx, input, emit)
	if err != nil {
		status := "failed"
		if errors.Is(err, context.Canceled) {
			status = "cancelled"
		}
		if saveErr := h.append(historyRecord{TurnID: id, Kind: "turn_end", Status: status}); saveErr != nil {
			return &historyError{saveErr}
		}
		return err
	}
	if err := h.append(
		historyRecord{TurnID: id, Kind: "assistant_message", Text: reply.Text, Output: reply.Output},
		historyRecord{TurnID: id, Kind: "turn_end", Status: "completed"},
	); err != nil {
		return &historyError{fmt.Errorf("Answer displayed, but saving history could not be confirmed: %w", err)}
	}
	return nil
}
