package eval

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func postResponses(t *testing.T, url, body string) (int, []map[string]any) {
	t.Helper()
	response, err := http.Post(url+"/responses", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var events []map[string]any
	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line[len("data: "):]), &event); err != nil {
			t.Fatalf("bad event %q: %v", line, err)
		}
		events = append(events, event)
	}
	return response.StatusCode, events
}

func TestFakeFollowsScriptAndReportsViolations(t *testing.T) {
	fake, err := StartFake([]Step{
		{Call: &ScriptCall{Tool: "bash", Arguments: json.RawMessage(`{"command":"ls"}`)}, ExpectInputContains: "zebra"},
		{Text: "Done."},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fake.Close()
	if response, err := http.Get(fake.URL() + "/models"); err != nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected route: %v %v", response, err)
	}
	status, events := postResponses(t, fake.URL(), `{"input":[{"role":"user","content":"List files"}]}`)
	if status != http.StatusOK || len(events) != 1 || events[0]["type"] != "response.completed" {
		t.Fatalf("call step: %d %v", status, events)
	}
	response := events[0]["response"].(map[string]any)
	item := response["output"].([]any)[0].(map[string]any)
	if item["type"] != "function_call" || item["call_id"] != "call_1" || item["name"] != "bash" || item["arguments"] != `{"command":"ls"}` || item["status"] != "completed" {
		t.Fatalf("function call item: %v", item)
	}
	if usage := response["usage"].(map[string]any); usage["input_tokens"].(float64) <= 0 || usage["total_tokens"].(float64) <= usage["input_tokens"].(float64) {
		t.Fatalf("usage: %v", usage)
	}
	if err := fake.Err(); err == nil || !strings.Contains(err.Error(), `"zebra"`) {
		t.Fatalf("expectation error: %v", err)
	}
	_, events = postResponses(t, fake.URL(), `{"input":[]}`)
	if len(events) != 2 || events[0]["type"] != "response.output_text.delta" || events[0]["delta"] != "Done." {
		t.Fatalf("text step: %v", events)
	}
	message := events[1]["response"].(map[string]any)["output"].([]any)[0].(map[string]any)
	if message["type"] != "message" || message["role"] != "assistant" || message["content"].([]any)[0].(map[string]any)["text"] != "Done." {
		t.Fatalf("message item: %v", message)
	}
	_, events = postResponses(t, fake.URL(), `{"input":[]}`)
	if len(events) != 1 || events[0]["type"] != "response.failed" || fake.Requests() != 3 {
		t.Fatalf("exhausted script: %v requests=%d", events, fake.Requests())
	}
	// The first violation is kept; exhaustion does not overwrite it.
	if !strings.Contains(fake.Err().Error(), "zebra") {
		t.Fatalf("first error replaced: %v", fake.Err())
	}
}
