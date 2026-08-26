package tools

import "github.com/voocel/ainovel-cli/internal/domain"

// knowledgeBoundaryBelief 是给 Writer 的净化错误信念视图。
type knowledgeBoundaryBelief struct {
	Character string `json:"character"`
	Content   string `json:"content"`
	FormedAt  int    `json:"formed_at"`
}

// knowledgeBoundary 是给 Writer 的净化知识视图，不是新的事实源。
// Truth 只有在当前角色或读者已知时才输出；角色可只看到自己的活跃错误信念。
type knowledgeBoundary struct {
	ID               string                    `json:"id"`
	Truth            string                    `json:"truth,omitempty"`
	EstablishedAt    int                       `json:"established_at,omitempty"`
	KnownBy          []domain.KnowledgeHolder  `json:"known_by,omitempty"`
	Beliefs          []knowledgeBoundaryBelief `json:"beliefs,omitempty"`
	ReaderRevealedAt int                       `json:"reader_revealed_at,omitempty"`
}

// selectKnowledgeBoundaries applies the deterministic visibility policy to an
// already-loaded projection. It performs no Store IO and returns only the
// bounded, sanitized view consumed by the Writer.
func selectKnowledgeBoundaries(
	knowledge []domain.KnowledgeEntry,
	matchedCharacters []string,
	chapter int,
) []knowledgeBoundary {
	if len(knowledge) == 0 || len(matchedCharacters) == 0 {
		return nil
	}

	wanted := make(map[string]bool, len(matchedCharacters))
	for _, character := range matchedCharacters {
		wanted[character] = true
	}

	const maxKnowledge = 8
	var selected []knowledgeBoundary
	for i := len(knowledge) - 1; i >= 0; i-- {
		entry := knowledge[i]
		if entry.EstablishedAt >= chapter {
			continue
		}
		boundary := knowledgeBoundary{ID: entry.ID, EstablishedAt: entry.EstablishedAt}
		for _, holder := range entry.KnownBy {
			if wanted[holder.Character] && holder.LearnedAt < chapter {
				boundary.KnownBy = append(boundary.KnownBy, holder)
			}
		}
		for _, belief := range entry.BelievedBy {
			activeBeforeChapter := belief.FormedAt < chapter && (belief.CorrectedAt == 0 || belief.CorrectedAt >= chapter)
			if wanted[belief.Character] && activeBeforeChapter {
				boundary.Beliefs = append(boundary.Beliefs, knowledgeBoundaryBelief{
					Character: belief.Character, Content: belief.Content, FormedAt: belief.FormedAt,
				})
			}
		}
		readerKnown := entry.ReaderRevealedAt > 0 && entry.ReaderRevealedAt < chapter
		if readerKnown {
			boundary.ReaderRevealedAt = entry.ReaderRevealedAt
		}
		if len(boundary.KnownBy) > 0 || readerKnown {
			boundary.Truth = entry.Truth
		}
		if boundary.Truth != "" || len(boundary.Beliefs) > 0 {
			selected = append(selected, boundary)
			if len(selected) == maxKnowledge {
				break
			}
		}
	}
	return selected
}
