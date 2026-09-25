package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestBashThroughToolContract(t *testing.T) {
	var tool Tool = testBash(t)
	definition := tool.Definition()
	if definition.Name != "bash" || !definition.Strict {
		t.Fatal("missing Bash definition")
	}
	if _, err := tool.Prepare(`{"command":"printf ok","cwd":"/"}`); err == nil {
		t.Fatal("accepted extra arguments")
	}
	call, err := tool.Prepare(`{"command":"printf ok"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(call.Arguments) || call.Directory == "" || len(call.Fields) == 0 {
		t.Fatal("missing execution details")
	}
	result := tool.Execute(context.Background(), call)
	if result.Status != "completed" || result.Output != "ok" {
		t.Fatalf("result=%+v", result)
	}
}
