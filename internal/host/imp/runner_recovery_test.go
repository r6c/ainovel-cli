package imp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/tools"
)

func TestPublishRejectsInvalidFullBookFactsBeforeWritingOfficialStore(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	norm, seg := analyzeFixture(t, 2)
	sourcePath := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(sourcePath, norm, 0o644); err != nil {
		t.Fatal(err)
	}
	ws, _, err := Ingest(dir, sourcePath, Intent{})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeArtifact(ws, fileSegmentation, "digest", *seg); err != nil {
		t.Fatal(err)
	}
	facts := []ImportedChapterFacts{
		{Chapter: 1, Title: seg.Chapters[0].Title, Summary: "建立并获知", CoreEvent: "身份确认", KeyEvents: []string{"身份确认"}, HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "真相"}, {ID: "k", Action: "learn", Character: "林墨"}}},
		{Chapter: 2, Title: seg.Chapters[1].Title, Summary: "非法误信", CoreEvent: "误信", KeyEvents: []string{"误信"}, HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"}}},
	}
	for _, f := range facts {
		if err := writeArtifact(ws, analysisPath(f.Chapter), "digest", ChapterAnalysisPayload{Facts: f}); err != nil {
			t.Fatal(err)
		}
	}
	var synthesis BookSynthesis
	if err := json.Unmarshal([]byte(synthesisFixtureJSON(2, storyClosed)), &synthesis); err != nil {
		t.Fatal(err)
	}
	if err := writeArtifact(ws, fileSynthesis, synthesisInputDigest(facts), synthesis); err != nil {
		t.Fatal(err)
	}
	r := &runner{
		deps:   Deps{Store: st, CommitChapter: tools.NewCommitChapterTool(st, tools.NewStyleStatsIndex(st))},
		events: make(chan Event, 32), ws: ws,
	}

	if err := r.publish(context.Background()); err == nil {
		t.Fatal("expected invalid full-book facts to block publish")
	}
	if book, err := st.Book.Load(); err != nil || book != nil {
		t.Fatalf("official book must remain empty: book=%+v err=%v", book, err)
	}
	if premise, err := st.Outline.LoadPremise(); err != nil || premise != "" {
		t.Fatalf("official premise must remain empty: premise=%q err=%v", premise, err)
	}
	progress, err := st.Progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	if progress != nil && len(progress.CompletedChapters) != 0 {
		t.Fatalf("official chapters must remain empty: %+v", progress.CompletedChapters)
	}
	if pending, err := st.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("publish gate must not create pending commit: pending=%+v err=%v", pending, err)
	}
	if meta, err := st.RunMeta.Load(); err != nil || (meta != nil && meta.AdvanceHold != nil) {
		t.Fatalf("publish gate must not create completion hold: meta=%+v err=%v", meta, err)
	}
	if !ws.has(analysisPath(1)) || ws.has(analysisPath(2)) || ws.has(fileSynthesis) {
		t.Fatal("publish gate must preserve valid prefix and invalidate illegal tail plus synthesis")
	}
}

func TestSynthesizeRejectsInvalidFullBookFactsBeforeModelCall(t *testing.T) {
	ws := &Workspace{dir: t.TempDir()}
	seg := Segmentation{Chapters: []ChapterSpan{{Number: 1, Title: "第一章"}, {Number: 2, Title: "第二章"}}}
	if err := writeArtifact(ws, fileSegmentation, "digest", seg); err != nil {
		t.Fatal(err)
	}
	facts := []ImportedChapterFacts{
		{Chapter: 1, Title: "第一章", Summary: "建立并获知", CoreEvent: "身份确认", KeyEvents: []string{"身份确认"}, HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "真相"}, {ID: "k", Action: "learn", Character: "林墨"}}},
		{Chapter: 2, Title: "第二章", Summary: "非法误信", CoreEvent: "误信", KeyEvents: []string{"误信"}, HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"}}},
	}
	for _, f := range facts {
		if err := writeArtifact(ws, analysisPath(f.Chapter), "digest", ChapterAnalysisPayload{Facts: f}); err != nil {
			t.Fatal(err)
		}
	}
	m := &mockModel{responses: []string{synthesisFixtureJSON(2, storyClosed)}}
	r := &runner{
		deps:   Deps{Synthesize: Caller{Model: m}, Budgets: DefaultRunBudgets(), Prompts: Prompts{Synthesize: "syn", Range: "range"}},
		events: make(chan Event, 16), ws: ws,
	}

	if err := r.synthesize(context.Background()); err == nil {
		t.Fatal("expected invalid full-book facts to block synthesis")
	}
	if m.i != 0 {
		t.Fatalf("synthesis model must not be called for invalid facts, calls=%d", m.i)
	}
	if ws.has(fileSynthesis) {
		t.Fatal("invalid facts must not produce synthesis artifact")
	}
	if !ws.has(analysisPath(1)) || ws.has(analysisPath(2)) {
		t.Fatal("synthesis gate must preserve valid prefix and invalidate illegal tail")
	}
}

