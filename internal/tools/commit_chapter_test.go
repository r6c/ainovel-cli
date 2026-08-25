package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/llmcontract"
	"github.com/voocel/ainovel-cli/internal/rules"
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

func TestChapterTargetMaxUsesBoundedOverflowSafeCalculation(t *testing.T) {
	if got := chapterTargetMax(rules.MaxChapterTargetChars); got != 1_200_000 {
		t.Fatalf("max chapter target upper bound=%d want=1200000", got)
	}
}

func TestCommitChapterRejectsPersistedChapterTargetAboveProductLimit(t *testing.T) {
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
	if err := s.UserRules.Save(&rules.Snapshot{Version: rules.SnapshotVersion, Status: rules.StatusReady,
		Structured: rules.Structured{ChapterTargetChars: rules.MaxChapterTargetChars + 1}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, "短正文。"); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "篇幅配置检查", "characters": []string{}, "key_events": []string{"事件"},
		"timeline_events": []any{}, "foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{}, "cast_intros": []any{}, "hook_type": nil, "dominant_strand": nil, "feedback": nil,
	})

	_, err := newTestCommitChapterTool(s).Execute(context.Background(), args)
	if err == nil || !errors.Is(err, errs.ErrToolPrecondition) || !strings.Contains(err.Error(), "篇幅目标非法") {
		t.Fatalf("oversized persisted target must fail before arithmetic, got %v", err)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending != nil {
		t.Fatalf("invalid target must fail before pending commit: pending=%+v err=%v", pending, loadErr)
	}
}

func TestCommitChapterRejectsChapterOverTargetBeforePendingCommit(t *testing.T) {
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
	snap := rules.BuildSnapshot([]rules.Candidate{{
		Source: "startup_prompt", Structured: rules.Structured{ChapterTargetChars: 100},
	}})
	if err := s.UserRules.Save(&snap); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, strings.Repeat("月", 121)); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "篇幅检查", "characters": []string{"林墨"}, "key_events": []string{"完成事件"},
		"timeline_events": []any{}, "foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{}, "cast_intros": []any{}, "hook_type": nil, "dominant_strand": nil, "feedback": nil,
	})

	_, err := newTestCommitChapterTool(s).Execute(context.Background(), args)
	if err == nil || !errors.Is(err, errs.ErrToolPrecondition) || !strings.Contains(err.Error(), "目标约 100 字") || !strings.Contains(err.Error(), "上限 120 字") || !strings.Contains(err.Error(), "当前 121 字") {
		t.Fatalf("over-target chapter should return actionable precondition error, got %v", err)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending != nil {
		t.Fatalf("length failure must precede pending commit: pending=%v err=%v", pending, loadErr)
	}
	if final, loadErr := s.Drafts.LoadChapterText(1); loadErr != nil || final != "" {
		t.Fatalf("length failure must not save final chapter: final=%q err=%v", final, loadErr)
	}
}

func TestCommitChapterDoesNotBlockChapterBelowTarget(t *testing.T) {
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
	snap := rules.BuildSnapshot([]rules.Candidate{{
		Source: "startup_prompt", Structured: rules.Structured{ChapterTargetChars: 100},
	}})
	if err := s.UserRules.Save(&snap); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, strings.Repeat("月", 80)); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "偏短但叙事完整", "characters": []string{"林墨"}, "key_events": []string{"完成事件"},
		"timeline_events": []any{}, "foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{}, "cast_intros": []any{}, "hook_type": nil, "dominant_strand": nil, "feedback": nil,
	})

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err != nil {
		t.Fatalf("chapter below target must not be padded by a mechanical minimum: %v", err)
	}
}

func TestCommitChapterRewriteRejectsChapterOverTargetBeforePendingCommit(t *testing.T) {
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
	oldFinal := "旧终稿。"
	if err := s.Drafts.SaveFinalChapter(1, oldFinal); err != nil {
		t.Fatal(err)
	}
	saveTestChapterRecord(t, s, 1, oldFinal)
	if err := s.Progress.MarkChapterComplete(1, len([]rune(oldFinal)), "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetPendingRewrites([]int{1}, "压缩篇幅"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatal(err)
	}
	snap := rules.BuildSnapshot([]rules.Candidate{{
		Source: "startup_prompt", Structured: rules.Structured{ChapterTargetChars: 100},
	}})
	if err := s.UserRules.Save(&snap); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, strings.Repeat("月", 121)); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "压缩篇幅", "characters": []string{"林墨"}, "key_events": []string{"完成重写"},
		"timeline_events": []any{}, "foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{}, "cast_intros": []any{}, "hook_type": nil, "dominant_strand": nil, "feedback": nil,
	})

	_, err := newTestCommitChapterTool(s).Execute(context.Background(), args)
	if err == nil || !errors.Is(err, errs.ErrToolPrecondition) || !strings.Contains(err.Error(), "当前 121 字") {
		t.Fatalf("rewrite over target should be rejected, got %v", err)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending != nil {
		t.Fatalf("rewrite length failure must precede pending commit: pending=%v err=%v", pending, loadErr)
	}
	if final, loadErr := s.Drafts.LoadChapterText(1); loadErr != nil || final != oldFinal {
		t.Fatalf("rewrite length failure must preserve old final: final=%q err=%v", final, loadErr)
	}
	progress, loadErr := s.Progress.Load()
	if loadErr != nil || !slices.Equal(progress.PendingRewrites, []int{1}) {
		t.Fatalf("rewrite queue must remain for retry: progress=%+v err=%v", progress, loadErr)
	}
}

