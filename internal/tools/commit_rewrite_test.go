package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestCommitChapterRejectsNonPendingRewrite(t *testing.T) {
	dir := t.TempDir()
	store := store.NewStore(dir)
	if err := store.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := store.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	if err := store.Progress.MarkChapterComplete(2, 3000, "", ""); err != nil {
		t.Fatalf("MarkChapterComplete: %v", err)
	}
	if err := store.Progress.SetPendingRewrites([]int{2}, "测试重写"); err != nil {
		t.Fatalf("SetPendingRewrites: %v", err)
	}
	if err := store.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatalf("SetFlow: %v", err)
	}
	if err := store.Drafts.SaveDraft(3, "这是错误章节的正文。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	tool := newTestCommitChapterTool(store)
	args, err := json.Marshal(map[string]any{
		"chapter":         3,
		"title":           "第三章",
		"summary":         "错误提交",
		"characters":      []string{"主角"},
		"key_events":      []string{"误提交"},
		"timeline_events": []any{},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if _, err := tool.Execute(context.Background(), args); err == nil {
		t.Fatal("expected commit to be rejected during rewrite flow")
	}

	if _, err := os.Stat(dir + "/chapters/03.md"); !os.IsNotExist(err) {
		t.Fatalf("chapter should not be persisted, stat err=%v", err)
	}

	progress, err := store.Progress.Load()
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if len(progress.CompletedChapters) != 1 || progress.CompletedChapters[0] != 2 {
		t.Fatalf("completed chapters should only contain original chapter 2, got %v", progress.CompletedChapters)
	}
	if progress.CurrentChapter != 3 {
		t.Fatalf("current chapter should not advance beyond original progress, got %d", progress.CurrentChapter)
	}
}

func TestCommitChapterRejectsAutomaticRewriteOfImportedChapter(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChapterRecords.Accept(2, domain.ChapterOriginImported, "用户导入原文。", domain.ChapterFacts{
		Title: "第二章", Summary: "导入摘要", KeyEvents: []string{"导入事件"},
	}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(2, 100, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetPendingRewrites([]int{2}, "升级前残留返工队列"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "Writer 自动重写正文。"); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "自动重写",
		"characters": []string{"主角"}, "key_events": []string{"覆盖原文"},
	})

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil || !strings.Contains(err.Error(), "imported") {
		t.Fatalf("automatic rewrite of imported chapter must be rejected, got %v", err)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("rejection must happen before PendingCommit: pending=%+v err=%v", pending, err)
	}
	record, err := s.ChapterRecords.Load(2)
	if err != nil || record == nil || record.Origin != domain.ChapterOriginImported || record.Content != "用户导入原文。" {
		t.Fatalf("imported record changed: record=%+v err=%v", record, err)
	}
}

func TestCommitChapterAllowsPendingRewrite(t *testing.T) {
	dir := t.TempDir()
	store := store.NewStore(dir)
	if err := store.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := store.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	if err := store.Progress.MarkChapterComplete(2, 3000, "", ""); err != nil {
		t.Fatalf("MarkChapterComplete: %v", err)
	}
	if err := store.Progress.SetPendingRewrites([]int{2}, "测试重写"); err != nil {
		t.Fatalf("SetPendingRewrites: %v", err)
	}
	if err := store.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatalf("SetFlow: %v", err)
	}
	if err := store.Drafts.SaveDraft(2, "这是正确待重写章节的正文。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	tool := newTestCommitChapterTool(store)
	args, err := json.Marshal(map[string]any{
		"chapter":         2,
		"title":           "第二章",
		"summary":         "正确提交",
		"characters":      []string{"主角"},
		"key_events":      []string{"完成重写"},
		"timeline_events": []any{},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if _, err := os.Stat(dir + "/chapters/02.md"); err != nil {
		t.Fatalf("chapter should be persisted: %v", err)
	}

	progress, err := store.Progress.Load()
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if len(progress.CompletedChapters) != 1 || progress.CompletedChapters[0] != 2 {
		t.Fatalf("unexpected completed chapters: %v", progress.CompletedChapters)
	}
	pending, err := store.Signals.LoadPendingCommit()
	if err != nil {
		t.Fatalf("LoadPendingCommit: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected pending commit cleared, got %+v", pending)
	}
}

func TestCommitChapterRefreshesSharedStyleStatsAfterRewrite(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	completed := []int{1, 2, 3, 4, 5}
	for _, chapter := range completed {
		content := fmt.Sprintf("# 第%d章\n普通正文。\n故事继续。", chapter)
		if err := s.Drafts.SaveFinalChapter(chapter, content); err != nil {
			t.Fatal(err)
		}
		saveTestChapterRecord(t, s, chapter, content)
		if err := s.Progress.MarkChapterComplete(chapter, 100, "", ""); err != nil {
			t.Fatal(err)
		}
	}

	styleStats := NewStyleStatsIndex(s)
	before, err := styleStats.Snapshot(completed, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if before == nil {
		t.Fatal("expected initialized style stats")
	}

	if err := s.Progress.SetPendingRewrites([]int{2}, "测试增量风格统计"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(2, "# 第二章\n他不是退缩，而是在等待。\n改写后的故事继续。"); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter":    2,
		"title":      "第二章",
		"summary":    "完成增量统计测试重写",
		"characters": []string{"主角"},
		"key_events": []string{"完成重写"},
	})
	if _, err := NewCommitChapterTool(s, styleStats).Execute(context.Background(), args); err != nil {
		t.Fatal(err)
	}

	after, err := styleStats.Snapshot(completed, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pattern := range after.Patterns {
		if strings.HasPrefix(pattern.Name, "矫正句") && pattern.Total == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("rewrite did not refresh shared style stats: %+v", after.Patterns)
	}
}

func TestCommitChapterRejectsTamperedRewriteIntent(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveFinalChapter(1, "第一章旧终稿"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(1, 100, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetPendingRewrites([]int{1}, "测试密封返工"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"重写标题","summary":"重写摘要","characters":["林墨"],"key_events":["重写事件"]}`)
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Rewrite: true, RewriteMode: "rewrite",
		Payload: payload, DraftContent: "第一章已经完成的重写正文",
	})
	if err != nil {
		t.Fatal(err)
	}
	pending.Rewrite = false
	pending.RewriteMode = ""
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil || !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("tampered rewrite intent must fail integrity check, got %v", err)
	}
	got, loadErr := s.Signals.LoadPendingCommit()
	if loadErr != nil || got == nil {
		t.Fatalf("tampered rewrite pending must remain: pending=%+v err=%v", got, loadErr)
	}
}

func TestCommitChapterRewriteRecoveryUsesFrozenDraft(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatalf("UpdatePhase: %v", err)
	}
	if err := s.Drafts.SaveFinalChapter(2, "第二章旧终稿"); err != nil {
		t.Fatalf("SaveFinalChapter: %v", err)
	}
	if err := s.Progress.MarkChapterComplete(2, 100, "", ""); err != nil {
		t.Fatalf("MarkChapterComplete: %v", err)
	}
	if err := s.Progress.SetPendingRewrites([]int{2}, "测试重写恢复"); err != nil {
		t.Fatalf("SetPendingRewrites: %v", err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatalf("SetFlow: %v", err)
	}

	persistedArgs, err := json.Marshal(map[string]any{
		"chapter": 2, "title": "冻结标题", "summary": "冻结摘要", "characters": []string{"主角"}, "key_events": []string{"冻结事件"},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 2, Stage: domain.CommitStageStarted, Rewrite: true, RewriteMode: "rewrite",
		Payload: persistedArgs, DraftContent: "第二章已经完成的重写正文",
	}); err != nil {
		t.Fatalf("SavePendingCommit: %v", err)
	}
	if err := s.Drafts.SaveDraft(2, "重启后被错误覆盖的草稿"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	tool := newTestCommitChapterTool(s)
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"chapter":2,"summary":"新参数不得采用"}`)); err != nil {
		t.Fatalf("Execute recovery: %v", err)
	}
	final, err := s.Drafts.LoadChapterText(2)
	if err != nil {
		t.Fatalf("LoadChapterText: %v", err)
	}
	if final != "第二章已经完成的重写正文" {
		t.Fatalf("rewrite recovery used overwritten draft: %q", final)
	}
	summary, err := s.Summaries.LoadSummary(2)
	if err != nil {
		t.Fatalf("LoadSummary: %v", err)
	}
	if summary == nil || summary.Summary != "冻结摘要" {
		t.Fatalf("rewrite recovery used regenerated args: %+v", summary)
	}
}

func TestCommitChapterNonLayeredRecompletesAfterRework(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	// 两章写完并完结。第 2 章备齐 drafts/chapters，供返工提交。
	ch1 := "第一章原始正文。"
	ch2 := "第二章原始正文，用于模拟已提交终稿。"
	if err := s.Drafts.SaveFinalChapter(1, ch1); err != nil {
		t.Fatalf("SaveFinalChapter(1): %v", err)
	}
	if err := s.Drafts.SaveDraft(2, ch2); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	if err := s.Drafts.SaveFinalChapter(2, ch2); err != nil {
		t.Fatalf("SaveFinalChapter: %v", err)
	}
	saveTestChapterRecord(t, s, 1, ch1)
	saveTestChapterRecord(t, s, 2, ch2)
	if err := s.Progress.MarkChapterComplete(1, 100, "", ""); err != nil {
		t.Fatalf("MarkChapterComplete(1): %v", err)
	}
	if err := s.Progress.MarkChapterComplete(2, len([]rune(ch2)), "", ""); err != nil {
		t.Fatalf("MarkChapterComplete(2): %v", err)
	}
	if err := s.Progress.MarkComplete(); err != nil {
		t.Fatalf("MarkComplete: %v", err)
	}

	// reopen 第 2 章 → phase 回 writing、PendingRewrites=[2]、flow=rewriting
	if err := s.Progress.Reopen([]int{2}, "返工"); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	// 返工提交（草稿需与终稿不同才放行）
	if err := s.Drafts.SaveDraft(2, ch2+"\n\n返工新增段落。"); err != nil {
		t.Fatalf("SaveDraft (reworked): %v", err)
	}
	tool := newTestCommitChapterTool(s)
	args, _ := json.Marshal(map[string]any{
		"chapter":    2,
		"title":      "第二章",
		"summary":    "返工后摘要",
		"characters": []string{"主角"},
		"key_events": []string{"清理"},
	})
	raw, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute rework commit: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if payload["book_complete"] != true {
		t.Errorf("book_complete = %v, want true", payload["book_complete"])
	}

	p, _ := s.Progress.Load()
	if p.Phase != domain.PhaseComplete {
		t.Errorf("phase = %s, want complete (应自动重新收尾)", p.Phase)
	}
	if len(p.PendingRewrites) != 0 {
		t.Errorf("PendingRewrites = %v, want empty", p.PendingRewrites)
	}
}

func TestCommitChapterLayeredReopenRecompletesDespiteOpenThread(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(0); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	// 单卷单弧两章，全部展开
	foundation := NewSaveFoundationTool(s)
	layeredArgs, _ := json.Marshal(map[string]any{
		"type": "layered_outline",
		"content": []map[string]any{{
			"index": 1, "title": "卷一", "theme": "主题",
			"arcs": []map[string]any{{
				"index": 1, "title": "弧一", "goal": "目标",
				"chapters": []map[string]any{
					{"title": "首章", "core_event": "起", "hook": "续"},
					{"title": "次章", "core_event": "承", "hook": "终"},
				},
			}},
		}},
		"scale": "long",
	})
	if _, err := foundation.Execute(context.Background(), layeredArgs); err != nil {
		t.Fatalf("Execute layered: %v", err)
	}

	// 两章写完落盘并完结
	ch2 := "第二章原始正文，模拟已提交终稿。"
	for ch, body := range map[int]string{1: "第一章正文。", 2: ch2} {
		if err := s.Drafts.SaveDraft(ch, body); err != nil {
			t.Fatalf("SaveDraft %d: %v", ch, err)
		}
		if err := s.Drafts.SaveFinalChapter(ch, body); err != nil {
			t.Fatalf("SaveFinalChapter %d: %v", ch, err)
		}
		saveTestChapterRecord(t, s, ch, body)
		if err := s.Progress.MarkChapterComplete(ch, len([]rune(body)), "", ""); err != nil {
			t.Fatalf("MarkChapterComplete %d: %v", ch, err)
		}
	}
	if err := s.Progress.MarkComplete(); err != nil {
		t.Fatalf("MarkComplete: %v", err)
	}

	// 模拟"返工扰动了长线"：compass 仍有未收束的 open thread
	if err := s.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "主角归乡", OpenThreads: []string{"宿敌未除"}}); err != nil {
		t.Fatalf("SaveCompass: %v", err)
	}

	// reopen 第 2 章 → 返工提交（草稿需与终稿不同才放行）
	if err := s.Progress.Reopen([]int{2}, "返工"); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if err := s.Drafts.SaveDraft(2, ch2+"\n\n返工新增段落。"); err != nil {
		t.Fatalf("SaveDraft reworked: %v", err)
	}
	tool := newTestCommitChapterTool(s)
	args, _ := json.Marshal(map[string]any{
		"chapter": 2, "title": "第二章", "summary": "返工摘要", "characters": []string{"主角"}, "key_events": []string{"清理"},
	})
	raw, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute rework commit: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if bc, _ := out["book_complete"].(bool); !bc {
		t.Error("reopen 返工排空后应按结构完整重新完结（即便长线未收束）")
	}
	p, _ := s.Progress.Load()
	if p.Phase != domain.PhaseComplete {
		t.Errorf("phase = %s, want complete", p.Phase)
	}
	if p.ReopenedFromComplete {
		t.Error("重新完结后 ReopenedFromComplete 应被清除")
	}
}

func TestCommitChapterRejectsPolishWithoutDraftChange(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	// 模拟第 2 章已正常完成：drafts 与 chapters 内容相同。
	original := "第二章原始正文内容，用于模拟已提交终稿。"
	if err := s.Drafts.SaveDraft(2, original); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	if err := s.Drafts.SaveFinalChapter(2, original); err != nil {
		t.Fatalf("SaveFinalChapter: %v", err)
	}
	if err := s.Progress.MarkChapterComplete(2, len([]rune(original)), "mystery", "quest"); err != nil {
		t.Fatalf("MarkChapterComplete: %v", err)
	}
	if err := s.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 2, Title: "第二章", Summary: "原摘要"}); err != nil {
		t.Fatalf("SaveSummary: %v", err)
	}

	// 进入打磨队列：Flow=Polishing, PendingRewrites=[2]
	if err := s.Progress.SetPendingRewrites([]int{2}, "测试打磨"); err != nil {
		t.Fatalf("SetPendingRewrites: %v", err)
	}
	if err := s.Progress.SetFlow(domain.FlowPolishing); err != nil {
		t.Fatalf("SetFlow: %v", err)
	}

	tool := newTestCommitChapterTool(s)
	args, _ := json.Marshal(map[string]any{
		"chapter":    2,
		"title":      "第二章",
		"summary":    "假装打磨了",
		"characters": []string{"主角"},
		"key_events": []string{"无改动"},
	})
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected commit to be rejected when drafts equals final content")
	}

	// 再写一版不同的草稿 → 应该通过
	polished := original + "\n\n打磨后新增段落。"
	if err := s.Drafts.SaveDraft(2, polished); err != nil {
		t.Fatalf("SaveDraft (polished): %v", err)
	}
	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute after real polish: %v", err)
	}
}

