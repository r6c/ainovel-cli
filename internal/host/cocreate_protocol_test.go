package host

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestCoCreateStreamRecordsThinkingUsage(t *testing.T) {
	final := agentcore.Message{
		Role: agentcore.RoleAssistant,
		Content: []agentcore.ContentBlock{agentcore.TextBlock(`<reply>继续澄清</reply>
<draft>## 核心定位
- 悬疑
## 深度定制
- 待确认
## 书名与简介
- 待确认
## 规划确认
- 待确认</draft>
<stage>core</stage>
<ready>false</ready>
<suggestions></suggestions>`)},
		StopReason: agentcore.StopReasonStop,
		Usage:      &agentcore.Usage{Input: 100, Output: 20, TotalTokens: 120},
	}
	model := &streamUsageTestModel{plainTrackedTestModel: &plainTrackedTestModel{}, message: final}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}
	usage := NewUsageTracker(models, nil)

	reply, err := coCreateStream(t.Context(), models, nil, usage, coCreateSystemPrompt, []CoCreateMessage{{Role: "user", Content: "写悬疑小说"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(reply.Message) == "" {
		t.Fatal("streamed reply must still reach the caller")
	}
	perAgent := usage.PerAgent()
	if len(perAgent) != 1 || perAgent[0].Role != "thinking" || perAgent[0].Input != 100 || perAgent[0].Output != 20 {
		t.Fatalf("thinking usage not recorded: %+v", perAgent)
	}
}

func TestParseCoCreateResponseExtractsInterviewStage(t *testing.T) {
	raw := `<reply>核心定位已清楚，接下来聊定制。</reply>
<draft>## 核心定位
- 悬疑</draft>
<stage>customization</stage>
<ready>false</ready>
<suggestions>- 第一人称
- 冷峻基调</suggestions>`
	reply, err := parseCoCreateResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Stage != "customization" || reply.Ready {
		t.Fatalf("parsed reply wrong: %+v", reply)
	}
	if preview := extractReplyPreview(raw); strings.Contains(preview, "stage") || strings.Contains(preview, "customization") {
		t.Fatalf("stream preview leaked protocol stage: %q", preview)
	}
}

func TestParseCoCreateResponseLeavesMissingOrInvalidStageEmpty(t *testing.T) {
	for _, raw := range []string{
		`<reply>继续聊</reply><draft>草稿</draft><ready>false</ready><suggestions></suggestions>`,
		`<reply>继续聊</reply><draft>草稿</draft><stage>unknown</stage><ready>true</ready><suggestions></suggestions>`,
	} {
		reply, err := parseCoCreateResponse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if reply.Stage != "" {
			t.Fatalf("invalid/missing stage must be empty: %+v", reply)
		}
	}
}

// thinkingOnlyStreamModel 模拟 Provider 只返回 reasoning/thinking、没有最终答案。
type thinkingOnlyStreamModel struct {
	*plainTrackedTestModel
	thinking string
}

func (m *thinkingOnlyStreamModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	ch := make(chan agentcore.StreamEvent, 2)
	ch <- agentcore.StreamEvent{Type: agentcore.StreamEventThinkingDelta, Delta: m.thinking}
	ch <- agentcore.StreamEvent{
		Type: agentcore.StreamEventDone,
		Message: agentcore.Message{
			Role:       agentcore.RoleAssistant,
			Content:    []agentcore.ContentBlock{agentcore.ThinkingBlock(m.thinking)},
			StopReason: agentcore.StopReasonStop,
		},
	}
	close(ch)
	return ch, nil
}

func TestCoCreateStreamDoesNotExposeThinkingAsReply(t *testing.T) {
	const hidden = "内部推理不应出现在用户回复"
	model := &thinkingOnlyStreamModel{
		plainTrackedTestModel: &plainTrackedTestModel{},
		thinking:              hidden,
	}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}

	reply, err := coCreateStream(
		t.Context(), models, nil, nil, coCreateSystemPrompt,
		[]CoCreateMessage{{Role: "user", Content: "继续写"}}, nil,
	)
	if err == nil {
		t.Fatalf("thinking-only response should fail, reply=%+v", reply)
	}
	if strings.Contains(reply.Message, hidden) || strings.Contains(reply.Prompt, hidden) {
		t.Fatalf("thinking leaked into reply: %+v", reply)
	}
}

type finalTextOnlyStreamModel struct {
	*plainTrackedTestModel
	message agentcore.Message
}

func (m *finalTextOnlyStreamModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	ch := make(chan agentcore.StreamEvent, 1)
	ch <- agentcore.StreamEvent{Type: agentcore.StreamEventDone, Message: m.message, StopReason: m.message.StopReason}
	close(ch)
	return ch, nil
}

func TestCoCreateSessionLogDoesNotPersistThinkingText(t *testing.T) {
	const hidden = "绝密内部推理内容"
	final := agentcore.Message{
		Role: agentcore.RoleAssistant,
		Content: []agentcore.ContentBlock{agentcore.TextBlock(`<reply>用户可见内容</reply>
<draft>草稿</draft>
<ready>false</ready>
<suggestions></suggestions>`)},
		StopReason: agentcore.StopReasonStop,
	}
	model := &thinkingAndFinalStreamModel{
		plainTrackedTestModel: &plainTrackedTestModel{},
		thinking:              hidden,
		message:               final,
	}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}
	dir := t.TempDir()
	sessions := store.NewStore(dir).Sessions

	if _, err := coCreateStream(
		t.Context(), models, sessions, nil, coCreateSystemPrompt,
		[]CoCreateMessage{{Role: "user", Content: "继续写"}}, nil,
	); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "meta/sessions/cocreate.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), hidden) {
		t.Fatalf("session log persisted thinking text: %s", data)
	}
	if got, ok := entry["thinking_len"].(float64); !ok || int(got) != len([]rune(hidden)) {
		t.Fatalf("thinking length missing or wrong: %#v", entry["thinking_len"])
	}
}

