package headless

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/assets"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
)

func TestRunWithoutPromptOrRecoverableSessionFailsCleanlyAndExportsDiagnostics(t *testing.T) {
	dir := t.TempDir()
	disabled := false
	cfg := bootstrap.Config{
		OutputDir: dir,
		Provider:  "ollama",
		ModelName: "offline-test",
		Providers: map[string]bootstrap.ProviderConfig{
			"ollama": {Models: []bootstrap.ModelConfig{{Name: "offline-test"}}},
		},
		Notify: bootstrap.NotifyConfig{Enabled: &disabled},
	}
	var stdout, stderr bytes.Buffer

	err := Run(cfg, assets.Bundle{}, Options{Stdout: &stdout, Stderr: &stderr})
	if err == nil || !strings.Contains(err.Error(), "需要 --prompt") || !strings.Contains(err.Error(), dir) {
		t.Fatalf("empty headless workspace should return actionable error, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed resume should not emit model output: %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "headless 恢复") {
		t.Fatalf("failed resume must not claim success: %q", stderr.String())
	}
	if _, statErr := os.Stat(filepath.Join(dir, "meta", "diag-export.md")); statErr != nil {
		t.Fatalf("headless exit should export diagnostics: %v", statErr)
	}
}
