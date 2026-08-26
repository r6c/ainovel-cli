package imp

import (
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
)

func TestValidateImportedFactSequenceRejectsBeliefAfterLearnAcrossBatches(t *testing.T) {
	facts := []ImportedChapterFacts{
		{
			Chapter: 1, Title: "第一章", Summary: "建立并获知", KeyEvents: []string{"身份确认"},
			HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{
				{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长"},
				{ID: "k_shadow", Action: "learn", Character: "林墨"},
			},
		},
		{
			Chapter: 2, Title: "第二章", Summary: "错误误信", KeyEvents: []string{"误认身份"},
			HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{
				{ID: "k_shadow", Action: "believe", Character: "林墨", Belief: "黑影是杀兄仇人"},
			},
		},
	}

	if err := validateImportedFactSequence(facts); err == nil {
		t.Fatal("expected full-book fact replay to reject belief after learn")
	}
}

func TestBuildLedgerIncludesReaderRevealState(t *testing.T) {
	ledger := buildLedger([]ImportedChapterFacts{
		{Chapter: 1, KnowledgeUpdates: []domain.KnowledgeUpdate{{
			ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长",
		}}},
		{Chapter: 2, KnowledgeUpdates: []domain.KnowledgeUpdate{{
			ID: "k_shadow", Action: "reveal_to_reader",
		}}},
	})
	if !strings.Contains(ledger, "k_shadow") || !strings.Contains(ledger, "读者已知") {
		t.Fatalf("knowledge ledger missing reader reveal state:\n%s", ledger)
	}
}

func TestBuildLedgerTracksActiveAndCorrectedBeliefs(t *testing.T) {
	active := buildLedger([]ImportedChapterFacts{
		{Chapter: 1, KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "黑影是兄长"}}},
		{Chapter: 2, KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "believe", Character: "林墨", Belief: "黑影是仇人"}}},
	})
	if !strings.Contains(active, "林墨误信：黑影是仇人") {
		t.Fatalf("ledger missing active belief:\n%s", active)
	}
	corrected := buildLedger([]ImportedChapterFacts{
		{Chapter: 1, KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "黑影是兄长"}}},
		{Chapter: 2, KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "believe", Character: "林墨", Belief: "黑影是仇人"}}},
		{Chapter: 3, KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "learn", Character: "林墨"}}},
	})
	if strings.Contains(corrected, "林墨误信") || !strings.Contains(corrected, "已知角色：林墨") {
		t.Fatalf("learn did not clear active belief in ledger:\n%s", corrected)
	}
}

func TestBuildLedgerIncludesKnowledgeContinuity(t *testing.T) {
	ledger := buildLedger([]ImportedChapterFacts{
		{Chapter: 1, KnowledgeUpdates: []domain.KnowledgeUpdate{{
			ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长",
		}}},
		{Chapter: 2, KnowledgeUpdates: []domain.KnowledgeUpdate{{
			ID: "k_shadow", Action: "learn", Character: "林墨",
		}}},
	})
	for _, want := range []string{"k_shadow", "黑影是林墨的兄长", "林墨"} {
		if !strings.Contains(ledger, want) {
			t.Fatalf("knowledge ledger missing %q:\n%s", want, ledger)
		}
	}
}

func TestValidateBatchReaderRevealOnlyAcceptsID(t *testing.T) {
	_, seg := analyzeFixture(t, 1)
	for _, update := range []domain.KnowledgeUpdate{
		{ID: "k_shadow", Action: "reveal_to_reader", Truth: "不应携带的真相"},
		{ID: "k_shadow", Action: "reveal_to_reader", Character: "林墨"},
	} {
		batch := &AnalysisBatchResult{Chapters: []ImportedChapterFacts{{
			Chapter: 1, Summary: "揭示", CoreEvent: "揭示", HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{
				{ID: "k_shadow", Action: "establish", Truth: "真相"}, update,
			},
		}}}
		if err := validateBatch(batch, seg, 0, 1); err == nil {
			t.Fatalf("expected import reader reveal with extra fields to fail: %+v", update)
		}
	}
}

func TestValidateBatchReaderRevealContinuity(t *testing.T) {
	_, seg := analyzeFixture(t, 1)
	valid := &AnalysisBatchResult{Chapters: []ImportedChapterFacts{{
		Chapter: 1, Summary: "揭示真相", CoreEvent: "读者得知身份", HookType: "mystery", DominantStrand: "quest",
		KnowledgeUpdates: []domain.KnowledgeUpdate{
			{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长"},
			{ID: "k_shadow", Action: "reveal_to_reader"},
		},
	}}}
	if err := validateBatch(valid, seg, 0, 1); err != nil {
		t.Fatalf("establish then reader reveal should be valid: %v", err)
	}
	unknown := &AnalysisBatchResult{Chapters: []ImportedChapterFacts{{
		Chapter: 1, Summary: "错误揭示", CoreEvent: "错误引用", HookType: "mystery", DominantStrand: "quest",
		KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_missing", Action: "reveal_to_reader"}},
	}}}
	if err := validateBatch(unknown, seg, 0, 1); err == nil {
		t.Fatal("revealing unknown knowledge should fail validation")
	}
}

func TestValidateBatchBeliefContinuityAndFields(t *testing.T) {
	_, seg := analyzeFixture(t, 1)
	base := func(updates []domain.KnowledgeUpdate) *AnalysisBatchResult {
		return &AnalysisBatchResult{Chapters: []ImportedChapterFacts{{
			Chapter: 1, Summary: "认知变化", CoreEvent: "形成并纠正误解", HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: updates,
		}}}
	}
	valid := base([]domain.KnowledgeUpdate{
		{ID: "k", Action: "establish", Truth: "真相"},
		{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"},
		{ID: "k", Action: "learn", Character: "林墨"},
	})
	if err := validateBatch(valid, seg, 0, 1); err != nil {
		t.Fatalf("valid belief lifecycle rejected: %v", err)
	}
	invalid := [][]domain.KnowledgeUpdate{
		{{ID: "missing", Action: "believe", Character: "林墨", Belief: "误解"}},
		{{ID: "k", Action: "establish", Truth: "真相"}, {ID: "k", Action: "believe", Character: "林墨", Belief: "真相"}},
		{{ID: "k", Action: "establish", Truth: "真相"}, {ID: "k", Action: "believe", Character: "林墨", Belief: "误解", Truth: "多余"}},
	}
	for _, updates := range invalid {
		if err := validateBatch(base(updates), seg, 0, 1); err == nil {
			t.Fatalf("invalid belief updates accepted: %+v", updates)
		}
	}
}

func TestValidateBatchKnowledgeContinuity(t *testing.T) {
	_, seg := analyzeFixture(t, 2)
	valid := &AnalysisBatchResult{Chapters: []ImportedChapterFacts{
		{
			Chapter: 1, Summary: "建立真相", CoreEvent: "确认身份", HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长"}},
		},
		{
			Chapter: 2, Summary: "角色获知", CoreEvent: "承认身份", HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "learn", Character: "林墨"}},
		},
	}}
	if err := validateBatch(valid, seg, 0, 2); err != nil {
		t.Fatalf("establish then learn should be valid: %v", err)
	}

	unknown := &AnalysisBatchResult{Chapters: []ImportedChapterFacts{
		{
			Chapter: 1, Summary: "错误获知", CoreEvent: "错误引用", HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_missing", Action: "learn", Character: "林墨"}},
		},
	}}
	if err := validateBatch(unknown, seg, 0, 1); err == nil {
		t.Fatal("learning unknown knowledge should fail validation")
	}
}
