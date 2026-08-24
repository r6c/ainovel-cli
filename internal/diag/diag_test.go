package diag

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestAnalyzeAggregatesKnowledgeState(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{
		{
			ID: "k1", Truth: "真相一", EstablishedAt: 1, ReaderRevealedAt: 5,
			KnownBy: []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 3}, {Character: "苏晚", LearnedAt: 4}},
			BelievedBy: []domain.KnowledgeBelief{
				{Character: "阿云", Content: "误解一", FormedAt: 2},
				{Character: "老周", Content: "已纠正误解", FormedAt: 2, CorrectedAt: 6},
			},
		},
		{
			ID: "k2", Truth: "真相二", EstablishedAt: 2,
			KnownBy:    []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 7}},
			BelievedBy: []domain.KnowledgeBelief{{Character: "苏晚", Content: "误解二", FormedAt: 8}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	stats := Analyze(st).Stats
	if stats.KnowledgeFacts != 2 || stats.KnowledgeKnownBy != 3 || stats.KnowledgeReaderKnown != 1 || stats.KnowledgeActiveBeliefs != 2 {
		t.Fatalf("knowledge stats wrong: %+v", stats)
	}
}

func TestAnalyzeAggregatesKnowledgeWithoutProgress(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{{
		ID: "k", Truth: "真相", EstablishedAt: 1, ReaderRevealedAt: 2,
		KnownBy:    []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 2}},
		BelievedBy: []domain.KnowledgeBelief{{Character: "苏晚", Content: "误解", FormedAt: 2}},
	}}); err != nil {
		t.Fatal(err)
	}

	stats := Analyze(st).Stats
	if stats.KnowledgeFacts != 1 || stats.KnowledgeKnownBy != 1 || stats.KnowledgeReaderKnown != 1 || stats.KnowledgeActiveBeliefs != 1 {
		t.Fatalf("knowledge stats must survive missing progress: %+v", stats)
	}
}

func TestAnalyzeKnowledgeStateMissingIsEmptyAndCorruptIsLoadError(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		st := store.NewStore(t.TempDir())
		if err := st.Init(); err != nil {
			t.Fatal(err)
		}
		if err := st.Progress.Init(1); err != nil {
			t.Fatal(err)
		}
		report := Analyze(st)
		if report.Stats.KnowledgeFacts != 0 || report.Stats.KnowledgeKnownBy != 0 || report.Stats.KnowledgeReaderKnown != 0 || report.Stats.KnowledgeActiveBeliefs != 0 {
			t.Fatalf("missing knowledge state must remain zero: %+v", report.Stats)
		}
		for _, finding := range report.Findings {
			if finding.Rule == "LoadError" && strings.Contains(finding.Title, "knowledge_state") {
				t.Fatalf("missing knowledge state must not be a load error: %+v", finding)
			}
		}
	})

	t.Run("corrupt", func(t *testing.T) {
		st := store.NewStore(t.TempDir())
		if err := st.Init(); err != nil {
			t.Fatal(err)
		}
		if err := st.Progress.Init(1); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(st.Dir(), "knowledge_state.json"), []byte("{"), 0o644); err != nil {
			t.Fatal(err)
		}
		report := Analyze(st)
		if report.Stats.KnowledgeFacts != 0 || report.Stats.KnowledgeActiveBeliefs != 0 {
			t.Fatalf("corrupt knowledge state must not invent stats: %+v", report.Stats)
		}
		if !slices.ContainsFunc(report.Findings, func(f Finding) bool {
			return f.Rule == "LoadError" && strings.Contains(f.Title, "knowledge_state")
		}) {
			t.Fatalf("corrupt knowledge state must produce LoadError: %+v", report.Findings)
		}
	})
}

func TestAnalyzeFindsOnlyLongActiveKnowledgeBeliefs(t *testing.T) {
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
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{
		{ID: "stale", Truth: "绝不能出现在诊断证据里的作者真相", EstablishedAt: 1, BelievedBy: []domain.KnowledgeBelief{{Character: "林墨", Content: "绝不能出现在诊断证据里的错误内容", FormedAt: 2}}},
		{ID: "recent", Truth: "近期真相", EstablishedAt: 1, BelievedBy: []domain.KnowledgeBelief{{Character: "苏晚", Content: "近期误解", FormedAt: 28}}},
		{ID: "corrected", Truth: "已纠正真相", EstablishedAt: 1, BelievedBy: []domain.KnowledgeBelief{{Character: "老周", Content: "旧误解", FormedAt: 2, CorrectedAt: 5}}},
	}); err != nil {
		t.Fatal(err)
	}

	report := Analyze(st)
	var finding *Finding
	for i := range report.Findings {
		if report.Findings[i].Rule == "StaleKnowledgeBelief" {
			finding = &report.Findings[i]
			break
		}
	}
	if finding == nil {
		t.Fatal("expected stale knowledge belief finding")
	}
	if finding.Severity != SevInfo || finding.Confidence != ConfMedium || finding.AutoLevel != AutoNone {
		t.Fatalf("stale belief policy wrong: %+v", finding)
	}
	if !strings.Contains(finding.Evidence, "stale/林墨") || strings.Contains(finding.Evidence, "recent") || strings.Contains(finding.Evidence, "corrected") {
		t.Fatalf("finding must include only long-active belief identity: %+v", finding)
	}
	if strings.Contains(finding.Evidence, "作者真相") || strings.Contains(finding.Evidence, "错误内容") {
		t.Fatalf("finding leaked truth or belief content: %+v", finding)
	}
}

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
