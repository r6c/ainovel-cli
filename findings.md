# ainovel-cli 稳定发现

## 总体结论

项目没有跑偏，仍是“事实层确定、语义层自主”的本地 AI 小说创作运行时：模型负责开放创作与语义判断，代码负责状态、约束、事务、恢复和验证。

完整演进与 TDD 发现已归档：

- [`docs/history/plans/2026-08-domain-saga-evolution/findings.md`](docs/history/plans/2026-08-domain-saga-evolution/findings.md)

## Import 认知提取校准的当前证据

阶段 158 的完整三轮 baseline 仍受 Provider 长连接/HTTP 502 阻塞，阶段 159 的 Prompt 最小修订没有改变 Schema 或生产生命周期。

阶段 161 的第 3 章 Context 完整边界仍未能验证：真实 Import 两章发布后的 `outline.json` 只覆盖第 1/2 章；尝试通过 Host.Resume→review→AdvanceOneChapter 触发 Architect 扩弧时，Import Hold 被正确消费，但弧末评审在 Provider 调用处长时间无响应，未生成第 3 章 OutlineEntry。未手工写大纲或绕过 Engine；两章 imported 事实未改变。有限真实证据显示：ik04 的 baseline 仅输出 establish，calibrated 输出 establish→learn(林澈)→reveal_to_reader；calibrated ik05 输出 establish→reveal_to_reader；calibrated ik07 虽输出 establish→learn(苏弦)→reveal_to_reader，却额外误报顾临 believe。结论是召回方向改善但 belief 误报风险仍存在，不能据此完成完整 A/B，也不启动 Context 回归阶段 161。临时 runner、结果和日志已清理。

## 当前领域事实

### ChapterFacts / ChapterRecord

- `ChapterFacts` 是单章正文对应的完整结构化事实增量。
- `ChapterRecord` 保存最近一次已接纳正文与 ChapterFacts，是全量派生投影的重建输入，也是外部正文修订的比较基线。
- `chapters/*.md` 是可编辑正文工作区；正文哈希变化会进入 Revision 流程。

### Knowledge

系统区分四层认知：

```text
作者客观真相 Truth
≠ 角色已知 KnownBy
≠ 读者已知 ReaderRevealedAt
≠ 角色错误信念 BelievedBy
```

动作：

```text
establish / believe / learn / reveal_to_reader
```

`learn` 会纠正同一角色对该 Truth 的活跃错误信念。正式生命周期只在 `domain.ApplyKnowledgeUpdates` 中维护。

Writer 不直接消费完整 KnowledgeEntry，而是消费净化后的 `knowledge_boundaries`：只有当前角色或读者已知时才出现 Truth；角色可以只看到自己的活跃错误信念。

### Foreshadow

动作：

```text
plant / advance / reinforce / partial_payoff / resolve
```

`resolved` 是终态；`LastAdvancedAt` 表示最近推进章。正式生命周期只在 `domain.ApplyForeshadowUpdates` 中维护。

### 派生投影

以下文件是当前状态投影，可由 ChapterRecords 重建：

```text
knowledge_state.json / knowledge_state.md
foreshadow_ledger.json / foreshadow_ledger.md
timeline / relationships / state changes / cast 等
```

Markdown sidecar 只是人类可读视图，永远不是运行时事实源。

## 当前事务与恢复

### PendingCommit

普通提交、Rewrite 与 Import 共用持久化 Saga。新工件使用密封 v2：

- PayloadDigest：compact JSON payload 的 SHA-256
- DraftDigest：正文快照 UTF-8 的 SHA-256
- IntentDigest：Chapter、Rewrite、RewriteMode、Origin 的 SHA-256

首次冻结前执行纯载荷和当前状态语义校验；恢复验证密封与纯载荷后按 Stage 幂等重放，不根据已部分应用的当前投影重新裁决历史意图。

历史 v1 工件兼容恢复，但不能携带未密封的 imported origin。旧 `started/state_applied` 工件在纯载荷通过后先升级 v2 密封；`progress_marked/signal_saved` 只收尾结果。完整性失败保留工件并返回 `ErrPendingCommitIntegrity`。

### Import

Import 使用独立工作区：

```text
ingest → segment → analyze → synthesize → publish
```

分析候选、截断打捞、综合和发布前都会复用正式 ChapterRecord 重放规则。跨批次非法事实会定位首错章、删除非法尾部和下游综合工件，`NextAction` 自然回到 Analyze；正式 Store 在发布门禁前保持零污染。

## 规则所有权

