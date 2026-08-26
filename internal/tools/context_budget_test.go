package tools

import (
	"slices"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
)

func TestTrimByBudgetRecordsKnowledgeBoundaries(t *testing.T) {
	result := map[string]any{
		"episodic_memory": map[string]any{
			"knowledge_boundaries": []domain.KnowledgeEntry{{
				ID: "k_shadow", Truth: strings.Repeat("很长的作者真相", 50), EstablishedAt: 1,
				KnownBy: []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 2}},
			}},
		},
	}

	trimByBudget(result, 80)

	episodic := result["episodic_memory"].(map[string]any)
	if _, ok := episodic["knowledge_boundaries"]; ok {
		t.Fatal("expected knowledge boundaries to be trimmed")
	}
	trimmed, ok := result["_trimmed"].([]string)
	if !ok || !slices.Contains(trimmed, "knowledge_boundaries") {
		t.Fatalf("trimmed keys missing knowledge boundaries: %#v", result["_trimmed"])
	}
}

func TestTrimByBudgetCanRemovePlatformRubricWithReferences(t *testing.T) {
	result := map[string]any{
		"reference_pack": map[string]any{
			"references": map[string]string{"platform_rubric": strings.Repeat("番茄软评价", 100)},
		},
	}
	trimByBudget(result, 80)
	pack := result["reference_pack"].(map[string]any)
	if _, ok := pack["references"]; ok {
		t.Fatalf("platform rubric must remain budget-trimmable: %#v", result)
	}
	trimmed, ok := result["_trimmed"].([]string)
	if !ok || !slices.Contains(trimmed, "references") {
		t.Fatalf("trimmed keys must record references: %#v", result["_trimmed"])
	}
}

func TestTrimByBudgetRemovesCanonicalMemoryKeys(t *testing.T) {
	result := map[string]any{
		"reference_pack": map[string]any{
			"references": map[string]string{
				"a": strings.Repeat("x", 200),
				"b": strings.Repeat("y", 200),
			},
			"style_rules": []string{"克制"},
		},
	}

	trimByBudget(result, 80)

	pack, ok := result["reference_pack"].(map[string]any)
	if !ok {
		t.Fatal("expected reference_pack to remain available")
	}
	if _, ok := pack["references"]; ok {
		t.Fatal("expected references to be trimmed from reference_pack")
	}
}

func TestTrimByBudgetKeepsStyleStats(t *testing.T) {
	styleStats := map[string]any{
		"chapters": 200,
		"patterns": []map[string]any{
			{"name": "矫正句", "total": 80, "per_chapter": 0.4},
		},
	}
	result := map[string]any{
		"reference_pack": map[string]any{
			"references": strings.Repeat("x", 500),
		},
		"episodic_memory": map[string]any{
			"style_stats": styleStats,
		},
	}

	trimByBudget(result, 100)

	episodic := result["episodic_memory"].(map[string]any)
	if _, ok := episodic["style_stats"]; !ok {
		t.Fatal("style_stats must remain in episodic_memory")
	}
	if trimmed, ok := result["_trimmed"].([]string); ok && slices.Contains(trimmed, "style_stats") {
		t.Fatal("style_stats must not be reported as trimmed")
	}
}
