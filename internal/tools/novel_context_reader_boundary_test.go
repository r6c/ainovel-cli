package tools

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestContextToolExposesReaderKnownAndCharacterKnownWithOutline(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{
		{Name: "苏弦", Role: "主角"},
		{Name: "顾临", Role: "同伴"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{
		{Chapter: 3, Title: "苏弦调查红灯", CoreEvent: "苏弦与顾临前往北侧冷阱"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{
		{
			ID: "k_reader", Truth: "红灯是求救信标", EstablishedAt: 1, ReaderRevealedAt: 2,
		},
		{
			ID: "k_character", Truth: "铜钥匙能打开北侧冷阱", EstablishedAt: 1,
			KnownBy: []domain.KnowledgeHolder{{Character: "苏弦", LearnedAt: 2}},
		},
		{
			ID: "k_hidden", Truth: "顾临曾修改过日志", EstablishedAt: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}

	raw, err := newTestContextTool(st, References{}, "default").Execute(
		context.Background(), json.RawMessage(`{"chapter":3}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Episodic map[string]json.RawMessage `json:"episodic_memory"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	var boundaries []struct {
		ID               string                   `json:"id"`
		Truth            string                   `json:"truth"`
		KnownBy          []domain.KnowledgeHolder `json:"known_by"`
		ReaderRevealedAt int                      `json:"reader_revealed_at"`
	}
	if err := json.Unmarshal(payload.Episodic["knowledge_boundaries"], &boundaries); err != nil {
		t.Fatalf("decode knowledge boundaries: %v; raw=%s", err, raw)
	}

	var reader, character *struct {
		ID               string                   `json:"id"`
		Truth            string                   `json:"truth"`
		KnownBy          []domain.KnowledgeHolder `json:"known_by"`
		ReaderRevealedAt int                      `json:"reader_revealed_at"`
	}
	for i := range boundaries {
		b := &boundaries[i]
		switch b.ID {
		case "k_reader":
			reader = &struct {
				ID               string                   `json:"id"`
				Truth            string                   `json:"truth"`
				KnownBy          []domain.KnowledgeHolder `json:"known_by"`
				ReaderRevealedAt int                      `json:"reader_revealed_at"`
			}{b.ID, b.Truth, b.KnownBy, b.ReaderRevealedAt}
		case "k_character":
			character = &struct {
				ID               string                   `json:"id"`
				Truth            string                   `json:"truth"`
				KnownBy          []domain.KnowledgeHolder `json:"known_by"`
				ReaderRevealedAt int                      `json:"reader_revealed_at"`
			}{b.ID, b.Truth, b.KnownBy, b.ReaderRevealedAt}
		}
	}
	if reader == nil || reader.Truth != "红灯是求救信标" || reader.ReaderRevealedAt != 2 || len(reader.KnownBy) != 0 {
		t.Fatalf("reader-known boundary wrong: %#v", reader)
	}
	if character == nil || character.Truth != "铜钥匙能打开北侧冷阱" || character.ReaderRevealedAt != 0 || !slices.ContainsFunc(character.KnownBy, func(h domain.KnowledgeHolder) bool {
		return h.Character == "苏弦" && h.LearnedAt == 2
	}) {
		t.Fatalf("character-known boundary wrong: %#v", character)
	}
	for _, boundary := range boundaries {
		if boundary.ID == "k_hidden" {
			t.Fatalf("hidden truth leaked into outlined context: %#v", boundary)
		}
	}
}
