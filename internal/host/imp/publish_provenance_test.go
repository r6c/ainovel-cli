package imp

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/tools"
)

func TestPublishChaptersPersistsKnowledgeState(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(3); err != nil {
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
			Chapter: 2, Title: "第二章", Summary: "角色误解", KeyEvents: []string{"误认身份"},
			HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{
				{ID: "k_shadow", Action: "believe", Character: "林墨", Belief: "黑影是杀兄仇人"},
			},
		},
		{
			Chapter: 3, Title: "第三章", Summary: "角色获知", KeyEvents: []string{"承认身份"},
			HookType: "mystery", DominantStrand: "quest",
			KnowledgeUpdates: []domain.KnowledgeUpdate{
				{ID: "k_shadow", Action: "learn", Character: "林墨"},
				{ID: "k_shadow", Action: "reveal_to_reader"},
			},
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
	if len(entries) != 1 || entries[0].EstablishedAt != 1 || len(entries[0].KnownBy) != 1 || entries[0].KnownBy[0].LearnedAt != 3 ||
		entries[0].ReaderRevealedAt != 3 || len(entries[0].BelievedBy) != 1 || entries[0].BelievedBy[0].FormedAt != 2 || entries[0].BelievedBy[0].CorrectedAt != 3 {
		t.Fatalf("published knowledge state wrong: %+v", entries)
	}
}

func TestImportedFactsMappingMatchesPublishedCommitFacts(t *testing.T) {
	facts := ImportedChapterFacts{
		Chapter: 7, Title: "第七章", Summary: "真相推进", CoreEvent: "林墨确认密令",
		Characters:          []string{"林墨", "苏晚"},
		TimelineEvents:      []domain.TimelineEvent{{Time: "当夜", Event: "密令送达", Characters: []string{"林墨"}}},
		ForeshadowUpdates:   []domain.ForeshadowUpdate{{ID: "f1", Action: "plant", Description: "残缺密令"}},
		RelationshipChanges: []domain.RelationshipEntry{{CharacterA: "林墨", CharacterB: "苏晚", Relation: "暂时结盟"}},
		StateChanges:        []domain.StateChange{{Entity: "林墨", Field: "resolve", NewValue: "坚定", Reason: "确认密令"}},
		KnowledgeUpdates:    []domain.KnowledgeUpdate{{ID: "k1", Action: "establish", Truth: "城主下令封城"}},
		HookType:            "mystery", DominantStrand: "quest",
	}

	mapped := importedChapterFacts(facts)
	raw, err := json.Marshal(commitArgs(facts.Chapter, facts))
	if err != nil {
		t.Fatal(err)
	}
	var published struct {
		Chapter int `json:"chapter"`
		domain.ChapterFacts
	}
	if err := json.Unmarshal(raw, &published); err != nil {
		t.Fatal(err)
	}
	if published.Chapter != facts.Chapter || !reflect.DeepEqual(published.ChapterFacts, mapped) {
		t.Fatalf("validation mapping differs from published facts: mapped=%+v published=%+v", mapped, published.ChapterFacts)
	}
	if !reflect.DeepEqual(mapped.KeyEvents, []string{facts.CoreEvent}) {
		t.Fatalf("empty key_events must use core_event fallback, got %#v", mapped.KeyEvents)
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
