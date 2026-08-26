# ainovel-cli 当前演进计划

## 当前状态

- 总体状态：`blocked_provider`
- 当前里程碑：U——Import 认知事实提取校准
- 当前阶段：阶段 158——当前 Import Prompt 三轮基线（blocked_provider）
- 基线提交：`3fddb1e 修复：保护导入与用户正文不被自动返工覆盖`
- 完整历史：[`docs/history/plans/2026-08-domain-saga-evolution/`](docs/history/plans/2026-08-domain-saga-evolution/)

## 已完成里程碑

| 里程碑 | 结果 | 提交 |
|---|---|---|
| A | 伏笔生命周期升级 | `13a775b` |
| B | 作者真相与角色知情 | `6a7b7f7` |
| C1 | 读者揭示与信息差 | `03bf271` |
| C2a | 角色错误信念与纠正 | `3ee475c` |
| D | Import 发布前全书事实重放门禁 | `cafb752` |
| E1 | Knowledge 纯 Apply/Replay 规则收敛 | `a041b5c` |
| E2 | Foreshadow 纯 Apply/Replay 规则收敛 | `429bb4a` |
| F | PendingCommit 冻结载荷完整性 | `f5e91de` |
| G | 稳定文档与规划历史归档 | `2f6768b` |
| H | 确定性重复段落 Prose Lint | `83fbb92` |
| I | Knowledge 最小诊断与脱敏统计 | `25a43d5` |
| J | 番茄平台 Rubric 试点 | `6b4050f` |
| K | 现有 cocreate 阶段化访谈 | `91b0224` |
| L1 | 本地拆文独立命令 | `4ed5e6b` |
| M | Linux 与无头环境兼容性验收 | `ade8108` |
| N | 端到端创作验收与发布就绪检查 | `efcef9f` |
| N2 | sss 真实 Provider 最小 Headless 验收 | `9c0324b` |
| O1 | Headless 已完结会话 | `652acbd` |
| O2 | Markdown 提交前格式门禁 | `3ab06d0` |
| O3 | 结构化单章篇幅目标 | `f99af1b` |
| O4 | sss 问题修复真实回归 | `bdba5d1` |
| Q | 终态恢复与正文 provenance 收口 | `37df0ca` |
| P1 | 真实外部正文 Revision 验收 | `4fd8d15` |
| R | AI 味判据证据校准 | `8d14f9f` |
| S | 真实 Cocreate 五阶段验收与流式用量修复 | `3a6b457` |
| T | 真实 Import 后续写验收与原文写权限修复 | `3fddb1e` |

## 稳定架构边界

1. `ChapterRecord.Facts` 是章节派生事实的重建输入；章节正文与事实变更通过 Commit/Revision 接纳。
2. `knowledge_state.json`、`foreshadow_ledger.json` 等是可由 ChapterRecords 全量重建的当前投影，不是第二事实源。
3. Knowledge 与 Foreshadow 生命周期分别由专用纯 Apply 函数唯一裁决；Store、Commit、Projector 是适配器。
4. Import 在分析、综合和发布前按章重放正式 ChapterFacts；非法尾部失效后自然回到 Analyze。
5. PendingCommit 首次冻结前执行纯载荷 + 当前状态校验；恢复执行密封 + 纯载荷校验后幂等重放，不按部分应用后的当前投影重新裁决。
6. Writer 只消费净化后的 `knowledge_boundaries`，不能看到当前角色与读者均未知的作者真相。
7. 不引入数据库、通用状态机、CRUD Service 或并行相邻章节写作。

# 里程碑 G：稳定文档与规划历史归档

## 阶段 64：归档完整历史

状态：`complete`

完整快照和索引已写入：

```text
docs/history/plans/2026-08-domain-saga-evolution/
```

## 阶段 65：压缩根规划工作记忆

状态：`complete`

根目录三份规划文件只保留稳定现状、当前路线、最近验证和下一候选。

## 阶段 66：建立稳定词汇表

状态：`complete`

新增根目录 `CONTEXT.md`，记录领域术语、事实源/投影边界、关键代码入口和修改纪律。

## 阶段 67：同步 README 与架构导航

状态：`complete`

补充认知事实、密封提交恢复和稳定文档入口，不改变产品定位。

## 阶段 68：验证与中文提交

状态：`complete`

验收：

```text
Markdown 相对链接可解析
根规划文件显著缩小
无 Go 生产代码变更
git diff --check
```

提交信息：

```text
文档：归档演进计划并补充领域词汇表
```

# 里程碑 H：确定性重复段落 Prose Lint

## 行为边界

- 公共接缝：`rules.Lint(text) []Violation`；Commit/Revision 继续自动消费。
- 段落：修剪首尾空白后的非空、非 `#` 标题单行。
- 仅检测完全相同段落；不做相似度或语义判定。
- 最小长度：24 个 Unicode 字符；短对话、拟声词不报。
- 同段出现至少 2 次时返回 warning：`rule=duplicate_paragraph`、`limit=1`、`actual=出现次数`。
- Target 只保存有限长度示例，避免把整段正文复制进诊断。

