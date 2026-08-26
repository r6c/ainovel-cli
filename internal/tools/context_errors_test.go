package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/store"
)

func TestContextToolWarnsWhenOptionalDataIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "meta", "simulation_profile.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := newTestContextTool(st, References{}, "default").Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	warnings, _ := got["_warnings"].([]any)
	if len(warnings) == 0 || !strings.Contains(warnings[0].(string), "simulation_profile") {
		t.Fatalf("可选资料损坏必须显式告警: %+v", got["_warnings"])
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestContextToolRejectsCorruptCoreState(t *testing.T) {
	dir := t.TempDir()
	store := store.NewStore(dir)
	if err := store.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "meta", "progress.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write progress.json: %v", err)
	}

	tool := newTestContextTool(store, References{}, "default")
	args, err := json.Marshal(map[string]any{"chapter": 2})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	_, err = tool.Execute(context.Background(), args)
	if err == nil || !strings.Contains(err.Error(), "progress") {
		t.Fatalf("核心事实损坏必须终止上下文装配: %v", err)
	}
}