func TestCommitChapterRejectsMarkdownResidueBeforePendingCommit(t *testing.T) {
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
	if err := s.Drafts.SaveDraft(1, "# 第一章\n\n**这行不应以 Markdown 加粗进入最终小说正文。**\n"); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "发现格式残留", "characters": []string{"林墨"}, "key_events": []string{"林墨发现格式残留"},
		"timeline_events": []any{}, "foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{}, "cast_intros": []any{}, "hook_type": nil, "dominant_strand": nil, "feedback": nil,
	})

	_, err := newTestCommitChapterTool(s).Execute(context.Background(), args)
	if err == nil || !errors.Is(err, errs.ErrToolPrecondition) || !strings.Contains(err.Error(), `Markdown 标记 "**"`) || !strings.Contains(err.Error(), "2 处") {
		t.Fatalf("markdown residue should return an actionable precondition error, got %v", err)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending != nil {
		t.Fatalf("markdown residue must fail before pending commit: pending=%v err=%v", pending, loadErr)
	}
	if final, loadErr := s.Drafts.LoadChapterText(1); loadErr != nil || final != "" {
		t.Fatalf("markdown residue must not save final chapter: final=%q err=%v", final, loadErr)
	}
}

func TestCommitChapterRewriteRejectsMarkdownResidueBeforePendingCommit(t *testing.T) {
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
	oldFinal := "# 第一章\n\n旧终稿。\n"
	if err := s.Drafts.SaveFinalChapter(1, oldFinal); err != nil {
		t.Fatal(err)
	}
	saveTestChapterRecord(t, s, 1, oldFinal)
	if err := s.Progress.MarkChapterComplete(1, len([]rune(oldFinal)), "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetPendingRewrites([]int{1}, "清理格式"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.SetFlow(domain.FlowRewriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveDraft(1, "# 第一章\n\n## 不应进入正文的副标题\n\n重写正文。\n"); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "清理格式", "characters": []string{"林墨"}, "key_events": []string{"完成重写"},
		"timeline_events": []any{}, "foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{}, "cast_intros": []any{}, "hook_type": nil, "dominant_strand": nil, "feedback": nil,
	})

	_, err := newTestCommitChapterTool(s).Execute(context.Background(), args)
	if err == nil || !errors.Is(err, errs.ErrToolPrecondition) || !strings.Contains(err.Error(), `Markdown 标记 "#"`) {
		t.Fatalf("rewrite markdown residue should be rejected, got %v", err)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending != nil {
		t.Fatalf("rewrite format failure must precede pending commit: pending=%v err=%v", pending, loadErr)
	}
	if final, loadErr := s.Drafts.LoadChapterText(1); loadErr != nil || final != oldFinal {
		t.Fatalf("rewrite format failure must preserve old final: final=%q err=%v", final, loadErr)
	}
	progress, loadErr := s.Progress.Load()
	if loadErr != nil || !slices.Equal(progress.PendingRewrites, []int{1}) {
		t.Fatalf("rewrite queue must remain for retry: progress=%+v err=%v", progress, loadErr)
	}
}

func TestCommitChapterPersistsDuplicateParagraphViolationWithoutBlocking(t *testing.T) {
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
	paragraph := "雨水沿着破旧窗棂缓慢滑落，林墨站在黑暗里听见远处钟声再次响起。"
	if err := s.Drafts.SaveDraft(1, "# 第一章\n"+paragraph+"\n他推门走入长廊。\n"+paragraph); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "林墨听见钟声", "characters": []string{"林墨"}, "key_events": []string{"林墨听见钟声"},
		"timeline_events": []any{}, "foreshadow_updates": []any{}, "relationship_changes": []any{}, "state_changes": []any{},
		"knowledge_updates": []any{}, "cast_intros": []any{}, "hook_type": nil, "dominant_strand": nil, "feedback": nil,
	})
	raw, err := newTestCommitChapterTool(s).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("duplicate paragraph warning must not block commit: %v", err)
	}
	var output struct {
		Committed      bool `json:"committed"`
		RuleViolations []struct {
			Rule string `json:"rule"`
		} `json:"rule_violations"`
	}
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatal(err)
	}
	if !output.Committed || !slices.ContainsFunc(output.RuleViolations, func(v struct {
		Rule string `json:"rule"`
	}) bool {
		return v.Rule == "duplicate_paragraph"
	}) {
		t.Fatalf("commit output missing duplicate paragraph fact: %+v", output)
	}
	stored := s.World.LoadRuleViolations(1)
	if !slices.ContainsFunc(stored, func(v rules.Violation) bool { return v.Rule == "duplicate_paragraph" }) {
		t.Fatalf("stored violations missing duplicate paragraph: %+v", stored)
	}
}