## 阶段 69：首个完全重复长段落

状态：`complete`

写 `rules.Lint` 公开行为失败测试并做最小实现。

## 阶段 70：低误报边界

状态：`complete`

覆盖短段、标题、不同长段、首尾空白、三个以上重复和多个重复组。

## 阶段 71：Violation 契约与隐私边界

状态：`complete`

固定 warning、limit/actual、Target 截断和确定性输出顺序。

## 阶段 72：Commit 与 Revision 集成

状态：`complete`

验证普通提交和 Revision Projector 都将重复段落事实写入既有 rule violations，不阻断流程。

## 阶段 73：Editor 语义消费

状态：`complete`

同步规则注释/文档，确认 Editor 复用 `rule_violations`，不新增 verdict 或 Route。

## 阶段 74：误报样本与范围审计

状态：`complete`

覆盖对话复沓、刻意短句、CRLF、空白行；不加入模糊匹配、跨章累计或可配置阈值。

## 阶段 75：全量验证与中文提交

状态：`complete`

```text
go test ./internal/rules -count=1
go test ./internal/tools -run 'CommitChapter|RuleViolation' -count=1
go test ./internal/revision -run 'Projector|RuleViolation' -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

提交信息：

```text
功能：检测章节中的完全重复段落
```

# 里程碑 I：Knowledge 最小诊断与脱敏导出

## 边界

- 公共接缝：`diag.Analyze(store)`、TUI `/diag` 渲染、`diag.RenderExport`。
- 统计：Truth 数、角色知情关系数、读者已知 Truth 数、活跃错误信念数。
- `diag-export.md` 只输出聚合数字，不输出 Truth、Belief、角色名或 Knowledge ID。
- TXT/EPUB 是读者成品导出，本批禁止加入任何 Knowledge 数据。
- 第一批不自动修复、不新增事实源、不修改 Knowledge 生命周期。

## 阶段 76：Knowledge 聚合统计

状态：`complete`

通过 `diag.Analyze(store)` 锁定四项聚合统计并接入 Snapshot。

## 阶段 77：空数据与加载失败

状态：`complete`

无 knowledge_state 时保持零值；损坏文件进入既有 LoadError，不伪造统计。

## 阶段 78：长期未纠正信念 Finding

状态：`complete`

以活跃 belief 的 FormedAt 与最新完成章计算停滞，仅输出 ID/角色到本地 Report；中等置信、仅建议、不自动处理。

## 阶段 79：TUI 诊断摘要

状态：`complete`

在既有概览中显示聚合指标，不新增页面或交互状态。

## 阶段 80：脱敏 diag export

状态：`complete`

只导出聚合数字；测试证明 sentinel Truth/Belief/角色名不出包。

## 阶段 81：成品导出隔离与文档

状态：`complete`

锁定 TXT/EPUB 不含 Knowledge；同步 CONTEXT/observability，不扩展读者成品格式。

## 阶段 82：全量验证与中文提交

状态：`complete`

```text
go test ./internal/diag -count=1
go test ./internal/entry/tui -run 'Diag|Report' -count=1
go test ./internal/host/exp -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

提交信息：

```text
功能：增加知识状态诊断与脱敏统计
```

# 里程碑 J：番茄平台 Rubric 试点

## 边界

- 显式输入：`user_rules.structured.platform = fanqie`；未指定平台时不注入。
- 资源：单个版本化番茄 rubric，不创建通用 Pack 框架。
- 评分：映射到现有七维，不新增第八维、Verdict、Route 或平台算法分。
- 官方事实与产品启发式分栏；不把第三方“黄金三章公式”写成官方阈值。
- 资源允许全局/本书覆盖；旧 user_rules 快照零值兼容。

## 阶段 83：显式平台偏好

状态：`complete`

扩展 `rules.Structured`、Snapshot 合并与版本兼容，先锁定显式 `fanqie` 覆盖。

## 阶段 84：规则归一化

状态：`complete`

仅当用户明确指定番茄时输出 `platform=fanqie`；未指定/含糊时为空，不自行猜测。

## 阶段 85：番茄 Rubric 资源

状态：`complete`

新增官方事实来源、软评价维度与禁用伪阈值说明；支持内置/全局/本书覆盖。

## 阶段 86：Context 条件注入

状态：`complete`

只有平台为 fanqie 时注入 `reference_pack.references.platform_rubric`，并接入既有预算裁剪。

## 阶段 87：Editor 消费纪律

状态：`complete`

映射到 pacing/hook/aesthetic/consistency 等现有维度；不得新增平台维度或机械改写。

## 阶段 88：Writer/Architect 软目标

状态：`complete`

有 rubric 时作为软适配参考，用户偏好与章节合同优先；无 rubric 时行为逐字节不变。

## 阶段 89：文档、来源与范围审计

状态：`complete`

记录官方来源日期、非官方启发式边界、旧快照兼容和不复制外部文案原则。

## 阶段 90：全量验证与中文提交

状态：`complete`

