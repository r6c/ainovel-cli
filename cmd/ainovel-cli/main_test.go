package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunSubcommandHandlesTopLevelHelpBeforeConfig(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "help"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			handled, code := runSubcommand([]string{arg}, &stdout, &stderr)
			if !handled || code != 0 {
				t.Fatalf("handled=%v code=%d stderr=%q", handled, code, stderr.String())
			}
			if !strings.Contains(stdout.String(), "用法：ainovel-cli") ||
				!strings.Contains(stdout.String(), "deconstruct <本地语料目录>") {
				t.Fatalf("unexpected help: %q", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("help should not write stderr: %q", stderr.String())
			}
		})
	}
}

func TestRunSubcommandDispatchesDeconstructBeforeNormalFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	handled, code := runSubcommand([]string{"deconstruct", "--help"}, &stdout, &stderr)
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d stderr=%q", handled, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "deconstruct <本地语料目录>") {
		t.Fatalf("unexpected help: %q", stdout.String())
	}

	handled, _ = runSubcommand([]string{"--headless"}, &stdout, &stderr)
	if handled {
		t.Fatal("normal flags must remain in existing parser")
	}
}
