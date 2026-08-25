package host

import (
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
)

func TestCoCreateEntrypointsRejectExhaustedBudgetBeforeModel(t *testing.T) {
	newHost := func() *Host {
		return &Host{budget: NewBudgetSentinel(
			bootstrap.BudgetConfig{BookUSD: 1, WarnRatio: 0.8, HardStop: true},
			func() float64 { return 1.25 },
			func(string) {},
			func(string, string) {},
		)}
	}
	for name, call := range map[string]func(*Host) error{
		"cold_start": func(h *Host) error {
			_, err := h.CoCreateStream(t.Context(), []CoCreateMessage{{Role: "user", Content: "继续访谈"}}, nil)
			return err
		},
		"stage": func(h *Host) error {
			_, err := h.StageCoCreateStream(t.Context(), []CoCreateMessage{{Role: "user", Content: "继续规划"}}, nil)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(newHost()); err == nil || !strings.Contains(err.Error(), "预算上限") {
				t.Fatalf("err=%v want budget precondition", err)
			}
		})
	}
}

func TestColdStartCoCreatePromptDefinesOrderedInterviewStages(t *testing.T) {
	for _, phrase := range []string{
		"core → customization → title → confirmation → ready",
		"题材、主角、核心冲突、规模倾向",
		"世界观、视角、基调、目标读者",
		"书名与无剧透简介候选",
		"用户明确选择或授权",
		"完整创作指令",
		"## 核心定位", "## 深度定制", "## 书名与简介", "## 规划确认",
		"每轮最多提出 1 到 2 个",
		"<stage>",
		"下一轮当前阶段",
		"最多前进一格",
	} {
		if !strings.Contains(coCreateSystemPrompt, phrase) {
			t.Fatalf("cold cocreate prompt missing %q", phrase)
		}
	}
	if strings.Contains(stageCoCreateSystemPrompt, "core → customization") || strings.Contains(stageCoCreateSystemPrompt, "<stage>") {
		t.Fatal("running stage cocreate must not inherit cold-start interview stages")
	}
}
