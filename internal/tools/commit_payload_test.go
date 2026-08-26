package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/llmcontract"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
)

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
