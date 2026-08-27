package domain

import "testing"

func TestWordCountUsesNormalizedChapterContent(t *testing.T) {
	content := "\ufeff月面\r\n维修"
	want := len([]rune(NormalizeChapterContent(content)))

	if got := WordCount(content); got != want {
		t.Fatalf("WordCount(%q) = %d, want normalized rune count %d", content, got, want)
	}
}
