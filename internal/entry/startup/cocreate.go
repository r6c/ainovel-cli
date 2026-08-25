package startup

import (
	"fmt"
	"strings"

	"github.com/voocel/ainovel-cli/internal/host"
)

// CoCreateSession 承载共创模式的非 UI 状态。
type CoCreateSession struct {
	history        []host.CoCreateMessage
	draftPrompt    string
	ready          bool
	streamReply    string
	streamThinking string
	suggestions    []string
	stage          string
	staged         bool // true=运行中阶段共创，不应用冷启动访谈门禁
}

func NewCoCreateSession(initial string) *CoCreateSession {
	return &CoCreateSession{
		stage: "core",
		history: []host.CoCreateMessage{
			{Role: "user", Content: strings.TrimSpace(initial)},
		},
	}
}

// NewStageCoCreateSession 创建运行中阶段共创会话，保留原有“有草稿即可应用”语义。
func NewStageCoCreateSession(initial string) *CoCreateSession {
	s := NewCoCreateSession(initial)
	s.stage = ""
	s.staged = true
	return s
}

func (s *CoCreateSession) History() []host.CoCreateMessage {
	if s == nil {
		return nil
	}
	return append([]host.CoCreateMessage(nil), s.history...)
}

func (s *CoCreateSession) ApplyReply(reply host.CoCreateReply) {
	if s == nil {
		return
	}
	s.streamReply = ""
	s.streamThinking = ""
	// 仅当本轮带来非空 Prompt 才覆盖；格式降级时保留上一轮累计草稿。
	if prompt := strings.TrimSpace(reply.Prompt); prompt != "" {
		s.draftPrompt = prompt
	}
	if !s.staged {
		s.advanceStage(reply.Stage)
		s.ready = reply.Ready && s.stage == "ready" && completeInterviewDraft(s.draftPrompt)
	} else {
		s.ready = reply.Ready
	}
	// history 里保存完整协议，让下一轮模型看到累计草稿。冷启动会先由代码裁决
	// stage/ready，再写回规范化协议，避免模型跳级与确定性 Session 状态分叉。
	text := strings.TrimSpace(reply.Raw)
	if !s.staged && strings.TrimSpace(reply.Message) != "" {
		reply.Prompt = s.draftPrompt
		text = normalizedColdStartReply(reply, s.stage, s.ready)
	}
	if text == "" {
		text = strings.TrimSpace(reply.Message)
	}
	if text != "" {
		s.history = append(s.history, host.CoCreateMessage{Role: "assistant", Content: text})
	}
	// suggestions 直接覆盖（包括覆盖为空）：每轮的引导只对当下有意义。
	s.suggestions = append(s.suggestions[:0], reply.Suggestions...)
}

func normalizedColdStartReply(reply host.CoCreateReply, stage string, ready bool) string {
	var suggestions strings.Builder
	for _, suggestion := range reply.Suggestions {
		if suggestion = strings.TrimSpace(suggestion); suggestion != "" {
			suggestions.WriteString("- ")
			suggestions.WriteString(suggestion)
			suggestions.WriteByte('\n')
		}
	}
	return fmt.Sprintf("<reply>%s</reply>\n<draft>%s</draft>\n<stage>%s</stage>\n<ready>%t</ready>\n<suggestions>%s</suggestions>",
		strings.TrimSpace(reply.Message), strings.TrimSpace(reply.Prompt), stage, ready, strings.TrimSpace(suggestions.String()))
}

func (s *CoCreateSession) AppendUser(text string) {
	if s == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	// 用户已经决定下一句要说什么，suggestions 立即作废，避免 AI 还没回复时
	// 旧建议挂在输入框上误导。
	s.suggestions = nil
	s.history = append(s.history, host.CoCreateMessage{Role: "user", Content: text})
}

// ApplyDelta 接收流式累积；kind="thinking" 写入推理流，"reply" 写入回复预览。
// 两路分别累积，UI 可分块染色显示，让用户在 thinking 阶段也看到 LLM 在工作。
func (s *CoCreateSession) ApplyDelta(kind, text string) {
	if s == nil {
		return
	}
	text = strings.TrimSpace(text)
	switch kind {
	case host.CoCreateProgressThinking:
		s.streamThinking = text
	case host.CoCreateProgressReply:
		s.streamReply = text
	}
}

func (s *CoCreateSession) StreamReply() string {
	if s == nil {
		return ""
	}
	return s.streamReply
}

func (s *CoCreateSession) StreamThinking() string {
	if s == nil {
		return ""
	}
	return s.streamThinking
}

func (s *CoCreateSession) DraftPrompt() string {
	if s == nil {
		return ""
	}
	return s.draftPrompt
}

func (s *CoCreateSession) Suggestions() []string {
	if s == nil {
		return nil
	}
	return s.suggestions
}

// Stage 返回冷启动访谈当前阶段；运行中阶段共创返回空字符串。
func (s *CoCreateSession) Stage() string {
	if s == nil {
		return ""
	}
	return s.stage
}

func (s *CoCreateSession) advanceStage(reported string) {
	reported = strings.TrimSpace(reported)
	if reported == "" || reported == s.stage {
		return
	}
	next := map[string]string{
		"core":          "customization",
		"customization": "title",
		"title":         "confirmation",
		"confirmation":  "ready",
	}
	if next[s.stage] == reported {
		s.stage = reported
	}
}

func (s *CoCreateSession) Ready() bool {
	if s == nil {
		return false
	}
	return s.ready
}

func (s *CoCreateSession) CanStart() bool {
	if s == nil || strings.TrimSpace(s.DraftPrompt()) == "" {
		return false
	}
	if s.staged {
		return true
	}
	return s.stage == "ready" && s.ready && completeInterviewDraft(s.draftPrompt)
}

func completeInterviewDraft(draft string) bool {
	if strings.Contains(draft, "待确认") {
		return false
	}
	headings := make(map[string]struct{})
	for _, line := range strings.Split(strings.ReplaceAll(draft, "\r\n", "\n"), "\n") {
		headings[strings.TrimSpace(line)] = struct{}{}
	}
	for _, heading := range []string{"## 核心定位", "## 深度定制", "## 书名与简介", "## 规划确认"} {
		if _, ok := headings[heading]; !ok {
			return false
		}
	}
	return true
}

func (s *CoCreateSession) InitialInput() string {
	if s == nil || len(s.history) == 0 {
		return ""
	}
	return strings.TrimSpace(s.history[0].Content)
}

func (s *CoCreateSession) BuildPrompt() (string, error) {
	if s == nil || !s.CanStart() {
		return "", fmt.Errorf("cocreate interview must be confirmed with a complete draft")
	}
	return s.DraftPrompt(), nil
}