```text
go test ./internal/rules ./internal/userrules ./assets ./internal/tools -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

提交信息：

```text
功能：试点番茄平台创作评审参考
```

# 里程碑 K：现有 cocreate 阶段化访谈

## 边界

- 不新增第三种启动模式；继续使用 `startupModeCoCreate`。
- 冷启动阶段：`core → customization → title → confirmation → ready`。
- 运行中阶段共创保持原协议，不套冷启动访谈。
- 阶段由 `startup.CoCreateSession` 确定性维护；模型只回报本轮完成阶段，不得自行跳过。
- 旧/降级回复缺阶段时保持当前阶段；Draft 非空但未到 ready 不允许 Ctrl+S 启动。
- 最终仍产出一段现有创作指令，继续走 `StartPrepared`，不新增 Foundation 事实源。

## 阶段 91：访谈阶段状态

状态：`complete`

定义稳定阶段、顺序推进与冷启动/阶段共创兼容，首个测试锁定不能跳级和只有 ready 可启动。

## 阶段 92：Host 输出协议

状态：`complete`

新增 `<stage>` 标签和值域；解析缺失/非法阶段时保守保持，不破坏旧降级路径。

## 阶段 93：阶段覆盖条件

状态：`complete`

Prompt 明确各阶段最低信息：核心定位、深度定制、标题简介候选、规划确认；每轮最多问 1—2 个关键问题。

## 阶段 94：TUI 阶段可见性与门禁

状态：`complete`

显示当前阶段/进度，Ctrl+S 仅在 ready 放行；阶段共创保持既有提示与完成行为。

## 阶段 95：最终指令完整性

状态：`complete`

BuildPrompt 必须保留已确认核心定位、定制项、选定标题/简介与规划确认，不创建第二份 Foundation。

## 阶段 96：恢复、取消与流式回归

状态：`complete`

覆盖请求失败、缺标签、建议快捷键、取消、阶段共创、流式预览，不持久化半成品为正式事实。

## 阶段 97：文档与范围审计

状态：`complete`

同步 README/CONTEXT；确认不新增 Engine、Worker、Store schema 或启动模式。

## 阶段 98：全量验证与中文提交

状态：`complete`

```text
go test ./internal/entry/startup ./internal/host ./internal/entry/tui -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

提交信息：

```text
功能：为共创模式增加阶段化访谈
```

# 里程碑 L1：本地拆文独立命令

## 边界与假设

- 命令：`ainovel-cli deconstruct <本地语料目录>`。
- 只读取用户本地 `.txt/.md/.markdown`；不抓站、不收 URL、不提供扫榜功能。
- 复用现有 `host/sim`、`SimulationProfile`、结构化契约和 Agent Context；不新增 Benchmark DTO/Pipeline。
- 输出写入当前配置 `OutputDir/meta/simulation_profile.json`，与现有 `/simulate` 和 `/importsim` 兼容。
- 分析只保留抽象写法、结构、钩子、节奏与读者收益；不输出原文段落或模仿具体作者。
- 为保证 Linux/无头环境可用性，不规划需要 Chrome、登录态、网页抓取或平台页面适配的扫榜能力。

## 阶段 99：命令契约

状态：`complete`

先锁定参数、帮助、缺目录错误和退出码；不先改 sim runner。

## 阶段 100：任意本地目录运行

状态：`complete`

让 Host 现有 Simulate 支持显式 SourceDir；TUI `/simulate` 保持默认 `./simulate`。

## 阶段 101：命令执行与事件输出

状态：`complete`

构造现有 Host、消费 sim.Event、输出画像路径；不启动 Engine。

## 阶段 102：增量与错误回归

状态：`complete`

覆盖空目录、不支持文件、重复运行无新增、模型/Store 失败和取消。

## 阶段 103：合规命名与 Prompt 审计

状态：`complete`

用户可见文案从“仿写”收敛为“拆文/方法画像”，保持内部兼容名；确认不输出原文、作者模仿指令或签名短语。

## 阶段 104：文档与范围审计

状态：`complete`

同步 README/CONTEXT；确认没有网页抓取、排名事实、第二套画像协议或 Engine Route。

## 阶段 105：全量验证与中文提交

状态：`complete`

