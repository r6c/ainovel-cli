package sim

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSourcesOnlyReadsSupportedLocalTextFiles(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"a.txt":       "甲",
		"b.md":        "乙",
		"c.markdown":  "丙",
		"ignore.json": `{"content":"不应扫描"}`,
		"ignore.html": "<p>不应扫描</p>",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sources, err := scanSources(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 3 || sources[0].RelativePath != "a.txt" || sources[1].RelativePath != "b.md" || sources[2].RelativePath != "c.markdown" {
		t.Fatalf("unexpected sources: %+v", sources)
	}
}

func TestScanSourcesRejectsNonDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(path, []byte("正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := scanSources(path); err == nil {
		t.Fatal("file path must not be accepted as corpus directory")
	}
}
