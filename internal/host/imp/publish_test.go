package imp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/tools"
)

// spyCommitter 记录 Execute 调用次数，供发布幂等/恢复路径测试。
type spyCommitter struct{ calls int }

func (s *spyCommitter) Execute(context.Context, json.RawMessage) (json.RawMessage, error) {
	s.calls++
	return json.RawMessage(`{}`), nil
}

func TestPublishChaptersPersistsKnowledgeState(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	commit := tools.NewCommitChapterTool(st, tools.NewStyleStatsIndex(st))
	chapters := []ImportedChapterFacts{
		{
			Chapter: 1, Title: "第一章", Summary: "建立真相", KeyEvents: []string{"确认身份"},
			HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长"}},
		},
		{
			Chapter: 2, Title: "第二章", Summary: "角色获知", KeyEvents: []string{"承认身份"},
			HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "learn", Character: "林墨"}},
		},
	}
	for i, facts := range chapters {
		chapter := i + 1
		if err := publishChapter(context.Background(), st, commit, chapter, "导入章节正文，内容足够用于正式发布。", facts); err != nil {
			t.Fatalf("publish chapter %d: %v", chapter, err)
		}
	}
	entries, err := st.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].EstablishedAt != 1 || len(entries[0].KnownBy) != 1 || entries[0].KnownBy[0].LearnedAt != 2 {
		t.Fatalf("published knowledge state wrong: %+v", entries)
	}
}

func TestCommitArgsIncludesKnowledgeUpdates(t *testing.T) {
	updates := []domain.KnowledgeUpdate{{
		ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长",
	}}
	args := commitArgs(1, ImportedChapterFacts{
		Title: "第一章", Summary: "确认身份", KeyEvents: []string{"确认身份"},
		HookType: "mystery", DominantStrand: "quest", KnowledgeUpdates: updates,
	})
	got, ok := args["knowledge_updates"].([]domain.KnowledgeUpdate)
	if !ok || len(got) != 1 || got[0].ID != "k_shadow" {
		t.Fatalf("commit args dropped knowledge updates: %#v", args["knowledge_updates"])
	}
}

func TestCheckFoundationConflictsNormalizesBookMetadata(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "测试书", Synopsis: "测试简介"}); err != nil {
		t.Fatal(err)
	}
	f := &Foundation{Book: domain.BookMetadata{Title: " 测试书 ", Synopsis: " 测试简介 "}}
	if err := checkFoundationConflicts(st, f); err != nil {
		t.Fatalf("规范化后相同的作品信息不应冲突: %v", err)
	}
}

// TestPublishChapterHandlesStalePendingCommit 守护发布崩溃窗口的恢复：崩溃落在
// MarkChapterComplete 与 ClearPendingCommit 之间会残留指向本章的 pending_commit。
// 已完成章若直接跳过会绕开 commit 工具的清理分支，下一章 Execute 以 ErrToolConflict
// 拒绝，导入每次重跑死在同一处——命中残留时必须仍走一次工具幂等路径。
func TestPublishChapterHandlesStalePendingCommit(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.StartChapter(1); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.MarkChapterComplete(1, 100, "mystery", "quest"); err != nil {
		t.Fatal(err)
	}
	f := ImportedChapterFacts{Chapter: 1, Summary: "s", CoreEvent: "c", HookType: "mystery", DominantStrand: "quest"}

	// 无残留：已完成章零成本跳过，不触发 commit。
	spy := &spyCommitter{}
	if err := publishChapter(context.Background(), st, spy, 1, "正文", f); err != nil {
		t.Fatalf("已完成章应幂等跳过：%v", err)
	}
	if spy.calls != 0 {
		t.Fatalf("无残留不应调用 commit，得 %d 次", spy.calls)
	}

	// 残留指向本章：必须走一次 commit 幂等路径完成清理。
	if err := st.Signals.SavePendingCommit(domain.PendingCommit{Chapter: 1}); err != nil {
		t.Fatal(err)
	}
	if err := publishChapter(context.Background(), st, spy, 1, "正文", f); err != nil {
		t.Fatalf("残留清理路径不应失败：%v", err)
	}
	if spy.calls != 1 {
		t.Fatalf("命中残留应恰好调用 commit 一次，得 %d 次", spy.calls)
	}
}
