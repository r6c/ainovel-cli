package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestCommitChapterRejectsUnknownForeshadowReferenceBeforePending(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "推进", "characters": []string{"主角"}, "key_events": []string{"发现线索"},
		"foreshadow_updates": []map[string]any{{"id": "missing", "action": "resolve"}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil || !strings.Contains(err.Error(), "unknown id") {
		t.Fatalf("expected unknown foreshadow rejection, got %v", err)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("invalid args must not create pending commit: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterReinforcesForeshadow(t *testing.T) {
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
	if err := s.World.UpdateForeshadow(1, []domain.ForeshadowUpdate{
		{ID: "f1", Action: "plant", Description: "黑影身份"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, "第一章正文，黑影再次出现。摘要需要再讲清楚一点。事件继续推进。"); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "黑影再次出现", "characters": []string{"主角"}, "key_events": []string{"黑影现身"},
		"foreshadow_updates": []map[string]any{{"id": "f1", "action": "reinforce"}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	ledger, err := s.World.LoadForeshadowLedger()
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger) != 1 || ledger[0].Status != "reinforced" || ledger[0].LastAdvancedAt != 1 {
		t.Fatalf("foreshadow not reinforced by commit: %+v", ledger)
	}
}

func TestCommitChapterRejectsAdvancingResolvedForeshadowBeforePending(t *testing.T) {
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
	if err := s.World.UpdateForeshadow(1, []domain.ForeshadowUpdate{
		{ID: "f1", Action: "plant", Description: "黑影身份"},
		{ID: "f1", Action: "resolve"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, "第一章正文，已回收的黑影伏笔被错误推进。摘要需要再讲清楚一点。事件继续推进。"); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "错误推进已回收伏笔",
		"characters": []string{"主角"}, "key_events": []string{"黑影再次出现"},
		"foreshadow_updates": []map[string]any{{"id": "f1", "action": "advance"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected advancing resolved foreshadow to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("invalid transition must not create pending commit: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsReinforcingResolvedForeshadowBeforePending(t *testing.T) {
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
	if err := s.World.UpdateForeshadow(1, []domain.ForeshadowUpdate{
		{ID: "f1", Action: "plant", Description: "黑影身份"},
		{ID: "f1", Action: "resolve"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, "第一章正文，黑影再次出现。摘要需要再讲清楚一点。事件继续推进。"); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "黑影再次出现", "characters": []string{"主角"}, "key_events": []string{"黑影现身"},
		"foreshadow_updates": []map[string]any{{"id": "f1", "action": "reinforce"}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
		t.Fatal("expected reinforcing resolved foreshadow to fail")
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("invalid transition must not create pending commit: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsActionsAfterResolveInSamePayloadBeforePending(t *testing.T) {
	tests := []struct {
		name    string
		seed    bool
		actions []map[string]any
	}{
		{
			name: "resolve then reinforce existing entry", seed: true,
			actions: []map[string]any{{"id": "f1", "action": "resolve"}, {"id": "f1", "action": "reinforce"}},
		},
		{
			name: "resolve then partially pay off existing entry", seed: true,
			actions: []map[string]any{{"id": "f1", "action": "resolve"}, {"id": "f1", "action": "partial_payoff"}},
		},
		{
			name: "plant then resolve then reinforce",
			actions: []map[string]any{
				{"id": "f1", "action": "plant", "description": "黑影身份"},
				{"id": "f1", "action": "resolve"},
				{"id": "f1", "action": "reinforce"},
			},
		},
		{
			name: "resolve then repeat plant then reinforce", seed: true,
			actions: []map[string]any{
				{"id": "f1", "action": "resolve"},
				{"id": "f1", "action": "plant", "description": "黑影身份"},
				{"id": "f1", "action": "reinforce"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if tt.seed {
				if err := s.World.UpdateForeshadow(1, []domain.ForeshadowUpdate{{
					ID: "f1", Action: "plant", Description: "黑影身份",
				}}); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Drafts.SaveDraft(1, "第一章正文，黑影身份已经揭晓，却又被错误推进。摘要需要足够长。事件继续推进。"); err != nil {
				t.Fatal(err)
			}

			args, err := json.Marshal(map[string]any{
				"chapter": 1, "title": "第一章", "summary": "黑影身份揭晓",
				"characters": []string{"主角"}, "key_events": []string{"揭晓黑影身份"},
				"foreshadow_updates": tt.actions,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil {
				t.Fatal("expected action after resolve in the same payload to fail")
			}
			if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
				t.Fatalf("invalid action sequence must not create pending commit: pending=%+v err=%v", pending, err)
			}
		})
	}
}

func TestCommitChapterRewriteRebuildsForeshadowLifecycle(t *testing.T) {
	const foreshadowID = "f_broken_sword"
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	chapters := []struct {
		chapter int
		content string
		facts   domain.ChapterFacts
	}{
		{chapter: 1, content: "第一章旧正文。", facts: domain.ChapterFacts{
			Title: "第一章", Summary: "发现断剑", KeyEvents: []string{"发现断剑"},
			ForeshadowUpdates: []domain.ForeshadowUpdate{{ID: foreshadowID, Action: "plant", Description: "断剑的真正来历"}},
		}},
		{chapter: 2, content: "第二章旧正文。", facts: domain.ChapterFacts{
			Title: "第二章", Summary: "断剑再次共鸣", KeyEvents: []string{"断剑共鸣"},
			ForeshadowUpdates: []domain.ForeshadowUpdate{{ID: foreshadowID, Action: "reinforce"}},
		}},
	}
	for _, item := range chapters {
		if _, err := s.ChapterRecords.Accept(item.chapter, domain.ChapterOriginGenerated, item.content, item.facts, domain.StyleDelta{}); err != nil {
			t.Fatalf("accept chapter %d: %v", item.chapter, err)
		}
		if err := s.Drafts.SaveFinalChapter(item.chapter, item.content); err != nil {
			t.Fatalf("save final chapter %d: %v", item.chapter, err)
		}
		if err := s.Progress.MarkChapterComplete(item.chapter, len([]rune(item.content)), "", ""); err != nil {
			t.Fatalf("complete chapter %d: %v", item.chapter, err)
		}
	}
	if err := s.World.SaveForeshadowLedger([]domain.ForeshadowEntry{{
		ID: foreshadowID, Description: "断剑的真正来历", PlantedAt: 1, Status: "reinforced", LastAdvancedAt: 2,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetPendingRewrites([]int{2}, "测试伏笔生命周期重建"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "这是重写后的第二章正文，断剑再次发出共鸣。"); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "重写后断剑再次共鸣",
		"characters": []string{"主角"}, "key_events": []string{"断剑再次共鸣"},
		"foreshadow_updates": []map[string]any{{"id": foreshadowID, "action": "reinforce"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("rewrite chapter with reinforced foreshadow: %v", err)
	}

	ledger, err := s.World.LoadForeshadowLedger()
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger) != 1 || ledger[0].Status != "reinforced" || ledger[0].LastAdvancedAt != 2 {
		t.Fatalf("rewrite rebuilt wrong foreshadow lifecycle: %+v", ledger)
	}
	progress, err := s.Progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(progress.PendingRewrites) != 0 {
		t.Fatalf("rewrite queue not drained: %v", progress.PendingRewrites)
	}
	pending, err := s.Signals.LoadPendingCommit()
	if err != nil {
		t.Fatal(err)
	}
	if pending != nil {
		t.Fatalf("rewrite left pending commit: %+v", pending)
	}
}

func TestCommitChapterRewriteKeepsOwnForeshadowPlant(t *testing.T) {
	const foreshadowID = "f_spillway_photo"
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	// 第 2 章首次提交后的落盘状态：记录里是 plant，账本里已建起条目。
	if _, err := s.ChapterRecords.Accept(2, domain.ChapterOriginGenerated, "旧版正文。", domain.ChapterFacts{
		Title: "第二章", Summary: "埋下线索", KeyEvents: []string{"发现旧照"},
		ForeshadowUpdates: []domain.ForeshadowUpdate{{ID: foreshadowID, Action: "plant", Description: "泄洪道旧照"}},
	}, domain.StyleDelta{}); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if err := s.World.SaveForeshadowLedger([]domain.ForeshadowEntry{
		{ID: foreshadowID, Description: "泄洪道旧照", PlantedAt: 2, Status: "planted"},
	}); err != nil {
		t.Fatalf("SaveForeshadowLedger: %v", err)
	}
	if err := s.Progress.MarkChapterComplete(2, 3000, "", ""); err != nil {
		t.Fatalf("MarkChapterComplete: %v", err)
	}
	if err := s.Progress.SetPendingRewrites([]int{2}, "测试重写"); err != nil {
		t.Fatalf("SetPendingRewrites: %v", err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatalf("SetFlow: %v", err)
	}
	if err := s.Drafts.SaveDraft(2, "这是重写后的第二章正文。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "重写后埋线索",
		"characters": []string{"主角"}, "key_events": []string{"重新发现旧照"},
		"foreshadow_updates": []map[string]any{{"id": foreshadowID, "action": "advance"}},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("重写种植章不应失败: %v", err)
	}

	record, err := s.ChapterRecords.Load(2)
	if err != nil || record == nil {
		t.Fatalf("Load record: %+v err=%v", record, err)
	}
	var planted bool
	for _, u := range record.Facts.ForeshadowUpdates {
		if u.ID == foreshadowID && u.Action == "plant" {
			planted = true
		}
	}
	if !planted {
		t.Fatalf("重写后应保留本章 plant，实际 %+v", record.Facts.ForeshadowUpdates)
	}

	ledger, err := s.World.LoadForeshadowLedger()
	if err != nil {
		t.Fatalf("LoadForeshadowLedger: %v", err)
	}
	if len(ledger) != 1 || ledger[0].ID != foreshadowID || ledger[0].PlantedAt != 2 {
		t.Fatalf("账本应保留第 2 章的种植事实，实际 %+v", ledger)
	}

	progress, err := s.Progress.Load()
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if len(progress.PendingRewrites) != 0 {
		t.Fatalf("返工队列应已排空，实际 %v", progress.PendingRewrites)
	}
}

func TestCommitChapterRewriteRejectsForwardForeshadowReference(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	for _, ch := range []int{2, 7} {
		if _, err := s.ChapterRecords.Accept(ch, domain.ChapterOriginGenerated, "旧版正文。", domain.ChapterFacts{
			Title: fmt.Sprintf("第%d章", ch), Summary: "摘要", KeyEvents: []string{"事件"},
		}, domain.StyleDelta{}); err != nil {
			t.Fatalf("Accept %d: %v", ch, err)
		}
		if err := s.Progress.MarkChapterComplete(ch, 3000, "", ""); err != nil {
			t.Fatalf("MarkChapterComplete %d: %v", ch, err)
		}
	}
	if err := s.World.SaveForeshadowLedger([]domain.ForeshadowEntry{
		{ID: "f_late", Description: "第七章才埋的线", PlantedAt: 7, Status: "planted"},
	}); err != nil {
		t.Fatalf("SaveForeshadowLedger: %v", err)
	}
	if err := s.Progress.SetPendingRewrites([]int{2}, "测试重写"); err != nil {
		t.Fatalf("SetPendingRewrites: %v", err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatalf("SetFlow: %v", err)
	}
	if err := s.Drafts.SaveDraft(2, "这是重写后的第二章正文。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	args, err := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "s",
		"characters": []string{"主角"}, "key_events": []string{"e"},
		"foreshadow_updates": []map[string]any{{"id": "f_late", "action": "advance"}},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	_, err = newTestCommitChapterTool(s).Execute(context.Background(), args)
	if err == nil {
		t.Fatal("引用后续章节才种下的伏笔必须被拒")
	}
	if !strings.Contains(err.Error(), "种植于第 7 章") {
		t.Fatalf("报错须指明种植章，模型才能自行修正，实际: %v", err)
	}
	// 关键：拦在落盘之前——章节记录和返工队列都不得被这次失败污染。
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("校验失败不得留下 pending commit: pending=%+v err=%v", pending, err)
	}
	record, err := s.ChapterRecords.Load(2)
	if err != nil || record == nil {
		t.Fatalf("Load record: %+v err=%v", record, err)
	}
	if record.Content != "旧版正文。" {
		t.Fatalf("校验失败不得覆盖章节记录，实际 %q", record.Content)
	}
	p, err := s.Progress.Load()
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if len(p.PendingRewrites) != 1 || p.PendingRewrites[0] != 2 {
		t.Fatalf("返工队列应原样保留待重试: %v", p.PendingRewrites)
	}
}

func TestCommitChapterReplayKeepsSameChapterAdvancedThenResolvedForeshadow(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	updates := []domain.ForeshadowUpdate{
		{ID: "f", Action: "plant", Description: "断剑来历"},
		{ID: "f", Action: "reinforce"},
		{ID: "f", Action: "partial_payoff"},
		{ID: "f", Action: "advance"},
		{ID: "f", Action: "resolve"},
	}
	if err := s.World.UpdateForeshadow(1, updates); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "断剑真相完整揭晓",
		"characters": []string{"林墨"}, "key_events": []string{"回收断剑伏笔"},
		"foreshadow_updates": updates,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload,
		DraftContent: "第一章正文连续推进并完整回收断剑来历，所有变化都发生在同一冻结提交中。",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err != nil {
		t.Fatalf("replay foreshadow commit: %v", err)
	}
	ledger, err := s.World.LoadForeshadowLedger()
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger) != 1 || ledger[0].Status != "resolved" || ledger[0].PlantedAt != 1 ||
		ledger[0].LastAdvancedAt != 1 || ledger[0].ResolvedAt != 1 {
		t.Fatalf("foreshadow changed after replay: %+v", ledger)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("pending commit not cleared: pending=%+v err=%v", pending, err)
	}
}
