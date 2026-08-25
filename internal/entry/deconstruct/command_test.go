package deconstruct

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/host/sim"
)

func TestWriteEventsReturnsFailureAndPrintsProgress(t *testing.T) {
	var stdout, stderr bytes.Buffer
	errBoom := errors.New("模型失败")
	events := make(chan sim.Event, 3)
	events <- sim.Event{Stage: sim.StageScan, Message: "扫描本地语料"}
	events <- sim.Event{Stage: sim.StageAnalyze, Current: 1, Total: 2, Message: "分析 a.txt"}
	events <- sim.Event{Stage: sim.StageError, Message: "拆文失败", Err: errBoom}
	close(events)
	code := writeEvents(events, &stderr)
	if code != 1 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "[analyze 1/2] 分析 a.txt") || !strings.Contains(stderr.String(), "拆文失败: 模型失败") {
		t.Fatalf("unexpected events: %q", stderr.String())
	}
}

func TestCommandValidatesArgumentsWithoutLoadingRuntime(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		wantCode int
		wantText string
	}{
		{name: "missing source", wantCode: 2, wantText: "用法"},
		{name: "too many arguments", args: []string{"a", "b"}, wantCode: 2, wantText: "只接受一个本地语料目录"},
		{name: "help", args: []string{"--help"}, wantCode: 0, wantText: "deconstruct <本地语料目录>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := Command(tc.args, &stdout, &stderr)
			if got != tc.wantCode {
				t.Fatalf("exit code=%d want=%d stdout=%q stderr=%q", got, tc.wantCode, stdout.String(), stderr.String())
			}
			if output := stdout.String() + stderr.String(); !strings.Contains(output, tc.wantText) {
				t.Fatalf("output missing %q: %q", tc.wantText, output)
			}
		})
	}
}
