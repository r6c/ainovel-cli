package host

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/assets"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/store"
)

type simulationUsageModel struct {
	calls atomic.Int32
}

func (m *simulationUsageModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	call := m.calls.Add(1)
	content := simulationSourceReportJSON
	if call > 1 {
		content = simulationSynthesisJSON
	}
	return &agentcore.LLMResponse{Message: agentcore.Message{
		Role:    agentcore.RoleAssistant,
		Content: []agentcore.ContentBlock{agentcore.TextBlock(content)},
		Usage:   &agentcore.Usage{Input: 11, Output: 7, TotalTokens: 18, Provider: "test", Model: "test"},
	}}, nil
}

func (*simulationUsageModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	ch := make(chan agentcore.StreamEvent)
	close(ch)
	return ch, nil
}

func (*simulationUsageModel) SupportsTools() bool { return false }

const simulationSourceReportJSON = `{"title":"样本","summary":"抽象结构摘要","style_observations":["近距离视角"],"common_words":["海"],"plot_patterns":["逐步升级"],"hook_patterns":["信息缺口"],"pacing_notes":["短场景"],"reader_appeal":["悬念"],"reusable_techniques":["延迟揭示"],"warnings":["不得复制原文"]}`

const simulationSynthesisJSON = `{"style":{"narrative_voice":["近距离"],"sentence_rhythm":["有变化"],"prose_texture":["具体"],"perspective":["限知"],"mood":["克制"],"do_not_copy":["原句"]},"lexicon":{"common_words":["意象"],"emotion_words":["紧张"],"scene_words":["封闭空间"],"transition_words":["随后"],"signature_phrases":["抽象特征"]},"plot_design":{"opening_patterns":["异常"],"escalation_patterns":["升级"],"turning_point_patterns":["发现"],"payoff_patterns":["回收"]},"hook_design":{"hook_types":["信息差"],"placement":["章末"],"cliffhanger_patterns":["未决"],"payoff_rules":["兑现"]},"pacing_density":{"scene_density":["中"],"information_release":["渐进"],"dialogue_action_ratio":["均衡"],"compression_rules":["删重复"]},"reader_engagement":{"methods":["悬念"],"emotional_drivers":["担忧"],"progression_rewards":["线索"],"anti_patterns":["复制"]},"role_guidance":{"architect":["控制升级"],"writer":["借鉴结构"],"editor":["检查复制"]}}`

func TestSimulateDirRecordsSimulationUsage(t *testing.T) {
	output := t.TempDir()
	st := store.NewStore(output)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	model := &simulationUsageModel{}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}
	usage := NewUsageTracker(models, nil)
	h := &Host{
		store:  st,
		runCtx: context.Background(),
		bundle: assets.Load("default", assets.LoadOptions{}),
		engine: &engine{},
		models: models,
		usage:  usage,
	}
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "a.md"), []byte("自建样本正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	events, err := h.SimulateDir(context.Background(), sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	for event := range events {
		if event.Err != nil {
			t.Fatal(event.Err)
		}
	}
	perAgent := usage.PerAgent()
	if len(perAgent) != 1 || perAgent[0].Role != "simulation" || perAgent[0].Input != 22 || perAgent[0].Output != 14 {
		t.Fatalf("simulation usage not recorded: %+v", perAgent)
	}
}
