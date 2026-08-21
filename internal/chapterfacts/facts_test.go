package chapterfacts

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
)

func TestValidateReaderRevealRejectsTruthAndCharacter(t *testing.T) {
	tests := []domain.KnowledgeUpdate{
		{ID: "k_shadow", Action: "reveal_to_reader", Truth: "不应携带的真相"},
		{ID: "k_shadow", Action: "reveal_to_reader", Character: "林墨"},
	}
	for _, update := range tests {
		facts := domain.ChapterFacts{
			Title: "第一章", Summary: "向读者揭示", KeyEvents: []string{"揭示身份"},
			KnowledgeUpdates: []domain.KnowledgeUpdate{update},
		}
		if err := Validate(facts); err == nil {
			t.Fatalf("expected reader reveal with extra fields to fail: %+v", update)
		}
	}
}

func TestValidateKnowledgeEstablishRequiresTruth(t *testing.T) {
	facts := domain.ChapterFacts{
		Title: "第一章", Summary: "建立作者真相", KeyEvents: []string{"发现黑影"},
		KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish"}},
	}
	if err := Validate(facts); err == nil {
		t.Fatal("expected establish without truth to fail")
	}
}
