package mino

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextWindowConfiguration(t *testing.T) {
	for _, value := range []string{"", "0", "64000", "-1", "1.5", `"128000"`, "null", "999999999999999999999999"} {
		t.Run("value="+value, func(t *testing.T) {
			path := isolateConfig(t)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			data := `{"base_url":"https://example.com/v1","api_key":"test","model":"test"`
			if value != "" {
				data += `,"context_window":` + value
			}
			data += "}"
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadConfig(nil)
			valid := value == "" || value == "0" || value == "64000"
			if !valid {
				if err == nil {
					t.Fatal("accepted invalid context_window")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := saveConfig(cfg); err != nil {
				t.Fatal(err)
			}
			dataBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var saved map[string]any
			if err := json.Unmarshal(dataBytes, &saved); err != nil {
				t.Fatal(err)
			}
			if value == "64000" && saved["context_window"] != float64(64000) {
				t.Fatal("configuration override was lost")
			}
		})
	}
}

func TestContextWindowStartupUsesLocalSettings(t *testing.T) {
	for _, window := range []int{0, 64000} {
		t.Run(fmt.Sprint(window), func(t *testing.T) {
			isolateConfig(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("startup queried model metadata") }))
			defer server.Close()
			if err := saveConfig(config{BaseURL: server.URL, APIKey: "test", Model: "test", ContextWindow: window}); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := run(context.Background(), strings.NewReader("/exit\n"), &output, &output); err != nil {
				t.Fatal(err)
			}
			want := "Context window: 128000 tokens (default)."
			if window > 0 {
				want = "Context window: 64000 tokens (config)."
			}
			if !strings.Contains(output.String(), want) {
				t.Fatalf("output=%q", output.String())
			}
		})
	}
}