func TestCommitChapterAllowsTitleOnlyRewrite(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	body := "正文无需修改，只有标题需要打磨。"
	if err := s.Drafts.SaveDraft(2, body); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveFinalChapter(2, body); err != nil {
		t.Fatal(err)
	}
	if err := s.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 2, Title: "旧标题", Summary: "原摘要"}); err != nil {
		t.Fatal(err)
	}
	before, err := s.Checkpoints.AppendArtifacts(
		domain.ChapterScope(2), "commit", "chapters/02.md", "summaries/02.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(2, len([]rune(body)), "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetPendingRewrites([]int{2}, "优化标题"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetFlow(domain.FlowPolishing); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{
		"chapter": 2, "title": "更准确的新标题", "summary": "原摘要",
		"characters": []string{"主角"}, "key_events": []string{"既有事件"},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("title-only rewrite failed: %v", err)
	}
	summary, err := s.Summaries.LoadSummary(2)
	if err != nil {
		t.Fatal(err)
	}
	if summary == nil || summary.Title != "更准确的新标题" {
		t.Fatalf("committed title = %+v", summary)
	}
	after := s.Checkpoints.LatestByStep(domain.ChapterScope(2), "commit")
	if after == nil || after.Seq <= before.Seq {
		t.Fatalf("title-only rewrite did not produce a new checkpoint: before=%+v after=%+v", before, after)
	}
	final, err := s.Drafts.LoadChapterText(2)
	if err != nil {
		t.Fatal(err)
	}
	if final != body {
		t.Fatalf("title-only rewrite changed body: %q", final)
	}
}

func TestCommitChapterLayeredRejectsOutOfRangeChapter(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(0); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	// 建一份 layered_outline，只有 1 卷 1 弧 1 章
	foundation := NewSaveFoundationTool(s)
	layeredArgs, _ := json.Marshal(map[string]any{
		"type": "layered_outline",
		"content": []map[string]any{{
			"index": 1, "title": "卷一", "theme": "主题",
			"arcs": []map[string]any{{
				"index": 1, "title": "弧一", "goal": "目标",
				"chapters": []map[string]any{
					{"title": "首章", "core_event": "起", "hook": "续"},
				},
			}},
		}},
		"scale": "long",
	})
	if _, err := foundation.Execute(context.Background(), layeredArgs); err != nil {
		t.Fatalf("Execute layered: %v", err)
	}
	_ = s.Progress.UpdatePhase(domain.PhaseWriting)

	// 越界章节 2 的 commit 必须硬失败
	if err := s.Drafts.SaveDraft(2, "越界章节正文，必须被拦下。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	tool := newTestCommitChapterTool(s)
	args, _ := json.Marshal(map[string]any{
		"chapter":    2,
		"title":      "第二章",
		"summary":    "越界章节",
		"characters": []string{"主角"},
		"key_events": []string{"不该被允许"},
	})
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected commit to fail when chapter out of layered outline range")
	}

	// 章节文件不应落盘、Progress 不应推进
	if _, statErr := os.Stat(dir + "/chapters/02.md"); !os.IsNotExist(statErr) {
		t.Fatalf("chapter 2 should not be persisted, stat err=%v", statErr)
	}
	progress, _ := s.Progress.Load()
	if len(progress.CompletedChapters) != 0 {
		t.Fatalf("CompletedChapters should stay empty, got %v", progress.CompletedChapters)
	}
}

func TestCommitChapterLayeredAutoCompletesWhenDone(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(0); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	// 单卷单弧两章，全部展开（无骨架弧）
	foundation := NewSaveFoundationTool(s)
	layeredArgs, _ := json.Marshal(map[string]any{
		"type": "layered_outline",
		"content": []map[string]any{{
			"index": 1, "title": "卷一", "theme": "主题",
			"arcs": []map[string]any{{
				"index": 1, "title": "弧一", "goal": "目标",
				"chapters": []map[string]any{
					{"title": "首章", "core_event": "起", "hook": "续"},
					{"title": "次章", "core_event": "承", "hook": "终"},
				},
			}},
		}},
		"scale": "long",
	})
	if _, err := foundation.Execute(context.Background(), layeredArgs); err != nil {
		t.Fatalf("Execute layered: %v", err)
	}
	// 指南针长线已收束（OpenThreads 空）
	if err := s.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "主角归乡"}); err != nil {
		t.Fatalf("SaveCompass: %v", err)
	}
	_ = s.Progress.UpdatePhase(domain.PhaseWriting)

	tool := newTestCommitChapterTool(s)
	commit := func(ch int) map[string]any {
		if err := s.Drafts.SaveDraft(ch, fmt.Sprintf("第 %d 章正文内容，用于测试确定性完结。", ch)); err != nil {
			t.Fatalf("SaveDraft %d: %v", ch, err)
		}
		args, _ := json.Marshal(map[string]any{
			"chapter": ch, "title": fmt.Sprintf("第%d章", ch), "summary": "摘要", "characters": []string{"主角"}, "key_events": []string{"事件"},
		})
		raw, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("Execute ch%d: %v", ch, err)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("Unmarshal ch%d: %v", ch, err)
		}
		return out
	}

	// 第 1 章：未写完，不应完结
	if bc, _ := commit(1)["book_complete"].(bool); bc {
		t.Fatal("写完第 1 章不应触发完结")
	}
	if p, _ := s.Progress.Load(); p.Phase == domain.PhaseComplete {
		t.Fatal("写完第 1 章 phase 不应为 complete")
	}

	// 第 2 章（最后一章）：应自动完结
	if bc, _ := commit(2)["book_complete"].(bool); !bc {
		t.Fatal("写完最后一章应自动完结")
	}
	if p, _ := s.Progress.Load(); p.Phase != domain.PhaseComplete {
		t.Fatalf("expected phase=complete, got %s", p.Phase)
	}
}

