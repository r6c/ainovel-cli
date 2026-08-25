package host

import (
	"fmt"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/host/imp"
	"github.com/voocel/ainovel-cli/internal/revision"
	storepkg "github.com/voocel/ainovel-cli/internal/store"
)

// ProbeCompleted 检查小说目录是否已达到可静默退出的干净终态。
// 探测期间持有与 Host 相同的跨进程目录租约，避免和另一个实例并发读写。
func ProbeCompleted(dir string) (string, bool, error) {
	lease, err := acquireBookLease(dir)
	if err != nil {
		return "", false, err
	}
	defer lease.Close()
	st := storepkg.NewStore(dir)
	version, err := st.LoadProjectFormatVersion()
	if err != nil {
		return "", false, err
	}
	if version != storepkg.CurrentProjectFormatVersion {
		return "", false, nil
	}
	return completedSummary(st)
}

// CompletedSummary 在 Host 已持有目录租约时复用同一终态判定。
func (h *Host) CompletedSummary() (string, bool, error) {
	return completedSummary(h.store)
}

func completedSummary(st *storepkg.Store) (string, bool, error) {
	progress, err := st.Progress.Load()
	if err != nil {
		return "", false, err
	}
	if progress == nil || progress.Phase != domain.PhaseComplete {
		return "", false, nil
	}
	pending, err := st.Signals.LoadPendingCommit()
	if err != nil {
		return "", false, err
	}
	if pending != nil {
		return "", false, nil
	}
	pendingRevision, err := st.Revisions.LoadPending()
	if err != nil {
		return "", false, err
	}
	if pendingRevision != nil {
		return "", false, fmt.Errorf("存在未完成的章节外部修订；请先执行 /sync 恢复")
	}
	activeImport, doneImport, err := imp.ResumeStatus(st)
	if err != nil {
		return "", false, fmt.Errorf("导入状态读取异常；请执行 /import 查看并修复: %w", err)
	}
	if activeImport && !doneImport {
		return "", false, fmt.Errorf("存在未完成的外部小说导入；请先执行 /import 恢复")
	}
	changes, err := revision.Scan(st)
	if err != nil {
		return "", false, err
	}
	if chapters := revision.ChangedChapters(changes); len(chapters) > 0 {
		return "", false, fmt.Errorf("检测到章节正文已被外部修改：%v；请先执行 /sync", chapters)
	}
	book, err := st.Book.Load()
	if err != nil {
		return "", false, err
	}
	title := "未命名作品"
	if book != nil {
		title = book.Title
	}
	return fmt.Sprintf("headless 完成: %s（《%s》，共 %d 章 %d 字）",
		st.Dir(), title, len(progress.CompletedChapters), progress.TotalWordCount), true, nil
}