func TestCommitChapterSchemaIncludesKnowledgeUpdates(t *testing.T) {
	tool := newTestCommitChapterTool(store.NewStore(t.TempDir()))
	if err := llmcontract.ValidateStrictReady(tool.Schema()); err != nil {
		t.Fatalf("commit_chapter schema is not strict-ready: %v", err)
	}
	props, ok := tool.Schema()["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties missing: %#v", tool.Schema()["properties"])
	}
	knowledge, ok := props["knowledge_updates"].(map[string]any)
	if !ok {
		t.Fatalf("knowledge_updates schema missing: %#v", props["knowledge_updates"])
	}
	items, ok := knowledge["items"].(map[string]any)
	if !ok {
		t.Fatalf("knowledge_updates items missing: %#v", knowledge["items"])
	}
	itemProps, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatalf("knowledge update properties missing: %#v", items["properties"])
	}
	action, ok := itemProps["action"].(map[string]any)
	if !ok || fmt.Sprint(action["enum"]) != "[establish believe learn reveal_to_reader]" {
		t.Fatalf("knowledge action enum = %#v", action["enum"])
	}
	if _, ok := itemProps["belief"].(map[string]any); !ok {
		t.Fatalf("knowledge belief field missing: %#v", itemProps["belief"])
	}
}

func TestCommitChapterSchemaDescribesFeedbackAsObject(t *testing.T) {
	tool := newTestCommitChapterTool(store.NewStore(t.TempDir()))
	if !tool.StrictSchema() {
		t.Fatal("commit_chapter must use strict schema")
	}
	if err := llmcontract.ValidateStrictReady(tool.Schema()); err != nil {
		t.Fatalf("commit_chapter schema is not strict-ready: %v", err)
	}
	schema := tool.Schema()
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties missing: %#v", schema["properties"])
	}
	feedback, ok := props["feedback"].(map[string]any)
	if !ok {
		t.Fatalf("feedback schema missing: %#v", props["feedback"])
	}
	desc, _ := feedback["description"].(string)
	if !strings.Contains(desc, "JSON object") || !strings.Contains(desc, "字符串化 JSON") {
		t.Fatalf("feedback description should warn against stringified JSON, got %q", desc)
	}
	if got := fmt.Sprint(feedback["type"]); got != "[object null]" {
		t.Fatalf("feedback type = %v, want nullable object", feedback["type"])
	}
}

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

func TestCommitChapterRejectsInvalidNestedFields(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "推进", "characters": []string{"主角"}, "key_events": []string{"发现线索"},
		"relationship_changes": []map[string]any{{"character_a": "主角", "character_b": "", "relation": "敌对"}},
	})
	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), args); err == nil || !strings.Contains(err.Error(), "relationship_changes[0]") {
		t.Fatalf("expected nested field rejection, got %v", err)
	}
}

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

// TestCommitChapterRewriteKeepsOwnForeshadowPlant 锁死 issue #110：重写伏笔的"种植章"时，
// Writer 看到账本里该伏笔已存在，自然只写 advance；旧实现整条覆盖章节记录，plant 随之丢失，
// Projector 全量重放时报"推进未知伏笔"并把返工队列锁死。种植事实必须被保留。
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

// TestCommitChapterRewriteRejectsForwardForeshadowReference 与上一个用例同源（issue #110）：
// 账本是全书投影，重写早期章节时里面还躺着后续章节才种下的伏笔。旧实现放行 → Projector
// 按章序重放时报"推进未知伏笔"，且此时章节记录已被覆盖，返工队列就此锁死。
// 必须在落盘前挡下，并把"种植于第几章"讲清楚，模型才改得动。
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

// TestCommitChapterUpdatesCastLedger 验证：commit_chapter 把本章 characters 累加进 cast_ledger，
// cast_intros 提供的 brief_role 被采用，且 characters.json 中的核心角色不进入 ledger。
func TestCommitChapterUpdatesCastLedger(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatalf("UpdatePhase: %v", err)
	}
	// 设定核心角色档案（这些不应进 cast_ledger）
	if err := s.Characters.Save([]domain.Character{
		{Name: "林墨", Role: "主角", Tier: "core"},
		{Name: "李清砚", Role: "导师", Tier: "important"},
	}); err != nil {
		t.Fatalf("Save core characters: %v", err)
	}
	if err := s.Drafts.SaveDraft(1, "第一章正文，林墨遇到客栈老板老周与小厮阿云。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	tool := newTestCommitChapterTool(s)
	args, _ := json.Marshal(map[string]any{
		"chapter":    1,
		"title":      "第一章",
		"summary":    "林墨入住客栈",
		"characters": []string{"林墨", "李清砚", "老周", "阿云"},
		"key_events": []string{"入住"},
		"cast_intros": []any{
			map[string]any{"name": "老周", "brief_role": "客栈老板"},
			map[string]any{"name": "阿云", "brief_role": "客栈小厮"},
		},
	})
	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	summary, err := s.Summaries.LoadSummary(1)
	if err != nil {
		t.Fatal(err)
	}
	if summary == nil || summary.Title != "第一章" {
		t.Fatalf("committed title = %+v", summary)
	}

	entries, err := s.Cast.Load()
	if err != nil {
		t.Fatalf("Cast.Load: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 ledger entries (老周/阿云), got %d: %+v", len(entries), entries)
	}
	byName := map[string]domain.CastEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if e, ok := byName["老周"]; !ok || e.BriefRole != "客栈老板" || e.FirstSeenChapter != 1 {
		t.Errorf("老周 entry wrong: %+v", e)
	}
	if e, ok := byName["阿云"]; !ok || e.BriefRole != "客栈小厮" || e.AppearanceCount != 1 {
		t.Errorf("阿云 entry wrong: %+v", e)
	}
	if _, ok := byName["林墨"]; ok {
		t.Errorf("核心角色 林墨 不应进 ledger")
	}
	if _, ok := byName["李清砚"]; ok {
		t.Errorf("核心角色 李清砚 不应进 ledger")
	}
}

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