func TestCommitChapterFinaleVolumeCompletesDespiteOpenThreads(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(0); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	foundation := NewSaveFoundationTool(s)
	layeredArgs, _ := json.Marshal(map[string]any{
		"type": "layered_outline",
		"content": []map[string]any{{
			"index": 1, "title": "卷一", "theme": "主题",
			"arcs": []map[string]any{{
				"index": 1, "title": "弧一", "goal": "目标",
				"chapters": []map[string]any{{"title": "首章", "core_event": "起", "hook": "续"}},
			}},
		}},
		"scale": "long",
	})
	if _, err := foundation.Execute(context.Background(), layeredArgs); err != nil {
		t.Fatalf("Execute layered: %v", err)
	}

	// 卷末宣告收官卷：append_volume 带 final:true
	appendArgs, _ := json.Marshal(map[string]any{
		"type":   "append_volume",
		"reason": "长线可在一卷内收完，宣告收官卷",
		"content": map[string]any{
			"index": 2, "title": "终卷", "theme": "收束", "final": true,
			"arcs": []map[string]any{{
				"index": 1, "title": "收官弧", "goal": "回收所有长线",
				"chapters": []map[string]any{{"title": "终章", "core_event": "合", "hook": "终"}},
			}},
		},
	})
	raw, err := foundation.Execute(context.Background(), appendArgs)
	if err != nil {
		t.Fatalf("Execute append_volume: %v", err)
	}
	var appendOut map[string]any
	if err := json.Unmarshal(raw, &appendOut); err != nil {
		t.Fatalf("Unmarshal append result: %v", err)
	}
	if appendOut["final_volume"] != true {
		t.Fatalf("append_volume 应返回 final_volume=true 事实, got %v", appendOut)
	}

	// 长线未收束（未宣告时这会阻止完结，见对照测试）
	if err := s.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "主角归乡", OpenThreads: []string{"宿敌未除"}}); err != nil {
		t.Fatalf("SaveCompass: %v", err)
	}
	_ = s.Progress.UpdatePhase(domain.PhaseWriting)

	tool := newTestCommitChapterTool(s)
	commit := func(ch int) map[string]any {
		if err := s.Drafts.SaveDraft(ch, fmt.Sprintf("第 %d 章正文内容，用于收官卷完结测试。", ch)); err != nil {
			t.Fatalf("SaveDraft %d: %v", ch, err)
		}
		args, _ := json.Marshal(map[string]any{
			"chapter": ch, "title": fmt.Sprintf("第%d章", ch), "summary": "摘要", "characters": []string{"主角"}, "key_events": []string{"事件"},
		})
		raw, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("Execute ch%d: %v", ch, err)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("Unmarshal ch%d: %v", ch, err)
		}
		return out
	}

	// 第 1 章（非终卷末章）：不应完结
	if bc, _ := commit(1)["book_complete"].(bool); bc {
		t.Fatal("收官卷尚未写完不应完结")
	}
	// 第 2 章（收官卷末章）：卷末收尾三连未齐，完结不得抢在 editor 评审/摘要之前
	if bc, _ := commit(2)["book_complete"].(bool); bc {
		t.Fatal("末章 commit 时三连未齐，不应完结")
	}
	if p, _ := s.Progress.Load(); p.Phase == domain.PhaseComplete {
		t.Fatal("完结不应发生在卷末评审与摘要之前")
	}

	// 卷末收尾三连：弧评审 + 弧摘要落盘后，卷摘要（save_volume_summary）是完结触发点
	if err := s.World.SaveReview(domain.ReviewEntry{Chapter: 2, Scope: "arc", Verdict: "accept", Summary: "末弧评审"}); err != nil {
		t.Fatalf("SaveReview: %v", err)
	}
	if err := s.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 2, Arc: 1, Title: "收官弧", Summary: "收束", KeyEvents: []string{"终局"}}); err != nil {
		t.Fatalf("SaveArcSummary: %v", err)
	}
	volTool := NewSaveVolumeSummaryTool(s)
	volArgs, _ := json.Marshal(map[string]any{
		"volume": 2, "title": "终卷", "summary": "全卷收束", "key_events": []string{"终局"},
	})
	volRaw, err := volTool.Execute(context.Background(), volArgs)
	if err != nil {
		t.Fatalf("Execute save_volume_summary: %v", err)
	}
	var volOut map[string]any
	if err := json.Unmarshal(volRaw, &volOut); err != nil {
		t.Fatalf("Unmarshal volume summary result: %v", err)
	}
	if volOut["book_complete"] != true {
		t.Fatalf("卷摘要落盘应触发收官完结并回显 book_complete, got %v", volOut)
	}
	if p, _ := s.Progress.Load(); p.Phase != domain.PhaseComplete {
		t.Fatalf("expected phase=complete, got %s", p.Phase)
	}
}

