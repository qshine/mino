package agent

import (
	"context"
	"errors"
	"fmt"
	"github.com/openai/openai-go/v3"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// Options contains only the explicitly configured model settings.
type Options struct{ BaseURL, APIKey, Model string }

const maxResponseBytes = 8 << 20 // 8 MiB，避免异常服务返回无限响应。

var errResponseTooLarge = errors.New("Response exceeds the 8 MiB limit")

type Agent struct {
	config       Options
	instructions string
	httpClient   *http.Client
	api          responses.ResponseService
}

func New(cfg Options, instructions string) *Agent {
	client := &Agent{
		config:       cfg,
		instructions: instructions,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
			// 不把包含密钥和问题的请求转发到重定向地址。
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
	// The Responses service constructor does not load OPENAI_* environment defaults.
	client.api = responses.NewResponseService(
		option.WithBaseURL(cfg.BaseURL),
		option.WithAPIKey(cfg.APIKey),
		option.WithHTTPClient(client.httpClient),
		option.WithMaxRetries(0), // Chapter 01 sends one request per question.
		option.WithMiddleware(checkResponse),
	)
	return client
}

// Handle sends one independent request and delivers each text delta immediately.
// https://developers.openai.com/api/docs/guides/streaming-responses
func (c *Agent) Handle(ctx context.Context, prompt string, emit func(string) error) error {
	var response *http.Response
	stream := c.api.NewStreaming(ctx, responses.ResponseNewParams{
		Model:        c.config.Model,
		Instructions: openai.String(c.instructions),
		Input:        responses.ResponseNewParamsInputUnion{OfString: openai.String(prompt)},
		Store:        openai.Bool(false),
	}, option.WithResponseInto(&response))
	defer stream.Close()

	hasText, refusing := false, false
	for stream.Next() {
		event := stream.Current()
		switch event.Type {
		case "response.output_text.delta", "response.refusal.delta":
			if strings.TrimSpace(event.Delta) != "" {
				hasText = true
			}
			delta := event.Delta
			if event.Type == "response.refusal.delta" && !refusing {
				delta = "Model refused: " + delta
				refusing = true
			}
			if err := emit(delta); err != nil {
				return err
			}
		case "response.completed":
			result := event.Response
			responseError := result.JSON.Error.Raw()
			if result.Status == "failed" || (responseError != "" && responseError != "null") {
				return errors.New("Model generation failed. Check the service status and model settings.")
			}
			if result.Status != "completed" {
				return errors.New("Model response is incomplete. Try again or shorten your question.")
			}
			if !hasText {
				return errors.New("The model returned no text")
			}
			return nil
		case "response.failed", "error":
			return errors.New("Model generation failed. Check the service status and model settings.")
		case "response.incomplete":
			return errors.New("Model response is incomplete. Try again or shorten your question.")
		}
	}
	if err := stream.Err(); err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return context.Canceled
		case errors.Is(err, context.DeadlineExceeded):
			return fmt.Errorf("Request timed out: %w", context.DeadlineExceeded)
		case errors.Is(err, errResponseTooLarge):
			return errResponseTooLarge
		case response != nil && (response.StatusCode < 200 || response.StatusCode >= 300):
			return fmt.Errorf("Responses API returned HTTP %d (%s). Check the URL, API key, model access, or quota.", response.StatusCode, http.StatusText(response.StatusCode))
		case response != nil:
			return errors.New("The server returned an invalid or unreadable Responses API stream. Check that your endpoint supports streaming (text/event-stream).")
		default:
			// SDK errors can include request URLs, response bodies, or credentials.
			return fmt.Errorf("Request failed. Check your connection and base_url in %s.", "~/.mino/config.json")
		}
	}
	// EOF or [DONE] alone does not establish that generation succeeded.
	return errors.New("Model response is incomplete: the stream ended before completion. Try again.")
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