func TestCommitChapterRejectsTamperedSealedPayloadBeforeRecoverySideEffects(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	originalPayload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "原始标题", "summary": "原始摘要",
		"characters": []string{"林墨"}, "key_events": []string{"原始事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tamperedPayload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "篡改标题", "summary": "篡改摘要",
		"characters": []string{"林墨"}, "key_events": []string{"篡改事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft := "第一章冻结正文，长度足以证明恢复不应接受被单独替换的结构化载荷。"
	pendingFile, err := json.MarshalIndent(map[string]any{
		"chapter": 1, "stage": domain.CommitStageStarted,
		"payload": json.RawMessage(tamperedPayload), "draft_content": draft,
		"seal_version":   1,
		"payload_digest": fmt.Sprintf("%x", sha256.Sum256(originalPayload)),
		"draft_digest":   fmt.Sprintf("%x", sha256.Sum256([]byte(draft))),
		"intent_digest":  digestPendingIntent(domain.PendingCommit{Chapter: 1}),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir(), "meta", "pending_commit.json"), pendingFile, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil {
		t.Fatal("tampered sealed payload must be rejected")
	}
	if !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("tampered payload error category = %v, want ErrPendingCommitIntegrity", err)
	}
	if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
		t.Fatalf("tampered recovery must not create chapter record: record=%+v err=%v", record, loadErr)
	}
	progress, loadErr := s.Progress.Load()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if len(progress.CompletedChapters) != 0 {
		t.Fatalf("tampered recovery advanced progress: %+v", progress.CompletedChapters)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending == nil {
		t.Fatalf("tampered pending must be preserved: pending=%+v err=%v", pending, loadErr)
	}
}

func TestCommitChapterRejectsTamperedSealedDraftBeforeRecoverySideEffects(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "原始标题", "summary": "原始摘要",
		"characters": []string{"林墨"}, "key_events": []string{"原始事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	originalDraft := "第一章原始冻结正文，恢复只能使用这一份经过密封的正文快照。"
	tamperedDraft := "第一章篡改正文，虽然仍是合法文本，但摘要与冻结时不一致。"
	pendingFile, err := json.MarshalIndent(map[string]any{
		"chapter": 1, "stage": domain.CommitStageStarted,
		"payload": json.RawMessage(payload), "draft_content": tamperedDraft,
		"seal_version":   1,
		"payload_digest": fmt.Sprintf("%x", sha256.Sum256(payload)),
		"draft_digest":   fmt.Sprintf("%x", sha256.Sum256([]byte(originalDraft))),
		"intent_digest":  digestPendingIntent(domain.PendingCommit{Chapter: 1}),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir(), "meta", "pending_commit.json"), pendingFile, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil {
		t.Fatal("tampered sealed draft must be rejected")
	}
	if !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("tampered draft error category = %v, want ErrPendingCommitIntegrity", err)
	}
	if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
		t.Fatalf("tampered draft recovery must not create chapter record: record=%+v err=%v", record, loadErr)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending == nil {
		t.Fatalf("tampered pending must be preserved: pending=%+v err=%v", pending, loadErr)
	}
}

func TestCommitChapterRejectsOriginOnUnsealedLegacyPending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":[],"key_events":["事件"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Origin: domain.ChapterOriginImported,
		Payload: payload, DraftContent: "冻结正文",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil || !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("unsealed legacy pending cannot acquire imported provenance, got %v", err)
	}
	got, loadErr := s.Signals.LoadPendingCommit()
	if loadErr != nil || got == nil || got.SealVersion != 0 {
		t.Fatalf("invalid legacy pending must remain unsealed: pending=%+v err=%v", got, loadErr)
	}
}

func TestCommitChapterRejectsOriginTamperOnLegacyV1Seal(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":[],"key_events":["事件"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	pending := domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: "冻结正文"}
	payloadDigest, err := digestPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	pending.SealVersion = 1
	pending.PayloadDigest = payloadDigest
	pending.DraftDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(pending.DraftContent)))
	pending.IntentDigest = digestPendingIntent(pending)
	pending.Origin = domain.ChapterOriginImported // v1 历史格式未密封 origin，必须拒绝而非升级权限。
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil || !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("v1 origin tamper must fail integrity validation, got %v", err)
	}
	if got, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || got == nil {
		t.Fatalf("tampered pending must remain: pending=%+v err=%v", got, loadErr)
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

func TestCommitChapterRecoversImportedPendingWithFrozenProvenance(t *testing.T) {
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
	if err := s.UserRules.Save(&rules.Snapshot{Version: rules.SnapshotVersion, Status: rules.StatusReady,
		Structured: rules.Structured{ChapterTargetChars: 10}}); err != nil {
		t.Fatal(err)
	}
	content := "# 第一章\n\n导入的**原始强调**必须保留。\n"
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"导入摘要","characters":[],"key_events":["导入事件"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Origin: domain.ChapterOriginImported,
		Payload: payload, DraftContent: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pending.SealVersion != 2 {
		t.Fatalf("imported pending seal version=%d want=2", pending.SealVersion)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err != nil {
		t.Fatalf("recover imported pending through ordinary resume entry: %v", err)
	}
	record, err := s.ChapterRecords.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Origin != domain.ChapterOriginImported || record.Content != content {
		t.Fatalf("imported provenance/content lost on recovery: %+v", record)
	}
	if got, err := s.Signals.LoadPendingCommit(); err != nil || got != nil {
		t.Fatalf("imported pending not cleared: pending=%+v err=%v", got, err)
	}
}

func TestCommitChapterRejectsMalformedPendingSealBeforeRecoverySideEffects(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "摘要",
		"characters": []string{"林墨"}, "key_events": []string{"事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft := "第一章合法冻结正文。"
	payloadDigest := fmt.Sprintf("%x", sha256.Sum256(payload))
	draftDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(draft)))
	tests := []struct {
		name    string
		pending domain.PendingCommit
	}{
		{
			name: "half sealed legacy format",
			pending: domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: draft,
				PayloadDigest: payloadDigest},
		},
		{
			name: "unknown seal version",
			pending: domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: draft,
				SealVersion: 2, PayloadDigest: payloadDigest, DraftDigest: draftDigest},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewStore(t.TempDir())
			if err := s.Init(); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.Init(1); err != nil {
				t.Fatal(err)
			}
			if err := s.Signals.SavePendingCommit(tt.pending); err != nil {
				t.Fatal(err)
			}
			_, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
			if err == nil {
				t.Fatal("malformed pending seal must be rejected")
			}
			if !errors.Is(err, errs.ErrPendingCommitIntegrity) {
				t.Fatalf("malformed seal error category = %v, want ErrPendingCommitIntegrity", err)
			}
			if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
				t.Fatalf("malformed seal must fail before chapter record: record=%+v err=%v", record, loadErr)
			}
			if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending == nil {
				t.Fatalf("malformed pending must be preserved: pending=%+v err=%v", pending, loadErr)
			}
		})
	}
}

