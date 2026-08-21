package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
)

func TestChapterRecordLoadKeepsLegacyFactsWithoutKnowledgeUpdatesCompatible(t *testing.T) {
	st := newTestStore(t)
	content := "第一章旧正文。"
	legacy := map[string]any{
		"version": 1, "chapter": 1, "revision": 1, "origin": "generated",
		"content": content, "content_sha256": domain.ChapterContentSHA256(content),
		"facts": map[string]any{
			"title": "第一章", "summary": "旧摘要", "characters": []string{"林墨"},
			"key_events": []string{"旧事件"}, "timeline_events": []any{},
			"foreshadow_updates": []any{}, "relationship_changes": []any{},
			"state_changes": []any{}, "cast_intros": []any{},
		},
		"style_delta": map[string]any{"prose": []any{}, "dialogue": []any{}, "taboos": []any{}},
		"accepted_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(st.Dir(), ChapterRecordPath(1))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := st.ChapterRecords.Load(1)
	if err != nil {
		t.Fatalf("load legacy chapter record: %v", err)
	}
	if record == nil || len(record.Facts.KnowledgeUpdates) != 0 {
		t.Fatalf("legacy knowledge updates should decode empty: %+v", record)
	}
}