```text
go test ./cmd/ainovel-cli ./internal/entry/deconstruct ./internal/host ./internal/host/sim -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

提交信息：

```text
功能：增加本地语料拆文命令
```

## 后续边界

- 扫榜功能已从路线图移除；不新增 Chrome/CDP、浏览器登录态、平台爬虫或排行榜抓取。

## 本批明确不做

- 排行榜抓取、浏览器自动化、URL 下载、反爬或扫榜路线图
- 第二套 Benchmark/Simulation 领域模型
- 新 Engine Route、Worker、Store schema 或数据库
- 自动注入原文、复制签名短语或模仿具体作者
- 多平台抓取器、插件系统或通用数据源框架

## 错误记录

| 错误 | 处理 |
|---|---|
| 搜索 Projector 声明时使用未闭合括号正则 | 停止重复该查询，改用字面搜索确认 `ValidateRecordSet` 与 `Projector.Apply` |

# 里程碑 M：Linux 与无头环境兼容性验收

## 边界与成功标准

- 公共接缝：顶层 CLI 帮助、`deconstruct --help`、现有通知 Adapter、Linux CI、Docker 镜像入口。
- `--help/-h/help` 必须在配置加载、首次引导、TTY 和模型初始化之前退出。
- CI 必须显式构建 Linux amd64/arm64，并在 Ubuntu 原生执行帮助命令。
- Docker 镜像必须能在无配置、无 TTY、无 DISPLAY 情况下执行帮助命令。
- Linux 缺少桌面通知能力时只降级日志，不影响创作控制流。
- 不新增浏览器、GUI 库、平台爬虫、配置框架或第二套发布系统。

## 阶段 106：顶层无配置帮助命令

状态：`complete`

先写顶层 CLI 公开行为失败测试：`--help/-h/help` 返回 0、写 stdout，并在常规配置解析前完成；`deconstruct --help` 保持现状。

## 阶段 107：Linux 跨编译门禁

状态：`complete`

在现有 CI 中显式构建 `linux/amd64` 与 `linux/arm64`，使用 `CGO_ENABLED=0`，不创建新的构建框架。

## 阶段 108：Ubuntu 无头与通知降级回归

状态：`complete`

CI 原生执行顶层/deconstruct 帮助；锁定通知缺少 system 通道时为 best-effort，不重写已满足的生产逻辑。

## 阶段 109：Docker 无头冒烟

状态：`complete`

复用现有 Dockerfile，在 Ubuntu CI 构建本地镜像并执行 `--help` 与 `deconstruct --help`；不调用模型、不挂用户配置。

## 阶段 110：Linux 文档与范围审计

状态：`complete`

同步 README/CONTEXT，说明无头帮助、通知降级和扫榜移除边界；确认无生产 `/tmp`、Chrome/CDP 或 GUI 强依赖。

## 阶段 111：全量验证与中文提交

状态：`complete`

```text
go test ./cmd/ainovel-cli ./internal/entry/deconstruct ./internal/notify -count=1
go test ./... -timeout=5m
go vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/ainovel-cli
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/ainovel-cli
git diff --check
```

预定提交信息：

```text
测试：加固 Linux 与无头环境兼容性
```

# 里程碑 N：端到端创作验收与发布就绪检查

## 边界与成功标准

- 公共路径：Quick、Cocreate、Headless 恢复、Import、Deconstruct、TXT/EPUB 导出。
- 自动化只使用现有 fake/mock 模型，不调用用户 Provider、不消耗真实额度。
- 复用现有 Engine/Host/Store 测试接缝，不建立新的 E2E 框架。
- 新增发布验收文档，区分自动化证据、人工真实模型步骤与阻塞等级。
- 只有真实覆盖缺口才新增测试；已有强覆盖只引用现有测试命令。
- 本批不因验收过程顺带增加产品功能或领域状态。

## 阶段 112：验收矩阵与 Headless 首个冒烟

状态：`complete`

盘点五条路径现有覆盖；针对当前唯一明显空白 `internal/entry/headless`，先用公开 `Run` 写一个 fake-model 冒烟红灯。

## 阶段 113：Quick/Engine 与导出证据

状态：`complete`

复用现有 Engine 完整书和 TXT/EPUB 隔离测试；只在入口接线存在真实缺口时补一条最小测试。

## 阶段 114：Cocreate 启动验收

状态：`complete`

复用 Session/Host/TUI 阶段契约，验证完整确认后的 BuildPrompt 仍进入现有启动主链；不新增启动模式。

## 阶段 115：Import 发布与恢复验收

状态：`complete`

运行现有全书门禁、原子发布、首错回退和继续写作接缝；只补缺失的公开路径证明。

## 阶段 116：Deconstruct 增量与合规验收

状态：`complete`

运行现有 CLI/Runner 增量与合规测试，确认第二次运行零模型调用和画像无原文模仿指令。

## 阶段 117：发布验收文档与人工清单

状态：`complete`

新增 `docs/release-acceptance.md`，记录自动化命令、人工真实模型步骤、工件位置与 P0/P1/P2 判定；人工项保持未执行，不自动消费额度。

## 阶段 118：全量验证与中文提交

状态：`complete`

```text
go test ./internal/entry/headless ./internal/entry/startup ./internal/host ./internal/host/imp ./internal/host/sim ./internal/host/exp -count=1 -timeout=5m
go test ./... -timeout=5m
go vet ./...
git diff --check
```

预定提交信息：

```text
测试：建立端到端创作发布验收基线
```

# 里程碑 N2：真实 Provider 人工验收

## 边界

- 使用用户已明确授权的 `sss / gpt-5.6-sol`，会产生真实 API 用量。
- 在系统临时目录运行，不污染仓库或正式小说目录。
- 首个切片仅跑单章约 1200 字科幻短篇 Headless 闭环；15 分钟超时。
- 项目级配置关闭通知并设置小额硬预算；不修改用户全局配置、不输出密钥。
- 先检查创作工件、PendingCommit、Checkpoint、Progress、诊断和日志，再决定是否扩大到强杀恢复或 Cocreate。

## 阶段 119：sss 最小 Headless 闭环

状态：`complete`

构建当前二进制，在独立工作区真实运行 `--headless --prompt`，记录耗时、退出码和工件完整性。

## 阶段 120：工件与质量检查

状态：`complete`

检查章节、ChapterRecord、Knowledge/Foreshadow、Progress、Checkpoint、PendingCommit、诊断、规则违规和读者成品；不把内部创作事实写入公开日志。

## 阶段 121：恢复与共创决策

状态：`complete`

根据首轮成本与结果决定是否继续真实强杀恢复/Cocreate；未经再次成本判断不自动扩大调用。

## 阶段 122：记录结果

状态：`complete`

只提交脱敏验收结论和文档状态，不提交临时小说、日志、配置或任何凭证。

# 里程碑 O：真实创作验收问题收敛

## 公共接缝与成功标准

1. O1：`headless.Run`——已完结工作区无 Prompt 启动返回成功摘要，不调用模型、不改工件/费用；空工作区仍报错。
2. O2：`CommitChapterTool.Execute`——正文含确定性 Markdown 残留时，在创建 PendingCommit 前返回可修正错误；重复段落等审美 warning 仍不阻断。
3. O3：`UserRules → novel_context/Chapter Contract → Commit`——明确篇幅目标结构化传递，并在提交前给出有界、可修正约束；不建立新配置系统。

每项独立红→绿并独立提交；未完成 O1 前不修改 O2/O3。

## 阶段 123：Headless 完结态

状态：`complete`

首个失败测试：已完结 Store 调用 `headless.Run`，期望返回 nil、输出标题/章数/字数摘要，usage、Progress、章节与 PendingCommit 不变。

## 阶段 124：Markdown 提交前格式门禁

状态：`complete`

只拦截确定性 Markdown 残留，发生在 PendingCommit 创建前；保留 `rules.Lint` 的其他 warning 语义。

## 阶段 125：结构化篇幅目标

状态：`complete`

先盘点 UserRules/Chapter Contract 既有字段与字符计数口径，再确定最小模型；不得用脆弱正则替代已有 Normalizer。

## 阶段 126：真实 sss 回归决策

状态：`complete`

已在全新隔离目录复用 `sss / gpt-5.6-sol` 做同类单章回归：目标 1200、实际 1311 字，Markdown/其它规则违规为 0，无 PendingCommit，完结态二次启动零写入，全量事实重放与 TXT/EPUB 隔离通过；费用约 `$0.161`。临时正文、日志和配置不进入仓库。

# 里程碑 Q：终态恢复与正文接纳 seam 收口

## 目标

在继续真实 Revision 验收前关闭全项目复审发现的两个 S1：

1. `phase=complete` 仍可能存在 PendingCommit/PendingRevision/Import 等恢复工作，不能直接静默终止。
2. Import 用户原文复用生成正文格式门禁，可能在正式 Foundation/Hold 写入后卡死发布。

原则：复用现有 Commit Saga、Revision 和 Import 工作区；不新建通用工作流框架，不静默修改用户原文。

## 阶段 127：终态恢复红灯

状态：`complete`

公共 seam：`headless.Run`、`Host.Resume`、TUI bootstrap、`CommitChapterTool.Execute`。

首个场景：构造 `phase=complete + sealed PendingCommit(progress_marked)`，Headless/TUI 必须先补 checkpoint、清 PendingCommit，再显示完成；不得直接命中完成摘要。追加：`signal_saved`、密封损坏、不同章冲突、无 Pending 的纯终态回归。

## 阶段 128：确定性 PendingCommit 恢复优先级

状态：`complete`

优先方案：Host Resume 在生成 resume label 前，直接通过现有 `CommitChapterTool.Execute` 幂等收尾冻结 PendingCommit，不调用 Writer 模型；完成后重新读取 Progress。若测试证明该位置破坏职责，再设计 `flow.State.PendingCommit` 路由，不在 Host/Flow 两处重复恢复规则。

验收：任意 phase 都能恢复四个 Commit Stage；完整性失败保留工件并返回稳定错误；不消耗预算、不改冻结正文。

## 阶段 129：真正可静默终止判定

状态：`complete`

终态摘要只有同时满足以下条件才允许：

```text
当前项目格式
phase=complete
无 PendingCommit
无 PendingRevision
无未完成 Import 工作区
无外部正文哈希变化
取得书目录独占租约
```

具体 interface 由阶段 127/128 结果决定；Headless 与 TUI 必须复用同一 Store 派生判定，不各自维护文件清单。外部修订只给出 `/sync` 指引，不在无授权情况下自动调用模型。

## 阶段 130：Import Markdown/篇幅政策红灯

状态：`complete`

公共 seam：`imp.Run/publish` + 正式 `CommitChapterTool`。

构造用户源章节含 `**`、内部 `##`，以及已有书级 `chapter_target_chars` 小于原文章节长度。要求：原文逐字保留发布，Lint 事实可记录，但不能使用生成正文的 Markdown/篇幅硬门禁；若存在其它不可接纳问题，必须在任何正式 Foundation/Hold/章节写入前失败并回退可修复状态。

