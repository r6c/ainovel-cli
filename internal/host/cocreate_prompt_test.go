package host

import (
	"strings"
	"testing"
)

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
