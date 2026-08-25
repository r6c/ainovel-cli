package host

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	storepkg "github.com/voocel/ainovel-cli/internal/store"
)

func TestUpgradeProjectMigratesLegacyBook(t *testing.T) {
	dir := t.TempDir()
	st := storepkg.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	premise := "# 《寿元账》\n\n## 核心冲突\n\n凡人以寿元换取灵性，在求生与守住人性之间挣扎。\n\n## 主角目标\n\n活下去。"
	if err := st.Outline.SavePremise(premise); err != nil {
		t.Fatalf("SavePremise: %v", err)
	}
	progress := []byte(`{"novel_name":"寿元账"}`)
	if err := os.WriteFile(filepath.Join(dir, "meta", "progress.json"), progress, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := upgradeProject(st); err != nil {
		t.Fatalf("upgradeProject: %v", err)
	}
	book, err := st.Book.Load()
	if err != nil {
		t.Fatalf("Load book: %v", err)
	}
	if book == nil || book.Title != "寿元账" || book.Synopsis != "凡人以寿元换取灵性，在求生与守住人性之间挣扎。" {
		t.Fatalf("unexpected migrated book: %+v", book)
	}
	if checkpoint := st.Checkpoints.LatestByStep(domain.GlobalScope(), "book"); checkpoint == nil {
		t.Fatal("book checkpoint was not recorded")
	}
	version, err := st.LoadProjectFormatVersion()
	if err != nil || version != storepkg.CurrentProjectFormatVersion {
		t.Fatalf("format version = %d, err = %v", version, err)
	}
}

func TestUpgradeProjectRemovesProtectedOriginsFromLegacyRewriteQueue(t *testing.T) {
	for _, tc := range []struct {
		name      string
		origins   []domain.ChapterOrigin
		wantQueue []int
		wantFlow  domain.FlowState
	}{
		{name: "mixed", origins: []domain.ChapterOrigin{domain.ChapterOriginImported, domain.ChapterOriginGenerated, domain.ChapterOriginUser}, wantQueue: []int{2}, wantFlow: domain.FlowRewriting},
		{name: "protected_only", origins: []domain.ChapterOrigin{domain.ChapterOriginImported, domain.ChapterOriginUser}, wantFlow: domain.FlowWriting},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := storepkg.NewStore(t.TempDir())
			if err := st.Init(); err != nil {
				t.Fatal(err)
			}
			if err := st.SaveProjectFormatVersion(storepkg.CurrentProjectFormatVersion); err != nil {
				t.Fatal(err)
			}
			if err := st.Progress.Init(len(tc.origins)); err != nil {
				t.Fatal(err)
			}
			queue := make([]int, 0, len(tc.origins))
			for i, origin := range tc.origins {
				chapter := i + 1
				if _, err := st.ChapterRecords.Accept(chapter, origin, "已接纳正文", domain.ChapterFacts{}, domain.StyleDelta{}); err != nil {
					t.Fatal(err)
				}
				if err := st.Progress.MarkChapterComplete(chapter, 100, "", ""); err != nil {
					t.Fatal(err)
				}
				queue = append(queue, chapter)
			}
			if err := st.Progress.SetPendingRewrites(queue, "升级前遗留队列"); err != nil {
				t.Fatal(err)
			}
			if err := st.Progress.SetFlow(domain.FlowRewriting); err != nil {
				t.Fatal(err)
			}

			if err := upgradeProject(st); err != nil {
				t.Fatal(err)
			}
			progress, err := st.Progress.Load()
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(progress.PendingRewrites, tc.wantQueue) || progress.Flow != tc.wantFlow {
				t.Fatalf("flow=%s queue=%v want flow=%s queue=%v", progress.Flow, progress.PendingRewrites, tc.wantFlow, tc.wantQueue)
			}
		})
	}
}

func TestInterventionStopsWhenPersistenceFails(t *testing.T) {
	dir := t.TempDir()
	st := storepkg.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := st.RunMeta.Init("default", "test", "model"); err != nil {
		t.Fatalf("RunMeta.Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "meta", "run.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &Host{store: st, events: make(chan Event, 4)}
	err := h.doIntervention("修改主角性格", false)
	if err == nil || !strings.Contains(err.Error(), "持久化失败") {
		t.Fatalf("expected persistence error, got %v", err)
	}
	// 公共 Steer 必须等待异步任务并把同一业务错误返回给 TUI；不能只表示 goroutine
	// 启动成功，否则界面永远收不到真实失败。
	err = h.Steer("修改主角性格")
	if err == nil || !strings.Contains(err.Error(), "持久化失败") {
		t.Fatalf("Steer should return persistence error, got %v", err)
	}
}

func TestCloseWaitsForRegisteredAsyncWork(t *testing.T) {
	h := &Host{
		observer: &observer{},
		engine:   &engine{},
		events:   make(chan Event, 1),
		streamCh: make(chan string, 1),
		done:     make(chan struct{}, 1),
	}
	started := make(chan struct{})
	release := make(chan struct{})
	if !h.launchAsync(func() {
		close(started)
		<-release
	}) {
		t.Fatal("launchAsync unexpectedly refused")
	}
	<-started
	closed := make(chan struct{})
	go func() {
		h.Close()
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Close returned before async work finished")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after async work finished")
	}
}
