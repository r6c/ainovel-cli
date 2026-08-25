package headless

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	content := "# 第一章\n\n正文。\n"
	if err := st.Drafts.SaveFinalChapter(1, content); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, content, domain.ChapterFacts{
		Title: "第一章", Summary: "正文完成", KeyEvents: []string{"完成"},
	}, domain.StyleDelta{}); err != nil {
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

func TestRunCompletedBookFinishesProgressMarkedPendingBeforeReturningSummary(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveProjectFormatVersion(store.CurrentProjectFormatVersion); err != nil {
		t.Fatal(err)
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "待收尾作品", Synopsis: "一章完结的测试作品"}); err != nil {
		t.Fatal(err)
	}
	content := "# 第一章\n\n已经完成但提交尚未收尾的正文。\n"
	facts := domain.ChapterFacts{Title: "第一章", Summary: "完成危机", KeyEvents: []string{"危机解除"}}
	if err := st.Drafts.SaveFinalChapter(1, content); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 1, Title: facts.Title, Summary: facts.Summary, KeyEvents: facts.KeyEvents}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, content, facts, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{
		Phase: domain.PhaseComplete, CurrentChapter: 2, TotalChapters: 1,
		CompletedChapters: []int{1}, ChapterWordCounts: map[int]int{1: domain.WordCount(content)}, TotalWordCount: domain.WordCount(content),
	}); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"完成危机","characters":[],"key_events":["危机解除"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	pending := sealedPendingForHeadlessTest(t, domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageProgressMarked, Payload: payload, DraftContent: content,
		Result: &domain.CommitResult{Chapter: 1, Committed: true, BookComplete: true},
		Output: json.RawMessage(`{"chapter":1,"committed":true,"book_complete":true}`),
	})
	if err := st.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	disabled := false
	var stdout, stderr bytes.Buffer
	if err := Run(offlineConfig(dir, &disabled), assets.Bundle{}, Options{Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("completed pending recovery: %v", err)
	}
	if got, err := st.Signals.LoadPendingCommit(); err != nil || got != nil {
		t.Fatalf("completed pending must be cleared before terminal summary: pending=%+v err=%v", got, err)
	}
	reloaded := store.NewStore(dir)
	if checkpoint := reloaded.Checkpoints.LatestByStep(domain.ChapterScope(1), "commit"); checkpoint == nil {
		t.Fatal("completed pending recovery must append missing commit checkpoint")
	}
	if !strings.Contains(stderr.String(), "headless 完成") {
		t.Fatalf("recovery must end with completion summary: %q", stderr.String())
	}
}

func TestRunCompletedBookRecoversSignalSavedPending(t *testing.T) {
	dir, st, pending := completedPendingHeadlessFixture(t, domain.CommitStageSignalSaved)
	if err := st.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}
	disabled := false
	var stderr bytes.Buffer
	if err := Run(offlineConfig(dir, &disabled), assets.Bundle{}, Options{Stderr: &stderr}); err != nil {
		t.Fatalf("signal_saved recovery: %v", err)
	}
	if got, err := st.Signals.LoadPendingCommit(); err != nil || got != nil {
		t.Fatalf("signal_saved pending must be cleared: pending=%+v err=%v", got, err)
	}
	if !strings.Contains(stderr.String(), "headless 完成") {
		t.Fatalf("signal_saved recovery must end completed: %q", stderr.String())
	}
}

func TestRunCompletedBookRejectsTamperedPendingBeforeTerminalSummary(t *testing.T) {
	dir, st, pending := completedPendingHeadlessFixture(t, domain.CommitStageProgressMarked)
	pending.DraftContent += "已被修改"
	if err := st.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}
	disabled := false
	var stderr bytes.Buffer
	err := Run(offlineConfig(dir, &disabled), assets.Bundle{}, Options{Stderr: &stderr})
	if err == nil || !strings.Contains(err.Error(), "完整性校验失败") {
		t.Fatalf("tampered completed pending must fail integrity validation, got %v", err)
	}
	if strings.Contains(stderr.String(), "headless 完成") {
		t.Fatalf("tampered pending must not claim terminal completion: %q", stderr.String())
	}
	if got, loadErr := st.Signals.LoadPendingCommit(); loadErr != nil || got == nil {
		t.Fatalf("tampered pending must remain for inspection: pending=%+v err=%v", got, loadErr)
	}
}