func TestCommitChapterRecoversSealedStateAppliedPending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":["林墨"],"key_events":["事件"]}`)
	draft := "第一章已经完成正文与状态落盘，恢复只应从进度阶段继续。"
	if err := s.Drafts.SaveFinalChapter(1, draft); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, draft, domain.ChapterFacts{
		Title: "第一章", Summary: "摘要", Characters: []string{"林墨"}, KeyEvents: []string{"事件"},
	}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Summaries.SaveSummary(domain.ChapterSummary{
		Chapter: 1, Title: "第一章", Summary: "摘要", Characters: []string{"林墨"}, KeyEvents: []string{"事件"},
	}); err != nil {
		t.Fatal(err)
	}
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStateApplied, Payload: payload, DraftContent: draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err != nil {
		t.Fatalf("recover state_applied: %v", err)
	}
	progress, err := s.Progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(progress.CompletedChapters, 1) {
		t.Fatalf("state_applied recovery did not mark progress: %+v", progress)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("state_applied recovery did not clear pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRecoversSealedSignalSavedPending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(1, 100, "", ""); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":["林墨"],"key_events":["事件"]}`)
	output := json.RawMessage(`{"chapter":1,"committed":true,"sealed":true}`)
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageSignalSaved, Payload: payload, DraftContent: "第一章冻结正文", Output: output,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	got, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err != nil {
		t.Fatalf("recover signal_saved: %v", err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, got); err != nil {
		t.Fatal(err)
	}
	if compact.String() != string(output) {
		t.Fatalf("signal_saved output=%s want=%s", got, output)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("signal_saved recovery did not clear pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsTamperedSealAtEveryRecoveryStage(t *testing.T) {
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":["林墨"],"key_events":["事件"]}`)
	originalDraft := "原始冻结正文"
	for _, stage := range []domain.CommitStage{
		domain.CommitStageStarted,
		domain.CommitStageStateApplied,
		domain.CommitStageProgressMarked,
		domain.CommitStageSignalSaved,
	} {
		t.Run(string(stage), func(t *testing.T) {
			s := store.NewStore(t.TempDir())
			if err := s.Init(); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.Init(1); err != nil {
				t.Fatal(err)
			}
			pending, err := sealPendingCommit(domain.PendingCommit{
				Chapter: 1, Stage: stage, Payload: payload, DraftContent: originalDraft,
			})
			if err != nil {
				t.Fatal(err)
			}
			pending.DraftContent = "篡改冻结正文"
			if err := s.Signals.SavePendingCommit(pending); err != nil {
				t.Fatal(err)
			}
			_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
			if err == nil || !strings.Contains(err.Error(), "draft 摘要不匹配") {
				t.Fatalf("stage %s must reject tampered seal, got %v", stage, err)
			}
			got, loadErr := s.Signals.LoadPendingCommit()
			if loadErr != nil || got == nil || got.Stage != stage {
				t.Fatalf("stage %s pending not preserved: pending=%+v err=%v", stage, got, loadErr)
			}
		})
	}
}

