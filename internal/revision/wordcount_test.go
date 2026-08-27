package revision

import (
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
)

func TestProjectorUsesNormalizedWordCount(t *testing.T) {
	st := newRevisionTestStore(t, 1)
	content := "\ufeff月面\r\n维修"
	records := []domain.ChapterRecord{
		testRecord(1, content, domain.ChapterFacts{
			Title: "第一章", Summary: "摘要", KeyEvents: []string{"事件"},
		}, domain.StyleDelta{}, time.Now()),
	}

	if err := NewProjector(st).Apply(records); err != nil {
		t.Fatal(err)
	}
	progress, err := st.Progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := progress.ChapterWordCounts[1]; got != domain.WordCount(content) {
		t.Fatalf("projected chapter word count = %d, want normalized count %d", got, domain.WordCount(content))
	}
}
