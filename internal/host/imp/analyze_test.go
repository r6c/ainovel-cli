package imp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/internal/domain"
)

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

func TestPlanBatchOutputBudgetCaps(t *testing.T) {
	_, seg := analyzeFixture(t, 10)
	// 输入宽松，但可见输出预算只够 2 章（#83 批次粒度守卫，§20.4.2）。
	b := AnalyzeBudget{ContextBytes: 1 << 20, MaxOutputTokens: 250, PerChapterOutput: 100, PromptOverhead: 0}
	end := planBatch(seg.Chapters, 0, 0, b)
	if end != 2 {
		t.Fatalf("输出预算应把批次限到 2 章，得 end=%d", end)
	}
}

func TestPlanBatchInputBudgetCaps(t *testing.T) {
	_, seg := analyzeFixture(t, 10)
	// 输出宽松，但输入字节预算只够约 1 章。
	one := chapterBytes(seg.Chapters, 0)
	b := AnalyzeBudget{ContextBytes: one + 1, MaxOutputTokens: 1 << 20, PerChapterOutput: 1, PromptOverhead: 0}
	end := planBatch(seg.Chapters, 0, 0, b)
	if end != 1 {
		t.Fatalf("输入预算应把批次限到 1 章，得 end=%d", end)
	}
}

