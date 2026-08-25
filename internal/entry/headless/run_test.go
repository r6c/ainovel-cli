package headless

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/assets"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestRunCompletedBookWithoutPromptReturnsSummaryWithoutChangingFacts(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.SaveProjectFormatVersion(store.CurrentProjectFormatVersion); err != nil {
		t.Fatal(err)
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "已完结作品", Synopsis: "简介"}); err != nil {
		t.Fatal(err)
	}
	progress := &domain.Progress{
		Phase:             domain.PhaseComplete,
		CurrentChapter:    2,
		TotalChapters:     1,
		CompletedChapters: []int{1},
		TotalWordCount:    1234,
		ChapterWordCounts: map[int]int{1: 1234},
	}
	if err := st.Progress.Save(progress); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, "# 第一章\n\n正文。\n"); err != nil {
		t.Fatal(err)
	}
	if err := st.Usage.Save(domain.UsageState{
		UpdatedAt: time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC),
		Overall:   domain.AgentUsageTotals{Input: 100, Output: 20, Cost: 0.12},
		PerAgent:  map[string]domain.AgentUsageTotals{"writer": {Input: 100, Output: 20, Cost: 0.12}},
	}); err != nil {
		t.Fatal(err)
	}

	progressBefore, err := os.ReadFile(filepath.Join(dir, "meta", "progress.json"))
	if err != nil {
		t.Fatal(err)
	}
	usageBefore, err := os.ReadFile(filepath.Join(dir, "meta", "usage.json"))
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	cfg := offlineConfig(dir, &disabled)
	var stdout, stderr bytes.Buffer

	if err := Run(cfg, assets.Bundle{}, Options{Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("completed headless workspace should be a successful terminal state: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("completed workspace should not emit model output: %q", stdout.String())
	}
	for _, want := range []string{"headless 完成", "已完结作品", "1 章", "1234 字"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("completion summary missing %q: %q", want, stderr.String())
		}
	}
	progressAfter, err := os.ReadFile(filepath.Join(dir, "meta", "progress.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(progressBefore, progressAfter) {
		t.Fatal("completed headless check must not rewrite progress")
	}
	usageAfter, err := os.ReadFile(filepath.Join(dir, "meta", "usage.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(usageBefore, usageAfter) {
		t.Fatal("completed headless check must not rewrite usage")
	}
	if pending, err := st.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("completed headless check must not create pending commit: pending=%v err=%v", pending, err)
	}
}

func TestRunWithoutPromptOrRecoverableSessionFailsCleanlyAndExportsDiagnostics(t *testing.T) {
	dir := t.TempDir()
	disabled := false
	cfg := offlineConfig(dir, &disabled)
	var stdout, stderr bytes.Buffer

	err := Run(cfg, assets.Bundle{}, Options{Stdout: &stdout, Stderr: &stderr})
	if err == nil || !strings.Contains(err.Error(), "需要 --prompt") || !strings.Contains(err.Error(), dir) {
		t.Fatalf("empty headless workspace should return actionable error, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed resume should not emit model output: %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "headless 恢复") {
		t.Fatalf("failed resume must not claim success: %q", stderr.String())
	}
	if _, statErr := os.Stat(filepath.Join(dir, "meta", "diag-export.md")); statErr != nil {
		t.Fatalf("headless exit should export diagnostics: %v", statErr)
	}
}

func offlineConfig(dir string, notifyEnabled *bool) bootstrap.Config {
	return bootstrap.Config{
		OutputDir: dir,
		Provider:  "ollama",
		ModelName: "offline-test",
		Providers: map[string]bootstrap.ProviderConfig{
			"ollama": {Models: []bootstrap.ModelConfig{{Name: "offline-test"}}},
		},
		Notify: bootstrap.NotifyConfig{Enabled: notifyEnabled},
	}
}
