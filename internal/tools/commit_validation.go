package tools

import (
	"fmt"

	"github.com/voocel/ainovel-cli/internal/chapterfacts"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/revision"
)

// validateCommitArgs 在创建 PendingCommit 前校验模型提交的完整语义载荷。
// 错误直接返回模型修正；不生成半成品状态，也不猜测缺失值。
func (t *CommitChapterTool) validateRewriteRecordSet(a commitArgs, progress *domain.Progress) error {
	target, err := t.store.ChapterRecords.Load(a.Chapter)
	if err != nil {
		return fmt.Errorf("load chapter record for rewrite validation: %w: %w", errs.ErrStoreRead, err)
	}
	// 兼容尚未迁移出 ChapterRecord 的旧书/测试夹具：没有可替换的事实基线时，
	// 保持既有返工路径；正式迁移完成的书会进入下方完整依赖验证。
	if target == nil {
		return nil
	}
	records, err := t.store.ChapterRecords.LoadCompleted(progress.CompletedChapters)
	if err != nil {
		return fmt.Errorf("load chapter records for rewrite validation: %w: %w", errs.ErrStoreRead, err)
	}
	found := false
	for i := range records {
		if records[i].Chapter != a.Chapter {
			continue
		}
		facts := a.ChapterFacts
		facts.ForeshadowUpdates = domain.RestoreOwnPlants(records[i].Facts.ForeshadowUpdates, facts.ForeshadowUpdates)
		records[i].Facts = facts
		found = true
		break
	}
	if !found {
		return fmt.Errorf("第 %d 章缺少接纳记录，无法验证返工事实: %w", a.Chapter, errs.ErrToolPrecondition)
	}
	if err := revision.ValidateRecordSet(records); err != nil {
		return fmt.Errorf("第 %d 章返工会破坏后续事实依赖: %v: %w", a.Chapter, err, errs.ErrToolPrecondition)
	}
	return nil
}

func (t *CommitChapterTool) validateCommitArgs(a commitArgs) error {
	if err := chapterfacts.Validate(a.ChapterFacts); err != nil {
		return fmt.Errorf("%v: %w", err, errs.ErrToolArgs)
	}

	if len(a.ForeshadowUpdates) > 0 {
		ledger, err := t.store.World.LoadForeshadowLedger()
		if err != nil {
			return fmt.Errorf("load foreshadow ledger: %w: %w", errs.ErrStoreRead, err)
		}
		// 完整投影保留未来同 ID 的证据；引用可见性、终态和同 payload 顺序
		// 均由 domain 纯 Apply 裁决。
		if _, err := domain.ApplyForeshadowUpdates(ledger, a.Chapter, a.ForeshadowUpdates); err != nil {
			return fmt.Errorf("foreshadow updates invalid: %v: %w", err, errs.ErrToolPrecondition)
		}
	}

	if len(a.KnowledgeUpdates) > 0 {
		entries, err := t.store.World.LoadKnowledgeState()
		if err != nil {
			return fmt.Errorf("load knowledge state: %w: %w", errs.ErrStoreRead, err)
		}
		// 章节可见性与生命周期都由纯 Apply 裁决；完整投影必须保留，才能在早期
		// 返工 establish 同 ID 时发现后续章节已建立的冲突 Truth。
		if _, err := domain.ApplyKnowledgeUpdates(entries, a.Chapter, a.KnowledgeUpdates); err != nil {
			return fmt.Errorf("knowledge updates invalid: %v: %w", err, errs.ErrToolPrecondition)
		}
	}
	return nil
}
