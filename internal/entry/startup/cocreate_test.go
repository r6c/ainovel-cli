package startup

import (
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/host"
)

func TestCoCreateSessionAdvancesInterviewOneStageAtATime(t *testing.T) {
	s := NewCoCreateSession("写一个悬疑故事")
	if s.Stage() != "core" || s.CanStart() {
		t.Fatalf("new session stage=%q canStart=%v", s.Stage(), s.CanStart())
	}

	s.ApplyReply(host.CoCreateReply{Prompt: "已有草稿", Stage: "title", Ready: true})
	if s.Stage() != "core" || s.CanStart() {
		t.Fatalf("jump must be rejected: stage=%q canStart=%v", s.Stage(), s.CanStart())
	}

	for _, stage := range []string{"customization", "title", "confirmation"} {
		s.ApplyReply(host.CoCreateReply{Prompt: "累计草稿", Stage: stage, Ready: true})
		if s.Stage() != stage || s.CanStart() {
			t.Fatalf("stage=%q got=%q canStart=%v", stage, s.Stage(), s.CanStart())
		}
	}

	complete := "## 核心定位\n- 悬疑\n\n## 深度定制\n- 冷峻\n\n## 书名与简介\n- 《雾》\n\n## 规划确认\n- 已确认"
	s.ApplyReply(host.CoCreateReply{Prompt: complete, Stage: "ready", Ready: true})
	if s.Stage() != "ready" || !s.CanStart() {
		t.Fatalf("ready session stage=%q canStart=%v", s.Stage(), s.CanStart())
	}
}

func TestCoCreateSessionHistoryReflectsAcceptedStageAndReadiness(t *testing.T) {
	s := NewCoCreateSession("写一本小说")
	s.ApplyReply(host.CoCreateReply{
		Message: "我已经跳到标题阶段",
		Prompt:  "不完整草稿",
		Stage:   "title",
		Ready:   true,
		Raw:     "<reply>我已经跳到标题阶段</reply><draft>不完整草稿</draft><stage>title</stage><ready>true</ready><suggestions></suggestions>",
	})
	if s.Stage() != "core" || s.Ready() || s.CanStart() {
		t.Fatalf("rejected jump leaked into state: stage=%q ready=%v canStart=%v", s.Stage(), s.Ready(), s.CanStart())
	}
	history := s.History()
	last := history[len(history)-1].Content
	if !strings.Contains(last, "<stage>core</stage>") || !strings.Contains(last, "<ready>false</ready>") {
		t.Fatalf("history must reflect accepted state, got %s", last)
	}
}

func TestCoCreateSessionNormalizedHistoryKeepsPreviousDraftWhenReplyOmitsIt(t *testing.T) {
	s := NewCoCreateSession("写一本小说")
	first := "## 核心定位\n- 悬疑\n\n## 深度定制\n- 待确认\n\n## 书名与简介\n- 待确认\n\n## 规划确认\n- 待确认"
	s.ApplyReply(host.CoCreateReply{Message: "先确认核心", Prompt: first, Stage: "customization"})
	s.ApplyReply(host.CoCreateReply{Message: "本轮格式降级，没有 draft", Stage: "customization", Raw: "本轮格式降级，没有 draft"})
	if s.DraftPrompt() != first {
		t.Fatalf("session draft drifted: %q", s.DraftPrompt())
	}
	last := s.History()[len(s.History())-1].Content
	if !strings.Contains(last, "<draft>"+first+"</draft>") {
		t.Fatalf("normalized history lost retained draft: %s", last)
	}
}

func TestStageCoCreateSessionPreservesRawHistory(t *testing.T) {
	s := NewStageCoCreateSession("规划后续")
	raw := "<reply>继续聊</reply><draft>后续方向</draft><ready>false</ready><suggestions></suggestions>"
	s.ApplyReply(host.CoCreateReply{Message: "继续聊", Prompt: "后续方向", Raw: raw})
	history := s.History()
	if history[len(history)-1].Content != raw {
		t.Fatalf("stage cocreate raw history changed: %q", history[len(history)-1].Content)
	}
}

func TestCoCreateSessionMissingOrInvalidStageKeepsCurrentStage(t *testing.T) {
	for _, reported := range []string{"", "unknown", "ready"} {
		s := NewCoCreateSession("写一本小说")
		s.ApplyReply(host.CoCreateReply{Prompt: "已有草稿", Stage: reported, Ready: true})
		if s.Stage() != "core" || s.CanStart() {
			t.Fatalf("reported=%q stage=%q canStart=%v", reported, s.Stage(), s.CanStart())
		}
	}
}

func TestCoCreateSessionReadyRequiresCompleteInterviewDraft(t *testing.T) {
	s := NewCoCreateSession("写一本小说")
	for _, stage := range []string{"customization", "title", "confirmation"} {
		s.ApplyReply(host.CoCreateReply{Prompt: "不完整草稿", Stage: stage})
	}
	s.ApplyReply(host.CoCreateReply{Prompt: "## 核心定位\n- 悬疑", Stage: "ready", Ready: true})
	if s.CanStart() {
		t.Fatal("ready stage with incomplete interview draft must not start")
	}
	if _, err := s.BuildPrompt(); err == nil {
		t.Fatal("BuildPrompt must explain incomplete interview draft")
	}

	inline := "- 文中提到 ## 核心定位\n- 文中提到 ## 深度定制\n- 文中提到 ## 书名与简介\n- 文中提到 ## 规划确认"
	s.ApplyReply(host.CoCreateReply{Prompt: inline, Stage: "ready", Ready: true})
	if s.CanStart() {
		t.Fatal("heading names embedded in body must not satisfy draft shape")
	}

	placeholder := "## 核心定位\n- 都市悬疑\n\n## 深度定制\n- 待确认\n\n## 书名与简介\n- 待确认\n\n## 规划确认\n- 待确认"
	s.ApplyReply(host.CoCreateReply{Prompt: placeholder, Stage: "ready", Ready: true})
	if s.CanStart() {
		t.Fatal("ready draft with pending placeholders must not start")
	}

	complete := "## 核心定位\n- 都市悬疑\n\n## 深度定制\n- 第一人称冷峻基调\n\n## 书名与简介\n- 《雾中来信》：无剧透简介\n\n## 规划确认\n- 用户已确认以上方向"
	s.ApplyReply(host.CoCreateReply{Prompt: complete, Stage: "ready", Ready: true})
	if !s.CanStart() {
		t.Fatal("complete confirmed interview draft should start")
	}
	got, err := s.BuildPrompt()
	if err != nil || got != complete {
		t.Fatalf("BuildPrompt=%q err=%v", got, err)
	}
}

func TestStageCoCreateSessionKeepsLegacyDraftGate(t *testing.T) {
	s := NewStageCoCreateSession("规划下一卷")
	s.ApplyReply(host.CoCreateReply{Prompt: "## 后续方向\n- 推进旧伏笔"})
	if !s.CanStart() {
		t.Fatal("stage cocreate should remain applicable once draft exists")
	}
}
