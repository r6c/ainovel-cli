# ainovel-cli 稳定上下文

本文件面向维护者和代码 Agent，记录当前稳定术语、事实边界与代码入口。它不是任务计划；历史决策过程见 [`docs/history/plans/`](docs/history/plans/)。

## 1. 产品与架构定位

`ainovel-cli` 是本地、文件系统驱动、可恢复的 AI 小说创作运行时。

核心纪律：

```text
模型负责开放创作与边界清晰的语义判断；
代码负责状态、约束、事务、恢复和验证。
```

- Engine 从 Store 读取事实，经纯 Route 决定下一 Worker。
- Architect / Writer / Editor 是三个自主创作 Worker。
- Arbiter 只处理启动选择、用户干预、失败/僵局等开放语义裁定。
- 章节按叙事依赖串行提交，不并行生成相邻正文。
- 文件 Store 是事实层；CLI/TUI 是 Adapter，不是事实源。

稳定架构详见 [`docs/architecture.md`](docs/architecture.md)；发布前自动化与人工验收见 [`docs/release-acceptance.md`](docs/release-acceptance.md)。

## 2. 事实源与投影

### 2.1 章节正文与结构化事实

- `chapters/*.md`：用户可编辑的章节正文工作区。
- `meta/chapter_records/*.json`：最近一次已接纳正文及其完整 `ChapterFacts`；是判断外部正文修订的基线，也是派生事实全量重建的输入。
- `ChapterFacts`：单章对应的结构化事实增量，包括时间线、伏笔、知识、关系、角色状态等。

关键类型：

```text
internal/domain/revision.go
  ChapterFacts
  ChapterRecord
  RevisionAnalysis
```

正文与记录哈希不一致时进入 Revision 管线；不要绕过接纳流程直接把派生状态当成新的章节事实。

### 2.2 当前投影

以下是可从 ChapterRecords 全量重建的当前状态，不是第二事实源：

```text
foreshadow_ledger.json / .md
knowledge_state.json / .md
timeline / relationships / state changes / cast 等
```

- JSON 是运行时当前投影。
- Markdown sidecar 只是人类可读视图。
- `revision.Projector` 负责按章重建投影。
- `revision.ValidateRecordSet` 只验证整组记录能否重建，不写 Store。

关键入口：

```text
internal/revision/projector.go
internal/store/world.go
```

## 3. Knowledge 认知模型

系统区分四个概念：

```text
Author Truth       作者认定的客观真相
Character Known    角色已明确获知 Truth
Reader Known       正文已向读者完整揭示 Truth
Character Belief   角色对该 Truth 持有的稳定错误认知
```

它们不能互相替代：

```text
Truth ≠ KnownBy ≠ ReaderRevealedAt ≠ BelievedBy
```

### 3.1 类型与动作

类型位于 `internal/domain/tracking.go`：

- `KnowledgeEntry`：某项 Truth 的当前投影。
- `KnowledgeHolder`：角色首次获知 Truth 的章节。
- `KnowledgeBelief`：错误信念的内容、形成章和纠正章。
- `KnowledgeUpdate`：章节知识增量。

动作：

```text
establish          建立作者客观真相
believe            角色形成稳定错误信念
learn              角色获知客观 Truth，并纠正其活跃错误信念
reveal_to_reader   正文向读者完整揭示 Truth
```

正式生命周期唯一实现在：

```text
internal/domain/knowledge.go
  ApplyKnowledgeUpdates
```

Store、Commit、Projector 不得重新复制 Knowledge 生命周期 switch。

### 3.2 Writer 的净化视图

Writer 不直接消费完整 `KnowledgeEntry`，而消费：

```text
episodic_memory.knowledge_boundaries
```

构造位置：

```text
internal/tools/novel_context.go
  selectKnowledgeForCurrentOutline  // 读取 Store 与匹配当前大纲角色

internal/tools/context_knowledge_policy.go
  selectKnowledgeBoundaries         // 纯选择、时间过滤、净化与 8 条上限

```

约束：

