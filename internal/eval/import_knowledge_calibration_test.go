package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type importKnowledgeSamples struct {
	Version int `json:"version"`
	Samples []struct {
		ID          string `json:"id"`
		KnowledgeID string `json:"knowledge_id"`
		Text        string `json:"text"`
	} `json:"samples"`
}

type importKnowledgeLabels struct {
	Version int `json:"version"`
	Labels  []struct {
		ID       string `json:"id"`
		Category string `json:"category"`
		Updates  []struct {
			ID        string `json:"id"`
			Action    string `json:"action"`
			Character string `json:"character,omitempty"`
		} `json:"updates"`
	} `json:"labels"`
}

func TestImportKnowledgeCalibrationCorpusIsAnonymousAndCoversActionBoundaries(t *testing.T) {
	root := importKnowledgeRoot(t)
	var samples importKnowledgeSamples
	readImportKnowledgeJSON(t, filepath.Join(root, "samples.json"), &samples)
	var labels importKnowledgeLabels
	readImportKnowledgeJSON(t, filepath.Join(root, "labels.json"), &labels)
	if samples.Version != 1 || labels.Version != 1 || len(samples.Samples) != 12 || len(labels.Labels) != 12 {
		t.Fatalf("corpus shape samples=%d labels=%d versions=%d/%d", len(samples.Samples), len(labels.Labels), samples.Version, labels.Version)
	}
	byID := make(map[string]string, len(samples.Samples))
	for _, sample := range samples.Samples {
		if sample.ID == "" || sample.KnowledgeID == "" || strings.TrimSpace(sample.Text) == "" || byID[sample.ID] != "" {
			t.Fatalf("invalid sample: %+v", sample)
		}
		byID[sample.ID] = sample.KnowledgeID
		for _, leaked := range []string{"establish_only", "establish_learn", "guess_negative", "预期动作", "金标"} {
			if strings.Contains(sample.Text, leaked) {
				t.Fatalf("sample %s leaks label %q", sample.ID, leaked)
			}
		}
	}
	categories := map[string]int{}
	for _, label := range labels.Labels {
		knowledgeID, ok := byID[label.ID]
		if !ok || label.Category == "" {
			t.Fatalf("label without sample/category: %+v", label)
		}
		categories[label.Category]++
		seen := map[string]bool{}
		for i, update := range label.Updates {
			if update.ID != knowledgeID || seen[update.Action] {
				t.Fatalf("label %s invalid update[%d]=%+v", label.ID, i, update)
			}
			seen[update.Action] = true
			switch update.Action {
			case "establish", "reveal_to_reader":
				if update.Character != "" {
					t.Fatalf("label %s action %s must not carry character", label.ID, update.Action)
				}
			case "learn", "believe":
				if update.Character == "" {
					t.Fatalf("label %s action %s requires character", label.ID, update.Action)
				}
			default:
				t.Fatalf("label %s invalid action %q", label.ID, update.Action)
			}
		}
	}
	want := map[string]int{
		"establish_only": 2, "establish_learn": 2, "establish_reveal": 2,
		"establish_learn_reveal": 2, "establish_learn_reveal_belief": 1,
		"guess_negative": 1, "lie_negative": 1, "disbelief_negative": 1,
	}
	if len(categories) != len(want) {
		t.Fatalf("categories=%v want=%v", categories, want)
	}
	for category, count := range want {
		if categories[category] != count {
			t.Fatalf("category %s=%d want=%d", category, categories[category], count)
		}
	}
}

func importKnowledgeRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "evals", "import-knowledge"))
}

func readImportKnowledgeJSON(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatal(err)
	}
}