func completedPendingHeadlessFixture(t *testing.T, stage domain.CommitStage) (string, *store.Store, domain.PendingCommit) {
	t.Helper()
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveProjectFormatVersion(store.CurrentProjectFormatVersion); err != nil {
		t.Fatal(err)
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "待收尾作品", Synopsis: "一章完结的测试作品"}); err != nil {
		t.Fatal(err)
	}
	content := "# 第一章\n\n已经完成但提交尚未收尾的正文。\n"
	facts := domain.ChapterFacts{Title: "第一章", Summary: "完成危机", KeyEvents: []string{"危机解除"}}
	if err := st.Drafts.SaveFinalChapter(1, content); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 1, Title: facts.Title, Summary: facts.Summary, KeyEvents: facts.KeyEvents}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, content, facts, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{
		Phase: domain.PhaseComplete, CurrentChapter: 2, TotalChapters: 1,
		CompletedChapters: []int{1}, ChapterWordCounts: map[int]int{1: domain.WordCount(content)}, TotalWordCount: domain.WordCount(content),
	}); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"完成危机","characters":[],"key_events":["危机解除"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	pending := sealedPendingForHeadlessTest(t, domain.PendingCommit{
		Chapter: 1, Stage: stage, Payload: payload, DraftContent: content,
		Result: &domain.CommitResult{Chapter: 1, Committed: true, BookComplete: true},
		Output: json.RawMessage(`{"chapter":1,"committed":true,"book_complete":true}`),
	})
	if stage == domain.CommitStageSignalSaved {
		if _, err := st.Checkpoints.AppendArtifacts(domain.ChapterScope(1), "commit", "chapters/01.md", "summaries/01.json", store.ChapterRecordPath(1)); err != nil {
			t.Fatal(err)
		}
	}
	return dir, st, pending
}

func sealedPendingForHeadlessTest(t *testing.T, pending domain.PendingCommit) domain.PendingCommit {
	t.Helper()
	var compact bytes.Buffer
	if err := json.Compact(&compact, pending.Payload); err != nil {
		t.Fatal(err)
	}
	pending.SealVersion = 1
	pending.PayloadDigest = fmt.Sprintf("%x", sha256.Sum256(compact.Bytes()))
	pending.DraftDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(pending.DraftContent)))
	intent := fmt.Sprintf("chapter=%d\x00rewrite=%t\x00rewrite_mode=%s", pending.Chapter, pending.Rewrite, pending.RewriteMode)
	pending.IntentDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(intent)))
	return pending
}

func TestRunCompletedBookWithExternalRevisionDoesNotClaimCleanTerminalState(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveProjectFormatVersion(store.CurrentProjectFormatVersion); err != nil {
		t.Fatal(err)
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "已修改作品", Synopsis: "测试外部修订"}); err != nil {
		t.Fatal(err)
	}
	baseline := "# 第一章\n\n原始正文。\n"
	facts := domain.ChapterFacts{Title: "第一章", Summary: "原始摘要", KeyEvents: []string{"原始事件"}}
	if err := st.Drafts.SaveFinalChapter(1, baseline); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, baseline, facts, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{
		Phase: domain.PhaseComplete, CurrentChapter: 2, TotalChapters: 1,
		CompletedChapters: []int{1}, ChapterWordCounts: map[int]int{1: domain.WordCount(baseline)}, TotalWordCount: domain.WordCount(baseline),
	}); err != nil {
		t.Fatal(err)
	}
	changed := "# 第一章\n\n用户手工修改后的正文。\n"
	if err := st.Drafts.SaveFinalChapter(1, changed); err != nil {
		t.Fatal(err)
	}

	disabled := false
	var stderr bytes.Buffer
	err := Run(offlineConfig(dir, &disabled), assets.Bundle{}, Options{Stderr: &stderr})
	if err == nil || !strings.Contains(err.Error(), "外部修改") || !strings.Contains(err.Error(), "/sync") {
		t.Fatalf("completed book with external revision must require sync, got %v", err)
	}
	if strings.Contains(stderr.String(), "headless 完成") {
		t.Fatalf("dirty completed book must not claim terminal completion: %q", stderr.String())
	}
	got, readErr := st.Drafts.LoadChapterText(1)
	if readErr != nil || got != changed {
		t.Fatalf("terminal probe must preserve external revision: got=%q err=%v", got, readErr)
	}
}

func TestRunCompletedBookWithPendingRevisionRequiresSync(t *testing.T) {
	dir, st, _ := completedPendingHeadlessFixture(t, domain.CommitStageSignalSaved)
	if err := st.Signals.ClearPendingCommit(); err != nil {
		t.Fatal(err)
	}
	if err := st.Revisions.SavePending(domain.PendingRevision{Stage: domain.RevisionStagePrepared}); err != nil {
		t.Fatal(err)
	}
	disabled := false
	var stderr bytes.Buffer
	err := Run(offlineConfig(dir, &disabled), assets.Bundle{}, Options{Stderr: &stderr})
	if err == nil || !strings.Contains(err.Error(), "/sync") {
		t.Fatalf("pending revision must block terminal summary with sync guidance, got %v", err)
	}
	if strings.Contains(stderr.String(), "headless 完成") {
		t.Fatalf("pending revision must not claim completion: %q", stderr.String())
	}
}

func TestRunCompletedBookWithActiveImportRequiresImportRecovery(t *testing.T) {
	dir, st, _ := completedPendingHeadlessFixture(t, domain.CommitStageSignalSaved)
	if err := st.Signals.ClearPendingCommit(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "meta", "import"), 0o755); err != nil {
		t.Fatal(err)
	}
	disabled := false
	var stderr bytes.Buffer
	err := Run(offlineConfig(dir, &disabled), assets.Bundle{}, Options{Stderr: &stderr})
	if err == nil || !strings.Contains(err.Error(), "/import") {
		t.Fatalf("active import must block terminal summary with import guidance, got %v", err)
	}
	if strings.Contains(stderr.String(), "headless 完成") {
		t.Fatalf("active import must not claim completion: %q", stderr.String())
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