- 当前角色或读者已知时才可输出 Truth。
- 角色可以只看到自己的活跃错误信念，此时必须隐藏 Truth。
- 不输出当前章或未来才形成、获知、揭示、纠正的信息。
- 最多 8 条，并参与上下文预算裁剪。

修改该视图时必须使用 JSON 结构级测试证明隐藏 Truth 没有泄漏。

## 4. Foreshadow 伏笔模型

类型位于 `internal/domain/review.go`，正式生命周期位于：

```text
internal/domain/foreshadow.go
  ApplyForeshadowUpdates
```

动作：

```text
plant
advance
reinforce
partial_payoff
resolve
```

核心语义：

- `PlantedAt` 是首次种植章。
- `LastAdvancedAt` 是最近推进、强化或部分兑现章。
- `ResolvedAt` 是完整回收章。
- `resolved` 是终态；不能再 advance/reinforce/partial_payoff。
- 同章冻结 payload 重放必须幂等。

Rewrite 的 `RestoreOwnPlants` 是候选 ChapterRecord 构造策略，不属于生命周期函数本身。

## 5. Chapter Commit Saga

普通提交与 Rewrite 共用 `PendingCommit`：

```text
冻结完整意图
→ 写正文、ChapterRecord 和状态投影
→ 推进 Progress
→ 写 checkpoint
→ 清除中间态与 PendingCommit
```

关键代码：

```text
internal/domain/commit.go
internal/tools/commit_chapter.go
internal/tools/commit_validation.go
internal/store/signals.go
```

### 5.1 两层校验

首次冻结前：

```text
纯载荷校验
＋ 当前 Store 投影语义校验
```

恢复时：

```text
密封完整性校验
＋ 纯载荷校验
＋ 按 Stage 幂等重放
```

恢复不能根据已部分应用后的当前投影重新裁决冻结意图。

### 5.2 密封 v2

新建 `PendingCommit` 保存三个 SHA-256：

- `PayloadDigest`：compact JSON payload。
- `DraftDigest`：冻结正文 UTF-8。
- `IntentDigest`：Chapter、Rewrite、RewriteMode、Origin。

`Stage`、`Output`、`Result` 和时间戳是 Saga 可变字段，不纳入摘要。历史 v1 工件继续兼容，但不能携带未密封的 imported origin。

完整性失败：

- 返回 `errs.ErrPendingCommitIntegrity`。
- 保留 `meta/pending_commit.json`。
- 不自动重签、不删除、不接受被改写内容。

旧 `started/state_applied` 工件只在纯载荷通过后升级 v2 密封；`progress_marked/signal_saved` 只做后段收尾。

### 5.3 真正可静默终止

`phase=complete` 不是充分条件。Headless/TUI 只有在目录租约可取得、无 PendingCommit、无 PendingRevision、无未完成 Import 且无外部正文修改时，才显示干净完成态。`Host.Resume` 会先用现有 Commit Saga 同步收尾 PendingCommit，再重新判定终态；外部修订与 Import 只给 `/sync`、`/import` 指引，不自动调用模型。

## 6. 正文来源与自动写权限

`ChapterRecord.Origin` 区分 `generated / imported / user`。Editor 可以在 chapter/global/arc Review 中记录任何来源章节的问题和完整 `affected_chapters`，但自动返工队列只允许 `origin=generated`（或旧兼容缺记录）的章节。`imported/user` 是作者原文，Writer 不得自动覆盖；需要修改时由作者编辑正文后执行 `/sync`。

`SaveReviewTool` 在生成控制状态时过滤非 generated 章节；`CommitChapterTool` 在冻结 Rewrite PendingCommit 前再次检查 provenance，防止升级前遗留脏返工队列绕过权限；Start/Resume 的 `upgradeProject` 修复接缝还会幂等清理历史残留的 imported/user 返工项，避免 Router 反复派 Writer。若 Review 只指向 imported/user，评审工件照常保存，但控制 Flow 回到 writing，不能制造空返工死循环。

## 7. Revision 与 Projector

Revision 用于接纳外部或用户正文修改，并重建后续派生事实。

关键入口：