type thinkingAndFinalStreamModel struct {
	*plainTrackedTestModel
	thinking  string
	message   agentcore.Message
	textDelta string
}

func (m *thinkingAndFinalStreamModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	ch := make(chan agentcore.StreamEvent, 3)
	ch <- agentcore.StreamEvent{Type: agentcore.StreamEventThinkingDelta, Delta: m.thinking}
	if m.textDelta != "" {
		ch <- agentcore.StreamEvent{Type: agentcore.StreamEventTextDelta, Delta: m.textDelta}
	}
	ch <- agentcore.StreamEvent{Type: agentcore.StreamEventDone, Message: m.message, StopReason: m.message.StopReason}
	close(ch)
	return ch, nil
}

func TestCoCreateStreamReplyProgressDoesNotExposeThinkBlock(t *testing.T) {
	const hidden = "流式隐藏思考"
	final := agentcore.Message{
		Role:       agentcore.RoleAssistant,
		Content:    []agentcore.ContentBlock{agentcore.TextBlock(`<reply>用户可见内容<think>` + hidden + `</think></reply>`)},
		StopReason: agentcore.StopReasonStop,
	}
	model := &thinkingAndFinalStreamModel{
		plainTrackedTestModel: &plainTrackedTestModel{},
		thinking:              "独立思考",
		message:               final,
		textDelta:             `<reply>用户可见内容<think>` + hidden + `</think></reply>`,
	}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}
	var progress []string

	if _, err := coCreateStream(
		t.Context(), models, nil, nil, coCreateSystemPrompt,
		[]CoCreateMessage{{Role: "user", Content: "继续写"}}, func(kind, text string) {
			if kind == CoCreateProgressReply {
				progress = append(progress, text)
			}
		}); err != nil {
		t.Fatal(err)
	}
	for _, text := range progress {
		if strings.Contains(text, hidden) {
			t.Fatalf("reply progress leaked thinking: %q", text)
		}
	}
}

func TestCoCreateStreamDropsUnclosedThinkBlock(t *testing.T) {
	const hidden = "未闭合的内部推理"
	final := agentcore.Message{
		Role:       agentcore.RoleAssistant,
		Content:    []agentcore.ContentBlock{agentcore.TextBlock(`<reply>用户可见内容<thinking>` + hidden)},
		StopReason: agentcore.StopReasonStop,
	}
	model := &finalTextOnlyStreamModel{plainTrackedTestModel: &plainTrackedTestModel{}, message: final}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}

	reply, err := coCreateStream(
		t.Context(), models, nil, nil, coCreateSystemPrompt,
		[]CoCreateMessage{{Role: "user", Content: "继续写"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Message != "用户可见内容" {
		t.Fatalf("reply = %q, want visible prefix only", reply.Message)
	}
	if strings.Contains(reply.Message, hidden) {
		t.Fatalf("unclosed thinking leaked: %q", reply.Message)
	}
}

func TestCoCreateStreamStripsThinkBlockFromFinalText(t *testing.T) {
	const hidden = "这是隐藏的思考"
	final := agentcore.Message{
		Role: agentcore.RoleAssistant,
		Content: []agentcore.ContentBlock{agentcore.TextBlock(`<reply>用户可见内容<think>` + hidden + `</think></reply>
<draft>草稿</draft>
<ready>false</ready>
<suggestions></suggestions>`)},
		StopReason: agentcore.StopReasonStop,
	}
	model := &finalTextOnlyStreamModel{plainTrackedTestModel: &plainTrackedTestModel{}, message: final}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}

	reply, err := coCreateStream(
		t.Context(), models, nil, nil, coCreateSystemPrompt,
		[]CoCreateMessage{{Role: "user", Content: "继续写"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Message != "用户可见内容" {
		t.Fatalf("final reply = %q, want thinking block removed", reply.Message)
	}
	if strings.Contains(reply.Raw, hidden) {
		t.Fatalf("raw final response should not be exposed through parsed reply: %q", reply.Raw)
	}
}