func TestCommitChapterRejectsIncoherentPendingMetadataBeforeRecoverySideEffects(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "摘要",
		"characters": []string{"林墨"}, "key_events": []string{"事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		draft       string
		rewrite     bool
		rewriteMode string
	}{
		{name: "empty draft"},
		{name: "ordinary commit with rewrite mode", draft: "冻结正文", rewriteMode: "rewrite"},
		{name: "rewrite with unknown mode", draft: "冻结正文", rewrite: true, rewriteMode: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewStore(t.TempDir())
			if err := s.Init(); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.Init(1); err != nil {
				t.Fatal(err)
			}
			pending, err := sealPendingCommit(domain.PendingCommit{
				Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: tt.draft,
				Rewrite: tt.rewrite, RewriteMode: tt.rewriteMode,
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Signals.SavePendingCommit(pending); err != nil {
				t.Fatal(err)
			}
			_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
			if err == nil {
				t.Fatal("incoherent pending metadata must be rejected")
			}
			if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
				t.Fatalf("incoherent pending must fail before chapter record: record=%+v err=%v", record, loadErr)
			}
		})
	}
}

func TestCommitChapterRejectsSealedPayloadWithInvalidFactsBeforeRecoverySideEffects(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "非法知识字段",
		"characters": []string{"林墨"}, "key_events": []string{"事件"},
		"knowledge_updates": []map[string]any{{"id": "k", "action": "establish"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft := "第一章冻结正文，结构化事实虽然摘要匹配，但字段矩阵本身非法。"
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil {
		t.Fatal("sealed payload with invalid chapter facts must be rejected")
	}
	if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
		t.Fatalf("invalid frozen payload must fail before chapter record: record=%+v err=%v", record, loadErr)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending == nil {
		t.Fatalf("invalid frozen pending must be preserved: pending=%+v err=%v", pending, loadErr)
	}
}

func TestCommitChapterSealsLegacyPendingBeforeReplayingState(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "推进未知伏笔",
		"characters": []string{"林墨"}, "key_events": []string{"事件"},
		"foreshadow_updates": []map[string]any{{"id": "missing", "action": "advance"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft := "第一章旧版冻结正文，恢复时会在伏笔状态应用阶段失败。"
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: draft,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err == nil {
		t.Fatal("state replay should fail on unknown foreshadow")
	}
	pending, err := s.Signals.LoadPendingCommit()
	if err != nil || pending == nil {
		t.Fatalf("legacy pending must remain after failed replay: pending=%+v err=%v", pending, err)
	}
	if pending.SealVersion != 2 || pending.Origin != domain.ChapterOriginGenerated || len(pending.PayloadDigest) != 64 || len(pending.DraftDigest) != 64 {
		t.Fatalf("legacy pending was not sealed before replay: %+v", pending)
	}
	wantPayload, err := digestPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	wantDraft := fmt.Sprintf("%x", sha256.Sum256([]byte(draft)))
	if pending.PayloadDigest != wantPayload || pending.DraftDigest != wantDraft {
		t.Fatalf("legacy pending seal does not match frozen inputs: %+v", pending)
	}
}

func TestCommitChapterDoesNotSealInvalidLegacyPayload(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","key_events":[]}`)
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: "冻结正文",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err == nil {
		t.Fatal("invalid legacy payload must be rejected")
	}
	pending, err := s.Signals.LoadPendingCommit()
	if err != nil || pending == nil {
		t.Fatalf("invalid legacy pending must remain: pending=%+v err=%v", pending, err)
	}
	if pending.SealVersion != 0 || pending.PayloadDigest != "" || pending.DraftDigest != "" {
		t.Fatalf("invalid legacy payload must not be sealed: %+v", pending)
	}
	if record, err := s.ChapterRecords.Load(1); err != nil || record != nil {
		t.Fatalf("invalid legacy payload created chapter record: record=%+v err=%v", record, err)
	}
}

func TestCommitChapterRejectsLegacyPayloadModifiedAfterAutomaticSeal(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":["林墨"],"key_events":["事件"],"foreshadow_updates":[{"id":"missing","action":"advance"}]}`)
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: "冻结正文",
	}); err != nil {
		t.Fatal(err)
	}
	tool := newTestCommitChapterTool(s)
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err == nil {
		t.Fatal("first replay must fail after sealing at state application")
	}
	sealed, err := s.Signals.LoadPendingCommit()
	if err != nil || sealed == nil || sealed.SealVersion != 2 || sealed.Origin != domain.ChapterOriginGenerated {
		t.Fatalf("legacy pending was not sealed: pending=%+v err=%v", sealed, err)
	}
	sealed.Payload = json.RawMessage(`{"chapter":1,"title":"篡改标题","summary":"摘要","characters":["林墨"],"key_events":["事件"]}`)
	if err := s.Signals.SavePendingCommit(*sealed); err != nil {
		t.Fatal(err)
	}

	_, err = tool.Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil || !strings.Contains(err.Error(), "摘要不匹配") {
		t.Fatalf("modified auto-sealed pending must fail integrity check, got %v", err)
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

func TestCommitChapterReplayAfterPartialCommitDoesNotDuplicateWorldState(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	if err := s.Drafts.SaveDraft(2, "第二章正文，林墨遇到黑影并突破。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	timeline := []domain.TimelineEvent{{
		Chapter:    2,
		Time:       "清晨",
		Event:      "林墨遇到黑影",
		Characters: []string{"林墨"},
	}}
	stateChanges := []domain.StateChange{{
		Chapter:  2,
		Entity:   "林墨",
		Field:    "realm",
		OldValue: "凡人",
		NewValue: "练气期",
	}}
	if err := s.World.UpdateForeshadow(1, []domain.ForeshadowUpdate{{
		ID: "f1", Action: "plant", Description: "黑影身份",
	}}); err != nil {
		t.Fatalf("plant foreshadow seed: %v", err)
	}
	foreshadow := []domain.ForeshadowUpdate{{
		ID:     "f1",
		Action: "reinforce",
	}}

	// 模拟 commit_chapter 已写入世界状态，但尚未 MarkChapterComplete 时进程崩溃。
	if err := s.World.AppendTimelineEvents(timeline); err != nil {
		t.Fatalf("AppendTimelineEvents seed: %v", err)
	}
	if err := s.World.AppendStateChanges(stateChanges); err != nil {
		t.Fatalf("AppendStateChanges seed: %v", err)
	}
	if err := s.World.UpdateForeshadow(2, foreshadow); err != nil {
		t.Fatalf("UpdateForeshadow seed: %v", err)
	}
	persistedArgs, _ := json.Marshal(map[string]any{
		"chapter":            2,
		"title":              "第二章",
		"summary":            "林墨遇到黑影并突破",
		"characters":         []string{"林墨"},
		"key_events":         []string{"遇到黑影", "突破"},
		"timeline_events":    timeline,
		"state_changes":      stateChanges,
		"foreshadow_updates": foreshadow,
	})
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter:      2,
		Stage:        domain.CommitStageStarted,
		Summary:      "半提交摘要",
		Payload:      persistedArgs,
		DraftContent: "第二章正文，林墨遇到黑影并突破。",
	}); err != nil {
		t.Fatalf("SavePendingCommit: %v", err)
	}
	if err := s.Drafts.SaveDraft(2, "重启后被新 Worker 覆盖、绝不能混入旧提交的正文。"); err != nil {
		t.Fatalf("overwrite draft: %v", err)
	}

	tool := newTestCommitChapterTool(s)
	// 模拟重启后的 Writer 重新生成了不同参数；恢复必须忽略它，使用 persistedArgs。
	args, _ := json.Marshal(map[string]any{
		"chapter":         2,
		"title":           "错误标题",
		"summary":         "错误的新摘要",
		"characters":      []string{"林墨"},
		"key_events":      []string{"错误事件"},
		"timeline_events": []domain.TimelineEvent{{Time: "夜晚", Event: "不应写入的新事件"}},
	})
	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute replay: %v", err)
	}

	events, _ := s.World.LoadTimeline()
	if len(events) != 1 {
		t.Fatalf("timeline duplicated after replay, got %d: %+v", len(events), events)
	}
	changes, _ := s.World.LoadStateChanges()
	if len(changes) != 1 {
		t.Fatalf("state changes duplicated after replay, got %d: %+v", len(changes), changes)
	}
	ledger, _ := s.World.LoadForeshadowLedger()
	if len(ledger) != 1 || ledger[0].Status != "reinforced" || ledger[0].LastAdvancedAt != 2 {
		t.Fatalf("foreshadow reinforce changed after replay: %+v", ledger)
	}
	pending, _ := s.Signals.LoadPendingCommit()
	if pending != nil {
		t.Fatalf("pending commit should be cleared, got %+v", pending)
	}
	if cp := s.Checkpoints.LatestByStep(domain.ChapterScope(2), "commit"); cp == nil {
		t.Fatal("commit checkpoint should be written")
	}
	final, err := s.Drafts.LoadChapterText(2)
	if err != nil {
		t.Fatalf("LoadChapterText: %v", err)
	}
	if final != "第二章正文，林墨遇到黑影并突破。" {
		t.Fatalf("recovery used overwritten draft: %q", final)
	}
}

