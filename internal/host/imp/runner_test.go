package imp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/tools"
)

// testDeps 构造三个语义函数同用一个 mock 档位的最小 Deps。
func testDeps(st *store.Store, m callModel) Deps {
	c := Caller{Model: m}
	return Deps{
		Store:         st,
		CommitChapter: tools.NewCommitChapterTool(st, tools.NewStyleStatsIndex(st)),
		Segment:       c,
		Analyze:       c,
		Synthesize:    c,
		Prompts:       Prompts{Segment: "seg", Analyze: "ana", Synthesize: "syn", Range: "range"},
	}
}

func TestConfirmNotesGate(t *testing.T) {
	newRunner := func(opts Options, notes []string) *runner {
		ws := &Workspace{dir: t.TempDir()}
		if err := ws.writeJSON(fileIntent, Intent{}); err != nil {
			t.Fatal(err)
		}
		seg := Segmentation{Chapters: []ChapterSpan{{Number: 1, Title: "第一章", End: 10}}, Notes: notes}
		if err := writeArtifact(ws, fileSegmentation, "d", seg); err != nil {
			t.Fatal(err)
		}
		return &runner{opts: opts, events: make(chan Event, 8), ws: ws}
	}
	r := newRunner(Options{AutoConfirm: true}, []string{"空正文占位并入前段"})
	if r.confirm() {
		t.Fatal("--yes 不应放行带容错说明的切分")
	}
	if ev := <-r.events; !strings.Contains(ev.Message, "未自动放行") {
		t.Fatalf("预览应说明未放行原因：%q", ev.Message)
	}
	if !newRunner(Options{AutoConfirm: true}, nil).confirm() {
		t.Fatal("--yes 应放行无容错说明的切分")
	}
	r = newRunner(Options{AcceptSegmentation: true}, []string{"空正文占位并入前段"})
	if !r.confirm() {
		t.Fatal("预览后的人工 y 应放行带容错说明的切分")
	}
	conf, err := readArtifact[Confirmation](r.ws, fileConfirmation)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Payload.Method != confirmMethodUser {
		t.Fatalf("人工确认应记 user_confirmed，得 %q", conf.Payload.Method)
	}
}

// TestStoryChoiceIgnoresStaleResolution 守护 #5：重新综合后旧故事裁定失效，
// storyChoice 不得把旧 open/closed 静默套到新 synthesis 上（否则用户不会被重新征询）。
func TestStoryChoiceIgnoresStaleResolution(t *testing.T) {
	ws := OpenWorkspace(t.TempDir())
	if err := ws.writeJSON(fileIntent, Intent{}); err != nil {
		t.Fatal(err)
	}
	if err := writeArtifact(ws, fileSynthesis, "d", BookSynthesis{Premise: "p1", StoryStatus: storyUncertain}); err != nil {
		t.Fatal(err)
	}
	raw, _ := ws.readBytes(fileSynthesis)
	if err := writeArtifact(ws, fileStoryResolve, Digest(raw), StoryResolution{Choice: storyClosed}); err != nil {
		t.Fatal(err)
	}
	r := &runner{ws: ws}
	if got, err := r.storyChoice(); err != nil || got != storyClosed {
		t.Fatalf("绑定当前 synthesis 的裁定应返回 closed，得 %q", got)
	}
	// 重新综合：改写 synthesis → 旧裁定 InputDigest 失配，应被忽略，回到"需重新征询"（返回空）。
	if err := writeArtifact(ws, fileSynthesis, "d", BookSynthesis{Premise: "p2", StoryStatus: storyUncertain}); err != nil {
		t.Fatal(err)
	}
	if got, err := r.storyChoice(); err != nil || got != "" {
		t.Fatalf("重新综合后旧裁定应失效返回空，得 %q", got)
	}
}

// TestBudgetsFromDepsPerTier 守护档位旋钮（RFC §13.1）：各语义函数预算按各自档位派生，
// 廉价档位的小窗口只约束它自己的函数，不拖累其它阶段。
func TestBudgetsFromDepsPerTier(t *testing.T) {
	small := ModelRuntime{ContextTokens: 32000, MaxOutputTokens: 4000}
	big := ModelRuntime{ContextTokens: 200000, MaxOutputTokens: 16000}
	b := budgetsFromDeps(Deps{
		Segment:    Caller{Runtime: small},
		Analyze:    Caller{Runtime: big},
		Synthesize: Caller{Runtime: big},
	})
	if b.SegmentChunkBytes >= b.Analyze.ContextBytes {
		t.Fatalf("segment 小档位窗口应只约束自身：seg=%d analyze=%d", b.SegmentChunkBytes, b.Analyze.ContextBytes)
	}
	if b.Analyze.MaxOutputTokens != 16000 || b.SegmentMaxTokens != 4000 {
		t.Fatalf("输出预算应各取自身档位上限：analyze=%d segment=%d", b.Analyze.MaxOutputTokens, b.SegmentMaxTokens)
	}
}

