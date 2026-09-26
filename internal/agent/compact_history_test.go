package agent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedCompactTurns(t *testing.T, s *Session, count, size int) {
	t.Helper()
	for n := 0; n < count; n++ {
		id := newHistoryID()
		if err := s.append(
			historyRecord{TurnID: id, Kind: "user_message", Text: "Goal " + strings.Repeat("x", size)},
			historyRecord{TurnID: id, Kind: "model_response", Text: "Done " + strings.Repeat("y", size)},
			historyRecord{TurnID: id, Kind: "turn_end", Status: "completed"},
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompactionRecoversInterruptedTurnAfterLegacyRecords(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenSession(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	seedCompactTurns(t, s, 4, 200)
	s.Close()
	path := filepath.Join(dir, "history.jsonl")
	data, _ := os.ReadFile(path)
	data = bytes.ReplaceAll(data, []byte(`"v":3`), []byte(`"v":2`))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s, err = OpenSession(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.append(historyRecord{TurnID: newHistoryID(), Kind: "user_message", Text: "Interrupted new question."}, historyRecord{TurnID: newHistoryID(), Kind: "context_compaction", Text: "Earlier goals and work.", CoveredSeq: 6}); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenSession(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.state.summary != "Earlier goals and work." || len(s.state.input) != 4 || s.state.pending.id != "" || !strings.Contains(s.notice, "interrupted") {
		t.Fatal("crash recovery lost summary or replayed unfinished question")
	}
	restored, _ := os.ReadFile(path)
	if !bytes.HasPrefix(restored, data) {
		t.Fatal("legacy records rewritten")
	}
}

func TestCompactionHistoryRestoresSummaryAndRetainsOriginalRecords(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenSession(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedCompactTurns(t, s, 4, 200)
	original, _ := os.ReadFile(s.file.Name())
	if err := s.append(historyRecord{TurnID: newHistoryID(), Kind: "context_compaction", Text: "Goal and completed work.", CoveredSeq: 6}); err != nil {
		t.Fatal(err)
	}
	if len(s.state.input) != 4 || s.state.summary != "Goal and completed work." || s.state.coveredSeq != 6 {
		t.Fatalf("bad compacted state: %+v", s.state)
	}
	want, _ := json.Marshal(s.state.input)
	data, _ := os.ReadFile(s.file.Name())
	if !bytes.HasPrefix(data, original) || !bytes.Contains(data[len(original):], []byte(`"v":3`)) {
		t.Fatal("original records changed or summary version missing")
	}
	s.Close()
	s, err = OpenSession(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, _ := json.Marshal(s.state.input)
	if !bytes.Equal(got, want) || s.state.summary != "Goal and completed work." || s.state.coveredSeq != 6 {
		t.Fatal("restart did not restore compacted context")
	}
	seedCompactTurns(t, s, 1, 200)
	if err := s.append(historyRecord{TurnID: newHistoryID(), Kind: "context_compaction", Text: "Updated goal and work.", CoveredSeq: 9}); err != nil {
		t.Fatal(err)
	}
	if len(s.state.input) != 4 || s.state.summary != "Updated goal and work." {
		t.Fatal("repeated compaction duplicated old context")
	}
}

func TestCompactionHistoryRejectsInvalidCoverage(t *testing.T) {
	for _, mode := range []string{"missing", "middle of turn", "future", "old version", "extra fields", "unacknowledged", "unresolved call", "duplicate id", "repeat coverage"} {
		t.Run(mode, func(t *testing.T) {
			s, err := OpenSession(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			seedCompactTurns(t, s, 4, 50)
			r := historyRecord{Version: 3, Seq: s.state.seq + 1, TurnID: newHistoryID(), Kind: "context_compaction", Text: "Summary.", CoveredSeq: 6}
			switch mode {
			case "missing":
				r.CoveredSeq = 0
			case "middle of turn":
				r.CoveredSeq = 5
			case "future":
				r.CoveredSeq = 99
			case "old version":
				r.Version = 2
			case "extra fields":
				r.Status = "completed"
			case "unacknowledged":
				s.state.uncertain = map[string]bool{newHistoryID(): true}
			case "unresolved call":
				s.state.pending = pendingTurn{id: newHistoryID(), calls: []pendingCall{{call: toolCall{CallID: "a"}}}}
			case "duplicate id":
				for id := range s.state.turns {
					r.TurnID = id
					break
				}
			case "repeat coverage":
				if err := s.append(r); err != nil {
					t.Fatal(err)
				}
				r.Seq, r.TurnID = s.state.seq+1, newHistoryID()
			}
			before, _ := json.Marshal(s.state.input)
			if err := s.state.apply(r); err == nil {
				t.Fatal("invalid summary accepted")
			}
			after, _ := json.Marshal(s.state.input)
			if !bytes.Equal(before, after) {
				t.Fatal("invalid record changed context")
			}
		})
	}
}

func TestCompactionHistoryWriteFailurePreservesMemory(t *testing.T) {
	for _, mode := range []string{"write", "sync", "short"} {
		t.Run(mode, func(t *testing.T) {
			s, err := OpenSession(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			seedCompactTurns(t, s, 4, 50)
			before, _ := json.Marshal(s.state.input)
			s.writer = failingHistoryWriter{s.file, mode}
			if err := s.append(historyRecord{TurnID: newHistoryID(), Kind: "context_compaction", Text: "Summary.", CoveredSeq: 6}); err == nil {
				t.Fatal("write failure accepted")
			}
			after, _ := json.Marshal(s.state.input)
			if !bytes.Equal(before, after) || s.state.summary != "" || !s.broken {
				t.Fatal("memory committed after failed save")
			}
		})
	}
}
