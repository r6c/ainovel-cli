package host

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	storepkg "github.com/voocel/ainovel-cli/internal/store"
)

func TestHostSnapshotUsesLatestCompletedChapterForResumeLabel(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{
		Phase:             domain.PhaseWriting,
		Flow:              domain.FlowWriting,
		Layered:           true,
		CompletedChapters: []int{1, 3, 2},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{
		Index: 1,
		Arcs: []domain.ArcOutline{{
			Index: 1,
			Chapters: []domain.OutlineEntry{
				{Title: "第一章"},
				{Title: "第二章"},
				{Title: "第三章"},
			},
		}},
	}}); err != nil {
		t.Fatal(err)
	}

	modelHost, _ := newModelConfigTestHost(t)
	modelHost.store = st
	modelHost.usage = NewUsageTracker(modelHost.models, nil)
	var events []Event
	modelHost.observer = testObserver(&events)

	snapshot := modelHost.Snapshot()
	if snapshot.RecoveryLabel != "恢复：弧末评审待处理（V1 A1）" {
		t.Fatalf("recovery label = %q, want chapter 3 arc-end label", snapshot.RecoveryLabel)
	}
}
