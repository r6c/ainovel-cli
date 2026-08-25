# ainovel-cli 当前演进计划

## 当前状态

- 总体状态：`in_progress`
- 当前里程碑：O——真实创作验收问题收敛
- 当前阶段：阶段 124——Markdown 提交前格式门禁
- 基线提交：`ade8108 测试：加固 Linux 与无头环境兼容性`
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

状态：`in_progress`

只拦截确定性 Markdown 残留，发生在 PendingCommit 创建前；保留 `rules.Lint` 的其他 warning 语义。

## 阶段 125：结构化篇幅目标

状态：`pending`

先盘点 UserRules/Chapter Contract 既有字段与字符计数口径，再确定最小模型；不得用脆弱正则替代已有 Normalizer。

## 阶段 126：真实 sss 回归决策

状态：`pending`

O1—O3 全部通过自动化后，再根据预算决定是否复用同一单章需求做一次真实回归；未经明确成本判断不调用 Provider。
