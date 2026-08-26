package imp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
)

func TestValidateImportedFactSequenceRejectsOutOfOrderArtifactChapters(t *testing.T) {
	facts := []ImportedChapterFacts{
		{Chapter: 1, Title: "第一章", Summary: "摘要", CoreEvent: "事件", KeyEvents: []string{"事件"}, HookType: "mystery", DominantStrand: "quest"},
		{Chapter: 3, Title: "第三章", Summary: "摘要", CoreEvent: "事件", KeyEvents: []string{"事件"}, HookType: "mystery", DominantStrand: "quest"},
	}
	if err := validateImportedFactSequence(facts); err == nil {
		t.Fatal("full-book import validation must reject a chapter number that differs from artifact order")
	}
}

func TestValidateImportedFactSequenceReplaysCrossBatchLifecycles(t *testing.T) {
	base := func(chapter int, knowledge []domain.KnowledgeUpdate, foreshadow []domain.ForeshadowUpdate) ImportedChapterFacts {
		return ImportedChapterFacts{
			Chapter: chapter, Title: "章节", Summary: "事实推进", KeyEvents: []string{"推进"},
			HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: knowledge, ForeshadowUpdates: foreshadow,
		}
	}

	tests := []struct {
		name    string
		facts   []ImportedChapterFacts
		wantErr bool
	}{
		{
			name: "合法知识和伏笔生命周期",
			facts: []ImportedChapterFacts{
				base(1, []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "真相"}}, []domain.ForeshadowUpdate{{ID: "f", Action: "plant", Description: "线索"}}),
				base(2, []domain.KnowledgeUpdate{{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"}}, []domain.ForeshadowUpdate{{ID: "f", Action: "reinforce"}}),
				base(3, []domain.KnowledgeUpdate{{ID: "k", Action: "learn", Character: "林墨"}}, []domain.ForeshadowUpdate{{ID: "f", Action: "partial_payoff"}}),
				base(4, []domain.KnowledgeUpdate{{ID: "k", Action: "reveal_to_reader"}}, []domain.ForeshadowUpdate{{ID: "f", Action: "resolve"}}),
			},
		},
		{
			name: "跨批次冲突真相",
			facts: []ImportedChapterFacts{
				base(1, []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "真相甲"}}, nil),
				base(2, []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "真相乙"}}, nil),
			},
			wantErr: true,
		},
		{
			name: "跨批次未知知识引用",
			facts: []ImportedChapterFacts{
				base(1, nil, nil),
				base(2, []domain.KnowledgeUpdate{{ID: "missing", Action: "learn", Character: "林墨"}}, nil),
			},
			wantErr: true,
		},
		{
			name: "伏笔回收后重新推进",
			facts: []ImportedChapterFacts{
				base(1, nil, []domain.ForeshadowUpdate{{ID: "f", Action: "plant", Description: "线索"}}),
				base(2, nil, []domain.ForeshadowUpdate{{ID: "f", Action: "resolve"}}),
				base(3, nil, []domain.ForeshadowUpdate{{ID: "f", Action: "advance"}}),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImportedFactSequence(tt.facts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateImportedFactSequence() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestInvalidWorkspaceFactsReturnStateToAnalyze(t *testing.T) {
	book := t.TempDir()
	sourcePath := filepath.Join(book, "book.txt")
	norm, seg := analyzeFixture(t, 3)
	if err := os.WriteFile(sourcePath, norm, 0o644); err != nil {
		t.Fatal(err)
	}
	ws, _, err := Ingest(book, sourcePath, Intent{})
	if err != nil {
		t.Fatal(err)
	}
	segDigest := segmentInputDigest(Digest(norm), "", segmentPromptVersion)
	if err := writeArtifact(ws, fileSegmentation, segDigest, *seg); err != nil {
		t.Fatal(err)
	}
	segRaw, err := ws.readBytes(fileSegmentation)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeArtifact(ws, fileConfirmation, Digest(segRaw), Confirmation{Method: confirmMethodAuto, Chapters: 3}); err != nil {
		t.Fatal(err)
	}
	facts := []ImportedChapterFacts{
		{Chapter: 1, Title: "第一章", Summary: "建立", CoreEvent: "建立", KeyEvents: []string{"建立"}, HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "真相"}, {ID: "k", Action: "learn", Character: "林墨"}}},
		{Chapter: 2, Title: "第二章", Summary: "非法", CoreEvent: "误信", KeyEvents: []string{"误信"}, HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"}}},
		{Chapter: 3, Title: "第三章", Summary: "下游", CoreEvent: "下游", KeyEvents: []string{"下游"}, HookType: "mystery", DominantStrand: "quest"},
	}
	for i, f := range facts {
		digest := chapterInputDigest(segDigest, analyzePromptVersion, seg, norm, i)
		if err := writeArtifact(ws, analysisPath(f.Chapter), digest, ChapterAnalysisPayload{Facts: f}); err != nil {
			t.Fatal(err)
		}
	}
	if err := validateWorkspaceFacts(ws, facts, len(facts)); err == nil {
		t.Fatal("expected invalid workspace facts")
	}
	state, err := LoadState(ws)
	if err != nil {
		t.Fatal(err)
	}
	if state.AnalyzedChapters != 1 || NextAction(state) != ActionAnalyze {
		t.Fatalf("invalid tail must return recovery to analyze: state=%+v action=%s", state, NextAction(state))
	}
}

func TestValidateWorkspaceFactsInvalidatesFromFirstIllegalChapter(t *testing.T) {
	ws := &Workspace{dir: t.TempDir()}
	facts := []ImportedChapterFacts{
		{Chapter: 1, Title: "第一章", Summary: "建立", CoreEvent: "建立", KeyEvents: []string{"建立"}, HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "真相"}, {ID: "k", Action: "learn", Character: "林墨"}}},
		{Chapter: 2, Title: "第二章", Summary: "非法", CoreEvent: "误信", KeyEvents: []string{"误信"}, HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"}}},
		{Chapter: 3, Title: "第三章", Summary: "下游", CoreEvent: "下游", KeyEvents: []string{"下游"}, HookType: "mystery", DominantStrand: "quest"},
	}
	for _, f := range facts {
		if err := writeArtifact(ws, analysisPath(f.Chapter), "digest", ChapterAnalysisPayload{Facts: f}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeArtifact(ws, fileSynthesis, "digest", BookSynthesis{}); err != nil {
		t.Fatal(err)
	}
	if err := writeArtifact(ws, fileStoryResolve, "digest", StoryResolution{}); err != nil {
		t.Fatal(err)
	}

	if err := validateWorkspaceFacts(ws, facts, len(facts)); err == nil {
		t.Fatal("expected invalid workspace facts")
	}
	if !ws.has(analysisPath(1)) || ws.has(analysisPath(2)) || ws.has(analysisPath(3)) {
		t.Fatal("must preserve valid prefix and discard invalid chapter plus tail")
	}
	if ws.has(fileSynthesis) || ws.has(fileStoryResolve) {
		t.Fatal("downstream synthesis and story resolution must be invalidated")
	}
}

// TestDiscardAnalysesAfter 守护 #4a：清理越过新鲜前缀的旧分析工件，
// 保证"重分析某章即失效其后全部分析"，防止陈旧 ledger 随后续章节被复用。

func TestDiscardAnalysesAfter(t *testing.T) {
	ws := OpenWorkspace(t.TempDir())
	for c := 1; c <= 5; c++ {
		if err := writeArtifact(ws, analysisPath(c), "d", ChapterAnalysisPayload{Facts: ImportedChapterFacts{Chapter: c}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := discardAnalysesAfter(ws, 2, 5); err != nil {
		t.Fatalf("清理不应失败：%v", err)
	}
	for c := 1; c <= 2; c++ {
		if !ws.has(analysisPath(c)) {
			t.Fatalf("新鲜前缀章 %d 应保留", c)
		}
	}
	for c := 3; c <= 5; c++ {
		if ws.has(analysisPath(c)) {
			t.Fatalf("越过新鲜前缀的章 %d 应被清理", c)
		}
	}
}

// analyzeFixture 构造一份含 n 章、正文都很短的切分，用于批次/分析测试。
func analyzeFixture(t *testing.T, n int) ([]byte, *Segmentation) {
	t.Helper()
	var b strings.Builder
	for c := 1; c <= n; c++ {
		b.WriteString("第")
		b.WriteString(strings.Repeat("一", 1))
		b.WriteString("章\n正文\n")
	}
	norm := []byte(b.String())
	units := buildSourceUnits(norm, 0)
	var ds []BoundaryDecision
	for i := 0; i < len(units); i += 2 { // 每 2 行一章（标题行 + 正文行）
		ds = append(ds, BoundaryDecision{UnitID: units[i].ID, Kind: kindChapter, Title: units[i].Text})
	}
	seg, err := resolveSegmentation(norm, units, ds)
	if err != nil {
		t.Fatalf("fixture 切分失败：%v", err)
	}
	if len(seg.Chapters) != n {
		t.Fatalf("fixture 章数 %d != %d", len(seg.Chapters), n)
	}
	return norm, seg
}

func TestAnalyzeNextRejectsCrossBatchFactConflictBeforeWritingArtifact(t *testing.T) {
	norm, seg := analyzeFixture(t, 2)
	ws := &Workspace{dir: t.TempDir()}
	segID, promptVersion := "segid", "v1"
	prior := ImportedChapterFacts{
		Chapter: 1, Title: seg.Chapters[0].Title, Summary: "建立并获知", KeyEvents: []string{"身份确认"}, CoreEvent: "身份确认",
		HookType: "mystery", DominantStrand: "quest",
		KnowledgeUpdates: []domain.KnowledgeUpdate{
			{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长"},
			{ID: "k_shadow", Action: "learn", Character: "林墨"},
		},
	}
	if err := writeArtifact(ws, analysisPath(1), chapterInputDigest(segID, promptVersion, seg, norm, 0), ChapterAnalysisPayload{Facts: prior}); err != nil {
		t.Fatal(err)
	}
	candidate := map[string]any{
		"chapter": 2, "title": seg.Chapters[1].Title, "summary": "形成误解", "core_event": "误认身份",
		"key_events": []string{"误认身份"}, "hook": nil, "scenes": []string{}, "characters": []string{"林墨"},
		"character_evidence": []any{}, "world_evidence": []any{}, "timeline_events": []any{},
		"foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{map[string]any{
			"id": "k_shadow", "action": "believe", "truth": nil, "character": "林墨", "belief": "黑影是杀兄仇人",
		}},
		"hook_type": "mystery", "dominant_strand": "quest",
	}
	invalidResponse, err := json.Marshal(map[string]any{"chapters": []any{candidate}})
	if err != nil {
		t.Fatal(err)
	}
	candidate["summary"] = "修正认知"
	candidate["core_event"] = "林墨维持已知事实"
	candidate["key_events"] = []string{"维持认知"}
	candidate["knowledge_updates"] = []any{}
	validResponse, err := json.Marshal(map[string]any{"chapters": []any{candidate}})
	if err != nil {
		t.Fatal(err)
	}
	m := &mockModel{responses: []string{string(invalidResponse), string(validResponse)}}
	budget := AnalyzeBudget{ContextBytes: 1 << 20, MaxOutputTokens: 1 << 20, PerChapterOutput: 10, PromptOverhead: 0}

	if got, err := AnalyzeNext(context.Background(), m, "sys", ws, norm, seg, segID, promptVersion, budget, callProfile{}); err != nil || got != 1 {
		t.Fatalf("corrected second batch should succeed once: got=%d err=%v", got, err)
	}
	if m.i != 2 {
		t.Fatalf("invalid cumulative facts should trigger exactly one correction, calls=%d", m.i)
	}
	art, err := readArtifact[ChapterAnalysisPayload](ws, analysisPath(2))
	if err != nil {
		t.Fatal(err)
	}
	if art.Payload.Facts.Summary != "修正认知" || len(art.Payload.Facts.KnowledgeUpdates) != 0 {
		t.Fatalf("only corrected facts may be written: %+v", art.Payload.Facts)
	}
	if got := analyzedChapters(ws, seg, norm, segID, promptVersion); got != 2 {
		t.Fatalf("corrected fact sequence should have two chapters, got %d", got)
	}
}