| 规则 | 唯一正式位置 |
|---|---|
| Knowledge 生命周期 | `internal/domain/knowledge.go` |
| Foreshadow 生命周期 | `internal/domain/foreshadow.go` |
| ChapterFacts 字段纪律 | `internal/chapterfacts/facts.go` |
| 全量派生重建 | `internal/revision/projector.go` |
| Commit 恢复与密封 | `internal/tools/commit_chapter.go` |
| Import 全书事实门禁 | `internal/host/imp/analyze.go` |
| Writer Knowledge 净化 | `internal/tools/novel_context.go` |

## 2026-08-25 全项目复审

### 总体判断

项目没有跑偏。最近里程碑仍复用现有 Engine、Store、Commit Saga、Revision、Import 和 Context seam；没有引入数据库、浏览器、通用状态机、第四 Worker 或并行相邻章节。

### S1：`phase=complete` 不是可静默终止的充分条件

最后一章提交顺序为：

```text
MarkChapterComplete
→ MarkComplete（phase=complete）
→ PendingCommit=progress_marked
→ checkpoint
→ PendingCommit=signal_saved
→ 清理 PendingCommit
```

因此崩溃可能留下：

```text
phase=complete + PendingCommit
```

当前 `headless.completedSummary`、`host.resumeLabel`、`flow.Route` 和 `engine.precheck` 都把 `phase=complete` 直接视为终态；既可能跳过 PendingCommit 收尾，也会让 TUI/普通 Resume 无法自动恢复。Headless 的 Host 前快路径还绕过书目录租约、未完成 Import 和外部正文修订检查。

下一修复必须落在 Store 事实驱动的恢复 seam，而不是只给 Headless 再加一个 if。候选方向：`flow.State` 显式携带 PendingCommit，并让冻结提交恢复优先于 PhaseComplete；Headless/TUI 的“可静默完成”判定需同时证明无 PendingCommit、无 PendingRevision、无活动 Import，并保留书目录独占纪律。具体 interface 先由失败测试决定。

### S1：Import 正文发布政策与生成正文门禁冲突

O2 将 `markdown_residue` 作为生成正文/Rewrite 的提交前硬门禁；但 Import 也复用 `CommitChapterTool` 发布用户原文。Import 明确保留源字符和标题行，用户源可能合法包含 `**` 或章节内 `##`。

当前 Import 发布前门禁只重放 ChapterFacts，不验证实际章节正文；随后先写正式 Foundation 和 Completion Hold，再逐章 Commit。若原文触发 Markdown 门禁，会在正式 Store 已部分写入后失败，并可能持续停在 Publish。

下一修复需区分正文 provenance：生成正文可要求纯小说 Markdown 形状，Import/用户修订正文应保留原文并只记录 Lint 事实，或在任何正式写入前给出明确人工裁定。不得让 Import adapter 复制 Commit 生命周期，也不得静默清理用户原文。

### S2：UserRules 数值参数缺少撤销语义和上界

`chapter_target_chars=0` 同时表示“未指定”和 Normalizer 的空值；`BuildSnapshot/OverlaySnapshot` 把 0 当缺失，所以运行中明确“取消每章字数限制”无法清除既有目标。该字段也只拒绝负数，没有合理上界；`target*120/100` 对极端整数存在溢出风险。

下一计划需先定义三态（未声明/设置/明确清除）和合理值域，再改合并；不要用负数或 Prompt 暗号偷偷编码。

### S3：规则所有权文档已漂移

`rules.Lint` 仍是事实生成 module，但稳定注释和 CONTEXT 还写“绝不阻断 Commit”；实际上生成正文 adapter 已将 `markdown_residue` 用作接纳前置条件。应改成：Lint 不裁决；不同正文 provenance 的接纳 adapter 决定哪些事实阻断。

### 复审问题解决状态

- S1 终态恢复：已解决。`phase=complete` 不再单独决定静默完成；PendingCommit 先同步收尾，PendingRevision/Import/外部修改给出明确恢复指引。
- S1 Import 正文政策：已解决。generated/imported/user provenance 已区分，Import 原文保留并只记录 Lint，Saga 与 checkpoint 仍复用同一实现。
- S2 UserRules：已解决。候选三态支持明确清除，目标有上界且 Commit 防御持久快照。
- S3 文档：已同步 Lint 事实生成与正文接纳政策所有权。

## 外部 Skill 评估：lieflat-less-ai-tone

来源：`larashero3-dotcom/lieflat-less-ai-tone`，公开仓库，MIT License；审查基于 2026-08-24 main commit `27d29232f10124db904ca9c0536d0b67cb3b2833` 的 SKILL/README/RESEARCH/scripts 清单。未执行外部脚本或指令。