// TestRunEndToEnd 用 mock 模型驱动完整管线 ingest→segment→analyze→synthesize→publish，
// 经真实 commit_chapter 落盘，验证正式 Foundation 与全部章节就绪。

func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatalf("store init: %v", err)
	}
	src := filepath.Join(dir, "book.txt")
	source := "第一章\n正文一含**强调**\n第二章\n## 场内标识\n正文二\n"
	if err := os.WriteFile(src, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	seg := boundariesJSON(
		boundaryFixture("L1", "", kindChapter, "第一章"),
		boundaryFixture("L3", "", kindChapter, "第二章"),
	)
	ana := `{"chapters":[` + factsJSON(1, "第一章") + `,` + factsJSON(2, "第二章") + `]}`
	syn := synthesisFixtureJSON(2, storyClosed)
	m := &mockModel{responses: []string{seg, ana, syn}}

	ch, err := Run(context.Background(), testDeps(st, m), Options{SourcePath: src, AutoConfirm: true, ContinueAfter: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	var runErr error
	var doneSeen bool
	for ev := range ch {
		if ev.Stage == StageError {
			runErr = ev.Err
		}
		if ev.Stage == StageDone {
			doneSeen = true
		}
	}
	if runErr != nil {
		t.Fatalf("管线失败：%v", runErr)
	}
	if !doneSeen {
		t.Fatal("未收到 StageDone")
	}
	// 正式状态就绪：作品信息、premise 与覆盖全章的扁平大纲已落盘（world_rules 合法为空，不做要求）。
	if book, _ := st.Book.Load(); book == nil || book.Synopsis == "" {
		t.Fatalf("作品信息未落盘: %+v", book)
	}
	if p, _ := st.Outline.LoadPremise(); p == "" {
		t.Fatal("premise 未落盘")
	}
	if o, _ := st.Outline.LoadOutline(); len(o) != 2 {
		t.Fatalf("扁平大纲应覆盖 2 章，得 %d", len(o))
	}
	prog, _ := st.Progress.Load()
	if prog == nil || len(prog.CompletedChapters) != 2 {
		t.Fatalf("应完成 2 章：%+v", prog)
	}
	if active, done, err := ResumeStatus(st); err != nil || !active || !done {
		t.Fatalf("ResumeStatus 应为 active&done，得 active=%v done=%v", active, done)
	}
	for chapter := 1; chapter <= 2; chapter++ {
		record, err := st.ChapterRecords.Load(chapter)
		if err != nil || record == nil || record.Origin != domain.ChapterOriginImported {
			t.Fatalf("第 %d 章应保留 imported provenance: record=%+v err=%v", chapter, record, err)
		}
	}
	first, err := st.Drafts.LoadChapterText(1)
	if err != nil || !strings.Contains(first, "**强调**") {
		t.Fatalf("第一章 Markdown 原文未保留: %q err=%v", first, err)
	}
	second, err := st.Drafts.LoadChapterText(2)
	if err != nil || !strings.Contains(second, "## 场内标识") {
		t.Fatalf("第二章内部标题未保留: %q err=%v", second, err)
	}
	// --continue：不设导入完成 Hold（交由 host 自动接力）。
	if meta, _ := st.RunMeta.Load(); meta != nil && meta.AdvanceHold != nil {
		t.Fatalf("--continue 不应留下导入完成 Hold：%+v", meta.AdvanceHold)
	}
}

func TestRunPreservesImportedMarkdownWithoutGeneratedDraftGate(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.UserRules.Save(&rules.Snapshot{
		Version: rules.SnapshotVersion, Status: rules.StatusReady,
		Structured: rules.Structured{ChapterTargetChars: 10},
	}); err != nil {
		t.Fatal(err)
	}
	source := "第一章\n正文中的**强调**必须保留。\n## 场内标识\n结尾。\n"
	src := filepath.Join(dir, "book.md")
	if err := os.WriteFile(src, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	seg := boundariesJSON(boundaryFixture("L1", "", kindChapter, "第一章"))
	ana := `{"chapters":[` + factsJSON(1, "第一章") + `]}`
	syn := synthesisFixtureJSON(1, storyClosed)
	m := &mockModel{responses: []string{seg, ana, syn}}

	ch, err := Run(context.Background(), testDeps(st, m), Options{SourcePath: src, AutoConfirm: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for ev := range ch {
		if ev.Stage == StageError {
			t.Fatalf("imported markdown must be preserved instead of rejected as generated draft: %v", ev.Err)
		}
	}
	got, err := st.Drafts.LoadChapterText(1)
	if err != nil {
		t.Fatal(err)
	}
	if got != source {
		t.Fatalf("imported chapter content changed:\n got=%q\nwant=%q", got, source)
	}
	record, err := st.ChapterRecords.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Origin != domain.ChapterOriginImported {
		t.Fatalf("imported chapter must preserve provenance, got %+v", record)
	}
	violations := st.World.LoadRuleViolations(1)
	if len(violations) == 0 || violations[0].Rule != "markdown_residue" {
		t.Fatalf("imported markdown should remain observable as lint facts: %+v", violations)
	}
}

// TestRunSetsCompletionHold 验证非 --continue 导入完成后设置 boundary Hold（RFC §12.4）。
// Hold 是"导入后不误续写"的唯一保障，必须在发布路径持久化。

func TestRunSetsCompletionHold(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatalf("store init: %v", err)
	}
	src := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(src, []byte("第一章\n正文一\n第二章\n正文二\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seg := boundariesJSON(
		boundaryFixture("L1", "", kindChapter, "第一章"),
		boundaryFixture("L3", "", kindChapter, "第二章"),
	)
	ana := `{"chapters":[` + factsJSON(1, "第一章") + `,` + factsJSON(2, "第二章") + `]}`
	syn := synthesisFixtureJSON(2, storyClosed)
	m := &mockModel{responses: []string{seg, ana, syn}}

	ch, err := Run(context.Background(), testDeps(st, m), Options{SourcePath: src, AutoConfirm: true}) // 无 --continue
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for ev := range ch {
		if ev.Stage == StageError {
			t.Fatalf("管线失败：%v", ev.Err)
		}
	}
	meta, err := st.RunMeta.Load()
	if err != nil {
		t.Fatalf("load run meta: %v", err)
	}
	if meta == nil || meta.AdvanceHold == nil {
		t.Fatalf("导入完成应设置 boundary Hold，得 %+v", meta)
	}
}

// TestRunRejectsDifferentSource 守护换源拦截（RFC §12.1/§18.2）：工作区进行中传入不同
// 内容的源文件必须明确报错——ingest 只在无工作区时执行，不比对会静默从旧书断点继续、
// 把旧书发布完毕而新文件一个字节都没读。同一文件重复传路径是常见恢复习惯，按内容摘要比对放行。

func TestRunRejectsDifferentSource(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(a, []byte("第一章\n正文一\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Ingest(dir, a, Options{}.intent()); err != nil {
		t.Fatalf("建立工作区：%v", err)
	}
	b := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(b, []byte("完全不同的另一本书\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ch, err := Run(context.Background(), testDeps(st, &mockModel{responses: []string{"{}"}}), Options{SourcePath: b})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	var runErr error
	for ev := range ch {
		if ev.Stage == StageError {
			runErr = ev.Err
		}
	}
	if runErr == nil || !strings.Contains(runErr.Error(), "内容不同") {
		t.Fatalf("不同源文件应被明确拒绝，得 %v", runErr)
	}
}

// TestConfirmNotesGate 守护 --yes 的容错门槛：语义容错（Notes 非空）发生过的切分结构
// 被确定性改写过，不由未看预览的 --yes 盲放行；TUI 预览后按 y（AcceptSegmentation）放行，
// 确认方法记 user_confirmed 溯源。
