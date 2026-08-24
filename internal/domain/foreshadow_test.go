package domain

import (
	"reflect"
	"testing"
)

func TestApplyForeshadowUpdatesPlantsEntryWithoutMutatingInput(t *testing.T) {
	current := []ForeshadowEntry{{ID: "existing", Description: "既有伏笔", PlantedAt: 1, Status: "planted"}}
	before := append([]ForeshadowEntry(nil), current...)

	got, err := ApplyForeshadowUpdates(current, 3, []ForeshadowUpdate{{ID: "f", Action: "plant", Description: "断剑来历"}})
	if err != nil {
		t.Fatalf("ApplyForeshadowUpdates: %v", err)
	}
	want := []ForeshadowEntry{
		{ID: "existing", Description: "既有伏笔", PlantedAt: 1, Status: "planted"},
		{ID: "f", Description: "断剑来历", PlantedAt: 3, Status: "planted"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("planted projection wrong:\nwant=%+v\ngot=%+v", want, got)
	}
	if !reflect.DeepEqual(current, before) {
		t.Fatalf("input mutated: before=%+v after=%+v", before, current)
	}
}

func TestApplyForeshadowUpdatesRejectsMalformedPlantAtomically(t *testing.T) {
	current := []ForeshadowEntry{{ID: "existing", Description: "既有", PlantedAt: 1, Status: "planted"}}
	before := append([]ForeshadowEntry(nil), current...)
	for _, bad := range []ForeshadowUpdate{
		{Action: "plant", Description: "缺 ID"},
		{ID: "bad", Action: "plant"},
	} {
		got, err := ApplyForeshadowUpdates(current, 3, []ForeshadowUpdate{
			{ID: "new", Action: "plant", Description: "先成功"},
			bad,
		})
		if err == nil || got != nil {
			t.Fatalf("malformed plant must fail atomically: bad=%+v got=%+v err=%v", bad, got, err)
		}
		if !reflect.DeepEqual(current, before) {
			t.Fatalf("failed apply mutated input: before=%+v after=%+v", before, current)
		}
	}
}

func TestApplyForeshadowUpdatesPreservesNilProjectionWithoutUpdates(t *testing.T) {
	got, err := ApplyForeshadowUpdates(nil, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("nil projection without updates must remain nil, got %#v", got)
	}
}

func TestApplyForeshadowUpdatesAdvancesVisibleActiveEntry(t *testing.T) {
	current := []ForeshadowEntry{{ID: "f", Description: "伏笔", PlantedAt: 1, Status: "planted"}}
	got, err := ApplyForeshadowUpdates(current, 3, []ForeshadowUpdate{{ID: "f", Action: "advance"}})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != "advanced" || got[0].LastAdvancedAt != 3 || got[0].PlantedAt != 1 {
		t.Fatalf("advance wrong: %+v", got[0])
	}
}

func TestApplyForeshadowUpdatesRejectsInvalidAdvanceAtomically(t *testing.T) {
	tests := []struct {
		name    string
		current []ForeshadowEntry
	}{
		{name: "unknown"},
		{name: "future", current: []ForeshadowEntry{{ID: "f", Description: "伏笔", PlantedAt: 5, Status: "planted"}}},
		{name: "resolved", current: []ForeshadowEntry{{ID: "f", Description: "伏笔", PlantedAt: 1, Status: "resolved", ResolvedAt: 2}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := append([]ForeshadowEntry(nil), tt.current...)
			got, err := ApplyForeshadowUpdates(tt.current, 3, []ForeshadowUpdate{{ID: "f", Action: "advance"}})
			if err == nil || got != nil {
				t.Fatalf("invalid advance must fail atomically: got=%+v err=%v", got, err)
			}
			if !reflect.DeepEqual(tt.current, before) {
				t.Fatalf("invalid advance mutated input: before=%+v after=%+v", before, tt.current)
			}
		})
	}
}

func TestApplyForeshadowUpdatesAdvancesLifecycleStates(t *testing.T) {
	tests := []struct {
		action string
		status string
	}{
		{action: "reinforce", status: "reinforced"},
		{action: "partial_payoff", status: "partially_paid"},
	}
	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			current := []ForeshadowEntry{{ID: "f", Description: "伏笔", PlantedAt: 1, Status: "advanced", LastAdvancedAt: 2}}
			got, err := ApplyForeshadowUpdates(current, 4, []ForeshadowUpdate{{ID: "f", Action: tt.action}})
			if err != nil {
				t.Fatal(err)
			}
			if got[0].Status != tt.status || got[0].LastAdvancedAt != 4 || got[0].ResolvedAt != 0 {
				t.Fatalf("%s wrong: %+v", tt.action, got[0])
			}
		})
	}
}

