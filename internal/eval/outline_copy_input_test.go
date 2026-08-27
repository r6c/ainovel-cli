package eval

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestOutlineCopyInputMatrixHasStableSameChapterPair(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	entry := domain.OutlineEntry{
		Chapter:   3,
		Title:     "北侧冷阱",
		CoreEvent: "苏弦前往北侧冷阱寻找第三枚中继器",
		Hook:      "冷阱里传来第二个求救信号",
		Scenes:    []string{"确认铜钥匙", "进入冷阱", "发现中继器"},
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{entry}); err != nil {
		t.Fatalf("SaveOutline: %v", err)
	}
	content := "苏弦前往北侧冷阱寻找第三枚中继器。"
	if _, err := st.ChapterRecords.Accept(3, domain.ChapterOriginGenerated, content, domain.ChapterFacts{
		Title: entry.Title, Summary: "苏弦进入冷阱", Characters: []string{"苏弦"}, KeyEvents: []string{entry.CoreEvent},
	}, domain.StyleDelta{}); err != nil {
		t.Fatalf("Accept: %v", err)
	}

	gotOutline, err := st.Outline.GetChapterOutline(3)
	if err != nil {
		t.Fatalf("GetChapterOutline: %v", err)
	}
	gotRecord, err := st.ChapterRecords.Load(3)
	if err != nil {
		t.Fatalf("LoadChapterRecord: %v", err)
	}
	if gotOutline == nil || gotRecord == nil {
		t.Fatal("same-chapter outline and record must both be readable")
	}
	if gotOutline.Chapter != gotRecord.Chapter {
		t.Fatalf("same-chapter pair must share chapter number: outline=%d record=%d", gotOutline.Chapter, gotRecord.Chapter)
	}
	if gotOutline.CoreEvent == "" || gotRecord.Content == "" {
		t.Fatal("same-chapter pair must contain comparable outline and prose text")
	}
}