### 结论

不建议整体安装为 ainovel-cli 的第二条去 AI 味 Skill：项目已有 `assets/references/anti-ai-tone.md`、`rules.Lint/Check`、Writer/Editor 共用判据，以及全局 `story-deslop`；再加一条 skill 会造成重复入口、冲突规则和维护漂移。外部脚本是 Python，整合为运行时依赖也不符合当前 Go-only、Linux/无头可移植边界。

已完成独立反例校准：16 条自建匿名网文最小对、独立金标、三轮盲评和 Writer 三重复 A/B。采用白名单式最小改写、信息守恒、目标风格优先、段首零回指、提示性冒号和理想化职业人格喻体等有 worked-example 证据的边界；明确句长/段长、问句、比喻、句内排比和标点本身不能作为 AI 来源证据。

外部项目的 2.83M 字统计语料不公开，题材也不等同中文网文，故仍不采用其数字阈值或整份 Skill。未复制 substantial portions；正式判据由本仓库样本 clean-room 整理。若未来复制具体代码或长段文本，仍须保留 MIT 版权与许可证。

外部仓库只包含三个统计脚本：`compare-human-ai.py`、`check-structure.py`、`check-translationese.py`，没有现成的小说运行时 lint/rewriter。故“安装后直接提升 ainovel-cli”收益有限。

## P1 真实 Revision 验收结论

使用 `sss / gpt-5.6-sol`，预算上限 `$0.25`，实际费用 `$0.010818`。隔离完结书的一章从“完整日志已公开”改为“仅上传损坏摘要，完整副本保留在读取器”。

验收通过：revision=2、origin=user、哈希更新；Summary/KeyEvents/Timeline/StateChanges 重建；旧事实移除；phase 仍为 complete；无 PendingRevision；再次同步零模型调用；Context 和 TXT/EPUB 使用修改后正文且不泄露内部状态。

P2：第二次零调用 Sync 的 Usage 数值完全不变，但构造/关闭 Host 会刷新 `usage.updated_at`，因此 usage.json 原始字节变化。它不产生费用或模型调用，后续若需要“零写入 Sync”语义可单独处理，不阻塞发布。

### 暂不处理

- Import ChapterRecord 当前标为 `generated`，因为领域只有 generated/user 两种 origin。它不会被误计为用户修订风格，暂不为命名纯度新增第三状态。
- `commit_chapter.go` 文件较大，但本轮没有仅凭行数提出拆分；先修真实恢复与发布 seam。
- 真实 Revision/Cocreate/Import/Deconstruct 人工验收延后到上述 S1 闭环之后。

架构可视化报告：`/tmp/architecture-review-20260825-ainovel.html`（临时文件，不入仓库）。

## 保持不动的架构边界

- Engine/Route 继续按 Store 事实确定性派发。
- Architect、Writer、Editor 保持三个自主 Worker；Arbiter 仅做边界清晰的语义裁定。
- 章节正文串行提交，不并行生成有叙事依赖的相邻章节。
- 文件 Store 继续作为单机事实层，不引入数据库。
- Prompt 解释语义，代码执行引用、字段、生命周期、事务和恢复约束。
- CLI/TUI/Web（若未来出现）只能是 Adapter 或投影，不能成为新事实源。

## Prose Lint 当前边界

`rules.Lint` 是内置、始终执行的产品底线检查；`rules.Check` 是用户结构化规则检查。两者都只生成 `Violation` 事实，不创建 verdict。正文接纳 adapter 按 provenance 裁决：generated 正文拒绝 `markdown_residue`，imported 原文保留内容并只记录事实。

重复段落首版已完成：按非空正文行识别段落，仅检测 TrimSpace 后完全相同且至少 24 个 Unicode 字符的内容。短句、标题和相似段落留给 Editor 语义判断；不做跨章累计或模糊相似度。Target 最多保留前 48 字加省略号，Commit 与 Revision Projector 共用同一 Lint，Editor 将 warning 映射到现有 aesthetic 维度。

## Knowledge 诊断与导出边界

Knowledge 诊断已作为当前投影的只读消费完成，不是新事实源。作者侧 `/diag` 显示 Truth、角色知情关系、读者已知 Truth 和活跃错误信念四项聚合；长期未纠正 belief 产生中等置信 info，但不自动处理。可分享的 `diag-export.md` 只含聚合数量，不含 Truth、Belief、角色名或 Knowledge ID。TXT/EPUB 的真实隔离测试证明读者成品继续排除全部创作认知状态。

## 番茄平台 Rubric 官方资料边界