func TestCommitChapterFinaleSkeletonArcBlocksCompletion(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(0); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	foundation := NewSaveFoundationTool(s)
	// 收官卷：第一弧展开 1 章，第二弧仍是骨架
	layeredArgs, _ := json.Marshal(map[string]any{
		"type": "layered_outline",
		"content": []map[string]any{{
			"index": 1, "title": "终卷", "theme": "收束", "final": true,
			"arcs": []map[string]any{
				{"index": 1, "title": "收官弧", "goal": "收线",
					"chapters": []map[string]any{{"title": "首章", "core_event": "起", "hook": "续"}}},
				{"index": 2, "title": "骨架弧", "goal": "待展开", "estimated_chapters": 5},
			},
		}},
		"scale": "long",
	})
	if _, err := foundation.Execute(context.Background(), layeredArgs); err != nil {
		t.Fatalf("Execute layered: %v", err)
	}
	if err := s.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "归乡"}); err != nil {
		t.Fatalf("SaveCompass: %v", err)
	}
	_ = s.Progress.UpdatePhase(domain.PhaseWriting)

	tool := newTestCommitChapterTool(s)
	if err := s.Drafts.SaveDraft(1, "第一章正文。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "摘要", "characters": []string{"主角"}, "key_events": []string{"事件"},
	})
	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// 三连齐备也不放行：骨架弧意味着规划内容还没写
	if err := s.World.SaveReview(domain.ReviewEntry{Chapter: 1, Scope: "arc", Verdict: "accept", Summary: "弧评审"}); err != nil {
		t.Fatalf("SaveReview: %v", err)
	}
	if err := s.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 1, Arc: 1, Title: "收官弧", Summary: "s", KeyEvents: []string{"e"}}); err != nil {
		t.Fatalf("SaveArcSummary: %v", err)
	}
	volTool := NewSaveVolumeSummaryTool(s)
	volArgs, _ := json.Marshal(map[string]any{
		"volume": 1, "title": "终卷", "summary": "s", "key_events": []string{"e"},
	})
	volRaw, err := volTool.Execute(context.Background(), volArgs)
	if err != nil {
		t.Fatalf("Execute save_volume_summary: %v", err)
	}
	var volOut map[string]any
	_ = json.Unmarshal(volRaw, &volOut)
	if volOut["book_complete"] == true {
		t.Fatal("收官卷仍有骨架弧时不得完结")
	}
	if p, _ := s.Progress.Load(); p.Phase == domain.PhaseComplete {
		t.Fatal("骨架弧未展开，phase 不应为 complete")
	}
}

