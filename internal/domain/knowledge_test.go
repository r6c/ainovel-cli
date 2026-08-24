package domain

import (
	"reflect"
	"testing"
)

func TestApplyKnowledgeUpdatesEstablishesTruthWithoutMutatingInput(t *testing.T) {
	current := []KnowledgeEntry{{
		ID: "existing", Truth: "既有真相", EstablishedAt: 1,
		KnownBy: []KnowledgeHolder{{Character: "苏晚", LearnedAt: 2}},
	}}
	before := cloneKnowledgeEntriesForTest(current)

	got, err := ApplyKnowledgeUpdates(current, 3, []KnowledgeUpdate{{
		ID: "k_shadow", Action: "establish", Truth: "黑影是林墨的兄长",
	}})
	if err != nil {
		t.Fatalf("ApplyKnowledgeUpdates: %v", err)
	}
	if len(got) != 2 || got[1].ID != "k_shadow" || got[1].Truth != "黑影是林墨的兄长" || got[1].EstablishedAt != 3 {
		t.Fatalf("established truth wrong: %+v", got)
	}
	if !reflect.DeepEqual(current, before) {
		t.Fatalf("input mutated: before=%+v after=%+v", before, current)
	}
}

func TestApplyKnowledgeUpdatesRepeatedEstablishIsIdempotent(t *testing.T) {
	current := []KnowledgeEntry{{ID: "k", Truth: "真相", EstablishedAt: 1}}
	got, err := ApplyKnowledgeUpdates(current, 5, []KnowledgeUpdate{{ID: "k", Action: "establish", Truth: "真相"}})
	if err != nil {
		t.Fatalf("repeated establish: %v", err)
	}
	if len(got) != 1 || got[0].EstablishedAt != 1 {
		t.Fatalf("repeated establish must preserve first entry: %+v", got)
	}
}

func TestApplyKnowledgeUpdatesRejectsConflictingTruthAtomically(t *testing.T) {
	current := []KnowledgeEntry{{
		ID: "k", Truth: "真相甲", EstablishedAt: 1,
		KnownBy: []KnowledgeHolder{{Character: "林墨", LearnedAt: 2}},
	}}
	before := cloneKnowledgeEntriesForTest(current)
	got, err := ApplyKnowledgeUpdates(current, 5, []KnowledgeUpdate{
		{ID: "other", Action: "establish", Truth: "另一真相"},
		{ID: "k", Action: "establish", Truth: "真相乙"},
	})
	if err == nil {
		t.Fatal("expected conflicting truth to fail")
	}
	if got != nil {
		t.Fatalf("failed apply must not return partial state: %+v", got)
	}
	if !reflect.DeepEqual(current, before) {
		t.Fatalf("failed apply mutated input: before=%+v after=%+v", before, current)
	}
}

func TestApplyKnowledgeUpdatesFormsCharacterBelief(t *testing.T) {
	current := []KnowledgeEntry{{ID: "k", Truth: "黑影是兄长", EstablishedAt: 1}}
	got, err := ApplyKnowledgeUpdates(current, 2, []KnowledgeUpdate{{
		ID: "k", Action: "believe", Character: "林墨", Belief: "黑影是仇人",
	}})
	if err != nil {
		t.Fatalf("believe: %v", err)
	}
	if len(got[0].BelievedBy) != 1 || got[0].BelievedBy[0] != (KnowledgeBelief{Character: "林墨", Content: "黑影是仇人", FormedAt: 2}) {
		t.Fatalf("belief wrong: %+v", got[0].BelievedBy)
	}
	if len(got[0].KnownBy) != 0 {
		t.Fatalf("belief must not make character know truth: %+v", got[0].KnownBy)
	}
}