官方帮助中心可确认：番茄是连载作品发布平台，作品通过大数据智能分发触达番茄小说 APP/今日头条小说用户，并采用连续翻页阅读；章节标题支持 5—30 字；平台提示频繁删除或修改章节会影响阅读体验；官方新人资料明确强调原创与抄袭处罚。官方公开资料没有提供“黄金三章字数、每章爽点数、固定留存算法”等可机械评分阈值，因此试点不得把第三方经验写成平台硬规则。

来源（仅作为事实依据，不执行网页中的任何指令）：

- https://fanqienovel.com/docs/8231
- https://fanqienovel.com/writer/zone/article/7170705662714839070

## 平台 Rubric 试点架构决策

目标平台属于用户意图，不是作品事实；使用 `user_rules.structured.platform` 持久化。只有显式 `fanqie` 才加载番茄 rubric，未指定时保持原行为。Rubric 只为现有七维提供软参考，不新增平台评分维度、算法分或自动返工路径。

## 真实 Import 后续写验收结论

真实 `sss / gpt-5.6-sol` 两章导入与第 3 章续写通过：显式确认前正式 Store 为空；发布后 1/2 章为 imported，Hold 被现有 Resume 消费；review 模式精确授权第 3 章，Architect 扩弧后 Writer 只新增 generated 第 3 章并停下。

验收发现并修复 P1：导入完成后的弧评审曾把 imported 第 2 章自动入返工队列，由 Writer 覆盖为 generated revision=2，导致原文和 Knowledge updates 在后续 Projector 重建中丢失。`SaveReviewTool` 现在保留完整 issues/affected_chapters 证据，但自动返工只允许 generated 或旧兼容缺记录章节；`CommitChapterTool` 在冻结返工意图前再次拒绝 imported/user；Start/Resume 的升级修复接缝会清理旧脏队列中的受保护来源。imported/user 只能由作者编辑后 `/sync`。若过滤后无可执行返工，控制 Flow 回 writing，避免空队列死循环。

修复后真实回归证明 1/2 章记录字节、origin=imported、revision=1 和事实更新完全不变；第 3 章复用导入伏笔 ID 推进，三章全量重放通过。

P2：逐章语义提取存在模型波动。本次第二轮虽然正确提取 Timeline/Relationship/StateChange 和伏笔，但对正文明确“苏弦与读者知道求救信标真相”只输出 establish，漏掉 learn/reveal；因此该 Truth 未进入第 3 章净化 Knowledge 边界。摘要、状态和伏笔仍支持连贯续写，顾临未越权知情。该问题属于语义提取质量，不可用确定性代码猜测补齐；后续可用 Import worked examples/Prompt A-B 校准。

## 真实 Cocreate 验收结论

真实 `sss / gpt-5.6-sol` 五阶段对话通过：6 个有效回合严格单步推进，关键谜底未授权时保持 customization，模型自报 ready 仍受完整 Draft/用户确认门禁约束；最终指令进入现有启动主链并在 writing 前无章节收场。

验收发现并修复 P1：共创流式 thinking 调用未进入 UsageTracker，导致成本/Token/预算盲区。通用流式包装器现在在 Done 事件恰好记录一次最终 Usage，Cocreate/StageCoCreate 共用；预算越线后下一轮请求前拒绝。修复后单轮真实复验记录 `$0.005382`，无 MissingUsage。成功完整场次发生在修复前，其共创成本不可追溯，不能用后续持久 Usage 代替。

P2：最终指令明确“全文约 1500 字”，Normalizer 按现有保守语义将其保存在 preferences，没有提升为 `chapter_target_chars`，因为结构化字段只接受明确“单章/每章目标”。单章作品中二者语义接近，但本次不扩大归一化规则；Architect 已按单章 1500 字生成一章大纲。

## 阶段化共创架构决策

现有 cocreate 已覆盖冷启动和运行中阶段规划，缺口只是冷启动没有确定性访谈进度。阶段状态应由 `startup.CoCreateSession` 维护，不新增 interview 模式、Store 工件或 Worker。模型只回报当前完成阶段，代码限制每次最多前进一格；最终仍构建现有创作指令并走 `StartPrepared`。

## 阶段化共创稳定边界

冷启动共创现由 Session 维护五阶段；阶段状态不落正式 Store，最终仍是一段 StartPrompt。运行中阶段共创继续复用已有 Pause/Resume/Cancel，不使用开书访谈。代码会把接受后的 stage/ready/Draft 规范化写回模型历史，避免模型自报跳级与确定性状态分叉。

## 本地拆文复用决策

