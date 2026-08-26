package eval

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestImportKnowledgeRunnerPersistsEachSampleAndResumes(t *testing.T) {
	dir := t.TempDir()
	samples := []CalibrationSample{
		{ID: "ik01", Category: "establish_only"},
		{ID: "ik02", Category: "establish_learn"},
		{ID: "ik03", Category: "establish_reveal"},
	}
	calls := make([]string, 0, len(samples))
	runner := NewImportKnowledgeRunner(dir, samples, ImportKnowledgeRunnerOptions{
		PromptName:   "calibrated",
		PromptDigest: "sha256:calibrated",
	})
	first, err := runner.Run(func(sample CalibrationSample) (ImportKnowledgeSampleResult, error) {
		calls = append(calls, sample.ID)
		return ImportKnowledgeSampleResult{
			SampleID: sample.ID,
			Category: sample.Category,
			Updates:  []KnowledgeActionResult{{ID: "K-" + sample.ID, Action: "establish"}},
			Usage:    UsageMetrics{UsageRecorded: true, Input: 10, Output: 5, CostUSD: 0.01},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Completed != 3 || first.Failed != 0 || !reflect.DeepEqual(calls, []string{"ik01", "ik02", "ik03"}) {
		t.Fatalf("first run summary=%+v calls=%v", first, calls)
	}

	calls = nil
	second, err := NewImportKnowledgeRunner(dir, samples, ImportKnowledgeRunnerOptions{
		PromptName:   "calibrated",
		PromptDigest: "sha256:calibrated",
	}).Run(func(sample CalibrationSample) (ImportKnowledgeSampleResult, error) {
		calls = append(calls, sample.ID)
		return ImportKnowledgeSampleResult{SampleID: sample.ID, Category: sample.Category}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Completed != 3 || second.Failed != 0 || len(calls) != 0 {
		t.Fatalf("resume summary=%+v calls=%v", second, calls)
	}
	if _, err := os.Stat(filepath.Join(dir, "results", "ik02.json")); err != nil {
		t.Fatalf("per-sample result missing: %v", err)
	}
}

func TestImportKnowledgeRunnerFailsLoudlyForMissingResult(t *testing.T) {
	dir := t.TempDir()
	samples := []CalibrationSample{{ID: "ik01", Category: "establish_only"}}
	runner := NewImportKnowledgeRunner(dir, samples, ImportKnowledgeRunnerOptions{
		PromptName:   "baseline",
		PromptDigest: "sha256:baseline",
	})
	_, err := runner.Run(func(sample CalibrationSample) (ImportKnowledgeSampleResult, error) {
		return ImportKnowledgeSampleResult{Category: sample.Category}, nil
	})
	if err == nil || !errors.Is(err, ErrImportKnowledgeResultInvalid) {
		t.Fatalf("missing sample id must fail loudly, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "results", "ik01.json")); !os.IsNotExist(statErr) {
		t.Fatalf("invalid result must not be persisted, stat=%v", statErr)
	}
}

func TestImportKnowledgeRunnerDoesNotReuseResultFromAnotherPrompt(t *testing.T) {
	dir := t.TempDir()
	samples := []CalibrationSample{{ID: "ik01", Category: "establish_only"}}
	calls := 0
	first := NewImportKnowledgeRunner(dir, samples, ImportKnowledgeRunnerOptions{
		PromptName: "baseline", PromptDigest: "sha256:baseline",
	})
	if _, err := first.Run(func(sample CalibrationSample) (ImportKnowledgeSampleResult, error) {
		calls++
		return ImportKnowledgeSampleResult{SampleID: sample.ID, Category: sample.Category}, nil
	}); err != nil {
		t.Fatal(err)
	}

	second := NewImportKnowledgeRunner(dir, samples, ImportKnowledgeRunnerOptions{
		PromptName: "calibrated", PromptDigest: "sha256:calibrated",
	})
	if _, err := second.Run(func(sample CalibrationSample) (ImportKnowledgeSampleResult, error) {
		calls++
		return ImportKnowledgeSampleResult{SampleID: sample.ID, Category: sample.Category}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("prompt identity change must rerun sample, calls=%d", calls)
	}
}

func TestImportKnowledgeRunnerFailsLoudlyForCorruptExistingResult(t *testing.T) {
	dir := t.TempDir()
	samples := []CalibrationSample{{ID: "ik01", Category: "establish_only"}}
	resultDir := filepath.Join(dir, "results")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resultDir, "ik01.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	_, err := NewImportKnowledgeRunner(dir, samples, ImportKnowledgeRunnerOptions{
		PromptName: "baseline", PromptDigest: "sha256:baseline",
	}).Run(func(sample CalibrationSample) (ImportKnowledgeSampleResult, error) {
		calls++
		return ImportKnowledgeSampleResult{SampleID: sample.ID, Category: sample.Category}, nil
	})
	if err == nil || calls != 0 || !strings.Contains(err.Error(), "decode result") {
		t.Fatalf("corrupt existing result must fail without rerun, calls=%d err=%v", calls, err)
	}
}

func TestImportKnowledgeRunnerRecordsErrorsWithoutWritingRawResponse(t *testing.T) {
	dir := t.TempDir()
	samples := []CalibrationSample{{ID: "ik01", Category: "establish_only"}, {ID: "ik02", Category: "establish_only"}}
	runner := NewImportKnowledgeRunner(dir, samples, ImportKnowledgeRunnerOptions{
		PromptName:   "baseline",
		PromptDigest: "sha256:baseline",
	})
	first, err := runner.Run(func(sample CalibrationSample) (ImportKnowledgeSampleResult, error) {
		if sample.ID == "ik01" {
			return ImportKnowledgeSampleResult{}, errors.New("provider unavailable")
		}
		return ImportKnowledgeSampleResult{SampleID: sample.ID, Category: sample.Category}, nil
	})
	if err == nil || first.Completed != 1 || first.Failed != 1 {
		t.Fatalf("expected one recorded failure, summary=%+v err=%v", first, err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "errors", "ik01.json")); statErr != nil {
		t.Fatalf("error record missing: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "responses", "ik01.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("raw response must not be persisted, stat=%v", statErr)
	}

	second, err := NewImportKnowledgeRunner(dir, samples, ImportKnowledgeRunnerOptions{
		PromptName:   "baseline",
		PromptDigest: "sha256:baseline",
	}).Run(func(sample CalibrationSample) (ImportKnowledgeSampleResult, error) {
		return ImportKnowledgeSampleResult{SampleID: sample.ID, Category: sample.Category}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Completed != 2 || second.Failed != 0 {
		t.Fatalf("retry summary=%+v", second)
	}
}
