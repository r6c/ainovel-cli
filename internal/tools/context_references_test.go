package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestContextToolInjectsFanqieRubricOnlyWhenExplicitlySelected(t *testing.T) {
	for _, tc := range []struct {
		name     string
		platform string
		want     bool
	}{
		{name: "fanqie", platform: "fanqie", want: true},
		{name: "unspecified", platform: "", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := store.NewStore(t.TempDir())
			if err := st.Init(); err != nil {
				t.Fatal(err)
			}
			if err := st.Progress.Init(1); err != nil {
				t.Fatal(err)
			}
			snap := rules.BuildSnapshot([]rules.Candidate{{Source: "test", Structured: rules.Structured{Platform: tc.platform}}})
			if err := st.UserRules.Save(&snap); err != nil {
				t.Fatal(err)
			}
			raw, err := newTestContextTool(st, References{FanqieRubric: "番茄平台软评价"}, "default").Execute(t.Context(), json.RawMessage(`{"chapter":1}`))
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatal(err)
			}
			pack := payload["reference_pack"].(map[string]any)
			references := pack["references"].(map[string]any)
			_, got := references["platform_rubric"]
			if got != tc.want {
				t.Fatalf("platform_rubric present=%v want=%v: %#v", got, tc.want, references)
			}
		})
	}
}

func TestContextToolUnspecifiedPlatformDoesNotLeakLoadedFanqieRubric(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	const sentinel = "SENTINEL_FANQIE_RUBRIC_MUST_NOT_APPEAR"
	raw, err := newTestContextTool(st, References{FanqieRubric: sentinel}, "default").Execute(t.Context(), json.RawMessage(`{"chapter":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), sentinel) || strings.Contains(string(raw), "platform_rubric") {
		t.Fatalf("unspecified platform leaked loaded fanqie rubric: %s", raw)
	}
}

func TestContextToolArchitectModeInjectsExplicitFanqieRubric(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	snap := rules.BuildSnapshot([]rules.Candidate{{Source: "test", Structured: rules.Structured{Platform: "fanqie"}}})
	if err := st.UserRules.Save(&snap); err != nil {
		t.Fatal(err)
	}
	raw, err := newTestContextTool(st, References{FanqieRubric: "番茄平台软评价"}, "default").Execute(t.Context(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	references := payload["reference_pack"].(map[string]any)["references"].(map[string]any)
	if references["platform_rubric"] != "番茄平台软评价" {
		t.Fatalf("architect platform rubric missing: %#v", references)
	}
}