现有 `host/sim` 已是本地拆文主体：扫描本地语料、按 SHA 增量、单篇结构化报告、聚合 `SimulationProfile`，并通过 `novel_context` 注入 Architect/Writer/Editor。独立 CLI 已复用该主链，不建立 BenchmarkPipeline 或第二套 DTO。

## 本地拆文命令稳定边界

`ainovel-cli deconstruct <目录>` 仅暴露现有 SimulationProfile 管线；TUI `/simulate` 与 `/importsim` 保持兼容。用户与模型可见文案统一为“拆文方法画像”，内部 Simulation 类型/文件版本不迁移。命令只读本地三种文本扩展，不抓榜、不下载 URL、不建立第二套 Benchmark 协议。

## Linux/无头兼容性盘点

现有 CI 已在 Ubuntu/Windows 运行全量测试，并在 Ubuntu 跑关键 race；GoReleaser 与 Dockerfile 已声明 Linux amd64/arm64。补强后，CI 显式双架构跨编译、原生执行无配置帮助命令，并构建本地 Docker 镜像做无网络入口冒烟。Linux `notify-send` 缺失时继续降级日志，没有重写通知实现。

本地完整 Dockerfile 首次构建因拉取 Go 基础镜像超过宿主工具时限而未完成；随后使用 Alpine 容器真实运行 arm64 静态二进制和 Linux 通知测试，均通过。完整镜像构建与 ENTRYPOINT 冒烟由 Ubuntu CI 作为正式门禁。

## 扫榜能力决策

扫榜已明确从产品路线移除。原因是项目需要稳定运行于 Linux、服务器、NAS 和无头环境；自动访问番茄、起点、晋江等站点通常引入浏览器、登录态、页面适配、反爬和平台条款维护，和当前本地创作运行时的可移植边界不匹配。

保留的市场/对标能力只有用户主动提供本地文本后的抽象拆文，不自动访问任何平台。历史归档中的扫榜候选是当时讨论快照，不代表当前路线。

暂不增加 `doubt/suspect/forget/reader belief` 等认知动作，除非出现明确产品需求。

## 端到端验收覆盖盘点

- Engine 文档已明确 `host/engine_test.go` 使用 fake 模型与真实工具覆盖完整书、失败/僵局裁定、返工、Hold 和退出竞态。
- Cocreate 已在 startup/Host/TUI 三层覆盖阶段顺序、协议、Ctrl+S、失败与取消。
- Import 已覆盖全书事实门禁、正式 Store 零污染、原子发布、恢复状态和 Knowledge 发布。
- Deconstruct 已覆盖显式目录、扫描扩展名、增量复用、合规 Prompt 和画像导入。
- TXT/EPUB 已覆盖不泄露 Knowledge。
- `internal/entry/headless` 原先没有测试；现已补齐空 Prompt/空工作区的真实入口失败边界和脱敏诊断导出契约，没有为测试增加 Host 注入框架。
- 代表性自动化路径全部通过，未发现 P0/P1。
- 用户授权后已用 `sss / gpt-5.6-sol` 完成真实 Headless 单章闭环：强杀恢复、预算硬停、模型自修正提交、事实重放、TXT/EPUB 隔离均通过，总费用约 `$0.410`。
- 真实验收发现 3 项 P2：目标约 1200 字实际 2092 字；正文残留 6 个 `**`（Lint 已报告）；完结态无 Prompt Headless 重启使用错误退出码和不够准确的文案。
- 首轮真实验收发现的 3 项 P2 已在里程碑 O 修复，并用同一 `sss / gpt-5.6-sol` 做二次单章回归：目标 1200、实际 1311 字，Markdown/其它规则违规为 0，无 PendingCommit，完结态重启零写入，全量重放和 TXT/EPUB 隔离通过，费用约 `$0.161`。
- 2026-08-25 全项目复审发现阻塞恢复窗口：最后一章 `MarkComplete` 早于 PendingCommit 进入 `progress_marked/signal_saved` 并清理；Headless 完结快路径若只看 `phase=complete`，可能跳过仍需收尾的 PendingCommit。下一计划必须先让终态判定同时证明无 PendingCommit，再谈新的真实协同验收。
- 本轮架构审查尝试读取 `docs/adr/`，目录不存在；结论为当前无 ADR 可核对，不将其视为产品错误，也不再重复访问。
- Cocreate、PendingCommit 中间 Stage 强杀、Revision、Import 后续写和 Deconstruct 文学效果仍未做真实模型验证。

## 里程碑 U：Import 认知事实提取校准（2026-08-26）