func TestCommitChapterRecoversProgressMarkedWindowWithExactOutput(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatalf("UpdatePhase: %v", err)
	}
	if err := s.Progress.StartChapter(1); err != nil {
		t.Fatalf("StartChapter: %v", err)
	}
	if err := s.Drafts.SaveFinalChapter(1, "第一章终稿"); err != nil {
		t.Fatalf("SaveFinalChapter: %v", err)
	}
	if err := s.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 1, Title: "第一章", Summary: "摘要"}); err != nil {
		t.Fatalf("SaveSummary: %v", err)
	}
	if _, err := s.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "第一章终稿", domain.ChapterFacts{
		Title: "第一章", Summary: "摘要", KeyEvents: []string{"事件"},
	}, domain.StyleDelta{}); err != nil {
		t.Fatalf("SaveChapterRecord: %v", err)
	}
	if err := s.Progress.MarkChapterComplete(1, 100, "mystery", "quest"); err != nil {
		t.Fatalf("MarkChapterComplete: %v", err)
	}

	want := json.RawMessage(`{"chapter":1,"committed":true,"recovered":"exact"}`)
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1,
		Stage:   domain.CommitStageProgressMarked,
		Output:  want,
	}); err != nil {
		t.Fatalf("SavePendingCommit: %v", err)
	}

	tool := newTestCommitChapterTool(s)
	got, err := tool.Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err != nil {
		t.Fatalf("Execute recovery: %v", err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, got); err != nil {
		t.Fatalf("compact recovered output: %v", err)
	}
	if compact.String() != string(want) {
		t.Fatalf("recovered output = %s, want exact document %s", got, want)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("pending commit should be cleared, pending=%+v err=%v", pending, err)
	}
	if cp := s.Checkpoints.LatestByStep(domain.ChapterScope(1), "commit"); cp == nil {
		t.Fatal("commit checkpoint should be repaired")
	}
	p, err := s.Progress.Load()
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if p.InProgressChapter != 0 {
		t.Fatalf("in-progress chapter should be cleared, got %d", p.InProgressChapter)
	}
}

