package deconstruct

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/voocel/ainovel-cli/assets"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/host/sim"
)

const usage = "用法：ainovel-cli deconstruct <本地语料目录>"

func writeEvents(events <-chan sim.Event, stderr io.Writer) int {
	for ev := range events {
		prefix := fmt.Sprintf("[%s]", ev.Stage)
		if ev.Total > 0 {
			prefix = fmt.Sprintf("[%s %d/%d]", ev.Stage, ev.Current, ev.Total)
		}
		if ev.Err != nil {
			fmt.Fprintf(stderr, "%s %s: %v\n", prefix, ev.Message, ev.Err)
			return 1
		}
		fmt.Fprintf(stderr, "%s %s\n", prefix, ev.Message)
	}
	return 0
}

// Command 执行本地语料拆文子命令。返回 0=成功，1=执行失败，2=用法/配置错误。
func Command(args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(stderr, "deconstruct 只接受一个本地语料目录")
		fmt.Fprintln(stderr, usage)
		return 2
	}
	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "deconstruct: 加载配置失败: %v\n", err)
		return 2
	}
	cfg.FillDefaults()
	bundle := assets.Load(cfg.Style, assets.DefaultLoadOptions(cfg.OutputDir))
	eng, err := host.New(cfg, bundle, host.WithFileLog("deconstruct.log", false))
	if err != nil {
		fmt.Fprintf(stderr, "deconstruct: 初始化运行时失败: %v\n", err)
		return 2
	}
	defer eng.Close()
	if logErr := eng.FileLogError(); logErr != nil {
		fmt.Fprintf(stderr, "警告：文件日志不可用，继续运行：%v\n", logErr)
	}

	events, err := eng.SimulateDir(context.Background(), args[0])
	if err != nil {
		fmt.Fprintf(stderr, "deconstruct: 启动失败: %v\n", err)
		return 1
	}
	if code := writeEvents(events, stderr); code != 0 {
		return code
	}
	fmt.Fprintln(stdout, filepath.Join(eng.Dir(), "meta", "simulation_profile.json"))
	return 0
}