阶段 157 已完成：建立 12 条完全自建中文片段和独立金标，覆盖 establish-only、establish+learn、establish+reveal、三动作、belief，以及猜测/谎言/明确不相信负例。确定性数据契约通过。语义约定是：`establish` 建立作者层 Truth；`reveal_to_reader` 仅用于此前隐藏 Truth 的完整揭晓，不机械伴随普通世界设定。

阶段 158 当前为 `blocked_provider`。使用 `sss / gpt-5.6-sol` 的单批两条通道验证成功（约 31 秒，普通事实均正确提取为 establish，Usage 有记录），但完整质量基线没有产出有效结果：一次长时间停在本地代理 TCP 连接；改为每批两条后，第 1 轮第 3 批出现不一致 Knowledge ID，正式 Import 校验进入未知引用自愈，随后遇到 HTTP 502，并在 5 分钟 context timeout 结束。未生成完整结果文件，因此不计入 precision/recall、漏报/误报或 Prompt 质量证据。

完整三轮基线仍未取得，因此不能计算完整 precision/recall 或一致性；阶段 159 已基于定向探针完成最小 Prompt 修订，但不把有限结果包装成完整 A/B。Provider 通道稳定后可从阶段 158/160 继续，临时 live runner、结果、日志和一次性进程均已删除。

阶段 163/164 代码层替代验证已完成：在现有 `novel_context` 测试夹具中提供第 3 章 OutlineEntry，确认 ReaderKnown 但当前角色未知的 Truth 可注入，苏弦已知的 Truth 保留 KnownBy，顾临独知且读者未知的 Truth 不泄露。该测试不替代真实 Architect 扩弧；真实扩弧此前两次均在 Provider 请求处阻塞，未生成第 3 章大纲。

里程碑 V 的跨进程恢复审查：仓库没有现成故障注入钩子；直接新增生产环境变量会扩大运行时边界，因此没有采用。新增测试子进程只执行已有 `CommitChapterTool.Execute`，父进程通过重新打开磁盘 Store 验证 `progress_marked` 收尾。现有四个 Commit Stage、密封完整性、Imported provenance、Knowledge/Foreshadow 重放矩阵和 Race 已覆盖；没有发现需要新增生产修复的恢复缺口。

## 里程碑 W：本地拆文方法画像真实验收

离线基线复用现有 `internal/host/sim`、`internal/entry/deconstruct`、SimulationProfile 和 Context 测试，全绿。真实自建语料验收覆盖：首次 2 篇、未修改二次零分析、新增 `.markdown` 单篇增量、修改 `.txt` 单篇增量；非文本 JSON 被忽略。

首次真实 Provider 运行暴露一个真实 P1：`Host.SimulateDir` 直接使用 architect 模型，未经过 `UsageTracker`，导致画像调用实际发生但 `usage.json` 为零。已用 fake-model Host 公共测试先取得红灯，再以现有 `newUsageTrackedModel` 复用方式接入 `simulation` agent；修复后真实复跑记录 `input=22084/output=15499/cost≈$0.199158/missing_usage=0`。

真实画像安全扫描没有发现连续自建原文、人物专名或地点专名。关键词“仿写/签名短语”仅出现在模型生成的禁止复制与抽象语言特征字段中；具体内容明确禁止复制，不是模仿指令。Context 仍只注入 compact profile，不注入 `source_reports`；TXT/EPUB 隔离回归通过。

## 里程碑 X：发布候选稳定化与交付检查

阶段 174—179 已完成。干净 HOME 下 `--help/-h/help`、`--version/version`、`deconstruct --help` 通过；linux/amd64、linux/arm64、darwin/amd64、darwin/arm64、windows/amd64 的 CGO_DISABLED 构建通过。旧 UserRules、ChapterRecord、ProjectFormat、PendingCommit、SimulationProfile 兼容矩阵和未来版本拒绝行为通过。

八类模型入口 Architect、Writer、Editor、Arbiter、Import、Revision、Cocreate、Deconstruct 均已确认使用 UsageTracker；Deconstruct 先前绕过 `simulation` 记账的缺口已在 `e0cb158` 修复。Commit/Rewrite/Knowledge/Foreshadow/Imported provenance/complete/跨进程恢复和 TXT/EPUB 隔离矩阵通过。

新增交付文档：`CHANGELOG.md`、`docs/release-notes.md`、`docs/upgrade.md`。发布候选仍未创建版本标签；源码构建显示 `dev` 是未注入 GoReleaser ldflags 的预期行为。Import 认知完整三轮统计和真实 Architect 扩弧 Context 仍保留 Provider 阻塞限制，不影响已验证的主流程发布候选。

