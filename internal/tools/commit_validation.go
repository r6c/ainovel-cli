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
		// 账本是全书投影，而 Projector 按章序重放章节记录。重写早期章节时账本里
		// 还躺着后续章节才种下的伏笔——放行它们，提交前校验就与重放结论相反，
		// 模型无从修正，返工队列随之锁死。故一律以"本章可见"为准。
		plantedAt := make(map[string]int, len(ledger))
		status := make(map[string]string, len(ledger))
		for _, entry := range ledger {
			plantedAt[entry.ID] = entry.PlantedAt
			status[entry.ID] = entry.Status
		}
		for i, update := range a.ForeshadowUpdates {
			switch update.Action {
			case "plant":
				if _, known := plantedAt[update.ID]; !known {
					plantedAt[update.ID] = a.Chapter
					status[update.ID] = "planted"
				}
			case "advance", "reinforce", "partial_payoff", "resolve":
				at, known := plantedAt[update.ID]
				if !known {
					return fmt.Errorf("foreshadow_updates[%d] references unknown id %q: %w", i, update.ID, errs.ErrToolPrecondition)
				}
				if at > a.Chapter {
					return fmt.Errorf("foreshadow_updates[%d] 伏笔 %q 种植于第 %d 章，不能在第 %d 章推进或回收: %w",
						i, update.ID, at, a.Chapter, errs.ErrToolPrecondition)
				}
				if status[update.ID] == "resolved" && update.Action != "resolve" {
					return fmt.Errorf("foreshadow_updates[%d] 伏笔 %q 已回收，不能再次推进: %w",
						i, update.ID, errs.ErrToolPrecondition)
				}
				switch update.Action {
				case "advance":
					status[update.ID] = "advanced"
				case "reinforce":
					status[update.ID] = "reinforced"
				case "partial_payoff":
					status[update.ID] = "partially_paid"
				case "resolve":
					status[update.ID] = "resolved"
				}
			}
		}
	}

	if len(a.KnowledgeUpdates) > 0 {
		entries, err := t.store.World.LoadKnowledgeState()
		if err != nil {
			return fmt.Errorf("load knowledge state: %w: %w", errs.ErrStoreRead, err)
		}
		establishedAt := make(map[string]int, len(entries))
		truth := make(map[string]string, len(entries))
		for _, entry := range entries {
			establishedAt[entry.ID] = entry.EstablishedAt
			truth[entry.ID] = entry.Truth
		}
		for i, update := range a.KnowledgeUpdates {
			switch update.Action {
			case "establish":
				if _, known := establishedAt[update.ID]; known {
					if truth[update.ID] != update.Truth {
						return fmt.Errorf("knowledge_updates[%d] 真相 %q 与已建立内容冲突: %w", i, update.ID, errs.ErrToolPrecondition)
					}
					continue
				}
				establishedAt[update.ID] = a.Chapter
				truth[update.ID] = update.Truth
			case "learn":
				at, known := establishedAt[update.ID]
				if !known {
					return fmt.Errorf("knowledge_updates[%d] references unknown id %q: %w", i, update.ID, errs.ErrToolPrecondition)
				}
				if at > a.Chapter {
					return fmt.Errorf("knowledge_updates[%d] 真相 %q 建立于第 %d 章，不能在第 %d 章获知: %w",
						i, update.ID, at, a.Chapter, errs.ErrToolPrecondition)
				}
			}
		}
	}
	return nil
}
