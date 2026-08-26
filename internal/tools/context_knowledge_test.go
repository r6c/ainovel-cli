package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestContextToolBoundsKnowledgeForCurrentOutline(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(20); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{
		Chapter: 10, Title: "林墨整理线索", CoreEvent: "林墨回忆自己知道的真相",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{{
		Name: "林墨", Role: "主角", Description: "追查真相", Arc: "揭开秘密", Traits: []string{"执着"},
	}}); err != nil {
		t.Fatal(err)
	}
	var entries []domain.KnowledgeEntry
	for i := 1; i <= 12; i++ {
		entries = append(entries, domain.KnowledgeEntry{
			ID: fmt.Sprintf("k_%02d", i), Truth: fmt.Sprintf("第 %d 项作者真相", i), EstablishedAt: i,
			KnownBy: []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: i}},
		})
	}
	if err := st.World.SaveKnowledgeState(entries); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{"chapter": 10})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := newTestContextTool(st, References{}, "default").Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Episodic struct {
			Knowledge []domain.KnowledgeEntry `json:"knowledge_boundaries"`
		} `json:"episodic_memory"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Episodic.Knowledge) != 8 {
		t.Fatalf("want bounded 8 knowledge entries, got %d", len(payload.Episodic.Knowledge))
	}
}

func TestContextToolDoesNotExposeTruthUntilAfterReaderRevealChapter(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{
		Chapter: 4, Title: "林墨追查", CoreEvent: "林墨继续调查",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{{Name: "林墨", Role: "主角"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{{
		ID: "k_shadow", Truth: "黑影是林墨的兄长", EstablishedAt: 1, ReaderRevealedAt: 4,
	}}); err != nil {
		t.Fatal(err)
	}

	raw, err := newTestContextTool(st, References{}, "default").Execute(context.Background(), json.RawMessage(`{"chapter":4}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "k_shadow") || strings.Contains(string(raw), "黑影是林墨的兄长") {
		t.Fatalf("context exposed truth in its reveal chapter before prose established it: %s", raw)
	}
}