## 候选 2/3/4 规划决策（2026-08-26）

本次全项目 Review 后决定并行规划三条后续线，但不立即编码。阶段 180 已完成静态归属盘点和原始测试基线；阶段 181 进入 Commit 测试按 seam 拆分，仍不修改生产代码。首个 payload/Schema/篇幅/格式切片已完成：移动测试到 `commit_payload_test.go` 后所有基线通过，未修改生产代码。第二个 Knowledge/Belief/ReaderKnown 切片也已完成，移动到 `commit_knowledge_test.go` 后所有基线继续通过。第三个 Foreshadow 生命周期切片也已完成，移动到 `commit_foreshadow_test.go` 后所有基线继续通过。随后 Rewrite/Integrity/Completion/Side-effects 剩余测试也已按接缝拆分完成，全部基线继续通过。完整集合审计确认 75 个测试与 HEAD 完全一致：

### 候选 4：按接缝拆分测试资产

当前最大测试热点是 `commit_chapter_test.go`、`novel_context_test.go`、`world_test.go` 与 `engine_test.go`。拆分的目标是改善 locality 和 AI 可导航性，不改变测试语义或生产代码。先做函数归属表，再按 Commit、Context、Import 的 seam 移动；每次移动后保持测试名称、数量和 `-run` 筛选稳定。

### 候选 2：Context selection policy 深化

`novel_context.go` 与 builders 已集中处理 Knowledge 净化、相关章节/伏笔、References、SimulationProfile、平台 Rubric、预算裁剪和 JSON envelope。它值得研究但不应凭文件大小直接重构。先用候选 4 的测试面和 deletion test 判断“删除选择/净化/预算逻辑是否会集中复杂度”；只有证明存在真实深度机会，才形成更深的 selection policy。否则只保留测试拆分，不创建空壳接口或 Context Service。

### 候选 3：Import 认知动作完整 A/B

Import Prompt 已有定向修订和有限真实证据，但完整三轮 baseline/calibrated 仍被 Provider 长连接和 HTTP 502 阻塞。规划采用可断点、逐样本落盘 runner：缺结果必须失败，不得 skip 伪成功；连续 Provider 阻塞即停止。只有动作级 precision/recall、负例误报和 believe 不退化证据足够，才允许继续修改 Prompt。该线不阻塞 Release Candidate，也不引入运行时评测框架。

### 依赖与顺序

```text
候选 4 测试按 seam 拆分
→ 候选 2 deletion test / Context selection 评估
候选 3 Import A/B 独立并行、可暂停
```

明确不做：候选 4 不顺手重构生产；候选 2 不创建通用 Context Service/Repository；候选 3 不因 Provider 阻塞伪造质量结论或新增 Import Schema。
阶段 182 首个 Knowledge/信息边界切片已完成：6 个 Context 测试移动到 `context_knowledge_test.go`。完整 Context 测试集合与 HEAD 对比为 33 个唯一测试，未丢失、重复或新增；此前集合审计第一次漏算了独立的 `novel_context_reader_boundary_test.go` 与 `novel_context_simulation_test.go`，随后已用全部 Context 测试文件重新核对通过。生产代码无差异。

阶段 182 第二个 Recall 切片已完成：8 个 Recall/伏笔/Review 记忆测试移动到 `context_recall_test.go`。完整 Context 测试集合与 HEAD 保持 33 个唯一测试，未丢失、重复或新增；生产代码无差异。

阶段 182 第三个 Budget/References 切片已完成：4 个预算裁剪测试移动到 `context_budget_test.go`，3 个平台 Rubric/References 测试移动到 `context_references_test.go`。完整 Context 测试集合与 HEAD 保持 33 个唯一测试；生产代码无差异。

阶段 182 已完成：最后 10 个 Context 测试拆分到 `context_errors_test.go`、`context_modes_test.go`、`context_envelope_test.go`；`novel_context_test.go` 仅保留共享 helper，Simulation/Reader Boundary 独立文件保持不动。全部 Context 测试与 HEAD 保持 33 个唯一测试，生产代码无差异。

阶段 182 收口：最后 10 个 Context 测试已拆分到 `context_errors_test.go`、`context_modes_test.go`、`context_envelope_test.go`。`novel_context_test.go` 仅保留共享 helper；Simulation/Reader Boundary 测试保持独立。与 HEAD 对比为 33 个唯一测试，生产代码无差异；阶段 183 已进入进行中。

