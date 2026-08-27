package eval

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestSettingTermsDoNotCreateKnowledgeFacts(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := st.World.SaveWorldRules([]domain.WorldRule{{
		Category: "technology",
		Rule:     "潮汐锁定协议",
		Boundary: "只能在双向校验完成后启动",
	}}); err != nil {
		t.Fatalf("SaveWorldRules: %v", err)
	}
	if err := st.Cast.MergeAppearances(1, []string{"顾临"}, []domain.CastIntro{{
		Name: "顾临", BriefRole: "维修员",
	}}, nil); err != nil {
		t.Fatalf("MergeAppearances: %v", err)
	}

	knowledge, err := st.World.LoadKnowledgeState()
	if err != nil {
		t.Fatalf("LoadKnowledgeState: %v", err)
	}
	if knowledge != nil {
		t.Fatalf("world rules and cast entries must not implicitly create knowledge facts: %+v", knowledge)
	}
}
