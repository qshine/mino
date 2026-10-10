package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qshine/mino/internal/skills"
)

func TestSkillViewArgumentsAndResult(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".agents", "skills", "review")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: review\ndescription: Review code\n---\nRead tests first."), 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := skills.Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	view := NewSkillView(catalog)
	for _, raw := range []string{`{}`, `null`, `{"name":"../review"}`, `{"name":"review","name":"review"}`, `{"name":"review","path":"secret"}`, `{"name":1}`, `{"name":"review"} {}`} {
		if _, err := view.Prepare(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	call, err := view.Prepare(`{"name":"review"}`)
	if err != nil || !call.ReadOnly {
		t.Fatalf("prepare: %+v, %v", call, err)
	}
	result := view.Execute(context.Background(), call)
	if result.Status != "completed" || !strings.Contains(result.Output, "Read tests first.") {
		t.Fatalf("result: %+v", result)
	}
	call, err = view.Prepare(`{"name":"missing"}`)
	if err != nil {
		t.Fatal("history preparation must not depend on today's catalog")
	}
	if result := view.Execute(context.Background(), call); result.Status != "failed" {
		t.Fatalf("missing: %+v", result)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result := view.Execute(ctx, call); result.Status != "cancelled" {
		t.Fatalf("cancelled: %+v", result)
	}
}
