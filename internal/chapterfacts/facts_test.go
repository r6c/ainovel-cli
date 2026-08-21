package chapterfacts

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
)

func TestValidateKnowledgeEstablishRequiresTruth(t *testing.T) {
	facts := domain.ChapterFacts{
		Title: "第一章", Summary: "建立作者真相", KeyEvents: []string{"发现黑影"},
		KnowledgeUpdates: []domain.KnowledgeUpdate{{ID: "k_shadow", Action: "establish"}},
	}
	if err := Validate(facts); err == nil {
		t.Fatal("expected establish without truth to fail")
	}
}
