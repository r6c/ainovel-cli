package diag

import (
	"fmt"
	"strings"
)

// GhostCharacter 检测 core/important 角色长期未出现。
func GhostCharacter(snap *Snapshot) []Finding {
	if snap.Progress == nil || len(snap.Characters) == 0 || len(snap.Summaries) == 0 {
		return nil
	}
	completed := snap.CompletedCount()
	if completed < 5 {
		return nil
	}

	// 计算每个角色最后出现的章节号
	lastSeen := make(map[string]int)
	for ch, s := range snap.Summaries {
		for _, name := range s.Characters {
			if ch > lastSeen[name] {
				lastSeen[name] = ch
			}
		}
	}

	threshold := completed / 3
	if threshold < 5 {
		threshold = 5
	}
	latest := snap.LatestCompleted()

	var ghosts []string
	for _, c := range snap.Characters {
		if c.Tier != "core" && c.Tier != "important" {
			continue
		}
		seen, ok := lastSeen[c.Name]
		if !ok {
			// 也检查别名
			for _, alias := range c.Aliases {
				if s, exists := lastSeen[alias]; exists && s > seen {
					seen = s
					ok = true
				}
			}
		}
		gap := latest - seen
		if !ok {
			ghosts = append(ghosts, fmt.Sprintf("%s(从未出现在摘要中)", c.Name))
		} else if gap > threshold {
			ghosts = append(ghosts, fmt.Sprintf("%s(最后出现ch%d,已缺席%d章)", c.Name, seen, gap))
		}
	}
	if len(ghosts) == 0 {
		return nil
	}
	return []Finding{{
		Rule:       "GhostCharacter",
		Category:   CatContext,
		Severity:   SevInfo,
		Confidence: ConfMedium,
		AutoLevel:  AutoNone,
		Target:     "context.characters",
		Title:      fmt.Sprintf("角色消失: %d 个核心角色长期缺席", len(ghosts)),
		Evidence:   strings.Join(ghosts, "; "),
		Suggestion: "Writer 可能丢失了该角色的追踪。考虑直接在输入框提交干预指令重新引入该角色，或在 characters.json 中降级其 tier。",
	}}
}

// TimelineGaps 检测已完成章节缺少时间线事件。
func TimelineGaps(snap *Snapshot) []Finding {
	if snap.Progress == nil || len(snap.Progress.CompletedChapters) == 0 {
		return nil
	}
	if len(snap.Timeline) == 0 && snap.CompletedCount() > 0 {
		return []Finding{{
			Rule:       "TimelineGaps",
			Category:   CatContext,
			Severity:   SevInfo,
			Confidence: ConfMedium,
			AutoLevel:  AutoNone,
			Target:     "context.timeline",
			Title:      "时间线为空",
			Evidence:   fmt.Sprintf("completed=%d, timeline_events=0", snap.CompletedCount()),
			Suggestion: "commit_chapter 的时间线提取可能未生效。检查 Writer 输出是否包含 timeline 字段。",
		}}
	}

	// 建立章节→事件映射
	chaptersWithEvents := make(map[int]bool)
	for _, e := range snap.Timeline {
		chaptersWithEvents[e.Chapter] = true
	}

	var missing []int
	for _, ch := range snap.Progress.CompletedChapters {
		if !chaptersWithEvents[ch] {
			missing = append(missing, ch)
		}
	}
	// 允许少量缺失（某些过渡章可能确实无重大事件）
	if len(missing) == 0 || float64(len(missing))/float64(snap.CompletedCount()) < ThresholdTimelineGapRate {
		return nil
	}
	return []Finding{{
		Rule:       "TimelineGaps",
		Category:   CatContext,
		Severity:   SevInfo,
		Confidence: ConfMedium,
		AutoLevel:  AutoNone,
		Target:     "context.timeline",
		Title:      fmt.Sprintf("时间线缺口: %d 章无事件记录", len(missing)),
		Evidence:   fmt.Sprintf("missing=[%s]", intsToStr(missing)),
		Suggestion: "commit_chapter 的时间线提取可能部分失效。检查 Writer 输出的 timeline 字段格式。",
	}}
}

// StaleKnowledgeBelief 检测长期未纠正的活跃错误信念。
// 仅输出事实 ID 与角色名，不暴露作者 Truth 或错误信念正文。
func StaleKnowledgeBelief(snap *Snapshot) []Finding {
	if snap.Progress == nil || len(snap.Knowledge) == 0 {
		return nil
	}
	latest := snap.LatestCompleted()
	threshold := staleKnowledgeBeliefThreshold(snap.CompletedCount())
	var stale []string
	for _, entry := range snap.Knowledge {
		for _, belief := range entry.BelievedBy {
			if belief.CorrectedAt != 0 || belief.FormedAt <= 0 {
				continue
			}
			gap := latest - belief.FormedAt
			if gap > threshold {
				stale = append(stale, fmt.Sprintf("%s/%s(ch%d形成,已过%d章)", entry.ID, belief.Character, belief.FormedAt, gap))
			}
		}
	}
	if len(stale) == 0 {
		return nil
	}
	return []Finding{{
		Rule:       "StaleKnowledgeBelief",
		Category:   CatContext,
		Severity:   SevInfo,
		Confidence: ConfMedium,
		AutoLevel:  AutoNone,
		Target:     "context.knowledge",
		Title:      fmt.Sprintf("错误信念长期未纠正: %d 条超过 %d 章", len(stale), threshold),
		Evidence:   strings.Join(stale, "; "),
		Suggestion: "检查 knowledge_state 与近期正文，确认该错误信念仍在服务剧情；如角色已经获知真相，应核对 learn 事实是否漏记。",
	}}
}

func staleKnowledgeBeliefThreshold(completedChapters int) int {
	threshold := completedChapters / 3
	if threshold < 8 {
		return 8
	}
	return threshold
}

// RelationshipStagnation 检测关系数据停止更新。
func RelationshipStagnation(snap *Snapshot) []Finding {
	if snap.Progress == nil || len(snap.Relationships) == 0 {
		return nil
	}
	completed := snap.CompletedCount()
	if completed < 6 {
		return nil
	}

	// 找到关系数据的最新章节
	latestRelCh := 0
	for _, r := range snap.Relationships {
		if r.Chapter > latestRelCh {
			latestRelCh = r.Chapter
		}
	}

	// 如果最新关系数据在前 1/3，判定为停滞
	cutoff := snap.LatestCompleted() - completed/3
	if latestRelCh >= cutoff {
		return nil
	}
	return []Finding{{
		Rule:       "RelationshipStagnation",
		Category:   CatContext,
		Severity:   SevInfo,
		Confidence: ConfLow,
		AutoLevel:  AutoNone,
		Target:     "context.relationships",
		Title:      fmt.Sprintf("关系数据停滞: 最新更新在第 %d 章", latestRelCh),
		Evidence:   fmt.Sprintf("relationship_entries=%d, latest_update=ch%d, latest_completed=ch%d", len(snap.Relationships), latestRelCh, snap.LatestCompleted()),
		Suggestion: "commit_chapter 的关系更新可能停止工作，或故事关系确实无变化。检查 Writer 输出的 relationships 字段。",
	}}
}
