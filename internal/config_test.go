package mino

import (
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	for _, tc := range []struct {
		name, baseURL, key, model, wantURL, wantError string
	}{
		{"official", "https://api.openai.com/v1", "test-key", "test-model", "https://api.openai.com/v1", ""},
		{"custom", " https://gateway.example/api/v1/ ", " test-key ", " test-model ", "https://gateway.example/api/v1", ""},
		{"local", "http://127.0.0.1:8080/v1", "test-key", "test-model", "http://127.0.0.1:8080/v1", ""},
		{"missing key", "", " ", "test-model", "", "api_key"},
		{"missing model", "", "test-key", "", "", "model"},
		{"relative URL", "/v1", "test-key", "test-model", "", "base_url"},
		{"slashes only", "///", "test-key", "test-model", "", "base_url"},
		{"invalid URL", "https://%", "test-key", "test-model", "", "base_url"},
		{"URL credentials", "https://user:password@example.com/v1", "test-key", "test-model", "", "base_url"},
		{"query", "https://example.com/v1?key=secret", "test-key", "test-model", "", "base_url"},
		{"fragment", "https://example.com/v1#fragment", "test-key", "test-model", "", "base_url"},
		{"remote HTTP", "http://example.com/v1", "test-key", "test-model", "", "HTTPS"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config{tc.baseURL, tc.key, tc.model, 0}
			err := cfg.validate()
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "secret") {
					t.Fatalf("error exposed URL credentials: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.BaseURL != tc.wantURL || cfg.APIKey != "test-key" || cfg.Model != "test-model" {
				t.Fatal("configuration does not match the expected values")
			}
		})
	}
}
