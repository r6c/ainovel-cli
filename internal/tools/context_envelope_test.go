package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestBuildProgressStatusHidesLayeredCapacityEstimate(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{TotalChapters: 66, Layered: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{
		Index: 1, Arcs: []domain.ArcOutline{
			{Index: 1, Chapters: []domain.OutlineEntry{{Title: "一"}, {Title: "二"}}},
			{Index: 2, EstimatedChapters: 64},
		},
	}}); err != nil {
		t.Fatal(err)
	}

	result := map[string]any{}
	newTestContextTool(st, References{}, "default").buildProgressStatus(result, &contextReads{})
	status, ok := result["progress_status"].(map[string]any)
	if !ok {
		t.Fatalf("progress_status = %#v", result["progress_status"])
	}
	if status["dynamic_planning"] != true || status["outlined_chapters"] != 2 {
		t.Fatalf("动态规划进度错误: %#v", status)
	}
	if _, exists := status["total_chapters"]; exists {
		t.Fatalf("分层容量估算不得作为 total_chapters 暴露: %#v", status)
	}
}

func TestContextToolDoesNotInjectUserDirectives(t *testing.T) {
	// save_directive 已移除：novel_context 不再注入 working_memory.user_directives，
	// 长期写作要求统一走 user_rules。锁死这条，防止回归。
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(3); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}

	tool := newTestContextTool(s, References{}, "default")
	for name, chapter := range map[string]int{"writer": 1, "architect": 0} {
		args, _ := json.Marshal(map[string]any{"chapter": chapter})
		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("[%s] Execute: %v", name, err)
		}
		var payload map[string]any
		if err := json.Unmarshal(result, &payload); err != nil {
			t.Fatalf("[%s] Unmarshal: %v", name, err)
		}
		working, ok := payload["working_memory"].(map[string]any)
		if !ok {
			t.Fatalf("[%s] missing working_memory", name)
		}
		if _, exists := working["user_directives"]; exists {
			t.Errorf("[%s] working_memory 不应再有 user_directives（已统一到 user_rules）", name)
		}
		// user_rules 仍应稳定注入
		if _, ok := working["user_rules"].(map[string]any); !ok {
			t.Errorf("[%s] working_memory.user_rules 应稳定注入", name)
		}
	}
}

// TestContextToolInjectsRuleViolations 违规事实管道契约(第五轮评审):
// commit 落盘的机械违规必须经 novel_context(chapter=N) 真实注入——
// editor.md §机械检查映射消费的就是这个字段,管道断了 prompt 就成空头支票。

func TestContextToolInjectsRuleViolations(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := st.Progress.Save(&domain.Progress{TotalChapters: 3, Phase: domain.PhaseWriting}); err != nil {
		t.Fatalf("progress: %v", err)
	}
	if err := st.World.SaveRuleViolations(2, []rules.Violation{
		{Rule: "fatigue_words", Target: "不禁", Actual: 9, Severity: rules.SeverityWarning},
	}); err != nil {
		t.Fatalf("save violations: %v", err)
	}

	tool := newTestContextTool(st, References{}, "default")
	args, _ := json.Marshal(map[string]any{"chapter": 2})
	raw, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	vs, ok := result["rule_violations"].([]any)
	if !ok || len(vs) != 1 {
		t.Fatalf("rule_violations 必须注入章节上下文, got %v", result["rule_violations"])
	}

	// 无违规章节:字段缺省(editor.md 约定)
	args3, _ := json.Marshal(map[string]any{"chapter": 3})
	raw3, err := tool.Execute(context.Background(), args3)
	if err != nil {
		t.Fatalf("Execute ch3: %v", err)
	}
	var result3 map[string]any
	_ = json.Unmarshal(raw3, &result3)
	if _, has := result3["rule_violations"]; has {
		t.Fatal("无违规章节不应带 rule_violations 字段")
	}
}
