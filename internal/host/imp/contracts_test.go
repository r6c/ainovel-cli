package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
	"github.com/voocel/ainovel-cli/internal/llmcontract"
)

func TestStructuredContractsAreStrictReady(t *testing.T) {
	for _, contract := range []llmcontract.Contract{segmentContract, analysisContract, rangeContract, synthesisContract} {
		if err := llmcontract.ValidateStrictReady(contract.Schema); err != nil {
			t.Fatalf("%s: %v", contract.Name, err)
		}
	}
}

func TestAnalysisContractAcceptsKnowledgeActions(t *testing.T) {
	rootProps, ok := analysisContract.Schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("analysis properties missing: %#v", analysisContract.Schema["properties"])
	}
	chapters := rootProps["chapters"].(map[string]any)
	chapterItems := chapters["items"].(map[string]any)
	chapterProps := chapterItems["properties"].(map[string]any)
	knowledge, ok := chapterProps["knowledge_updates"].(map[string]any)
	if !ok {
		t.Fatalf("knowledge_updates schema missing: %#v", chapterProps["knowledge_updates"])
	}
	updateItems := knowledge["items"].(map[string]any)
	updateProps := updateItems["properties"].(map[string]any)
	action := updateProps["action"].(map[string]any)
	if fmt.Sprint(action["enum"]) != "[establish learn]" {
		t.Fatalf("knowledge action enum = %#v", action["enum"])
	}

	updates := []map[string]any{
		{"id": "k_shadow", "action": "establish", "truth": "黑影是林墨的兄长", "character": nil},
		{"id": "k_shadow", "action": "learn", "truth": nil, "character": "林墨"},
	}
	for _, update := range updates {
		t.Run(update["action"].(string), func(t *testing.T) {
			var facts map[string]any
			if err := json.Unmarshal([]byte(factsJSON(1, "第一章")), &facts); err != nil {
				t.Fatal(err)
			}
			facts["knowledge_updates"] = []any{update}
			body, err := json.Marshal(map[string]any{"chapters": []any{facts}})
			if err != nil {
				t.Fatal(err)
			}
			if err := llmcontract.ValidateJSON(analysisContract.Schema, body); err != nil {
				t.Fatalf("analysis contract rejected knowledge update: %v", err)
			}
		})
	}
}

func TestAnalysisContractAcceptsForeshadowLifecycleActions(t *testing.T) {
	for _, action := range []string{"reinforce", "partial_payoff"} {
		t.Run(action, func(t *testing.T) {
			var facts map[string]any
			if err := json.Unmarshal([]byte(factsJSON(1, "第一章")), &facts); err != nil {
				t.Fatal(err)
			}
			facts["foreshadow_updates"] = []any{map[string]any{
				"id": "f1", "action": action, "description": nil,
			}}
			body, err := json.Marshal(map[string]any{"chapters": []any{facts}})
			if err != nil {
				t.Fatal(err)
			}
			if err := llmcontract.ValidateJSON(analysisContract.Schema, body); err != nil {
				t.Fatalf("analysis contract rejected %s: %v", action, err)
			}
		})
	}
}

type nativeImportModel struct {
	*mockModel
	messages []agentcore.Message
	config   agentcore.CallConfig
}

func (m *nativeImportModel) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Provider:   "openai",
		Model:      "gpt-test",
		Structured: llm.StructuredCapabilities{JSONSchema: llm.SupportYes, Strict: llm.SupportYes},
	}
}

func (m *nativeImportModel) Generate(ctx context.Context, messages []agentcore.Message, tools []agentcore.ToolSpec, opts ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	m.messages = messages
	m.config = agentcore.ResolveCallConfig(opts)
	return m.mockModel.Generate(ctx, messages, tools, opts...)
}

func TestCallStructuredUsesNativeSchemaWithoutPromptDuplication(t *testing.T) {
	model := &nativeImportModel{mockModel: &mockModel{responses: []string{`{"boundaries":[]}`}}}
	const prompt = "判断真实边界。"
	_, err := callStructured[boundaryBatch](t.Context(), model, segmentContract, prompt, `{}`, 100, callProfile{}, nil)
	if err != nil {
		t.Fatalf("callStructured: %v", err)
	}
	format := model.config.ResponseFormat
	if format == nil || format.JSONSchema == nil || format.JSONSchema.Name != segmentContract.Name {
		t.Fatalf("response format = %#v", format)
	}
	if got := model.messages[0].TextContent(); got != prompt {
		t.Fatalf("native prompt 被重复注入 schema: %s", got)
	}
}

func TestCallStructuredPromptModeInjectsContract(t *testing.T) {
	model := &nativeImportModel{mockModel: &mockModel{responses: []string{`{"boundaries":[]}`}}}
	modelCaps := &promptImportModel{nativeImportModel: model}
	_, err := callStructured[boundaryBatch](t.Context(), modelCaps, segmentContract, "判断真实边界。", `{}`, 100, callProfile{}, nil)
	if err != nil {
		t.Fatalf("callStructured: %v", err)
	}
	if model.config.ResponseFormat != nil {
		t.Fatalf("prompt mode 不应发送 response_format: %#v", model.config.ResponseFormat)
	}
	if !strings.Contains(model.messages[0].TextContent(), "<output-json-schema>") {
		t.Fatalf("prompt mode 未注入契约: %s", model.messages[0].TextContent())
	}
}

type promptImportModel struct{ *nativeImportModel }

func (m *promptImportModel) Capabilities() llm.Capabilities { return llm.Capabilities{} }
