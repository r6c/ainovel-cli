package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type calibrationSamples struct {
	Version     int    `json:"version"`
	Instruction string `json:"instruction"`
	Samples     []struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"samples"`
}

type calibrationLabels struct {
	Version int `json:"version"`
	Labels  []struct {
		ID           string `json:"id"`
		Feature      string `json:"feature"`
		ShouldModify bool   `json:"should_modify"`
		Reason       string `json:"reason"`
	} `json:"labels"`
}

func TestAntiAIToneCalibrationCorpusIsAnonymousAndBalanced(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	var samples calibrationSamples
	readCalibrationJSON(t, filepath.Join(root, "evals", "anti-ai-tone", "samples.json"), &samples)
	var labels calibrationLabels
	readCalibrationJSON(t, filepath.Join(root, "evals", "anti-ai-tone", "labels.json"), &labels)

	if samples.Version != 1 || labels.Version != 1 || len(samples.Samples) != 16 || len(labels.Labels) != 16 {
		t.Fatalf("calibration corpus shape: samples=%d labels=%d versions=%d/%d", len(samples.Samples), len(labels.Labels), samples.Version, labels.Version)
	}
	byID := make(map[string]bool, len(labels.Labels))
	positive, negative := 0, 0
	for _, label := range labels.Labels {
		if label.ID == "" || label.Feature == "" || label.Reason == "" || byID[label.ID] {
			t.Fatalf("invalid or duplicate label: %+v", label)
		}
		byID[label.ID] = true
		if label.ShouldModify {
			positive++
		} else {
			negative++
		}
	}
	if positive != 5 || negative != 11 {
		t.Fatalf("unexpected gold balance: modify=%d preserve=%d", positive, negative)
	}
	for _, sample := range samples.Samples {
		if !byID[sample.ID] || strings.TrimSpace(sample.Text) == "" {
			t.Fatalf("sample without gold label or text: %+v", sample)
		}
		for _, leaked := range []string{"should_modify", "feature", "外部研究", "lieflat", "预期"} {
			if strings.Contains(sample.Text, leaked) {
				t.Fatalf("anonymous sample %s leaks label/source %q", sample.ID, leaked)
			}
		}
	}
}

func TestAntiAIToneCalibrationEvidenceIsComplete(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	var labels calibrationLabels
	readCalibrationJSON(t, filepath.Join(root, "evals", "anti-ai-tone", "labels.json"), &labels)
	gold := make(map[string]bool, len(labels.Labels))
	for _, label := range labels.Labels {
		gold[label.ID] = label.ShouldModify
	}

	var judged struct {
		Runs []struct {
			Round   int      `json:"round"`
			Order   []string `json:"order"`
			Results []struct {
				ID       string `json:"id"`
				Decision string `json:"decision"`
			} `json:"results"`
		} `json:"runs"`
	}
	readCalibrationJSON(t, filepath.Join(root, "evals", "anti-ai-tone", "judge-runs.json"), &judged)
	if len(judged.Runs) != 3 {
		t.Fatalf("judge runs=%d want=3", len(judged.Runs))
	}
	votes := make(map[string]int, len(gold))
	orders := make(map[string]bool, len(judged.Runs))
	for _, run := range judged.Runs {
		if len(run.Order) != len(gold) || len(run.Results) != len(gold) {
			t.Fatalf("round %d incomplete: order=%d results=%d", run.Round, len(run.Order), len(run.Results))
		}
		orders[strings.Join(run.Order, ",")] = true
		seen := make(map[string]bool, len(gold))
		for _, result := range run.Results {
			if _, ok := gold[result.ID]; !ok || seen[result.ID] {
				t.Fatalf("round %d unknown/duplicate id %q", run.Round, result.ID)
			}
			seen[result.ID] = true
			if result.Decision == "modify" {
				votes[result.ID]++
			} else if result.Decision != "preserve" {
				t.Fatalf("round %d invalid decision %q", run.Round, result.Decision)
			}
		}
	}
	if len(orders) != 3 {
		t.Fatalf("judge order variants=%d want=3", len(orders))
	}
	for id, want := range gold {
		if got := votes[id] >= 2; got != want {
			t.Fatalf("majority mismatch for %s: modify_votes=%d want_modify=%v", id, votes[id], want)
		}
	}

	var ab struct {
		Runs []struct {
			Arm     string `json:"arm"`
			Outcome string `json:"outcome"`
		} `json:"runs"`
		Score map[string]int `json:"blind_review_score"`
		Blind []struct {
			AFunction bool `json:"a_function_preserved"`
			BFunction bool `json:"b_function_preserved"`
		} `json:"blind_reviews"`
	}
	readCalibrationJSON(t, filepath.Join(root, "evals", "anti-ai-tone", "writer-ab.json"), &ab)
	if len(ab.Runs) != 6 || len(ab.Blind) != 9 {
		t.Fatalf("writer A/B shape: runs=%d blind=%d", len(ab.Runs), len(ab.Blind))
	}
	arms := map[string]int{}
	for _, run := range ab.Runs {
		if run.Outcome != "PASS" {
			t.Fatalf("writer A/B %s outcome=%s", run.Arm, run.Outcome)
		}
		arms[run.Arm]++
	}
	if arms["baseline"] != 3 || arms["calibrated"] != 3 {
		t.Fatalf("writer A/B arms=%v", arms)
	}
	for i, review := range ab.Blind {
		if !review.AFunction || !review.BFunction {
			t.Fatalf("blind review %d reports function loss", i)
		}
	}
	if ab.Score["calibrated"] != 8 || ab.Score["baseline"] != 1 || ab.Score["tie"] != 0 {
		t.Fatalf("unexpected blind score: %v", ab.Score)
	}
}

func readCalibrationJSON(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatal(err)
	}
}