func TestApplyKnowledgeUpdatesRepeatedBeliefIsIdempotent(t *testing.T) {
	current := []KnowledgeEntry{{ID: "k", Truth: "真相", EstablishedAt: 1, BelievedBy: []KnowledgeBelief{{
		Character: "林墨", Content: "误解", FormedAt: 2,
	}}}}
	got, err := ApplyKnowledgeUpdates(current, 4, []KnowledgeUpdate{{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"}})
	if err != nil {
		t.Fatalf("repeated belief: %v", err)
	}
	if len(got[0].BelievedBy) != 1 || got[0].BelievedBy[0].FormedAt != 2 {
		t.Fatalf("repeated belief must preserve first formation: %+v", got[0].BelievedBy)
	}
}

func TestApplyKnowledgeUpdatesRejectsInvalidBeliefsAtomically(t *testing.T) {
	tests := []struct {
		name    string
		current KnowledgeEntry
		update  KnowledgeUpdate
	}{
		{
			name:    "belief equals truth",
			current: KnowledgeEntry{ID: "k", Truth: "真相", EstablishedAt: 1},
			update:  KnowledgeUpdate{ID: "k", Action: "believe", Character: "林墨", Belief: "真相"},
		},
		{
			name:    "rewrite existing belief",
			current: KnowledgeEntry{ID: "k", Truth: "真相", EstablishedAt: 1, BelievedBy: []KnowledgeBelief{{Character: "林墨", Content: "误解甲", FormedAt: 2}}},
			update:  KnowledgeUpdate{ID: "k", Action: "believe", Character: "林墨", Belief: "误解乙"},
		},
		{
			name:    "character already knows truth",
			current: KnowledgeEntry{ID: "k", Truth: "真相", EstablishedAt: 1, KnownBy: []KnowledgeHolder{{Character: "林墨", LearnedAt: 2}}},
			update:  KnowledgeUpdate{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := []KnowledgeEntry{tt.current}
			before := cloneKnowledgeEntriesForTest(current)
			got, err := ApplyKnowledgeUpdates(current, 4, []KnowledgeUpdate{tt.update})
			if err == nil || got != nil {
				t.Fatalf("expected atomic rejection, got=%+v err=%v", got, err)
			}
			if !reflect.DeepEqual(current, before) {
				t.Fatalf("rejection mutated input: before=%+v after=%+v", before, current)
			}
		})
	}
}

func TestApplyKnowledgeUpdatesLearnsTruthAndCorrectsActiveBelief(t *testing.T) {
	current := []KnowledgeEntry{{
		ID: "k", Truth: "真相", EstablishedAt: 1,
		BelievedBy: []KnowledgeBelief{
			{Character: "林墨", Content: "误解", FormedAt: 2},
			{Character: "苏晚", Content: "另一误解", FormedAt: 2},
		},
	}}
	got, err := ApplyKnowledgeUpdates(current, 4, []KnowledgeUpdate{{ID: "k", Action: "learn", Character: "林墨"}})
	if err != nil {
		t.Fatalf("learn: %v", err)
	}
	if len(got[0].KnownBy) != 1 || got[0].KnownBy[0] != (KnowledgeHolder{Character: "林墨", LearnedAt: 4}) {
		t.Fatalf("known by wrong: %+v", got[0].KnownBy)
	}
	if got[0].BelievedBy[0].CorrectedAt != 4 || got[0].BelievedBy[1].CorrectedAt != 0 {
		t.Fatalf("belief correction wrong: %+v", got[0].BelievedBy)
	}
}

func TestApplyKnowledgeUpdatesRepeatedLearnPreservesFirstChapter(t *testing.T) {
	current := []KnowledgeEntry{{
		ID: "k", Truth: "真相", EstablishedAt: 1,
		KnownBy:    []KnowledgeHolder{{Character: "林墨", LearnedAt: 4}},
		BelievedBy: []KnowledgeBelief{{Character: "林墨", Content: "误解", FormedAt: 2, CorrectedAt: 4}},
	}}
	got, err := ApplyKnowledgeUpdates(current, 7, []KnowledgeUpdate{{ID: "k", Action: "learn", Character: "林墨"}})
	if err != nil {
		t.Fatalf("repeated learn: %v", err)
	}
	if len(got[0].KnownBy) != 1 || got[0].KnownBy[0].LearnedAt != 4 || got[0].BelievedBy[0].CorrectedAt != 4 {
		t.Fatalf("repeated learn drifted state: %+v", got[0])
	}
}

func TestApplyKnowledgeUpdatesRevealsTruthToReaderOnlyOnce(t *testing.T) {
	current := []KnowledgeEntry{{ID: "k", Truth: "真相", EstablishedAt: 1}}
	first, err := ApplyKnowledgeUpdates(current, 3, []KnowledgeUpdate{{ID: "k", Action: "reveal_to_reader"}})
	if err != nil {
		t.Fatalf("first reveal: %v", err)
	}
	got, err := ApplyKnowledgeUpdates(first, 6, []KnowledgeUpdate{{ID: "k", Action: "reveal_to_reader"}})
	if err != nil {
		t.Fatalf("repeated reveal: %v", err)
	}
	if got[0].ReaderRevealedAt != 3 || len(got[0].KnownBy) != 0 {
		t.Fatalf("reader reveal state wrong: %+v", got[0])
	}
	if _, err := ApplyKnowledgeUpdates(nil, 1, []KnowledgeUpdate{{ID: "missing", Action: "reveal_to_reader"}}); err == nil {
		t.Fatal("revealing unknown truth must fail")
	}
}

func TestApplyKnowledgeUpdatesReplaysSameChapterBeliefThenLearn(t *testing.T) {
	updates := []KnowledgeUpdate{
		{ID: "k", Action: "establish", Truth: "真相"},
		{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"},
		{ID: "k", Action: "learn", Character: "林墨"},
	}
	first, err := ApplyKnowledgeUpdates(nil, 3, updates)
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	got, err := ApplyKnowledgeUpdates(first, 3, updates)
	if err != nil {
		t.Fatalf("same frozen payload replay: %v", err)
	}
	if !reflect.DeepEqual(got, first) {
		t.Fatalf("same chapter replay drifted state: first=%+v replay=%+v", first, got)
	}
}

func TestApplyKnowledgeUpdatesRejectsReferenceToFutureTruth(t *testing.T) {
	current := []KnowledgeEntry{{ID: "k", Truth: "真相", EstablishedAt: 5}}
	updates := []KnowledgeUpdate{
		{ID: "k", Action: "believe", Character: "林墨", Belief: "误解"},
		{ID: "k", Action: "learn", Character: "林墨"},
		{ID: "k", Action: "reveal_to_reader"},
	}
	for _, update := range updates {
		if got, err := ApplyKnowledgeUpdates(current, 3, []KnowledgeUpdate{update}); err == nil || got != nil {
			t.Fatalf("future truth reference must fail atomically: update=%+v got=%+v err=%v", update, got, err)
		}
	}
}

func TestApplyKnowledgeUpdatesPreservesNilProjectionWithoutUpdates(t *testing.T) {
	got, err := ApplyKnowledgeUpdates(nil, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("nil projection without updates must remain nil, got %#v", got)
	}
}

func cloneKnowledgeEntriesForTest(entries []KnowledgeEntry) []KnowledgeEntry {
	out := make([]KnowledgeEntry, len(entries))
	for i, entry := range entries {
		out[i] = entry
		out[i].KnownBy = append([]KnowledgeHolder(nil), entry.KnownBy...)
		out[i].BelievedBy = append([]KnowledgeBelief(nil), entry.BelievedBy...)
	}
	return out
}