```text
internal/revision/service.go
internal/revision/projector.go
internal/revision/migration.go
```

- `ValidateRecordSet`：纯验证候选记录集。
- `Projector.Apply`：全量写入派生投影。
- Rewrite 创建 PendingCommit 前，必须先验证候选记录集不会破坏后续事实引用。
- 用户删除 reveal、learn、belief 等动作时允许由全量重建回退对应投影；不要强制恢复用户已删除的认知事实。

## 8. Import 工作区

Import 是独立的语义编译工作区：

```text
ingest → segment → analyze → synthesize → publish
```

文档：[`docs/import-pipeline.md`](docs/import-pipeline.md)

关键入口：

```text
internal/host/imp/analyze.go
  validateImportedFactSequence
  validateWorkspaceFacts

internal/host/imp/publish.go
  importedChapterFacts
```

约束：

- Import→ChapterFacts 映射只有一份，验证与正式发布共用。
- 候选分析、截断打捞、综合和发布前均重放正式 ChapterRecord 规则。
- 非法事实定位首错章，失效该章及后续分析与综合工件。
- `NextAction` 通过现有状态推导自然回到 Analyze，不新增验证 Stage。
- 发布门禁必须在正式 Foundation、Hold 或章节写入之前完成。

## 9. 规则所有权

| 规则 | 正式位置 |
|---|---|
| Knowledge 生命周期 | `internal/domain/knowledge.go` |
| Foreshadow 生命周期 | `internal/domain/foreshadow.go` |
| ChapterFacts 字段纪律 | `internal/chapterfacts/facts.go` |
| 全量派生验证/重建 | `internal/revision/projector.go` |
| Commit Saga 与密封 | `internal/tools/commit_chapter.go` |
| Commit 当前状态前置条件 | `internal/tools/commit_validation.go` |
| Import 全书事实门禁 | `internal/host/imp/analyze.go` |
| Writer Knowledge 净化 | `internal/tools/novel_context.go` |

动作枚举可在 Schema、Prompt、Import 局部反馈中出现，但正式跨章生命周期规则不得复制到这些 Adapter。

## 10. 修改纪律

### 新增或修改 ChapterFacts 字段

同步检查：

1. `domain.ChapterFacts`
2. `chapterfacts.Properties` 与 `Validate`
3. Commit strict schema
4. Revision strict schema
5. Import DTO/schema/publish 映射
6. ChapterRecord/旧 JSON 兼容
7. Projector 与 Rewrite
8. Writer/Editor/Revision/Import Prompt
9. Host 模拟严格响应夹具
10. Import 分析缓存版本

### 修改生命周期动作

必须覆盖：

```text
Domain Apply
WorldStore 增量
Commit Pending 前试运行
PendingCommit started 重放
Projector 全量重建
Rewrite 候选记录集
Import 全书门禁
Context/Diagnostics 消费
```

不要先在 Store、Commit、Projector 各写一套 switch。

### 修改 PendingCommit

必须区分：

- 不可变冻结意图
- Saga 正常可变 Stage/Output/Result
- legacy 前段与后段兼容
- 完整性错误时的零副作用与工件保留

## 11. 确定性 Prose Lint

`internal/rules.Lint` 是始终执行的产品底线检查；它只生成 `Violation` 事实，不自行裁决。正文接纳 adapter 按 provenance 应用政策：模型生成正文在 PendingCommit 前拒绝 `markdown_residue`，Import 原文逐字保留并只记录同一事实；两者都不新增 verdict 或 Route。

当前内置规则包括：

```text
markdown_residue
non_cjk_fragments
duplicate_paragraph
```

`duplicate_paragraph` 只检测同章内 TrimSpace 后完全相同、至少 24 个 Unicode 字符的非标题正文行。它不做模糊相似度、跨章累计或意图判断；Editor 根据 `rule_violations` 判断是有意复沓还是复制退化。Target 最多保留前 48 字加省略号。

Commit 和 Revision Projector 都必须复用同一个 `rules.Lint`，不要建立第二条 Prose Lint 管线。

### 10.1 AI 味语义判据

