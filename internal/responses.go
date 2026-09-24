package mino

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

const maxResponseBytes = 8 << 20 // 8 MiB，避免异常服务返回无限响应。

var errResponseTooLarge = errors.New("Response exceeds the 8 MiB limit")

type responsesClient struct {
	config       config
	instructions string
	httpClient   *http.Client
	api          responses.ResponseService
}

func newResponsesClient(cfg config, instructions string) *responsesClient {
	client := &responsesClient{
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

// Responses API 将 instructions 用作系统/开发者指令，input 字符串表示本轮用户输入。
// https://developers.openai.com/api/docs/guides/text
func (c *responsesClient) respond(ctx context.Context, prompt string) (string, error) {
	var response *http.Response
	result, err := c.api.New(ctx, responses.ResponseNewParams{
		Model:        c.config.Model,
		Instructions: openai.String(c.instructions),
		Input:        responses.ResponseNewParamsInputUnion{OfString: openai.String(prompt)},
		Store:        openai.Bool(false),
	}, option.WithResponseInto(&response))
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return "", context.Canceled
		case errors.Is(err, context.DeadlineExceeded):
			return "", fmt.Errorf("Request timed out: %w", context.DeadlineExceeded)
		case errors.Is(err, errResponseTooLarge):
			return "", errResponseTooLarge
		case response != nil && (response.StatusCode < 200 || response.StatusCode >= 300):
			return "", fmt.Errorf("Responses API returned HTTP %d (%s). Check the URL, API key, model access, or quota.", response.StatusCode, http.StatusText(response.StatusCode))
		case response != nil:
			return "", fmt.Errorf("The server returned invalid or unreadable Responses API JSON")
		default:
			// SDK errors can include request URLs, response bodies, or credentials.
			return "", fmt.Errorf("Request failed. Check your connection and base_url in %s.", configLocation)
		}
	}
	if result == nil {
		return "", fmt.Errorf("Model response is incomplete. Try again or shorten your question.")
	}
	responseError := result.JSON.Error.Raw()
	if result.Status == "failed" || (responseError != "" && responseError != "null") {
		return "", fmt.Errorf("Model generation failed. Check the service status and model settings.")
	}
	if result.Status != "completed" {
		return "", fmt.Errorf("Model response is incomplete. Try again or shorten your question.")
	}
	var text strings.Builder
	for _, item := range result.Output {
		if item.Type != "message" || item.Role != "assistant" {
			continue
		}
		for _, part := range item.Content {
			switch part.Type {
			case "output_text":
				text.WriteString(part.Text)
			case "refusal":
				text.WriteString("Model refused: " + part.Refusal)
			}
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return "", fmt.Errorf("The model returned no text")
	}
	return text.String(), nil
}

// checkResponse enforces Mino's HTTP limits before the SDK decodes the response.
func checkResponse(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	response, err := next(req)
	if err != nil {
		return response, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Do not read or display a server error body, which may contain private input.
		return response, errors.New("Unexpected HTTP status")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return response, err
	}
	if len(data) > maxResponseBytes {
		return response, errResponseTooLarge
	}
	if !json.Valid(data) {
		return response, errors.New("Invalid Responses API JSON")
	}
	response.Body = io.NopCloser(bytes.NewReader(data))
	// Compatible endpoints sometimes omit the JSON content type.
	response.Header.Set("Content-Type", "application/json")
	return response, nil
}
