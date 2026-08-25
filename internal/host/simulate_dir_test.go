package host

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/assets"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestSimulateDirUsesExplicitSourceDirectory(t *testing.T) {
	output := t.TempDir()
	st := store.NewStore(output)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	model := &plainTrackedTestModel{}
	h := &Host{
		store:  st,
		runCtx: context.Background(),
		bundle: assets.Bundle{},
		engine: &engine{},
		models: &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)},
	}
	source := filepath.Join(t.TempDir(), "missing-corpus")
	events, err := h.SimulateDir(context.Background(), source)
	if err != nil {
		t.Fatalf("SimulateDir should report scan failures as events: %v", err)
	}
	var got error
	for ev := range events {
		if ev.Err != nil {
			got = ev.Err
		}
	}
	if got == nil || !strings.Contains(got.Error(), source) {
		t.Fatalf("scan error must reference explicit source %q, got %v", source, got)
	}
}
