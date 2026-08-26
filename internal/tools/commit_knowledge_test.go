package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestCommitChapterRejectsConflictingFutureKnowledgeBeforePending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(5); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.StartChapter(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "# 第二章\n\n足够长的返工正文，用于测试未来真相冲突必须在 PendingCommit 前拒绝。"); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(5, []domain.KnowledgeUpdate{{ID: "k_future", Action: "establish", Truth: "未来真相"}}); err != nil {
		t.Fatal(err)
	}
	tool := NewCommitChapterTool(s, NewStyleStatsIndex(s))
	args := map[string]any{
		"chapter": 2, "title": "第二章", "summary": "冲突建立", "characters": []string{}, "key_events": []string{"冲突"},
		"knowledge_updates": []domain.KnowledgeUpdate{{ID: "k_future", Action: "establish", Truth: "冲突真相"}},
	}
	raw, _ := json.Marshal(args)
	if _, err := tool.Execute(context.Background(), raw); err == nil {
		t.Fatal("expected conflicting future truth to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("future truth conflict must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsConflictingKnowledgeBeforePending(t *testing.T) {
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
	if err := s.World.UpdateKnowledge(1, []domain.KnowledgeUpdate{{
		ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, "第一章正文，叙事错误地把已经确立的黑影身份改成了另一个人。摘要需要足够清楚。"); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "错误修改作者真相",
		"characters": []string{"林墨"}, "key_events": []string{"错误修改真相"},
		"knowledge_updates": []map[string]any{{
			"id": "k_shadow", "action": "establish", "truth": "黑影是林墨的父亲",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected conflicting knowledge to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("conflicting truth must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsLearningUnknownKnowledgeBeforePending(t *testing.T) {
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
	if err := s.Drafts.SaveDraft(1, "第一章正文，林墨错误地获知一个尚未建立的作者真相。摘要需要足够清楚。事件继续推进。"); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "错误引用未知真相",
		"characters": []string{"林墨"}, "key_events": []string{"错误获知"},
		"knowledge_updates": []map[string]any{{
			"id": "k_missing", "action": "learn", "character": "林墨",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected learning unknown knowledge to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("unknown knowledge must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsBelievingUnknownKnowledgeBeforePending(t *testing.T) {
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
	if err := s.Drafts.SaveDraft(1, "第一章正文，林墨错误地相信一个尚未建立的真相，并据此采取行动。摘要需要足够清楚。 "); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "错误引用未知真相",
		"characters": []string{"林墨"}, "key_events": []string{"形成错误认知"},
		"knowledge_updates": []map[string]any{{
			"id": "k_missing", "action": "believe", "character": "林墨", "belief": "黑影是仇人",
		}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected belief about unknown knowledge to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("unknown belief must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsTrueBeliefBeforePending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(1, []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish", Truth: "黑影是兄长"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(1, 1000, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "第二章正文错误地把客观真相当成错误信念提交。摘要需要足够清楚。 "); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "错误信念与真相相同",
		"characters": []string{"林墨"}, "key_events": []string{"错误提交认知"},
		"knowledge_updates": []map[string]any{{
			"id": "k_shadow", "action": "believe", "character": "林墨", "belief": "黑影是兄长",
		}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected belief equal to truth to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("true belief must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsBeliefAfterCharacterKnowsTruthBeforePending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(3); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(1, []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish", Truth: "黑影是兄长"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(2, []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "learn", Character: "林墨"}}); err != nil {
		t.Fatal(err)
	}
	for chapter := 1; chapter <= 2; chapter++ {
		if err := s.Progress.MarkChapterComplete(chapter, 1000, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Drafts.SaveDraft(3, "第三章正文错误地让已经知道真相的林墨重新形成相反认知。摘要需要足够清楚。 "); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 3, "title": "第三章", "summary": "已知角色错误形成信念",
		"characters": []string{"林墨"}, "key_events": []string{"错误认知"},
		"knowledge_updates": []map[string]any{{
			"id": "k_shadow", "action": "believe", "character": "林墨", "belief": "黑影是仇人",
		}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected known character belief to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("known character belief must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsBeliefAfterLearningInSamePayloadBeforePending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(1, []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish", Truth: "黑影是兄长"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(1, 1000, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "第二章正文先让林墨获知真相，却又错误提交相反信念。摘要需要足够清楚。 "); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "同章认知顺序冲突",
		"characters": []string{"林墨"}, "key_events": []string{"获知后错误形成信念"},
		"knowledge_updates": []map[string]any{
			{"id": "k_shadow", "action": "learn", "character": "林墨"},
			{"id": "k_shadow", "action": "believe", "character": "林墨", "belief": "黑影是仇人"},
		},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected learn then believe to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("same-payload learn then believe must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsChangingActiveBeliefBeforePending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(3); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(1, []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish", Truth: "黑影是兄长"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(2, []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "believe", Character: "林墨", Belief: "黑影是仇人"}}); err != nil {
		t.Fatal(err)
	}
	for chapter := 1; chapter <= 2; chapter++ {
		if err := s.Progress.MarkChapterComplete(chapter, 1000, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Drafts.SaveDraft(3, "第三章正文错误地把林墨的稳定错误信念改成另一种内容。摘要需要足够清楚。 "); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 3, "title": "第三章", "summary": "错误改写角色信念",
		"characters": []string{"林墨"}, "key_events": []string{"改写错误认知"},
		"knowledge_updates": []map[string]any{{
			"id": "k_shadow", "action": "believe", "character": "林墨", "belief": "黑影是陌生人",
		}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected changing active belief to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("belief rewrite must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsRevealingUnknownKnowledgeBeforePending(t *testing.T) {
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
	if err := s.Drafts.SaveDraft(1, "第一章正文错误地向读者揭示一个尚未建立的作者真相。摘要需要足够清楚。"); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "错误揭示未知真相",
		"characters": []string{"林墨"}, "key_events": []string{"错误揭示"},
		"knowledge_updates": []map[string]any{{"id": "k_missing", "action": "reveal_to_reader"}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected revealing unknown knowledge to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("unknown reader reveal must fail before pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRevealsKnowledgeToReader(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(1, []domain.KnowledgeUpdate{{
		ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(1, 1000, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "第二章正文直接向读者揭示黑影是林墨的兄长，但林墨本人仍不知道这个真相。"); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "向读者揭示黑影身份",
		"characters": []string{"林墨"}, "key_events": []string{"读者得知黑影身份"},
		"knowledge_updates": []map[string]any{{"id": "k_shadow", "action": "reveal_to_reader"}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("execute reader reveal: %v", err)
	}
	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ReaderRevealedAt != 2 || len(entries[0].KnownBy) != 0 {
		t.Fatalf("commit reader reveal wrong: %+v", entries)
	}
}

func TestCommitChapterRecordsCharacterFalseBelief(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(1, []domain.KnowledgeUpdate{{
		ID: "k_shadow", Action: "establish", Truth: "黑影是林墨失踪多年的兄长",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(1, 1000, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "第二章正文，林墨认定黑影就是杀兄仇人，并据此决定追杀对方。这个误解明确改变了他的行动。 "); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "林墨误认黑影身份",
		"characters": []string{"林墨"}, "key_events": []string{"林墨形成错误认知"},
		"knowledge_updates": []map[string]any{{
			"id": "k_shadow", "action": "believe", "character": "林墨", "belief": "黑影是杀兄仇人",
		}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("execute belief: %v", err)
	}
	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(entries[0].BelievedBy) != 1 || entries[0].BelievedBy[0].FormedAt != 2 || len(entries[0].KnownBy) != 0 {
		t.Fatalf("commit did not record false belief: %+v", entries)
	}
}

func TestCommitChapterEstablishesBeliefAndLearningInSamePayload(t *testing.T) {
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
	if err := s.Drafts.SaveDraft(1, "第一章正文先建立黑影真实身份，让林墨误认其为仇人，随后又通过证据得知黑影其实是兄长。认知变化完整发生。 "); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "林墨由误解转为获知真相",
		"characters": []string{"林墨"}, "key_events": []string{"形成误解并获知真相"},
		"knowledge_updates": []map[string]any{
			{"id": "k_shadow", "action": "establish", "truth": "黑影是兄长"},
			{"id": "k_shadow", "action": "believe", "character": "林墨", "belief": "黑影是仇人"},
			{"id": "k_shadow", "action": "learn", "character": "林墨"},
		},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("execute establish-believe-learn: %v", err)
	}
	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(entries[0].KnownBy) != 1 || len(entries[0].BelievedBy) != 1 || entries[0].BelievedBy[0].CorrectedAt != 1 {
		t.Fatalf("same-payload knowledge state wrong: %+v", entries)
	}
}

func TestCommitChapterRecordsCharacterLearningKnowledge(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.World.UpdateKnowledge(1, []domain.KnowledgeUpdate{{
		ID: "k_shadow", Action: "establish", Truth: "黑影是林墨失踪多年的兄长",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(1, 1000, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "第二章正文，黑影亲口承认身份，林墨终于知道他是失踪多年的兄长。真相当场揭开。"); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "林墨得知黑影身份",
		"characters": []string{"林墨"}, "key_events": []string{"黑影承认身份"},
		"knowledge_updates": []map[string]any{{
			"id": "k_shadow", "action": "learn", "character": "林墨",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("execute learn knowledge: %v", err)
	}

	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(entries[0].KnownBy) != 1 || entries[0].KnownBy[0].Character != "林墨" || entries[0].KnownBy[0].LearnedAt != 2 {
		t.Fatalf("commit did not record character learning: %+v", entries)
	}
}

func TestCommitChapterEstablishesAndRevealsKnowledgeInSamePayload(t *testing.T) {
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
	if err := s.Drafts.SaveDraft(1, "第一章正文直接向读者揭示黑影是林墨的兄长，但林墨本人仍未获知。"); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "向读者揭示黑影身份",
		"characters": []string{"林墨"}, "key_events": []string{"读者得知身份"},
		"knowledge_updates": []map[string]any{
			{"id": "k_shadow", "action": "establish", "truth": "黑影是林墨的兄长"},
			{"id": "k_shadow", "action": "reveal_to_reader"},
		},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("execute establish then reader reveal: %v", err)
	}
	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].EstablishedAt != 1 || entries[0].ReaderRevealedAt != 1 || len(entries[0].KnownBy) != 0 {
		t.Fatalf("same-payload reader reveal wrong: %+v", entries)
	}
}

func TestCommitChapterEstablishesAndLearnsKnowledgeInSamePayload(t *testing.T) {
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
	if err := s.Drafts.SaveDraft(1, "第一章正文，黑影当面承认自己是林墨失踪多年的兄长，作者真相与角色获知同时成立。"); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "黑影当面承认身份",
		"characters": []string{"林墨"}, "key_events": []string{"黑影承认身份"},
		"knowledge_updates": []map[string]any{
			{"id": "k_shadow", "action": "establish", "truth": "黑影是林墨失踪多年的兄长"},
			{"id": "k_shadow", "action": "learn", "character": "林墨"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("execute establish then learn: %v", err)
	}

	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].EstablishedAt != 1 || len(entries[0].KnownBy) != 1 || entries[0].KnownBy[0].LearnedAt != 1 {
		t.Fatalf("same-payload knowledge sequence not applied: %+v", entries)
	}
}

func TestCommitChapterEstablishesKnowledge(t *testing.T) {
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
	if err := s.Drafts.SaveDraft(1, "第一章正文，林墨发现黑影其实是失踪多年的兄长。这个真相暂时只有叙事层确认。"); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "确认黑影身份",
		"characters": []string{"林墨"}, "key_events": []string{"确认黑影身份"},
		"knowledge_updates": []map[string]any{{
			"id": "k_shadow", "action": "establish", "truth": "黑影是林墨失踪多年的兄长",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("execute establish knowledge: %v", err)
	}

	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "k_shadow" || entries[0].EstablishedAt != 1 {
		t.Fatalf("commit did not establish knowledge: %+v", entries)
	}
}

func TestCommitChapterRewriteRejectsRemovingKnowledgeRequiredByLaterChapterBeforePending(t *testing.T) {
	tests := []struct {
		name       string
		later      domain.KnowledgeUpdate
		projection domain.KnowledgeEntry
	}{
		{
			name:  "later character learn",
			later: domain.KnowledgeUpdate{ID: "k_shadow", Action: "learn", Character: "林墨"},
			projection: domain.KnowledgeEntry{
				ID: "k_shadow", Truth: "黑影是林墨的兄长", EstablishedAt: 1,
				KnownBy: []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 3}},
			},
		},
		{
			name:  "later reader reveal",
			later: domain.KnowledgeUpdate{ID: "k_shadow", Action: "reveal_to_reader"},
			projection: domain.KnowledgeEntry{
				ID: "k_shadow", Truth: "黑影是林墨的兄长", EstablishedAt: 1, ReaderRevealedAt: 3,
			},
		},
		{
			name:  "later character belief",
			later: domain.KnowledgeUpdate{ID: "k_shadow", Action: "believe", Character: "林墨", Belief: "黑影是仇人"},
			projection: domain.KnowledgeEntry{
				ID: "k_shadow", Truth: "黑影是林墨的兄长", EstablishedAt: 1,
				BelievedBy: []domain.KnowledgeBelief{{Character: "林墨", Content: "黑影是仇人", FormedAt: 3}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewStore(t.TempDir())
			if err := s.Init(); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.Init(3); err != nil {
				t.Fatal(err)
			}
			facts := []domain.ChapterFacts{
				{Title: "第一章", Summary: "建立真相", KeyEvents: []string{"确认身份"},
					KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长"}}},
				{Title: "第二章", Summary: "继续追查", KeyEvents: []string{"追查"}},
				{Title: "第三章", Summary: "后续依赖", KeyEvents: []string{"后续依赖"},
					KnowledgeUpdates: []domain.KnowledgeUpdate{tt.later}},
			}
			for i, chapterFacts := range facts {
				chapter := i + 1
				content := fmt.Sprintf("第%d章旧正文。", chapter)
				if _, err := s.ChapterRecords.Accept(chapter, domain.ChapterOriginGenerated, content, chapterFacts, domain.StyleDelta{}); err != nil {
					t.Fatal(err)
				}
				if err := s.Drafts.SaveFinalChapter(chapter, content); err != nil {
					t.Fatal(err)
				}
				if err := s.Progress.MarkChapterComplete(chapter, len([]rune(content)), "", ""); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.World.SaveKnowledgeState([]domain.KnowledgeEntry{tt.projection}); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.SetPendingRewrites([]int{1}, "删除已不成立的作者真相"); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
				t.Fatal(err)
			}
			if err := s.Drafts.SaveDraft(1, "重写后的第一章正文，不再确立黑影身份。"); err != nil {
				t.Fatal(err)
			}
			args, _ := json.Marshal(map[string]any{
				"chapter": 1, "title": "第一章", "summary": "删除黑影真相",
				"characters": []string{"林墨"}, "key_events": []string{"身份仍未知"},
			})
			if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
				t.Fatal("expected rewrite to reject removing truth required by later chapter")
			}
			if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
				t.Fatalf("invalid rewrite must fail before pending: pending=%+v err=%v", pending, err)
			}
			record, err := s.ChapterRecords.Load(1)
			if err != nil || record == nil {
				t.Fatalf("load original record: record=%+v err=%v", record, err)
			}
			if len(record.Facts.KnowledgeUpdates) != 1 || record.Facts.KnowledgeUpdates[0].Action != "establish" {
				t.Fatalf("invalid rewrite overwrote original establish: %+v", record.Facts.KnowledgeUpdates)
			}
		})
	}
}

func TestCommitChapterRewriteRebuildsKnowledgeState(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	chapters := []struct {
		chapter int
		facts   domain.ChapterFacts
	}{
		{chapter: 1, facts: domain.ChapterFacts{
			Title: "第一章", Summary: "建立黑影真相", KeyEvents: []string{"确认身份"},
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长"}},
		}},
		{chapter: 2, facts: domain.ChapterFacts{
			Title: "第二章", Summary: "林墨获知真相", KeyEvents: []string{"承认身份"},
			KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "learn", Character: "林墨"}},
		}},
	}
	for _, item := range chapters {
		content := fmt.Sprintf("第%d章旧正文。", item.chapter)
		if _, err := s.ChapterRecords.Accept(item.chapter, domain.ChapterOriginGenerated, content, item.facts, domain.StyleDelta{}); err != nil {
			t.Fatal(err)
		}
		if err := s.Drafts.SaveFinalChapter(item.chapter, content); err != nil {
			t.Fatal(err)
		}
		if err := s.Progress.MarkChapterComplete(item.chapter, len([]rune(content)), "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.World.SaveKnowledgeState([]domain.KnowledgeEntry{{
		ID: "k_shadow", Truth: "黑影是林墨的兄长", EstablishedAt: 1,
		KnownBy: []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 2}},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetPendingRewrites([]int{2}, "测试知识状态重建"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "重写后的第二章正文，黑影再次当面承认身份，林墨获知真相。"); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "重写后林墨获知真相",
		"characters": []string{"林墨"}, "key_events": []string{"黑影承认身份"},
		"knowledge_updates": []map[string]any{{"id": "k_shadow", "action": "learn", "character": "林墨"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("rewrite chapter with knowledge: %v", err)
	}

	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(entries[0].KnownBy) != 1 || entries[0].KnownBy[0].LearnedAt != 2 {
		t.Fatalf("rewrite rebuilt wrong knowledge: %+v", entries)
	}
	progress, err := s.Progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(progress.PendingRewrites) != 0 {
		t.Fatalf("rewrite queue not drained: %v", progress.PendingRewrites)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("rewrite left pending commit: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterReplayKeepsFirstReaderRevealChapter(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	updates := []domain.KnowledgeUpdate{
		{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长"},
		{ID: "k_shadow", Action: "reveal_to_reader"},
	}
	if err := s.World.UpdateKnowledge(1, updates); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "向读者揭示身份",
		"characters": []string{"林墨"}, "key_events": []string{"读者得知身份"},
		"knowledge_updates": updates,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload,
		DraftContent: "第一章正文向读者揭示黑影是林墨的兄长，但林墨仍不知道。",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err != nil {
		t.Fatalf("replay reader reveal: %v", err)
	}
	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ReaderRevealedAt != 1 || len(entries[0].KnownBy) != 0 {
		t.Fatalf("reader reveal changed after replay: %+v", entries)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("pending commit not cleared: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterReplayDoesNotDuplicateKnowledgeState(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	updates := []domain.KnowledgeUpdate{
		{ID: "k_shadow", Action: "establish", Truth: "黑影是林墨失踪多年的兄长"},
		{ID: "k_shadow", Action: "believe", Character: "林墨", Belief: "黑影是杀兄仇人"},
		{ID: "k_shadow", Action: "learn", Character: "林墨"},
	}
	if err := s.World.UpdateKnowledge(1, updates); err != nil {
		t.Fatal(err)
	}
	persistedArgs, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "黑影承认身份",
		"characters": []string{"林墨"}, "key_events": []string{"黑影承认身份"},
		"knowledge_updates": updates,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: persistedArgs,
		DraftContent: "第一章正文，黑影承认自己是林墨失踪多年的兄长，林墨当场获知真相。",
	}); err != nil {
		t.Fatal(err)
	}

	newArgs, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "错误新标题", "summary": "错误新摘要",
		"characters": []string{"林墨"}, "key_events": []string{"错误事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), newArgs); err != nil {
		t.Fatalf("replay knowledge commit: %v", err)
	}

	entries, err := s.World.LoadKnowledgeState()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].EstablishedAt != 1 || len(entries[0].KnownBy) != 1 || entries[0].KnownBy[0].LearnedAt != 1 ||
		len(entries[0].BelievedBy) != 1 || entries[0].BelievedBy[0].FormedAt != 1 || entries[0].BelievedBy[0].CorrectedAt != 1 {
		t.Fatalf("knowledge changed after replay: %+v", entries)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("pending commit not cleared: pending=%+v err=%v", pending, err)
	}
}
