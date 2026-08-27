package store

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
)

func TestLoadChapterContentUsesNormalizedWordCount(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	content := "\ufeff月面\r\n维修"
	if err := st.Drafts.SaveDraft(1, content); err != nil {
		t.Fatal(err)
	}

	gotContent, gotCount, err := st.Drafts.LoadChapterContent(1)
	if err != nil {
		t.Fatal(err)
	}
	if gotContent != content {
		t.Fatalf("content changed while reading draft: %q", gotContent)
	}
	if gotCount != domain.WordCount(content) {
		t.Fatalf("word count = %d, want normalized count %d", gotCount, domain.WordCount(content))
	}
}
