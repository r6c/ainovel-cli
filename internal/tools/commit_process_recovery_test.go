package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

const pendingCommitChildEnv = "AINOVEL_PENDING_COMMIT_CHILD"

func TestCommitChapterProgressMarkedRecoveryAcrossProcesses(t *testing.T) {
	if os.Getenv(pendingCommitChildEnv) == "1" {
		dir := os.Getenv("AINOVEL_PENDING_COMMIT_DIR")
		st := store.NewStore(dir)
		if err := st.Init(); err != nil {
			t.Fatal(err)
		}
		output, err := newTestCommitChapterTool(st).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
		if err != nil {
			t.Fatal(err)
		}
		if len(output) == 0 {
			t.Fatal("recovery returned empty output")
		}
		return
	}

	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.MarkChapterComplete(1, 100, "mystery", "quest"); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, "冻结正文"); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 1, Title: "第一章", Summary: "摘要"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "冻结正文", domain.ChapterFacts{
		Title: "第一章", Summary: "摘要", KeyEvents: []string{"事件"},
	}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":[],"key_events":["事件"]}`)
	output := json.RawMessage(`{"chapter":1,"committed":true,"cross_process":true}`)
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageProgressMarked, Payload: payload,
		DraftContent: "冻结正文", Output: output,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestCommitChapterProgressMarkedRecoveryAcrossProcesses$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		pendingCommitChildEnv+"=1",
		"AINOVEL_PENDING_COMMIT_DIR="+dir,
	)
	var outputLog bytes.Buffer
	cmd.Stdout = &outputLog
	cmd.Stderr = &outputLog
	if err := cmd.Run(); err != nil {
		t.Fatalf("child recovery failed: %v\noutput=%s", err, outputLog.String())
	}

	reopened := store.NewStore(dir)
	if err := reopened.Init(); err != nil {
		t.Fatal(err)
	}
	if pending, err := reopened.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("cross-process recovery left pending commit: pending=%+v err=%v", pending, err)
	}
	if cp := reopened.Checkpoints.LatestByStep(domain.ChapterScope(1), "commit"); cp == nil {
		t.Fatal("cross-process recovery did not persist commit checkpoint")
	}
	progress, err := reopened.Progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	if progress == nil || len(progress.CompletedChapters) != 1 || progress.CompletedChapters[0] != 1 {
		t.Fatalf("cross-process recovery changed completed chapters: %+v", progress)
	}
}