func TestCommitChapterLayeredNoAutoCompleteWithOpenThreads(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(0); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	foundation := NewSaveFoundationTool(s)
	layeredArgs, _ := json.Marshal(map[string]any{
		"type": "layered_outline",
		"content": []map[string]any{{
			"index": 1, "title": "卷一", "theme": "主题",
			"arcs": []map[string]any{{
				"index": 1, "title": "弧一", "goal": "目标",
				"chapters": []map[string]any{{"title": "首章", "core_event": "起", "hook": "续"}},
			}},
		}},
		"scale": "long",
	})
	if _, err := foundation.Execute(context.Background(), layeredArgs); err != nil {
		t.Fatalf("Execute layered: %v", err)
	}
	// 仍有未收束的活跃长线
	if err := s.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "主角归乡", OpenThreads: []string{"宿敌未除"}}); err != nil {
		t.Fatalf("SaveCompass: %v", err)
	}
	_ = s.Progress.UpdatePhase(domain.PhaseWriting)

	if err := s.Drafts.SaveDraft(1, "唯一一章的正文，但长线未收束。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	tool := newTestCommitChapterTool(s)
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "摘要", "characters": []string{"主角"}, "key_events": []string{"事件"},
	})
	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if p, _ := s.Progress.Load(); p.Phase == domain.PhaseComplete {
		t.Fatal("活跃长线未收束时不应自动完结")
	}
}