## 阶段 131：正文 provenance 接纳 module

状态：`complete`

深化现有章节接纳 seam，至少区分：

```text
generated  模型新生成正文：Markdown/篇幅政策生效
imported   用户导入原文：保留内容，只记录 Lint
user       外部人工修订：保留内容，经 Revision 语义重建
```

推荐为 `CommitChapterTool` 增加非模型调用的 imported 接口，并将 provenance 纳入冻结意图/ChapterRecord；普通模型 `Execute` 不能自行伪造 origin。Import 发布前验证与正式提交必须消费同一接纳政策，不复制 Saga。

## 阶段 132：Import 零污染与恢复矩阵

状态：`complete`

验证多章 Markdown 源、发布中断、stale PendingCommit、重新运行和 TXT/EPUB 原文保真。所有失败在正式写入前或由 Saga 可恢复；`NextAction` 不得永久停在 Publish。

## 阶段 133：UserRules 撤销语义与数值上界

状态：`complete`

先由测试固定三态：未声明 / 设置正值 / 明确清除。推荐 `*int` 语义（nil=未声明，0=清除，正值=设置），但只有 strict schema 能稳定表达 nullable 时采用；否则定义显式 action 字段。禁止用负数暗号。

为 `chapter_target_chars` 定义合理上界，并用溢出安全公式计算 120%；运行中“取消每章字数限制”必须清除旧快照值。旧 v1-v4 快照兼容。

