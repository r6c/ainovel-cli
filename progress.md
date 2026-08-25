# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-25
- 基线：`91b0224 功能：为共创模式增加阶段化访谈`
- 当前里程碑：L1——本地拆文独立命令
- 当前阶段：阶段 99—105 全部完成
- 公共接缝：独立命令入口、`Host` 显式语料目录、既有 `sim.Event`/`SimulationProfile`

## Karpathy 约束下的取舍

用户说“继续”可对应“扫榜 + 拆文”两个大模块，但二者边界完全不同。最小实现选择本地拆文：

```text
ainovel-cli deconstruct <本地语料目录>
→ 复用 host/sim
→ OutputDir/meta/simulation_profile.json
```

理由：仓库现有 SimulationProfile 已覆盖单篇报告、聚合画像、增量指纹和 Agent Context 消费；再建 BenchmarkPipeline 是重复抽象。扫榜需要实时网络、来源许可、站点适配和反爬，不应在本批静默加入。

## 成功标准

1. 参数错误在调用模型前失败。
2. 本地 txt/md/markdown 可从 CLI 独立运行。
3. 使用现有配置模型和 OutputDir，不启动 Engine。
4. 输出仍是现有 SimulationProfile，Agent 无需新接线。
5. 既有 TUI `/simulate` 继续默认读取 `./simulate`。
6. 无网页抓取、无原文注入、无作者模仿。

## 阶段 105 完成

全量测试、vet、race、diff、Markdown 链接和无网络/无第二 DTO 范围扫描全部通过。按 Karpathy 简化原则移除 `writeEvents` 未使用的 stdout 参数；事件只写 stderr，最终路径由 Command 单独写 stdout。

## 阶段 104 文档与范围

README/CONTEXT 增加独立命令、兼容入口、产物路径和本地/合规边界；不写扫榜设计。一次对 commands.go 单文件路径使用目录搜索返回 ENOTDIR，未重复。

## 阶段 103 完成

核心 Prompt/Runner 合规契约已绿。剩余用户可见 TUI/Context/import 文案统一为“拆文方法画像”；保留 `/simulate`、`/importsim` 和内部 Simulation 类型用于兼容。

## 阶段 102 完成

现有 Runner 增量测试直接满足重复运行零模型调用；新增 scanner 契约锁定只读 txt/md/markdown、忽略 JSON/HTML、拒绝单文件路径。无生产改动。

## 阶段 101 完成

命令尚无 sim.Event 消费的红灯已修复。实际执行体直接复用现有配置、Bundle、Host 与 SimulateDir；不新增依赖注入框架。成功时 stdout 只打印 `meta/simulation_profile.json` 路径，进度与错误写 stderr。

## 阶段 100 完成

`Host.SimulateDir` 缺失后已最小提取。首轮手工 Host 缺 engine/models；补齐后仍在 `superviseExclusive` 异步生命周期因其它私有字段 panic。按三次失败协议停止继续猜测并读取 panic 行，确认唯一缺口是 `superviseExclusive` 使用 nil `runCtx`。精确补 `context.Background()` 到测试 Host；目录校验仍由 sim scanner 唯一拥有，不在 Host 复制。

## 阶段 103 审计

现有数据结构已是抽象画像，无需改 DTO；合规红灯确认 Prompt/Runner 仍用“仿写”命名，signature_phrases 描述也过宽。现只收敛用户/模型可见文案和字段说明，内部类型/JSON/命令名保持兼容。两次对 README/CONTEXT 单文件路径使用目录搜索返回 ENOTDIR，未重复，改用直接读取/后续文档编辑。

## 错误记录

- 对 `cmd/ainovel-cli` 单文件路径调用目录搜索返回 ENOTDIR；改为直接读取 main.go。
- 猜测 `internal/host/sim/source.go` 返回 ENOENT；真实扫描实现是 `scanner.go`，不重复猜路径。

## 下一步

阶段 99 参数契约已绿；main 分发红灯证明缺少入口。现仅增加 deconstruct 顶层分发 helper，eval 和普通 flags 保持原路径。
