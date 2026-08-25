package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voocel/ainovel-cli/internal/host"
)

func TestCoCreatePromptPanelShowsColdStartInterviewProgress(t *testing.T) {
	state := newCoCreateState("写一本悬疑小说")
	state.awaiting = false
	got := renderCoCreatePromptPanel(50, 12, state)
	if !strings.Contains(got, "核心定位") || !strings.Contains(got, "1/5") {
		t.Fatalf("core progress missing: %q", got)
	}

	state.apply(host.CoCreateReply{Prompt: "累计草稿", Stage: "customization"})
	got = renderCoCreatePromptPanel(50, 12, state)
	if !strings.Contains(got, "深度定制") || !strings.Contains(got, "2/5") {
		t.Fatalf("customization progress missing: %q", got)
	}
}

func TestColdStartCoCreateCtrlSDoesNotStartBeforeReady(t *testing.T) {
	m := NewModel(nil, "")
	m.cocreate = newCoCreateState("写一本悬疑小说")
	m.cocreate.awaiting = false
	m.cocreate.apply(host.CoCreateReply{Prompt: "已有草稿", Stage: "customization", Ready: true})

	next, cmd := m.handleCoCreateKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	got := next.(Model)
	if cmd != nil || got.cocreate == nil || got.starting {
		t.Fatalf("Ctrl+S bypassed interview gate: cmd=%v cocreate=%v starting=%v", cmd, got.cocreate, got.starting)
	}
}

func TestCoCreateRequestFailureKeepsInterviewState(t *testing.T) {
	m := NewModel(nil, "")
	m.cocreate = newCoCreateState("写一本悬疑小说")
	m.cocreate.reqID = 7
	m.cocreate.awaiting = true
	m.cocreate.apply(host.CoCreateReply{Message: "核心已清楚", Prompt: "累计草稿", Stage: "customization"})
	m.cocreate.awaiting = true

	next, _ := m.handleCoCreateDoneMsg(cocreateDoneMsg{reqID: 7, err: errors.New("网络失败")})
	got := next.(Model)
	if got.cocreate == nil || got.cocreate.session.Stage() != "customization" || got.cocreate.draftPrompt() != "累计草稿" {
		t.Fatalf("request failure lost interview state: %+v", got.cocreate)
	}
	if got.cocreate.awaiting {
		t.Fatal("request failure must leave session editable")
	}
}

func TestExitColdStartCoCreateRestoresInitialInput(t *testing.T) {
	m := NewModel(nil, "")
	m.cocreate = newCoCreateState("写一本悬疑小说")
	next, _ := m.exitCoCreate()
	got := next.(Model)
	if got.cocreate != nil || got.textarea.Value() != "写一本悬疑小说" {
		t.Fatalf("exit did not restore initial input: cocreate=%v input=%q", got.cocreate, got.textarea.Value())
	}
}

func TestStageCoCreatePromptPanelDoesNotShowColdStartInterviewProgress(t *testing.T) {
	state := newStageCoCreateState()
	state.awaiting = false
	state.apply(host.CoCreateReply{Prompt: "后续方向"})
	got := renderCoCreatePromptPanel(50, 12, state)
	for _, forbidden := range []string{"核心定位", "1/5", "深度定制"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("stage cocreate leaked cold-start progress %q: %s", forbidden, got)
		}
	}
	if !state.canStart() {
		t.Fatal("stage cocreate legacy draft gate must remain enabled")
	}
}