阶段 183 首个切片已完成：`contracts_test.go` 原本已按职责独立，无需移动；`analyze_test.go` 中 8 个 Knowledge/认知连续性测试已迁移至 `analyze_knowledge_test.go`。Import 测试集合与 HEAD 保持 103 个唯一测试，无丢失、重复或新增；生产代码无差异。

阶段 183 第二个切片已完成：`publish_test.go` 中 4 个映射、Knowledge 发布与元数据规范化测试已迁移到 `publish_provenance_test.go`；`TestPublishChapterHandlesStalePendingCommit` 继续留在原文件作为发布恢复测试。Import 测试集合与 HEAD 保持 103 个唯一测试，生产代码无差异。

阶段 183 第三个切片已完成：`runner_test.go` 中 6 个 Import 主流程、全书事实发布前门禁、Synthesis 门禁、原文保真与 Completion Hold 测试已迁移至 `runner_recovery_test.go`；运行时配置、错误反馈和重分段测试保留在原文件。Import 测试集合与 HEAD 保持 103 个唯一测试，生产代码无差异。

阶段 183 第四个切片已完成：`analyze_test.go` 中 6 个全书事实校验与工作区失效测试已迁移至 `analyze_facts_test.go`；批次分析、Salvage、预算和 Schema 失效测试保留在原文件。Import 测试集合与 HEAD 保持 103 个唯一测试，生产代码无差异。


候选 4 收口结论：剩余 Import 测试文件已经按职责独立，继续拆分只会增加文件噪声。阶段 183—184 完成后，Commit、Context、Import 测试均按 seam 可导航，且 103 个 Import 测试与 HEAD 集合完全一致；未修改生产代码。

候选 2 阶段 185 结论：Context 生产逻辑主要集中在 `novel_context.go` 与 `novel_context_builders.go`，已有测试按 Knowledge、Recall、Budget、References、Modes、Envelope、Errors 分组。当前明确边界是 ContextTool 继续作为适配器；是否需要更深的选择策略模块，必须由 deletion test 证明。

阶段 186 deletion test 结论：Knowledge 选择/净化、预算裁剪、Envelope 装配均不可删除。第一版 Knowledge 变体因脚本破坏局部变量而无效；修正为可编译 no-op 后，相关边界测试稳定失败。由此只批准阶段 187 的 Knowledge 选择纯逻辑局部化，不批准 Context Service/Repository、Budget 模块或 Envelope 重构。

阶段 187—188 结论：Knowledge 选择与净化已集中到同包纯函数 `selectKnowledgeBoundaries`，只接收已加载投影、当前角色匹配结果和章节号，不执行 IO、预算裁剪或序列化。ContextTool 仍是唯一公共适配器；现有 JSON、泄漏、时间边界、8 条上限和预算行为均保持不变。

候选 3 阶段 189 结论：可断点 Runner 已完成且不进入运行时 Import。结果按样本落盘，成功结果携带 Prompt 身份和 Usage，错误只保存摘要；结果损坏或身份不匹配时不会静默跳过。阶段 190 仍需 Provider 稳定后再做有限探针，不能因 Runner 完成而伪报完整 A/B。

候选 3 阶段 190 真实探针补充：`ik03` baseline/calibrated 均成功；`ik04` 本次两版均成功输出三动作。此前同样本曾出现只输出 establish，说明 Provider/模型输出有随机性；阶段 190 需要累计有效双侧样本，不能用单次输出声称召回改善。

阶段 191 工具错误：完整 A/B 协调器首次启动前因摘要变量未导出而失败，未调用 Provider；不能计入质量结果。临时 Runner 需先修正每样本 Usage 与错误续跑，再重启。

候选 3 阶段 191—192 结论：完整 A/B 已取得 36/36 有效结果。calibrated Prompt 明显提高 learn recall，但降低总体 precision 和动作集合完全匹配；三个负例 baseline/calibrated 均 9/9 为空，believe precision 两版均为 0.5，说明当前修订没有改善该问题。按 Go/No-Go 保留已提交 calibrated Prompt，不继续叠加 Import 规则，也不把单一指标提升包装为整体改进。完整动作级统计与限制见 `evals/import-knowledge/ab-summary.json` 和 `report.md`。

候选 3 完整 A/B 收口：36/36 结果有效，Provider 错误/超时为 0。calibrated 只在 learn recall 上取得明显收益，整体 precision 与动作集合完全匹配下降；reveal_to_reader precision 下降至 0.7143，believe precision 两版均为 0.5，负例无新增动作。当前 Prompt 作为折中版本保留，后续不再基于这组 12 条样本继续堆叠规则；若未来需要优化，应扩大独立样本与人工金标，而不是局部追加禁令。