func TestContextToolExposesActiveBeliefWithoutLeakingHiddenTruth(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{
		Chapter: 4, Title: "林墨追杀黑影", CoreEvent: "林墨按自己的误解追击黑影",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{{Name: "林墨", Role: "主角"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{{
		ID: "k_shadow", Truth: "黑影是林墨的兄长", EstablishedAt: 1,
		BelievedBy: []domain.KnowledgeBelief{{Character: "林墨", Content: "黑影是杀兄仇人", FormedAt: 2}},
	}}); err != nil {
		t.Fatal(err)
	}
	raw, err := newTestContextTool(st, References{}, "default").Execute(context.Background(), json.RawMessage(`{"chapter":4}`))
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Episodic map[string]json.RawMessage `json:"episodic_memory"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	var boundaries []map[string]any
	if err := json.Unmarshal(payload.Episodic["knowledge_boundaries"], &boundaries); err != nil {
		t.Fatalf("decode knowledge boundaries: %v; raw=%s", err, raw)
	}
	if len(boundaries) != 1 {
		t.Fatalf("want one belief boundary, got %#v", boundaries)
	}
	if _, exists := boundaries[0]["truth"]; exists {
		t.Fatalf("hidden objective truth leaked into belief-only boundary: %#v", boundaries[0])
	}
	beliefs, ok := boundaries[0]["beliefs"].([]any)
	if !ok || len(beliefs) != 1 {
		t.Fatalf("active belief missing from boundary: %#v", boundaries[0])
	}
}

func TestContextToolSanitizesBeliefBoundariesByReaderCharacterAndTime(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 4, Title: "林墨追查", CoreEvent: "林墨重新判断黑影"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{{Name: "林墨", Role: "主角"}, {Name: "苏晚", Role: "盟友"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{
		{ID: "k_irony", Truth: "黑影是兄长", EstablishedAt: 1, ReaderRevealedAt: 2,
			KnownBy:    []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 4}},
			BelievedBy: []domain.KnowledgeBelief{{Character: "林墨", Content: "黑影是仇人", FormedAt: 2, CorrectedAt: 4}}},
		{ID: "k_other", Truth: "密令来自城主", EstablishedAt: 1,
			BelievedBy: []domain.KnowledgeBelief{{Character: "苏晚", Content: "密令来自皇帝", FormedAt: 2}}},
		{ID: "k_learned", Truth: "钥匙在塔顶", EstablishedAt: 1,
			KnownBy:    []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 3}},
			BelievedBy: []domain.KnowledgeBelief{{Character: "林墨", Content: "钥匙在地窖", FormedAt: 2, CorrectedAt: 3}}},
		{ID: "k_future", Truth: "门后是空城", EstablishedAt: 1,
			BelievedBy: []domain.KnowledgeBelief{{Character: "林墨", Content: "门后是敌军", FormedAt: 4}}},
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := newTestContextTool(st, References{}, "default").Execute(context.Background(), json.RawMessage(`{"chapter":4}`))
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Episodic map[string]json.RawMessage `json:"episodic_memory"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	var boundaries []map[string]any
	if err := json.Unmarshal(payload.Episodic["knowledge_boundaries"], &boundaries); err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]any{}
	for _, boundary := range boundaries {
		byID[boundary["id"].(string)] = boundary
	}
	if got := byID["k_irony"]; got == nil || got["truth"] != "黑影是兄长" || len(got["beliefs"].([]any)) != 1 {
		t.Fatalf("reader-known dramatic irony boundary wrong: %#v", got)
	} else if belief := got["beliefs"].([]any)[0].(map[string]any); belief["corrected_at"] != nil {
		t.Fatalf("current/future correction chapter leaked into active belief: %#v", belief)
	}
	if _, ok := byID["k_other"]; ok {
		t.Fatalf("unrelated belief leaked: %#v", byID["k_other"])
	}
	if got := byID["k_learned"]; got == nil || got["truth"] != "钥匙在塔顶" {
		t.Fatalf("learned truth missing: %#v", got)
	} else if _, ok := got["beliefs"]; ok {
		t.Fatalf("corrected belief remained active: %#v", got)
	}
	if _, ok := byID["k_future"]; ok {
		t.Fatalf("current-chapter belief leaked early: %#v", byID["k_future"])
	}
}

func TestContextToolExposesReaderKnownTruthWithoutTeachingCurrentCharacter(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{
		Chapter: 4, Title: "林墨追问黑影", CoreEvent: "林墨继续追查兄长下落",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{
		{Name: "林墨", Role: "主角", Description: "追查兄长", Arc: "接近真相", Traits: []string{"执着"}},
		{Name: "苏晚", Role: "盟友", Description: "掌握密令", Arc: "隐藏身份", Traits: []string{"谨慎"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{
		{ID: "k_shadow", Truth: "黑影是林墨的兄长", EstablishedAt: 1, ReaderRevealedAt: 2},
		{ID: "k_order", Truth: "苏晚奉命监视林墨", EstablishedAt: 1,
			KnownBy: []domain.KnowledgeHolder{{Character: "苏晚", LearnedAt: 2}}},
	}); err != nil {
		t.Fatal(err)
	}

	raw, err := newTestContextTool(st, References{}, "default").Execute(context.Background(), json.RawMessage(`{"chapter":4}`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "k_shadow") || !strings.Contains(text, "reader_revealed_at") {
		t.Fatalf("context missing reader-known truth: %s", text)
	}
	if strings.Contains(text, "k_order") || strings.Contains(text, "苏晚奉命监视林墨") {
		t.Fatalf("context leaked reader-unknown unrelated truth: %s", text)
	}
}

func TestContextToolSelectsKnowledgeForCharactersInCurrentOutline(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{
		Chapter: 3, Title: "林墨追问黑影", CoreEvent: "林墨向黑影确认兄长身份", Scenes: []string{"林墨追问"},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{
		{Name: "林墨", Role: "主角", Description: "追查兄长", Arc: "接近真相", Traits: []string{"执着"}},
		{Name: "苏晚", Role: "盟友", Description: "掌握密令", Arc: "隐藏身份", Traits: []string{"谨慎"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveKnowledgeState([]domain.KnowledgeEntry{
		{
			ID: "k_shadow", Truth: "黑影是林墨的兄长", EstablishedAt: 1,
			KnownBy: []domain.KnowledgeHolder{{Character: "林墨", LearnedAt: 2}},
		},
		{
			ID: "k_order", Truth: "城主密令要求苏晚监视林墨", EstablishedAt: 1,
			KnownBy: []domain.KnowledgeHolder{{Character: "苏晚", LearnedAt: 2}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]any{"chapter": 3})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := newTestContextTool(st, References{}, "default").Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "k_shadow") || !strings.Contains(text, "黑影是林墨的兄长") {
		t.Fatalf("context missing current character knowledge: %s", text)
	}
	if strings.Contains(text, "k_order") || strings.Contains(text, "城主密令要求苏晚监视林墨") {
		t.Fatalf("context leaked unrelated character knowledge: %s", text)
	}
}
