package host

import (
	"strings"
	"testing"

	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
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
