package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxResponseBytes = 8 << 20 // 8 MiB，避免异常服务返回无限响应。

type responsesClient struct {
	config       config
	instructions string
	httpClient   *http.Client
}

func newResponsesClient(cfg config, instructions string) *responsesClient {
	return &responsesClient{
		config:       cfg,
		instructions: instructions,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
			// 不把包含密钥和问题的请求转发到重定向地址。
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// Responses API 将 instructions 用作系统/开发者指令，input 字符串表示本轮用户输入。
// https://developers.openai.com/api/docs/guides/text
func (c *responsesClient) respond(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(struct {
		Model        string `json:"model"`
		Instructions string `json:"instructions"`
		Input        string `json:"input"`
		Store        bool   `json:"store"`
	}{c.config.Model, c.instructions, prompt, false})
	if err != nil {
		return "", fmt.Errorf("Failed to encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("Failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Request failed. Check your connection and base_url in %s: %w", configLocation, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 服务端错误正文可能包含密钥或私密输入，不直接打印。
		return "", fmt.Errorf("Responses API returned HTTP %d (%s). Check the URL, API key, model access, or quota.", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("Failed to read response: %w", err)
	}
	if len(data) > maxResponseBytes {
		return "", fmt.Errorf("Response exceeds the 8 MiB limit")
	}
	var result struct {
		Status string          `json:"status"`
		Error  json.RawMessage `json:"error"`
		Output []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Refusal string `json:"refusal"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("The server returned invalid Responses API JSON")
	}
	if result.Status == "failed" || (len(result.Error) > 0 && string(result.Error) != "null") {
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