// TestRunSavesFailureOnContractViolation 守护 §14.2：原生 Schema 契约违约
// 必须立即暴露，并把原始响应与元数据落 failures/。
func TestRunSavesFailureOnContractViolation(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(src, []byte("第一章\n正文\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &nativeImportModel{mockModel: &mockModel{responses: []string{"这不是 JSON"}}}
	ch, err := Run(context.Background(), testDeps(st, m), Options{SourcePath: src, AutoConfirm: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	var failed bool
	for ev := range ch {
		if ev.Stage == StageError {
			failed = true
		}
	}
	if !failed {
		t.Fatal("非法输出应以 StageError 结束")
	}
	ws := OpenWorkspace(dir)
	if !ws.has("failures/last-response.txt") {
		t.Fatal("应保存最后一次原始模型响应")
	}
	var meta FailureMeta
	if err := ws.readJSON("failures/last.json", &meta); err != nil {
		t.Fatalf("读失败元数据：%v", err)
	}
	if meta.Stage != string(ActionSegment) {
		t.Fatalf("失败元数据应标注 segment 阶段，得 %q", meta.Stage)
	}
}

// TestRunGuidanceResegments 守护 §18.3：恢复时携带 --guide 使旧切分自然失配，
// 按新指导重新识别并再次停在确认处；新切分 InputDigest 绑定指导文本。
func TestRunGuidanceResegments(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(src, []byte("第一章\n正文一\n第二章\n正文二\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	drain := func(ch <-chan Event) (awaiting bool) {
		for ev := range ch {
			if ev.Stage == StageError {
				t.Fatalf("管线失败：%v", ev.Err)
			}
			if ev.Stage == StageAwaitingConfirmation {
				awaiting = true
			}
		}
		return awaiting
	}
	// 首次交互导入：模型把全书切成 1 章，停在确认。
	one := boundariesJSON(boundaryFixture("L1", "", kindChapter, "第一章"))
	ch, err := Run(context.Background(), testDeps(st, &mockModel{responses: []string{one}}), Options{SourcePath: src})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !drain(ch) {
		t.Fatal("首次导入应停在切分确认")
	}
	// 带指导恢复：旧切分失配 → 重识别为 2 章，再次停在确认。
	two := boundariesJSON(
		boundaryFixture("L1", "", kindChapter, "第一章"),
		boundaryFixture("L3", "", kindChapter, "第二章"),
	)
	guidance := "第二章也是独立章节"
	ch2, err := Run(context.Background(), testDeps(st, &mockModel{responses: []string{two}}), Options{Guidance: guidance})
	if err != nil {
		t.Fatalf("恢复 Run: %v", err)
	}
	if !drain(ch2) {
		t.Fatal("重识别后应再次停在切分确认")
	}
	ws := OpenWorkspace(dir)
	art, err := readArtifact[Segmentation](ws, fileSegmentation)
	if err != nil {
		t.Fatalf("读切分工件：%v", err)
	}
	if len(art.Payload.Chapters) != 2 {
		t.Fatalf("应按指导切成 2 章，得 %d", len(art.Payload.Chapters))
	}
	norm, _ := ws.LoadSource()
	if art.InputDigest != segmentInputDigest(Digest(norm), guidance, segmentPromptVersion) {
		t.Fatal("新切分 InputDigest 应绑定指导文本")
	}
}

// TestBudgetsFromRuntime 验证双预算随模型真实容量放大，能力未知时回退保守默认（RFC §9.2/§21）。
func TestBudgetsFromRuntime(t *testing.T) {
	if got := budgetsFromRuntime(ModelRuntime{}); got != DefaultRunBudgets() {
		t.Fatal("能力未知应回退保守默认")
	}
	small := budgetsFromRuntime(ModelRuntime{ContextTokens: 32000, MaxOutputTokens: 4000})
	big := budgetsFromRuntime(ModelRuntime{ContextTokens: 200000, MaxOutputTokens: 16000})
	if big.Analyze.ContextBytes <= small.Analyze.ContextBytes {
		t.Fatalf("更大 context 应放大 analyze 输入预算：small=%d big=%d", small.Analyze.ContextBytes, big.Analyze.ContextBytes)
	}
	if big.Analyze.MaxOutputTokens != 16000 {
		t.Fatalf("输出预算应取模型 completion 上限，得 %d", big.Analyze.MaxOutputTokens)
	}
}

// TestProfileForKeyPolicy 守护事件合并范围：请求退避（带截止时刻）同 Key 原地跳动；
// 校验重问是跨调用的语义事件，不带 Key 各自成行——切分逐块调用，共用 Key 会让后块覆盖前块，
// 面板只剩一条 unit_id 不断变化的行，排查线索全丢；step 是普通进度事件（无警示级别）。
func TestProfileForKeyPolicy(t *testing.T) {
	r := &runner{events: make(chan Event, 3)}
	prof := r.profileFor(Caller{}, StageSegmenting)
	prof.notify("退避", time.Now().Add(time.Second))
	prof.notify("重问", time.Time{})
	prof.step(2, 12, "切分第 %d/%d 块...", 2, 12)
	backoff, reask, step := <-r.events, <-r.events, <-r.events
	if backoff.Key == "" || backoff.Level != "warn" || backoff.RetryAt.IsZero() {
		t.Fatalf("请求退避应为带 Key 与截止时刻的 warn 事件：%+v", backoff)
	}
	if reask.Key != "" || reask.Level != "warn" {
		t.Fatalf("校验重问应为不带 Key 的 warn 事件（独立成行）：%+v", reask)
	}
	if step.Level != "" || step.Current != 2 || step.Total != 12 {
		t.Fatalf("step 应为普通进度事件：%+v", step)
	}
}

// TestCallProfileOptions 验证 callProfile 只负责输出预算与 thinking；response_format
// 由 callStructured 根据模型事实和 Contract 选择，不能在 Profile 中重复组装。
func TestCallProfileOptions(t *testing.T) {
	if got := (callProfile{}).callOptions(100); len(got) != 1 {
		t.Fatalf("零值只应带 maxTokens，得 %d 个 option", len(got))
	}
	if got := (callProfile{thinking: "high"}).callOptions(100); len(got) != 2 {
		t.Fatalf("thinking 应带 2 个 option，得 %d", len(got))
	}
}
