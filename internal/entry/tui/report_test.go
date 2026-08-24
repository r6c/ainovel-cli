package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/diag"
)

func TestRenderReportTextShowsKnowledgeAggregatesWithoutContent(t *testing.T) {
	report := diag.Report{Stats: diag.Stats{
		CompletedChapters:      10,
		TotalChapters:          20,
		Phase:                  "writing",
		KnowledgeFacts:         4,
		KnowledgeKnownBy:       6,
		KnowledgeReaderKnown:   2,
		KnowledgeActiveBeliefs: 3,
	}}
	text := renderReportText(report, 100, "", nil, time.Time{}, time.Time{})
	for _, want := range []string{"知识", "真相4", "角色知情6", "读者已知2", "活跃误信3"} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q:\n%s", want, text)
		}
	}
	for _, secret := range []string{"机密作者真相", "角色错误信念正文"} {
		if strings.Contains(text, secret) {
			t.Fatalf("report leaked knowledge content %q:\n%s", secret, text)
		}
	}
}
