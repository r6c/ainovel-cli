package host

import (
	"context"
	"testing"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

type plainTrackedTestModel struct{}

func (*plainTrackedTestModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return &agentcore.LLMResponse{}, nil
}

func (*plainTrackedTestModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	ch := make(chan agentcore.StreamEvent)
	close(ch)
	return ch, nil
}

func (*plainTrackedTestModel) SupportsTools() bool { return true }

type capableTrackedTestModel struct {
	*plainTrackedTestModel
	caps llm.Capabilities
}

func (m *capableTrackedTestModel) Capabilities() llm.Capabilities { return m.caps }

type streamUsageTestModel struct {
	*plainTrackedTestModel
	message agentcore.Message
}

func (m *streamUsageTestModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	ch := make(chan agentcore.StreamEvent, 2)
	ch <- agentcore.StreamEvent{Type: agentcore.StreamEventTextDelta, Delta: "回复"}
	ch <- agentcore.StreamEvent{Type: agentcore.StreamEventDone, Message: m.message, StopReason: m.message.StopReason}
	close(ch)
	return ch, nil
}

func TestUsageTrackedModelRecordsStreamingDoneUsageOnce(t *testing.T) {
	final := agentcore.Message{
		Role:       agentcore.RoleAssistant,
		Content:    []agentcore.ContentBlock{agentcore.TextBlock("回复")},
		StopReason: agentcore.StopReasonStop,
		Usage:      &agentcore.Usage{Input: 120, Output: 30, TotalTokens: 150},
	}
	var recorded []agentcore.AgentMessage
	wrapped := newUsageTrackedModel(&streamUsageTestModel{plainTrackedTestModel: &plainTrackedTestModel{}, message: final}, "thinking", func(agentName, _ string, msg agentcore.AgentMessage) {
		if agentName != "thinking" {
			t.Fatalf("agentName=%q", agentName)
		}
		recorded = append(recorded, msg)
	})
	stream, err := wrapped.GenerateStream(t.Context(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var events []agentcore.StreamEvent
	for event := range stream {
		events = append(events, event)
	}
	if len(events) != 2 || events[0].Type != agentcore.StreamEventTextDelta || events[1].Type != agentcore.StreamEventDone {
		t.Fatalf("stream changed by usage wrapper: %+v", events)
	}
	if len(recorded) != 1 {
		t.Fatalf("recorded=%d want=1", len(recorded))
	}
	message, ok := recorded[0].(agentcore.Message)
	if !ok || message.Usage == nil || message.Usage.TotalTokens != 150 {
		t.Fatalf("recorded message=%+v", recorded[0])
	}
}

func TestUsageTrackedModelPreservesOptionalCapabilities(t *testing.T) {
	want := llm.Capabilities{
		Provider: "openai",
		Model:    "gpt-chat",
		Thinking: llm.ThinkingCapabilities{Supported: llm.SupportNo},
	}
	inner := &capableTrackedTestModel{plainTrackedTestModel: &plainTrackedTestModel{}, caps: want}
	wrapped := newUsageTrackedModel(inner, "arbiter", func(string, string, agentcore.AgentMessage) {})
	cp, ok := wrapped.(llm.CapabilityProvider)
	if !ok {
		t.Fatal("usage wrapper dropped CapabilityProvider")
	}
	if got := cp.Capabilities(); got.Provider != want.Provider || got.Model != want.Model || got.Thinking.Supported != llm.SupportNo {
		t.Fatalf("capabilities changed through wrapper: %+v", got)
	}

	plain := newUsageTrackedModel(&plainTrackedTestModel{}, "arbiter", func(string, string, agentcore.AgentMessage) {})
	if _, ok := plain.(llm.CapabilityProvider); ok {
		t.Fatal("wrapper must not invent capabilities for an unknown model")
	}
}

type overrideCapableTestModel struct {
	*capableTrackedTestModel
	override *bool
}

func (m *overrideCapableTestModel) JSONSchemaOverride() *bool { return m.override }

// usage 包装器必须透传 config json_schema 覆盖值；inner 未携带时返回 nil
// （"未配置"），不伪造能力。
func TestUsageTrackedModelForwardsJSONSchemaOverride(t *testing.T) {
	tr := true
	inner := &overrideCapableTestModel{
		capableTrackedTestModel: &capableTrackedTestModel{plainTrackedTestModel: &plainTrackedTestModel{}},
		override:                &tr,
	}
	wrapped := newUsageTrackedModel(inner, "arbiter", func(string, string, agentcore.AgentMessage) {})
	o, ok := wrapped.(interface{ JSONSchemaOverride() *bool })
	if !ok {
		t.Fatal("usage wrapper dropped JSONSchemaOverride")
	}
	if v := o.JSONSchemaOverride(); v == nil || !*v {
		t.Fatalf("override 未透传: %v", v)
	}

	capsOnly := newUsageTrackedModel(&capableTrackedTestModel{plainTrackedTestModel: &plainTrackedTestModel{}}, "arbiter", func(string, string, agentcore.AgentMessage) {})
	o, ok = capsOnly.(interface{ JSONSchemaOverride() *bool })
	if !ok {
		t.Fatal("capability wrapper should expose JSONSchemaOverride")
	}
	if v := o.JSONSchemaOverride(); v != nil {
		t.Fatalf("inner 无覆盖时应为 nil: %v", v)
	}
}
