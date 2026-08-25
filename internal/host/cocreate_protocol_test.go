package host

import (
	"strings"
	"testing"
)

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
