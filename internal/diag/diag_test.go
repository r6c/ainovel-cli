package diag

import (
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestAnalyzeMeasuresForeshadowStalenessFromLastAdvance(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(30); err != nil {
		t.Fatal(err)
	}
	for chapter := 1; chapter <= 30; chapter++ {
		if err := st.Progress.MarkChapterComplete(chapter, 1000, "", ""); err != nil {
			t.Fatalf("complete chapter %d: %v", chapter, err)
		}
	}
	if err := st.World.SaveForeshadowLedger([]domain.ForeshadowEntry{
		{ID: "stale", Description: "长期未推进", PlantedAt: 1, Status: "advanced", LastAdvancedAt: 2},
		{ID: "recent", Description: "近期刚强化", PlantedAt: 1, Status: "reinforced", LastAdvancedAt: 28},
	}); err != nil {
		t.Fatal(err)
	}

	report := Analyze(st)
	if report.Stats.ForeshadowStale != 1 {
		t.Fatalf("want 1 stale foreshadow, got %d", report.Stats.ForeshadowStale)
	}
	var stale *Finding
	for i := range report.Findings {
		if report.Findings[i].Rule == "StaleForeshadow" {
			stale = &report.Findings[i]
			break
		}
	}
	if stale == nil {
		t.Fatal("expected stale foreshadow finding")
	}
	if !strings.Contains(stale.Evidence, "stale") || strings.Contains(stale.Evidence, "recent") {
		t.Fatalf("stale finding should include only the long-unadvanced entry: %+v", stale)
	}
}

func TestAnalyzeCountsAllUnresolvedForeshadowAsOpen(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveForeshadowLedger([]domain.ForeshadowEntry{
		{ID: "planted", Description: "新伏笔", PlantedAt: 1, Status: "planted"},
		{ID: "advanced", Description: "已推进", PlantedAt: 1, Status: "advanced", LastAdvancedAt: 2},
		{ID: "reinforced", Description: "已强化", PlantedAt: 1, Status: "reinforced", LastAdvancedAt: 3},
		{ID: "partially_paid", Description: "部分兑现", PlantedAt: 1, Status: "partially_paid", LastAdvancedAt: 4},
		{ID: "resolved", Description: "已回收", PlantedAt: 1, Status: "resolved", ResolvedAt: 5},
	}); err != nil {
		t.Fatal(err)
	}

	report := Analyze(st)
	if report.Stats.ForeshadowOpen != 4 {
		t.Fatalf("want 4 open foreshadows, got %d", report.Stats.ForeshadowOpen)
	}
}