## 阶段 134：文档、全量验证与真实 Revision 计划门禁

状态：`complete`

同步规则所有权：Lint 只生成事实；正文接纳 adapter 按 provenance 决定哪些事实阻断。更新 CONTEXT、Import 文档、UserRules 文档和发布验收。

最终门禁：

```text
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/store ./internal/tools ./internal/revision ./internal/host ./internal/entry/headless ./internal/host/imp -count=1 -timeout=10m
git diff --check
```

里程碑 Q 完成后再规划 P1 真实外部正文 Revision；未经用户重新确认预算不调用 Provider。

## 本批明确不做

- 新认知动作、伏笔状态或 ChapterContract 字段
- 通用工作流/状态机框架
- 自动清洗用户导入正文
- 扫榜、浏览器或网络抓取
- 数据库、Web 事实源或第四 Worker
- 真实 Provider 调用

# 里程碑 P1：真实外部正文 Revision 验收

## 目标

验证作者手工修改已接纳正文后，Revision 能检测哈希变化、使用真实模型重建 ChapterFacts/StyleDelta、全量重建派生投影，并在崩溃恢复后保持幂等。真实调用只在用户确认预算后执行。

## 阶段 135：Revision 自动化验收基线

状态：`complete`

复跑并补齐：外部修改扫描、Prepared/RecordsApplied/ProjectionsApplied 三阶段恢复、候选全书事实验证、规则投影刷新、完结书 `/sync` 工作台路径。只补真实缺口，不复制现有 Revision 大矩阵。

## 阶段 136：隔离真实作品与修改方案

状态：`complete`

从临时验收作品复制到新的隔离目录，不修改原真实验收目录。选择一处明确但有限的事实变化，保存修改前 ChapterRecord、投影、Progress、Usage 与正文摘要；不记录 Provider 密钥。

## 阶段 137：用户确认预算

状态：`complete`

向用户确认本次 Revision 真实模型预算上限；未确认前停止。建议上限 `$0.25`，通知关闭，超限硬停。

## 阶段 138：真实 Revision 与恢复验证

状态：`complete`

执行 `/sync` 等价 Host 接口，必要时在 PendingRevision 中间阶段做一次可控强杀；验证 revision+1、origin=user、ContentSHA 更新、旧事实移除、新事实重建、PendingRevision 清理、再次同步零调用。

## 阶段 139：下游一致性与脱敏记录

状态：`complete`

验证 Context 使用新事实、TXT/EPUB 反映修改正文且不泄露内部状态；把费用、工件和 P0/P1/P2 脱敏写入发布验收，临时正文/日志/配置不进 Git。

# 里程碑 R：AI 味判据证据校准

## 目标

不安装外部 Skill、不新增第二条去 AI 味管线。使用本项目自有网文最小对与盲评，校准 `assets/references/anti-ai-tone.md` 和 Editor 的语义判据；确定性规则仍只进入现有 `rules`，未经样本证明不新增硬门禁。

## 阶段 140：校准语料与盲评协议

状态：`complete`

建立 8 组网文最小对：翻案腔、句间同构、句内排比、段落长度、问句、比喻、冒号、段首回指。每组一条应改、一条应保留；样本隐藏来源与预期标签，要求 Judge 只回答“是否应仅因 AI 味而修改”，并引用证据。固定随机顺序和 JSON schema，不把外部研究结论写进 Judge prompt。

## 阶段 141：真实盲评重复与人工复核

状态：`complete`

使用 `sss / gpt-5.6-sol` 对同一集合至少重复 3 轮，记录每样本 vote/confidence/reason 与费用；再按叙事功能人工复核。模型意见是辅助证据，不能替代 worked-example 标签。