func TestApplyForeshadowUpdatesRejectsInvalidLifecycleAdvance(t *testing.T) {
	for _, action := range []string{"reinforce", "partial_payoff"} {
		for _, current := range [][]ForeshadowEntry{
			nil,
			{{ID: "f", Description: "伏笔", PlantedAt: 5, Status: "planted"}},
			{{ID: "f", Description: "伏笔", PlantedAt: 1, Status: "resolved", ResolvedAt: 2}},
		} {
			got, err := ApplyForeshadowUpdates(current, 3, []ForeshadowUpdate{{ID: "f", Action: action}})
			if err == nil || got != nil {
				t.Fatalf("invalid %s must fail: current=%+v got=%+v err=%v", action, current, got, err)
			}
		}
	}
}

func TestApplyForeshadowUpdatesResolvesEntryAndReplaysSameChapter(t *testing.T) {
	current := []ForeshadowEntry{{ID: "f", Description: "伏笔", PlantedAt: 1, Status: "partially_paid", LastAdvancedAt: 3}}
	first, err := ApplyForeshadowUpdates(current, 5, []ForeshadowUpdate{{ID: "f", Action: "resolve"}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyForeshadowUpdates(first, 5, []ForeshadowUpdate{{ID: "f", Action: "resolve"}})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != "resolved" || got[0].ResolvedAt != 5 || got[0].LastAdvancedAt != 3 {
		t.Fatalf("resolve wrong: %+v", got[0])
	}
}

func TestApplyForeshadowUpdatesRejectsResolvingUnknownOrFutureEntry(t *testing.T) {
	for _, current := range [][]ForeshadowEntry{
		nil,
		{{ID: "f", Description: "伏笔", PlantedAt: 5, Status: "planted"}},
	} {
		got, err := ApplyForeshadowUpdates(current, 3, []ForeshadowUpdate{{ID: "f", Action: "resolve"}})
		if err == nil || got != nil {
			t.Fatalf("invalid resolve must fail: current=%+v got=%+v err=%v", current, got, err)
		}
	}
}

func TestApplyForeshadowUpdatesPreservesNonNilEmptyProjection(t *testing.T) {
	got, err := ApplyForeshadowUpdates([]ForeshadowEntry{}, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("non-nil empty projection must remain non-nil")
	}
}

func TestApplyForeshadowUpdatesReplaysSameChapterAdvancedThenResolvedPayload(t *testing.T) {
	updates := []ForeshadowUpdate{
		{ID: "f", Action: "plant", Description: "伏笔"},
		{ID: "f", Action: "reinforce"},
		{ID: "f", Action: "partial_payoff"},
		{ID: "f", Action: "advance"},
		{ID: "f", Action: "resolve"},
	}
	first, err := ApplyForeshadowUpdates(nil, 3, updates)
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	got, err := ApplyForeshadowUpdates(first, 3, updates)
	if err != nil {
		t.Fatalf("same frozen payload replay: %v", err)
	}
	if !reflect.DeepEqual(got, first) {
		t.Fatalf("same chapter replay drifted state: first=%+v replay=%+v", first, got)
	}
}

func TestApplyForeshadowUpdatesRepeatedPlantPreservesAndCompletesEntry(t *testing.T) {
	tests := []struct {
		name    string
		current ForeshadowEntry
		want    ForeshadowEntry
	}{
		{
			name:    "preserve existing identity and resolved state",
			current: ForeshadowEntry{ID: "f", Description: "原描述", PlantedAt: 1, Status: "resolved", ResolvedAt: 4},
			want:    ForeshadowEntry{ID: "f", Description: "原描述", PlantedAt: 1, Status: "resolved", ResolvedAt: 4},
		},
		{
			name:    "complete legacy empty fields",
			current: ForeshadowEntry{ID: "f"},
			want:    ForeshadowEntry{ID: "f", Description: "补充描述", PlantedAt: 3, Status: "planted"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyForeshadowUpdates([]ForeshadowEntry{tt.current}, 3, []ForeshadowUpdate{{ID: "f", Action: "plant", Description: "补充描述"}})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || !reflect.DeepEqual(got[0], tt.want) {
				t.Fatalf("repeated plant wrong: want=%+v got=%+v", tt.want, got)
			}
		})
	}
}