`assets/references/anti-ai-tone.md` 是 Writer/Editor 的语义参考，不是文本来源检测器。不得输出 AI 概率，也不得把句长、段长、问句、比喻、句内排比、冒号或“不是 A 而是 B”本身当作 AI 来源证据。

正式优先级：目标风格 → 叙事功能 → 信息守恒 → 最小改动。只有能定位空转提示、同义重复、机械同构、段首零回指或无机制支撑的职业人格喻体时才建议修改；一般审美问题继续映射现有七维并引用原文，不升级为 Commit 硬门禁。

校准证据位于 `evals/anti-ai-tone/`：16 条自建匿名网文最小对、独立金标、三轮盲评和 Writer 三重复 A/B。外部 `lieflat-less-ai-tone` 仅作为候选假设来源；未安装 Skill、未运行其 Python 脚本、未采用其不可复核统计阈值，也不新增第二条去 AI 味流程。

## 12. Knowledge 诊断与导出边界

作者侧 `diag.Analyze` 聚合：Truth 数、角色知情关系数、读者已知 Truth 数和活跃错误信念数。本地 `/diag` 可以显示长期未纠正 belief 的 Knowledge ID、角色与形成章，但不复制 Truth 或 Belief 正文。

可分享的 `meta/diag-export.md` 只输出上述聚合数字；创作类 Finding 不进入该文件。TXT/EPUB 是读者成品，禁止包含 `knowledge_state`、作者 Truth、角色错误信念或任何内部认知元数据。

Diagnostics 只是当前投影的只读 Adapter，不得修改 Knowledge、自动生成 `learn` 或成为新事实源。

## 13. 目标平台 Rubric 试点

目标平台属于用户意图，持久化在 `meta/user_rules.json` 的 `structured.platform`；当前只支持显式 `fanqie`。含糊的“免费阅读平台/移动端平台”不得猜测，未指定时不加载任何平台参考。

番茄试点资源位于 `assets/references/platforms/fanqie.md`，支持 `~/.ainovel/style/platforms/fanqie.md` 与本书 `style/platforms/fanqie.md` 追加覆盖。只有 `platform=fanqie` 时，`novel_context` 才把它作为 `reference_pack.references.platform_rubric` 注入 Writer、Editor 和 Architect；该资源参与既有预算裁剪。

Rubric 区分官方可核事实与 ainovel-cli 产品软评价，映射现有七维，不新增平台评分状态、Verdict 或 Route。官方公开资料未提供黄金三章字数、爽点数量或推荐算法阈值，禁止编造这些硬指标。用户偏好、章节合同与人物逻辑优先。

## 14. 单章篇幅目标

用户规则快照 v4 支持：

```text
structured.chapter_target_chars
```

仅当用户明确给出单章/每章正文的单一目标值时，由 UserRules Normalizer 以 `chapter_target_action=set` 提升；未声明为 `keep`，明确取消限制为 `clear`。区间、全书总字数和“短一点”等含糊要求继续留在 `preferences/uncertain`，不得用正则猜测或换算。目标上限为 1,000,000 字符。

Architect、Writer 和 Editor 通过现有 `working_memory.user_rules` 消费同一字段。Commit 使用 `domain.WordCount` 的现有 Unicode 字符口径，只在正文超过目标 120% 时于 PendingCommit 创建前拒绝；不设置机械下限，偏短章节仍由 Editor 在 pacing 维度判断，避免为达标注水。普通提交与 Rewrite 必须使用同一上限规则。

## 15. 阶段化冷启动共创

启动模式仍只有 quick 与 cocreate，没有第三种 interview 模式。冷启动 cocreate 由 `startup.CoCreateSession` 确定性维护：

```text
core → customization → title → confirmation → ready
```

模型通过 `<stage>` 回报下一轮当前阶段，代码只允许保持或前进一格；缺失、非法、跳级或回退都保持当前阶段。只有 ready、模型 ready=true，且 Draft 同时包含 `## 核心定位`、`## 深度定制`、`## 书名与简介`、`## 规划确认` 时，Ctrl+S 才放行。

