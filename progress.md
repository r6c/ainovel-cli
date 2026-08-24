# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-25
- 基线：`83fbb92 功能：检测章节中的完全重复段落`
- 当前里程碑：I——Knowledge 最小诊断与脱敏导出
- 当前阶段：阶段 76——Knowledge 聚合统计红灯
- 公共接缝：`diag.Analyze(store)`、TUI `/diag`、`diag.RenderExport`

## 基线盘点

- Knowledge 已进入 Store、Commit、Revision、Import 和 Writer Context，尚未进入 Diagnostics。
- `diag.Snapshot` 当前加载 Foreshadow，但不加载 Knowledge。
- `diag.Stats` 当前只有 ForeshadowOpen/ForeshadowStale，没有认知指标。
- `host/exp` 的 TXT/EPUB 是读者成品导出，明确排除创作蓝图；本批不向其注入作者 Truth 或错误信念。
- `diag-export.md` 是脱敏运行时报告，可安全加入聚合数量，但不得出现 Truth、Belief、角色名和 Knowledge ID。

## 第一批指标

```text
KnowledgeFacts           KnowledgeEntry 数
KnowledgeKnownBy         所有 KnownBy 关系数
KnowledgeReaderKnown     ReaderRevealedAt > 0 的 Truth 数
KnowledgeActiveBeliefs   CorrectedAt == 0 的错误信念数
```

## 错误记录

- 一次搜索 `RenderReport|diag.Report|Analyze(st` 使用未闭合括号正则失败；未重复，改为读取真实 TUI report.go 和字面搜索。

## 下一步

### 阶段 76 完成

首个红灯准确：公开 `Stats` 缺少四项 Knowledge 字段。Snapshot 现读取 `knowledge_state`，`buildStats` 聚合 Truth、KnownBy、ReaderKnown 与 active belief 数量；既有 Foreshadow 诊断不回归。

### 阶段 77 完成

无 `knowledge_state.json` 时统计保持零且不产生 LoadError；损坏 JSON 进入既有 LoadError Finding，Knowledge 统计不伪造。现有 Snapshot 错误语义直接满足。

### 阶段 78 完成

长期活跃 belief、近期 belief 和已纠正 belief 同时存在时，原诊断没有 Finding。现以 max(8, completed/3) 为阈值，仅长期活跃项产生中等置信 info；证据只含 ID、角色、形成章和间隔，不含 Truth/Belief 正文。

### 阶段 79 完成

TUI `/diag` 概览原先完全缺少 Knowledge 指标。现增加一行真相、角色知情、读者已知、活跃误信数量；不展示任何 ID、角色名或正文内容，TUI 全包通过。

### 阶段 80 完成

`diag-export.md` 原先没有 Knowledge 聚合。现环境段只增加四项数字；sentinel Truth、Belief、角色名和 ID 均未出包，创作类 Finding 继续只留在本地 `/diag`。

### 阶段 81 进度

真实 TXT/EPUB 导出在存在完整 sentinel Knowledge 投影时仍不含 Truth、Belief、角色或 ID；现有 Exporter 直接满足，无生产改动。观测手册同步作者本地投影、脱敏 diag 与读者成品三层边界，并修正旧伏笔路径。

收口审计红灯发现 `buildStats` 在 Progress 缺失时提前返回，吞掉可用 Knowledge 统计。现将不依赖 Progress 的聚合移到早返回前，保持部分工件损坏时的尽力诊断语义。

阶段 81 已完成：TXT/EPUB 隔离契约、CONTEXT 与观测手册均已同步；插入 Knowledge 专章后顺带修正后续章节编号和速查引用。

## 阶段 82 / 最终验证

```text
go test ./internal/diag -count=1
go test ./internal/entry/tui -run 'Diag|Report|RenderReport' -count=1
go test ./internal/host/exp -count=1
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/diag ./internal/entry/tui ./internal/host/exp -timeout=10m
git diff --check
```

全部通过。脱敏测试先证明本地 Finding 确实包含 sentinel ID/角色，再证明可分享导出移除 ID、角色、Truth 和 Belief。里程碑 I 完成。
