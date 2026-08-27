package tools

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestReconcileLayeredCompletionClosesCompleteFinaleFromPersistedFacts(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{
		Phase:             domain.PhaseWriting,
		Flow:              domain.FlowWriting,
		Layered:           true,
		CompletedChapters: []int{1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{
		Index: 1,
		Final: true,
		Arcs: []domain.ArcOutline{{
			Index:    1,
			Chapters: []domain.OutlineEntry{{Title: "终章", CoreEvent: "收束", Hook: "终"}},
		}},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 1, Scope: "arc", Verdict: "accept"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 1, Arc: 1, Summary: "弧已收束"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveVolumeSummary(domain.VolumeSummary{Volume: 1, Summary: "卷已收束"}); err != nil {
		t.Fatal(err)
	}

	complete, err := ReconcileLayeredCompletion(st)
	if err != nil || !complete {
		t.Fatalf("reconcile complete=%v err=%v, want complete", complete, err)
	}
	progress, err := st.Progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	if progress.Phase != domain.PhaseComplete {
		t.Fatalf("phase=%s, want complete", progress.Phase)
	}

	complete, err = ReconcileLayeredCompletion(st)
	if err != nil || !complete {
		t.Fatalf("second reconcile complete=%v err=%v, want idempotent complete", complete, err)
	}
}
