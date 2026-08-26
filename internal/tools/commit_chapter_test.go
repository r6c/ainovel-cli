package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func newTestCommitChapterTool(st *store.Store) *CommitChapterTool {
	return NewCommitChapterTool(st, NewStyleStatsIndex(st))
}

func saveTestChapterRecord(t *testing.T, st *store.Store, chapter int, content string) {
	t.Helper()
	if _, err := st.ChapterRecords.Accept(chapter, domain.ChapterOriginGenerated, content, domain.ChapterFacts{
		Title: fmt.Sprintf("第%d章", chapter), Summary: "既有摘要", KeyEvents: []string{"既有事件"},
	}, domain.StyleDelta{}); err != nil {
		t.Fatalf("SaveChapterRecord %d: %v", chapter, err)
	}
}

// TestCommitChapterRewriteKeepsOwnForeshadowPlant 锁死 issue #110：重写伏笔的"种植章"时，
// Writer 看到账本里该伏笔已存在，自然只写 advance；旧实现整条覆盖章节记录，plant 随之丢失，
// Projector 全量重放时报"推进未知伏笔"并把返工队列锁死。种植事实必须被保留。

// TestCommitChapterRewriteRejectsForwardForeshadowReference 与上一个用例同源（issue #110）：
// 账本是全书投影，重写早期章节时里面还躺着后续章节才种下的伏笔。旧实现放行 → Projector
// 按章序重放时报"推进未知伏笔"，且此时章节记录已被覆盖，返工队列就此锁死。
// 必须在落盘前挡下，并把"种植于第几章"讲清楚，模型才改得动。

// TestCommitChapterUpdatesCastLedger 验证：commit_chapter 把本章 characters 累加进 cast_ledger，
// cast_intros 提供的 brief_role 被采用，且 characters.json 中的核心角色不进入 ledger。

func TestSealPendingCommitAddsStablePayloadAndDraftDigests(t *testing.T) {
	payload := json.RawMessage(`{"chapter":1,"title":"第一章"}`)
	pending := domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: "冻结正文"}

	got, err := sealPendingCommit(pending)
	if err != nil {
		t.Fatal(err)
	}
	if got.SealVersion != 2 {
		t.Fatalf("seal version = %d, want 2", got.SealVersion)
	}
	if got.Origin != domain.ChapterOriginGenerated {
		t.Fatalf("new seal must freeze generated provenance by default, got %q", got.Origin)
	}
	wantPayload, err := digestPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	wantDraft := fmt.Sprintf("%x", sha256.Sum256([]byte("冻结正文")))
	if got.PayloadDigest != wantPayload || got.DraftDigest != wantDraft {
		t.Fatalf("wrong seal: payload=%q draft=%q", got.PayloadDigest, got.DraftDigest)
	}
	if len(got.PayloadDigest) != 64 || len(got.DraftDigest) != 64 || len(got.IntentDigest) != 64 {
		t.Fatalf("digests must be 64 lowercase hex characters: %+v", got)
	}
	if got.IntentDigest != digestPendingIntent(got) {
		t.Fatalf("wrong intent digest: %q", got.IntentDigest)
	}
	imported := got
	imported.Origin = domain.ChapterOriginImported
	if digestPendingIntent(imported) == got.IntentDigest {
		t.Fatal("v2 intent digest must protect chapter provenance")
	}
}

func TestExecuteImportedCannotChangeFrozenGeneratedProvenance(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.StartChapter(1); err != nil {
		t.Fatal(err)
	}
	content := "普通生成正文。"
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":[],"key_events":["事件"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Origin: domain.ChapterOriginGenerated,
		Payload: payload, DraftContent: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).ExecuteImported(context.Background(), json.RawMessage(`{"chapter":1}`)); err != nil {
		t.Fatalf("recover generated pending through imported adapter: %v", err)
	}
	record, err := s.ChapterRecords.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Origin != domain.ChapterOriginGenerated {
		t.Fatalf("frozen generated provenance changed: %+v", record)
	}
}

// TestCommitChapterRejectsPolishWithoutDraftChange 验证：已完成章节进入打磨/重写队列后，
// 若正文和标题都没有变化，commit_chapter 必须拒绝空返工。
// TestCommitChapterNonLayeredRecompletesAfterRework 验证非分层书完本后经 reopen 返工，
// 改完章节 commit、队列排空时能自动重新回到 complete（补 drain 后判完结的非分层分支）。

// TestCommitChapterLayeredReopenRecompletesDespiteOpenThread 验证收口：分层书经 reopen
// 返工后，即便 compass 仍有未收束长线（返工可能扰动），排空后也按"结构完整"重新完结——
// 不卡在 writing，杜绝终卷末越界续写死循环（§6.5 / known_outline_exhaustion 家族）。
// 反证：若 reopen 路径仍用质量级 layeredBookComplete，本例 open thread 会让其返 false、
// book_complete 为假，测试即失败。

// TestCommitChapterLayeredRejectsOutOfRangeChapter 验证分层模式下，
// 章号越出 layered_outline 的 commit 必须硬失败，而不是 slog.Warn 放行。
// 这是阻止"裁定误判后 writer 一路裸跑"的物理刹车（《凡骨》ch204..347 案例）。

// TestCommitChapterLayeredAutoCompletesWhenDone 验证分层模式确定性完结兜底：
// 大纲全部展开并写完 + 无骨架弧 + 无返工 + 活跃伏笔为零 + 指南针长线收束时，
// 最后一章 commit 自动推 Phase=Complete，不依赖架构师主动调 complete_book。
// 这是 9bf26a5 删掉分层自动完结后引入的 livelock 的修复（终卷末尾模型既不 append
// 也不 complete → 写手裸跑越界死循环）。

// TestCommitChapterFinaleVolumeCompletesDespiteOpenThreads 验证收官卷全链路：
// 已宣告收官卷（append_volume 带 final:true）后——
//  1. 末章 commit 不完结：完结不抢在卷末收尾三连（弧评审/弧摘要/卷摘要）之前，
//     结局必须过 editor 质量闸；
//  2. 三连齐备、卷摘要落盘（save_volume_summary 触发点）即完结，不再要求
//     伏笔/长线双归零——否则 estimated_scale 高估的书永远无法合法完本。
//
// 与下方 NoAutoCompleteWithOpenThreads 互为对照：同样带未收长线，未宣告不完结、已宣告完结。

// TestCommitChapterFinaleSkeletonArcBlocksCompletion 验证收官完结的结构闸门：
// 收官卷仍有骨架弧（规划内容未写）时，即使三连齐备也不得完结——这是防止
// "过早完结"的唯一防线（layeredStructurallyComplete 条件 2）。

// TestCommitChapterLayeredNoAutoCompleteWithOpenThreads 验证保守性：仍有活跃长线时
// 即使章节写满也不自动完结，把"是否继续"的裁定权留给架构师。

// TestCommitChapterLayeredNoAutoCompleteWithOpenThreads 验证保守性：仍有活跃长线时
// 即使章节写满也不自动完结，把"是否继续"的裁定权留给架构师。
