package tools

import (
	"fmt"

	"github.com/voocel/ainovel-cli/internal/chapterfacts"
	"github.com/voocel/ainovel-cli/internal/errs"
)

// validateCommitArgs 在创建 PendingCommit 前校验模型提交的完整语义载荷。
// 错误直接返回模型修正；不生成半成品状态，也不猜测缺失值。
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
	return nil
}
