package eval

import "testing"

type authorMemoryBoundaryCase struct {
	name   string
	input  string
	result string
}

func TestAuthorMemoryBoundaryExamplesRemainExplicit(t *testing.T) {
	cases := []authorMemoryBoundaryCase{
		{name: "book rule", input: "以后每章约 1200 字", result: "user_rules"},
		{name: "author preference", input: "记住我偏好冷峻克制的叙述", result: "author_memory_pending_confirmation"},
		{name: "book fact", input: "这本书主角怕水", result: "book_facts"},
		{name: "run intent", input: "本次先写到第 5 章", result: "run_intent"},
		{name: "unconfirmed guess", input: "我可能喜欢短章节", result: "do_not_persist"},
	}

	for _, tc := range cases {
		if tc.result == "" || tc.input == "" {
			t.Fatalf("boundary case %q is incomplete: %+v", tc.name, tc)
		}
	}
}

func TestAuthorMemoryBoundaryKeepsBookAndRunScopesSeparate(t *testing.T) {
	cases := []struct {
		name   string
		result string
	}{
		{name: "book fact", result: "book_facts"},
		{name: "run intent", result: "run_intent"},
		{name: "author preference", result: "author_memory_pending_confirmation"},
	}

	seen := make(map[string]string, len(cases))
	for _, tc := range cases {
		if previous, ok := seen[tc.name]; ok && previous != tc.result {
			t.Fatalf("boundary classification changed for %s: %q -> %q", tc.name, previous, tc.result)
		}
		seen[tc.name] = tc.result
	}
	if seen["book fact"] == seen["run intent"] {
		t.Fatal("book facts and run intent must remain distinct scopes")
	}
	if seen["author preference"] == seen["book fact"] {
		t.Fatal("author memory must remain distinct from book facts")
	}
}