## 阶段 142：基线误报/漏报矩阵

状态：`complete`

逐条对照当前 `anti-ai-tone.md` 与 `story-deslop`：统计会被现有规则标记但盲评/人工应保留的反例，以及外部候选能发现而当前遗漏的样本。优先验证句内排比、句长/段长、问句、比喻泛化是否误报；验证段首零回指、提示性冒号、理想化职业人格喻体是否有增量。

## 阶段 143：最小判据修订

状态：`complete`

仅修改有 worked-example 证据的语义资产：把“AI 来源判断”与“一般审美问题”分开；删除或收窄无证据泛化，加入小说体裁例外、目标风格优先、信息守恒。不得复制外部 SKILL 长段文本；若使用具体规则表达，保留来源与 MIT 说明。

## 阶段 144：Writer/Editor A/B 回归

状态：`complete`

使用现有 eval variant 或独立短篇真实调用，对基线与修订判据进行至少 3 次 A/B。门禁仍只认确定性事实；模型 Judge 只报告误报、保留叙事功能和审美偏好。确认新判据不会诱导删剧情、机械拆段、删除问句/比喻或统一口语化。

## 阶段 145：文档、全量验证与提交

状态：`complete`

同步 `CONTEXT.md`、AI 味来源与许可证记录、评测报告；运行 assets/rules/eval/全量测试、vet、race、格式与链接检查。建议中文提交：`校准：收窄去 AI 味判据并补充反例`。

## 明确不做

- 不整体安装 `lieflat-less-ai-tone`
- 不把 Python 脚本加入运行时或 Docker
- 不新增第二个 Skill/Service/Writer 后处理流程
- 不用 AI 检测概率作为目标
- 不把语义 Judge 变成 Commit 硬门禁
- 不因单次模型偏好改规则
- 不复制外部不可核验统计数字作为本项目阈值

# 里程碑 S：真实 Cocreate 五阶段验收

## 目标

使用 `sss / gpt-5.6-sol` 在隔离目录执行冷启动共创，验证模型协议、确定性 Session、用户选择/确认和现有启动主链。只做验收；发现真实 P0/P1 才进入修复，不为测试新建共创框架。

## 阶段 146：验收基线与隔离目录

状态：`complete`

复跑 Session/Host/TUI 代表性测试；创建不含凭证的隔离 OutputDir，加载现有全局 Provider 后只覆盖目录、预算和通知。

## 阶段 147：真实五阶段对话

状态：`complete`

最多 8 轮，按当前阶段提供固定用户回答。记录 stage、ready、Draft 标题完整性和 suggestions 数量，不保存完整模型思考/回复到 Git。阶段只能保持或前进一格；模型跳级由 Session 拒绝。

## 阶段 148：标题授权与最终指令

状态：`complete`

在 title 阶段由用户明确授权模型代选；confirmation 阶段明确确认。要求 `CanStart=true`、`BuildPrompt` 成功、四个独立二级标题齐全、无“待确认”，且创作指令保留题材、主角、冲突、规模、视角、基调和目标平台。

## 阶段 149：现有启动主链

状态：`complete`

用最终 Prompt 调用 `PrepareUserRules → StartPrepared`，只运行到 Foundation 完成并进入 writing 后 Abort；验证 Book/Premise/Outline/UserRules 已落盘，无正式章节、无 PendingCommit，不复制第二套 Foundation 生成器。

## 阶段 150：脱敏记录与提交

状态：`complete`

记录轮次、阶段轨迹、费用、工件和 P0/P1/P2；临时对话、正文、配置和日志不进 Git。运行相关测试、全量/vet/race/格式与链接检查后中文提交。

# 里程碑 T：真实 Import 后续写验收

## 目标

用 `sss / gpt-5.6-sol` 在隔离目录导入一篇自建两章未完小说，先确认切分，再完成分析/综合/发布；验证 imported provenance 和全书投影后，显式接力 Writer 续写第 3 章，确认继承导入事实。发现 P0/P1 时先修，不扩展 Import 管线。

## 阶段 151：自动化基线与自建源文本

状态：`complete`

复跑 Import 端到端、事实零污染、stale Pending、Knowledge/Foreshadow 发布和 continue 接力测试。创建不含第三方内容的两章科幻悬疑源文，明确故事未完，并设置可验证的角色状态、物件、地点、伏笔和作者真相。

## 阶段 152：真实切分与人工确认

状态：`complete`

首次 `AutoConfirm=false`，要求得到两章切分预览且无错误；检查边界/标题后用 `AcceptSegmentation=true` 显式确认。Source/Manifest/Segmentation/Confirmation 必须完整，正式 Store 仍为空。

## 阶段 153：真实分析、综合与正式发布

状态：`complete`

使用 `StoryResolution=open`、`ContinueAfter=false` 完成两章分析、综合和发布。验证 ChapterRecord 1/2 均为 imported、原文哈希一致、Progress/Checkpoint/Hold 正确、无 PendingCommit，Knowledge/Foreshadow/Timeline/Relationship/State 投影可全量重建。

## 阶段 154：下游 Context 一致性

