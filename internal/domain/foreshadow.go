package domain

import (
	"fmt"
	"strings"
)

// ApplyForeshadowUpdates 在内存中按顺序应用章节伏笔增量。
// 它不修改 current；调用方负责持久化、事务和错误包装。
func ApplyForeshadowUpdates(current []ForeshadowEntry, chapter int, updates []ForeshadowUpdate) ([]ForeshadowEntry, error) {
	entries := cloneForeshadowEntries(current)
	idx := make(map[string]int, len(entries))
	for i, entry := range entries {
		idx[entry.ID] = i
	}
	for updateIndex, update := range updates {
		if strings.TrimSpace(update.ID) == "" {
			return nil, fmt.Errorf("foreshadow id is required")
		}
		switch update.Action {
		case "plant":
			if strings.TrimSpace(update.Description) == "" {
				return nil, fmt.Errorf("plant foreshadow %q requires description", update.ID)
			}
			if i, exists := idx[update.ID]; exists {
				if entries[i].Description == "" {
					entries[i].Description = update.Description
				}
				if entries[i].PlantedAt == 0 {
					entries[i].PlantedAt = chapter
				}
				if entries[i].Status == "" {
					entries[i].Status = "planted"
				}
				continue
			}
			idx[update.ID] = len(entries)
			entries = append(entries, ForeshadowEntry{
				ID: update.ID, Description: update.Description, PlantedAt: chapter, Status: "planted",
			})
		case "advance":
			i, err := visibleForeshadowIndex(entries, idx, update.ID, chapter)
			if err != nil {
				return nil, err
			}
			if entries[i].Status == "resolved" {
				if isSameChapterResolvedReplay(entries[i], chapter, updates[updateIndex+1:]) {
					continue
				}
				return nil, fmt.Errorf("advance resolved foreshadow %q", update.ID)
			}
			entries[i].Status = "advanced"
			entries[i].LastAdvancedAt = chapter
		case "reinforce", "partial_payoff":
			i, err := visibleForeshadowIndex(entries, idx, update.ID, chapter)
			if err != nil {
				return nil, err
			}
			if entries[i].Status == "resolved" {
				if isSameChapterResolvedReplay(entries[i], chapter, updates[updateIndex+1:]) {
					continue
				}
				return nil, fmt.Errorf("%s resolved foreshadow %q", update.Action, update.ID)
			}
			if update.Action == "reinforce" {
				entries[i].Status = "reinforced"
			} else {
				entries[i].Status = "partially_paid"
			}
			entries[i].LastAdvancedAt = chapter
		case "resolve":
			i, err := visibleForeshadowIndex(entries, idx, update.ID, chapter)
			if err != nil {
				return nil, err
			}
			entries[i].Status = "resolved"
			entries[i].ResolvedAt = chapter
		default:
			return nil, fmt.Errorf("invalid foreshadow action %q", update.Action)
		}
	}
	return entries, nil
}

func cloneForeshadowEntries(entries []ForeshadowEntry) []ForeshadowEntry {
	if entries == nil {
		return nil
	}
	return append([]ForeshadowEntry{}, entries...)
}

func isSameChapterResolvedReplay(entry ForeshadowEntry, chapter int, remaining []ForeshadowUpdate) bool {
	if entry.ResolvedAt != chapter || entry.LastAdvancedAt != chapter {
		return false
	}
	for _, update := range remaining {
		if update.ID == entry.ID && update.Action == "resolve" {
			return true
		}
	}
	return false
}

func visibleForeshadowIndex(entries []ForeshadowEntry, idx map[string]int, id string, chapter int) (int, error) {
	i, exists := idx[id]
	if !exists {
		return 0, fmt.Errorf("foreshadow update references unknown id %q", id)
	}
	if entries[i].PlantedAt > chapter {
		return 0, fmt.Errorf("伏笔 %q 种植于第 %d 章，不能在第 %d 章推进或回收", id, entries[i].PlantedAt, chapter)
	}
	return i, nil
}