// TestCommitChapterRejectsPolishWithoutDraftChange 验证：已完成章节进入打磨/重写队列后，
// 若正文和标题都没有变化，commit_chapter 必须拒绝空返工。
// TestCommitChapterNonLayeredRecompletesAfterRework 验证非分层书完本后经 reopen 返工，
// 改完章节 commit、队列排空时能自动重新回到 complete（补 drain 后判完结的非分层分支）。
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

// TestCommitChapterLayeredReopenRecompletesDespiteOpenThread 验证收口：分层书经 reopen
// 返工后，即便 compass 仍有未收束长线（返工可能扰动），排空后也按"结构完整"重新完结——
// 不卡在 writing，杜绝终卷末越界续写死循环（§6.5 / known_outline_exhaustion 家族）。
// 反证：若 reopen 路径仍用质量级 layeredBookComplete，本例 open thread 会让其返 false、
// book_complete 为假，测试即失败。
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

// TestCommitChapterLayeredRejectsOutOfRangeChapter 验证分层模式下，
// 章号越出 layered_outline 的 commit 必须硬失败，而不是 slog.Warn 放行。
// 这是阻止"裁定误判后 writer 一路裸跑"的物理刹车（《凡骨》ch204..347 案例）。
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

// TestCommitChapterLayeredAutoCompletesWhenDone 验证分层模式确定性完结兜底：
// 大纲全部展开并写完 + 无骨架弧 + 无返工 + 活跃伏笔为零 + 指南针长线收束时，
// 最后一章 commit 自动推 Phase=Complete，不依赖架构师主动调 complete_book。
// 这是 9bf26a5 删掉分层自动完结后引入的 livelock 的修复（终卷末尾模型既不 append
// 也不 complete → 写手裸跑越界死循环）。
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

// TestCommitChapterFinaleVolumeCompletesDespiteOpenThreads 验证收官卷全链路：
// 已宣告收官卷（append_volume 带 final:true）后——
//  1. 末章 commit 不完结：完结不抢在卷末收尾三连（弧评审/弧摘要/卷摘要）之前，
//     结局必须过 editor 质量闸；
//  2. 三连齐备、卷摘要落盘（save_volume_summary 触发点）即完结，不再要求
//     伏笔/长线双归零——否则 estimated_scale 高估的书永远无法合法完本。
//
// 与下方 NoAutoCompleteWithOpenThreads 互为对照：同样带未收长线，未宣告不完结、已宣告完结。
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

// TestCommitChapterFinaleSkeletonArcBlocksCompletion 验证收官完结的结构闸门：
// 收官卷仍有骨架弧（规划内容未写）时，即使三连齐备也不得完结——这是防止
// "过早完结"的唯一防线（layeredStructurallyComplete 条件 2）。
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

// TestCommitChapterLayeredNoAutoCompleteWithOpenThreads 验证保守性：仍有活跃长线时
// 即使章节写满也不自动完结，把"是否继续"的裁定权留给架构师。

// TestCommitChapterLayeredNoAutoCompleteWithOpenThreads 验证保守性：仍有活跃长线时
// 即使章节写满也不自动完结，把"是否继续"的裁定权留给架构师。
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