func factsJSON(chapter int, title string) string {
	f := map[string]any{
		"chapter": chapter, "title": title, "summary": "摘要", "core_event": "核心事件",
		"key_events": []string{"事件"}, "hook": nil, "scenes": []string{}, "characters": []string{},
		"character_evidence": []any{}, "world_evidence": []any{}, "timeline_events": []any{},
		"foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{}, "hook_type": "mystery", "dominant_strand": "quest",
	}
	data, _ := json.Marshal(f)
	return string(data)
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

func TestValidateBatchRejections(t *testing.T) {
	_, seg := analyzeFixture(t, 2)
	// 数量不符
	bad := &AnalysisBatchResult{Chapters: []ImportedChapterFacts{{Chapter: 1}}}
	if err := validateBatch(bad, seg, 0, 2); err == nil {
		t.Fatal("数量不符应拒绝")
	}
	// hook_type 非法
	var f ImportedChapterFacts
	_ = json.Unmarshal([]byte(factsJSON(1, seg.Chapters[0].Title)), &f)
	f.HookType = "bogus"
	if err := validateBatch(&AnalysisBatchResult{Chapters: []ImportedChapterFacts{f}}, seg, 0, 1); err == nil {
		t.Fatal("非法 hook_type 应拒绝")
	}
	// 枚举大小写变体：校验通过并就地归一化为小写——commit_chapter 不复验枚举，
	// 变体直通正式状态会被精确串消费的逻辑视为未知类型。
	_ = json.Unmarshal([]byte(factsJSON(1, seg.Chapters[0].Title)), &f)
	f.HookType, f.DominantStrand = "Crisis", "QUEST"
	got := &AnalysisBatchResult{Chapters: []ImportedChapterFacts{f}}
	if err := validateBatch(got, seg, 0, 1); err != nil {
		t.Fatalf("大小写变体应通过校验：%v", err)
	}
	if got.Chapters[0].HookType != "crisis" || got.Chapters[0].DominantStrand != "quest" {
		t.Fatalf("枚举应归一化为小写落盘：%+v", got.Chapters[0])
	}
}

func TestAnalyzeNextPersistsWithRebatchOnTruncation(t *testing.T) {
	norm, seg := analyzeFixture(t, 2)
	book := t.TempDir()
	ws := &Workspace{dir: book}
	// 首批 2 章截断：第 1 章完整、第 2 章半截 → 打捞第 1 章连续前缀（§9.5）。
	truncated := `{"chapters":[` + factsJSON(1, seg.Chapters[0].Title) + `,{"chapter":2,"summary":"截断`
	m := &mockModel{
		responses: []string{truncated},
		stops:     []agentcore.StopReason{agentcore.StopReasonLength},
	}
	budget := AnalyzeBudget{ContextBytes: 1 << 20, MaxOutputTokens: 1000, PerChapterOutput: 10, PromptOverhead: 0}
	done, err := AnalyzeNext(context.Background(), m, "sys", ws, norm, seg, "segid", "v1", budget, callProfile{})
	if err != nil {
		t.Fatalf("AnalyzeNext: %v", err)
	}
	if done != 1 {
		t.Fatalf("截断应打捞第 1 章连续前缀，得 %d", done)
	}
	if !ws.has(analysisPath(1)) || ws.has(analysisPath(2)) {
		t.Fatal("应只落盘第 1 章")
	}
	if analyzedChapters(ws, seg, norm, "segid", "v1") != 1 {
		t.Fatal("已分析章数应为 1")
	}
	// failures/ 应保存原始响应与打捞状态（§14.2）。
	if !ws.has("failures/last-response.txt") || !ws.has("failures/last.json") {
		t.Fatal("应保存失败原始响应与元数据")
	}
}

func TestSalvagePrefixContiguous(t *testing.T) {
	_, seg := analyzeFixture(t, 3)
	// 前 2 章完整，第 3 章被截断。
	raw := `{"chapters":[` +
		factsJSON(1, seg.Chapters[0].Title) + `,` +
		factsJSON(2, seg.Chapters[1].Title) + `,` +
		`{"chapter":3,"summary":"截断`
	got := salvagePrefix(raw, seg, 0)
	if len(got) != 2 {
		t.Fatalf("应打捞前 2 章连续前缀，得 %d", len(got))
	}
	if got[0].Chapter != 1 || got[1].Chapter != 2 {
		t.Fatal("前缀章号不连续")
	}
}

func TestSalvagePrefixStopsAtGap(t *testing.T) {
	_, seg := analyzeFixture(t, 3)
	// 第 1 章后直接跳到第 3 章 → 打捞在跳号处停止，只返回第 1 章。
	raw := `{"chapters":[` + factsJSON(1, seg.Chapters[0].Title) + `,` + factsJSON(3, seg.Chapters[2].Title) + `]}`
	got := salvagePrefix(raw, seg, 0)
	if len(got) != 1 {
		t.Fatalf("跳号处应停止，得 %d", len(got))
	}
}

// TestAnalyzedChaptersInvalidatesOnUpstreamChange 验证切分身份或 prompt 版本变化使已落盘分析失效（不变量 1）。
// 这是 InputDigest 机制真正落地的核心：改上游即失效下游，而非只看文件是否存在。
func TestAnalyzedChaptersInvalidatesPreviousAnalysisSchemaVersion(t *testing.T) {
	norm, seg := analyzeFixture(t, 1)
	ws := &Workspace{dir: t.TempDir()}
	var old strings.Builder
	old.WriteString("analyze\x00")
	old.WriteString("v1")
	old.WriteString("\x00v4\x00")
	old.WriteString("segid")
	old.WriteString("\x00ch1\x00")
	old.WriteString(seg.Content(norm, 0))
	if err := writeArtifact(ws, analysisPath(1), Digest([]byte(old.String())), ChapterAnalysisPayload{
		Facts: ImportedChapterFacts{Chapter: 1},
	}); err != nil {
		t.Fatal(err)
	}

	if got := analyzedChapters(ws, seg, norm, "segid", "v1"); got != 0 {
		t.Fatalf("previous analysis schema cache must be invalidated, got %d reusable chapters", got)
	}
}

func TestAnalyzedChaptersInvalidatesOnUpstreamChange(t *testing.T) {
	norm, seg := analyzeFixture(t, 2)
	ws := &Workspace{dir: t.TempDir()}
	m := &mockModel{responses: []string{
		`{"chapters":[` + factsJSON(1, seg.Chapters[0].Title) + `,` + factsJSON(2, seg.Chapters[1].Title) + `]}`,
	}}
	budget := AnalyzeBudget{ContextBytes: 1 << 20, MaxOutputTokens: 1 << 20, PerChapterOutput: 10, PromptOverhead: 0}
	if _, err := AnalyzeNext(context.Background(), m, "sys", ws, norm, seg, "segid-A", "v1", budget, callProfile{}); err != nil {
		t.Fatalf("AnalyzeNext: %v", err)
	}
	if got := analyzedChapters(ws, seg, norm, "segid-A", "v1"); got != 2 {
		t.Fatalf("同身份/版本应认 2 章，得 %d", got)
	}
	if got := analyzedChapters(ws, seg, norm, "segid-B", "v1"); got != 0 {
		t.Fatalf("切分身份变化应使分析全部失效，得 %d", got)
	}
	if got := analyzedChapters(ws, seg, norm, "segid-A", "v2"); got != 0 {
		t.Fatalf("prompt 版本变化应使分析全部失效，得 %d", got)
	}
}