状态：`complete`

调用 `novel_context(chapter=3)`，确认只注入角色/读者可知信息，不泄露隐藏 Truth；当前角色状态、未回收伏笔和第 2 章结尾目标可供 Writer 使用。

## 阶段 155：真实续写第 3 章

状态：`complete`

显式接力现有 Engine，只允许新增第 3 章后暂停。验证 Writer 不重写 1/2 章，不违反导入 Knowledge 边界，推进或回收既有伏笔时复用 ID；ChapterRecord 3 为 generated，三章全量重放通过。

## 阶段 156：导出、脱敏记录与提交

状态：`complete`

验证 TXT/EPUB 含三章且不泄露内部状态；记录阶段、费用、工件和 P0/P1/P2，源文/正文/日志/配置不进 Git。运行全量/vet/race/格式/链接后中文提交。

# 里程碑 U：Import 认知事实提取校准

## 目标

用自建中文小说 worked examples 和 `sss / gpt-5.6-sol` 校准 Import 对 establish/believe/learn/reveal_to_reader 的提取。先跑实际 analysisContract 三轮基线；只有稳定漏报/误报才最小改 Prompt，再做同协议 A/B。禁止用确定性代码从正文猜测认知动作。

## 阶段 157：自建认知动作校准集

状态：`complete`

建立 12 条匿名片段与独立金标，覆盖 establish-only、establish+learn、establish+reveal、三动作、belief、猜测/谎言/不相信/模糊暗示等负例。每条使用稳定 Knowledge ID；样本不泄露标签，金标只比较 knowledge_updates。

## 阶段 158：当前 Import Prompt 三轮基线

状态：`blocked_provider`

每轮改变样本顺序，直接使用当前 import analysisContract 与严格 Schema；本次改为每次单片段调用，避免不同 Truth 的 Knowledge ID 在同一批次互相污染。统计动作级 precision/recall、完整集合准确率、三轮一致性、Token/费用。不得用简化 Judge 替代真实 Import 协议。

前次完整运行受本地代理长连接、模型不一致 Knowledge ID 自愈和 HTTP 502 阻塞，未计入证据；本次每次调用使用独立 5 分钟 context timeout。单批通道曾成功，但随后单片段三轮重跑再次长时间停在本地代理 TCP 连接，无结果文件；已终止，不计入模型证据。Provider 通道稳定后再从本阶段重跑。

最小单批通道验证成功：2 条普通事实约 31 秒，均正确输出 establish，Usage 有记录。完整三轮基线未取得有效结果：一次长时间停在本地代理 TCP 连接；改为每批 2 条后，第 1 轮第 3 批出现模型不一致 Knowledge ID，进入正式未知引用自愈，随后 HTTP 502 并在 5 分钟 context timeout 结束。没有完整结果文件，不计入模型质量证据。上述完整基线未形成有效结果，不据此计算质量指标；随后已通过阶段 159 的定向证据做最小 Prompt 修订，并由阶段 160 记录有限 A/B。

## 阶段 159：最小 Prompt 修订

状态：`complete`

已根据定向真实探针修订 import-analyze Prompt：明确同一 Truth 可在同章按正文顺序输出多个动作，并补充 `establish → learn → reveal_to_reader` 与猜测/不相信/部分兑现负例边界。随后根据负例复验补充：未经确认的说法、未决指控、故意说谎或广播未经证实的指控，不等于作者 Truth 或完整读者揭示。`ik03`、`ik04`、`ik05` 修订后分别补齐预期的 learn/reveal 动作；不复制新规则到 Go，不修改 Schema/DTO/生命周期。

## 阶段 160：同模型 Prompt A/B

状态：`partial_evidence`

完整三轮 A/B 暂未完成，不能计算完整 precision/recall 或一致性。有限证据如下：ik04 baseline 仅 establish，calibrated 输出 establish→learn→reveal_to_reader；ik05 baseline 输出空数组，calibrated 输出 establish→reveal_to_reader；ik07 baseline 与 calibrated 都输出 establish→learn→reveal_to_reader，并都额外输出顾临 believe，因此该误报不是本次修订引入。修订版负例 ik10（猜测）、修正后的 ik11（未经核验的说法）、ik12（明确不相信）均输出空数组。证据支持 learn/reveal 召回改善，且未在三个负例上新增误报，但既有 believe 误报风险仍待完整样本验证；不进入阶段 161。完整 A/B 仍需 Provider 稳定后从阶段 158/160 重跑。

## 阶段 161：真实两章 Import/Context 回归

状态：`pending`

复用自建两章源文，要求求救信标 Truth 的 EstablishedAt/苏弦 KnownBy/ReaderRevealedAt 正确，顾临未知；第3章大纲存在后 Context 显示 reader known 与苏弦 known，不泄露其它隐藏 Truth。无需再续写整章，除非 Context 结果不明确。

## 阶段 162：文档、全量验证与提交

状态：`pending`

保存脱敏数据、指标和局限，不提交完整模型响应/源文/凭证。运行 Import/Context/assets/eval、全量/vet/race/格式/链接/敏感信息门禁后中文提交。
