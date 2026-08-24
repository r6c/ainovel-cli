package domain

import (
	"fmt"
	"strings"
)

// ApplyKnowledgeUpdates 在内存中按顺序应用章节知识增量。
// 它不修改 current；调用方负责持久化、事务和错误包装。
func ApplyKnowledgeUpdates(current []KnowledgeEntry, chapter int, updates []KnowledgeUpdate) ([]KnowledgeEntry, error) {
	entries := cloneKnowledgeEntries(current)
	idx := make(map[string]int, len(entries))
	for i, entry := range entries {
		idx[entry.ID] = i
	}
	for _, update := range updates {
		if strings.TrimSpace(update.ID) == "" {
			return nil, fmt.Errorf("knowledge id is required")
		}
		switch update.Action {
		case "establish":
			if strings.TrimSpace(update.Truth) == "" {
				return nil, fmt.Errorf("establish knowledge %q requires truth", update.ID)
			}
			if strings.TrimSpace(update.Character) != "" || strings.TrimSpace(update.Belief) != "" {
				return nil, fmt.Errorf("establish knowledge %q only accepts id and truth", update.ID)
			}
			if i, exists := idx[update.ID]; exists {
				if entries[i].Truth != update.Truth {
					return nil, fmt.Errorf("knowledge %q truth conflicts with established truth", update.ID)
				}
				continue
			}
			idx[update.ID] = len(entries)
			entries = append(entries, KnowledgeEntry{
				ID: update.ID, Truth: update.Truth, EstablishedAt: chapter,
			})
		case "believe":
			if strings.TrimSpace(update.Character) == "" || strings.TrimSpace(update.Belief) == "" {
				return nil, fmt.Errorf("believe knowledge %q requires character and belief", update.ID)
			}
			if strings.TrimSpace(update.Truth) != "" {
				return nil, fmt.Errorf("believe knowledge %q cannot include truth", update.ID)
			}
			i, exists := idx[update.ID]
			if !exists {
				return nil, fmt.Errorf("believe unknown knowledge %q", update.ID)
			}
			if entries[i].EstablishedAt > chapter {
				return nil, fmt.Errorf("knowledge %q is established at chapter %d, after chapter %d", update.ID, entries[i].EstablishedAt, chapter)
			}
			if strings.TrimSpace(update.Belief) == strings.TrimSpace(entries[i].Truth) {
				return nil, fmt.Errorf("belief for knowledge %q must differ from objective truth", update.ID)
			}
			repeated := false
			for _, belief := range entries[i].BelievedBy {
				if belief.Character != update.Character {
					continue
				}
				if belief.Content == update.Belief && (belief.CorrectedAt == 0 ||
					(belief.FormedAt == chapter && belief.CorrectedAt == chapter)) {
					repeated = true
					break
				}
				return nil, fmt.Errorf("character %q already has a belief for knowledge %q", update.Character, update.ID)
			}
			if repeated {
				continue
			}
			for _, holder := range entries[i].KnownBy {
				if holder.Character == update.Character {
					return nil, fmt.Errorf("character %q already knows knowledge %q", update.Character, update.ID)
				}
			}
			entries[i].BelievedBy = append(entries[i].BelievedBy, KnowledgeBelief{
				Character: update.Character, Content: update.Belief, FormedAt: chapter,
			})
		case "learn":
			if strings.TrimSpace(update.Character) == "" {
				return nil, fmt.Errorf("learn knowledge %q requires character", update.ID)
			}
			if strings.TrimSpace(update.Truth) != "" || strings.TrimSpace(update.Belief) != "" {
				return nil, fmt.Errorf("learn knowledge %q only accepts id and character", update.ID)
			}
			i, exists := idx[update.ID]
			if !exists {
				return nil, fmt.Errorf("learn unknown knowledge %q", update.ID)
			}
			if entries[i].EstablishedAt > chapter {
				return nil, fmt.Errorf("knowledge %q is established at chapter %d, after chapter %d", update.ID, entries[i].EstablishedAt, chapter)
			}
			known := false
			for _, holder := range entries[i].KnownBy {
				if holder.Character == update.Character {
					known = true
					break
				}
			}
			for j := range entries[i].BelievedBy {
				if entries[i].BelievedBy[j].Character == update.Character && entries[i].BelievedBy[j].CorrectedAt == 0 {
					entries[i].BelievedBy[j].CorrectedAt = chapter
				}
			}
			if known {
				continue
			}
			entries[i].KnownBy = append(entries[i].KnownBy, KnowledgeHolder{
				Character: update.Character, LearnedAt: chapter,
			})
		case "reveal_to_reader":
			if strings.TrimSpace(update.Truth) != "" || strings.TrimSpace(update.Character) != "" || strings.TrimSpace(update.Belief) != "" {
				return nil, fmt.Errorf("reveal knowledge %q to reader only accepts id", update.ID)
			}
			i, exists := idx[update.ID]
			if !exists {
				return nil, fmt.Errorf("reveal unknown knowledge %q to reader", update.ID)
			}
			if entries[i].EstablishedAt > chapter {
				return nil, fmt.Errorf("knowledge %q is established at chapter %d, after chapter %d", update.ID, entries[i].EstablishedAt, chapter)
			}
			if entries[i].ReaderRevealedAt == 0 {
				entries[i].ReaderRevealedAt = chapter
			}
		default:
			return nil, fmt.Errorf("invalid knowledge action %q", update.Action)
		}
	}
	return entries, nil
}

func cloneKnowledgeEntries(entries []KnowledgeEntry) []KnowledgeEntry {
	if entries == nil {
		return nil
	}
	out := make([]KnowledgeEntry, len(entries))
	for i, entry := range entries {
		out[i] = entry
		out[i].KnownBy = append([]KnowledgeHolder(nil), entry.KnownBy...)
		out[i].BelievedBy = append([]KnowledgeBelief(nil), entry.BelievedBy...)
	}
	return out
}