阶段状态和半成品 Draft 只存在于当前内存会话与诊断性的 `meta/sessions/cocreate.jsonl`，不是正式事实源。最终 BuildPrompt 仍作为一段用户创作需求进入现有 `StartPrepared` / PlanStart / Architect 流程，不直接写 Book、Premise 或 Foundation。

运行中阶段共创使用 `NewStageCoCreateSession`，继续按“有后续方向 Draft 即可应用”的原协议，不套冷启动访谈阶段。

冷启动和运行中阶段共创的流式模型调用必须经过同一个 `UsageTracker`，归入 `thinking` 角色；否则书级成本、Token、缓存和预算会失明。每轮共创请求前复用 `BudgetSentinel.Refuse`，上一轮已越线时不得继续调用。瞬时流式失败由 Session 保持原阶段和 Draft，用户可重试；不得在失败时推进阶段或清空累计草稿。

## 16. 本地拆文方法画像

独立命令：

```text
ainovel-cli deconstruct <本地语料目录>
```

它只读取本地 `.txt/.md/.markdown`，复用现有 `internal/host/sim` 与 `SimulationProfile`，写入当前配置 `OutputDir/meta/simulation_profile.json`；不启动 Engine。TUI `/simulate` 继续委托 `Host.Simulate()` 读取 cwd/simulate，独立命令通过 `Host.SimulateDir()` 传显式目录。拆文模型调用必须经 `UsageTracker` 归入 `simulation` agent，并受现有预算哨兵约束；不得绕过用量和预算系统。

`SimulationProfile` 已是单篇报告、聚合方法画像、SHA 增量和 Agent Context 的唯一协议，不得另建 Benchmark DTO/Pipeline。内部 Simulation 命名为兼容保留；用户和 Prompt 统一称“拆文方法画像”。

安全边界：不抓排行榜、网页或 URL；不输出连续原文表达、签名短语、人物、地名或专有设定；不得模仿具体作者。

扫榜功能已从产品路线移除。为保持 Linux、服务器、NAS 和无头环境可移植性，不引入 Chrome/CDP、浏览器登录态、平台爬虫、反爬或番茄/起点/晋江页面适配。需要分析的资料必须由用户以本地文本主动提供。

## 17. Linux 与无头环境边界

顶层 `--help/-h/help` 和 `deconstruct --help` 必须在配置、首次引导、TTY、模型和 Host 初始化之前返回；它们可用于 Linux/Docker 无配置健康检查。

Linux amd64/arm64 发布目标均使用 `CGO_ENABLED=0`。CI 除 Ubuntu/Windows 测试外，显式跨编译两个 Linux 架构，并在 Ubuntu 原生执行帮助命令；Docker 入口以无网络、无挂载、无 TTY 方式冒烟。

桌面通知是 best-effort Adapter：Linux 缺少 `notify-send` 时只降级日志，不能影响 Engine、Route 或恢复流程。生产代码不得引入 Chrome/CDP、浏览器登录态、GUI 动态库或绝对临时目录依赖。

## 17.1 发布候选文档

发布候选的用户可读说明位于 `docs/release-notes.md`，升级兼容边界位于 `docs/upgrade.md`，变更记录位于 `CHANGELOG.md`。发布前逐项验收继续使用 `docs/release-acceptance.md`。Import 认知动作三轮 baseline/calibrated A/B 已完成；当前 calibrated 是经评测保留的折中版本。真实 Architect 扩弧后的第 3 章 Context 端到端验收仍未完成，必须与代码层 Context 回归分开记录。Import Analyze Prompt 版本现为 `analyze-v2`；每次语义 Prompt 修改必须递增该版本。

## 18. 常用验证

```bash
go test ./... -timeout=5m
go vet ./...

go test -race \
  ./internal/store \
  ./internal/tools \
  ./internal/revision \
  ./internal/host/imp \
  -timeout=10m

git diff --check
```

复杂改动继续使用根目录 `task_plan.md`、`findings.md`、`progress.md`；完成后将过程归档到 `docs/history/plans/`，根文件只保留当前工作记忆。
