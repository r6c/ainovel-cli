# ainovel-cli 当前稳定发现

## 当前基线

- Git 基线：`8ea849d 加固：建立发布基线一致性门禁`
- 当前修复：`/start` 基础设定审查循环已完成红→绿，待独立提交。
- 当前修复：`/start` 基础设定审查循环排查中；生产代码仅修改 `audit_foundation` 错误反馈与 `novel_context` 调度安全标记。
- 已确认：`FoundationFingerprint` 只读且连续读取稳定；问题来自可并发的 `novel_context` 与基础设定写工具同轮竞争，以及过期错误缺少当前 fingerprint。

- Git 基线：`a3a884f 评估：明确设定事实边界`
- 稳定版本：`v0.1.2`
- 工作区：AB7 收口待提交；生产代码无未提交变更。
- 项目定位：本地文件系统驱动、可恢复、可审计的 AI 小说创作运行时。
- 核心边界：模型负责开放语义；代码负责状态、约束、事务、恢复和验证。

## 外部更新审查（2026-08-27）

| 仓库 | 最新证据 | 许可证 | 决策 |
|---|---|---|---|
| `voocel/ainovel-cli` | `c090029`；v0.7.7；返工伏笔死锁、完成状态补偿、`LatestCompleted`、候选记录准备 | Apache-2.0 | 不直接 merge；做行为差异审查 |
| `PenglongHuang/chinese-novelist-skill` | `3db1e3b`；术语词典、文风基准；README 标注 v2.0 | GitHub API 未识别；README 有 MIT badge | 按未明确许可处理，不复制文本；只研究抽象边界 |
| `zenstory-ai/oh-story-claudecode` | `feb30aa`；v0.7.7；字数 checkpoint、跨会话作者记忆、细纲照搬检测 | MIT | 不安装 Skill；选择性借鉴设计 |
| `xiamuceer-j/MuMuAINovel` | `cb73692`；v1.5.4；推理内容隔离、重复生成防护 | GPL-3.0 | 不复制代码；优先做自有响应隔离测试 |
| `larashero3-dotcom/lieflat-less-ai-tone` | `27d2923`；冒号按用途重测、规则精简、信息守恒 | MIT | 不安装第二条管线；只作反例和规则参考 |

补充事实：

- 当前项目远端是 `r6c/ainovel-cli`，是 `voocel/ainovel-cli` 的 fork；当前 fork 已有自定义领域事实、Commit Saga、Import provenance、Context policy 和发布链，不能直接把上游默认分支当作可合并更新。
- `oh-story` 的最新作者记忆设计与当前 `UserRules` 有潜在重叠；若后续设计，必须保持“跨项目长期偏好”和“当前书可执行规则”分离。
- `chinese-novelist-skill` 的术语词典与当前 Knowledge/ReaderKnown 有交叉；应优先做诊断 advisory，不新增平行事实源。
- `lieflat` 最新冒号修订强调按用途区分正常总说分说与空转列表引导；当前项目已有反例校准方向，不新增第二个 Skill。

## AB2 收口结论

本阶段对上游 `voocel/ainovel-cli` 最新 `c090029` 做了行为级比较，不直接合并上游。`LatestCompleted()` 的最大完成章语义和分层完结补偿在当前 fork 中存在真实增量，已按当前模型 clean-room 实现并通过回归；`ChapterRecordStore.Prepare` 没有明确增量，当前 Rewrite/Revision 已有写前候选事实验证；返工伏笔恢复已有 `RestoreOwnPlants + ApplyForeshadowUpdates` 与回归覆盖。

新增行为回归：乱序完成章的 Flow/Host/Store 消费、卷末三件套补偿到 `PhaseComplete` 且重复调用幂等。

## 首要候选：推理内容隔离

AB1 已完成确定性核对并发现局部泄漏，处理结果如下：

```text
reasoning_content / reasoning_details
ContentThinking / StreamEventThinkingDelta
<think>...</think>
```

`agentcore.Message.TextContent()` 与 LiteLLM 的独立 thinking block 本身不会把 reasoning 自动并入 final text；但应用层曾在 Cocreate 的 thinking-only fallback、内嵌思考标签和会话日志投影中放宽边界。

AB1 的回归范围包括：

- 非流式 final content + reasoning content；
- 流式 reasoning delta + 正文 delta + tool-call delta + Done/Usage；
- 只有 reasoning、无 final content；
- final content 夹带 think block；
- reasoning 内含 JSON；
- Cocreate、结构化调用、普通章节写作的可见输出。

已取得 AB1 初步证据：

- `agentcore.Message.TextContent()` 只拼接 `ContentText`，不会拼接 `ContentThinking`；非流式结构化 JSON 解码目前安全，已补回归测试。
- `agentcore` LiteLLM adapter 会把 reasoning delta 转成独立 `ContentThinking` / `StreamEventThinkingDelta`，不会自动并入 final text。
- 当前项目的真实泄漏点位于 `internal/host/cocreate.go`：只有 thinking、无 final text 时曾把 thinking 当作用户回复；final text 中的 `<think>/<thinking>` 块也曾原样进入回复。两者已由测试驱动做局部修复。
- `meta/sessions/cocreate.jsonl` 原先会保存完整 thinking；现已改为只保存可见 raw、解析结果和 `thinking_len`。
- 普通 `SessionStore` 原先会序列化 ThinkingBlock；现已过滤 ThinkingBlock，仅保留可见文本、ToolCall、Usage 和 `thinking_len`。
- 流式 tool-call 只从 `ContentToolCall` 执行，thinking 中的伪造 JSON 不会污染工具参数。
- `llmcontract.Execute` 在 JSON 提取前清理内嵌 `<think>/<thinking>`，失败 Raw 和重问历史使用清理后的文本；原始思考不会进入 Import 失败工件。

AB1 已完成：用户可见回复、结构化 JSON、工具参数和持久化会话日志均与内部 reasoning 隔离；TUI 内部 thinking 进度与 Usage 保持可用。

实现仍为 clean-room 自有代码，没有复制 MuMu 的 Python/GPL-3.0 实现。

## AB3 字数口径复核结论

已确认章节字数只有一套口径：`domain.WordCount` 先执行 `NormalizeChapterContent`（去 BOM、CRLF/CR 统一为 LF），再按 Unicode rune 计数。DraftStore、`draft_chapter`、Commit、Progress 和 Revision Projector 已统一复用；剩余 `utf8.RuneCountInString` 仅用于会话、对话样本和风格锚点等非章节字数场景。

`generated` 正文继续在 PendingCommit 创建前执行明确单章目标的 120% 上限；`imported/user` 正文保留原文且不受 generated 篇幅门禁约束；不设置机械下限，不引入第二套 `visible_chars_v1`、Length Service 或自动压缩状态。Checkpoint 只记录工件摘要，不定义字数规则。

新增 BOM/CRLF 的 Domain、DraftStore、draft_chapter 和 Projector 回归测试。

## AB2 上游差异审查结论

已对 `voocel/ainovel-cli` 最新 `c090029` 做行为级比较，不直接合并上游。

### 已吸收的行为

- `Progress.LatestCompleted()`：当前 fork 原本已有领域方法，但 Flow、Host Resume、Host Snapshot、Store CheckConsistency 仍有末项读取；已统一改为最大完成章号。
- 分层完成补偿：当前 fork 原本缺少“卷末摘要已落盘、Progress 尚未 MarkComplete”的恢复接缝；已复用现有 `layeredComplete` 增加 `ReconcileLayeredCompletion`，Engine 在路由前调用。

### 未吸收的行为

- `ChapterRecordStore.Prepare`：当前 Rewrite 和 Revision 已分别具备写前候选记录/整组事实验证；新增 Prepare API 没有明确增量。
- 返工伏笔恢复：当前已有 `RestoreOwnPlants`、纯 `ApplyForeshadowUpdates`、Projector 和 Rewrite 回归，不复制上游实现。

AB2 的新回归覆盖：

- 乱序 `CompletedChapters=[1,3,2]` 的 Flow `LastCompleted`；
- Host Resume/Snapshot 使用最大完成章；
- Store CheckConsistency 检查最大完成章；
- 已落盘卷末三件套补偿到 `PhaseComplete`，并验证幂等。

## AB3 字数口径复核结论

当前章节字数采用单一口径：`domain.WordCount` 先执行 `NormalizeChapterContent`（去 BOM、CRLF/CR 统一为 LF），再按 Unicode rune 计数。DraftStore、`draft_chapter`、Commit 和 Revision Projector 已统一使用该函数；`generated/imported/user` 仅决定生成篇幅门禁是否适用。`generated` 继续只拒绝超过明确单章目标 120% 的正文，`imported/user` 保留原文且不受生成门禁约束；不引入第二套 `visible_chars_v1`、Length Service 或压缩状态。

新增回归覆盖 BOM/CRLF 的 Domain、Draft、Tool 和 Projector 路径。`Checkpoint` 只记录工件摘要，不定义字数口径。

## AB5 细纲照搬输入可得性结论

同章比较所需的 `OutlineEntry` 与 `ChapterRecord.Content` 已通过真实 Store 公共接口验证可读并按章节号配对。已建立 `docs/outline-copy-boundary.md`，锁定标题、专名、固定台词、系统提示、任务清单和事实锚点等误报边界。

当前 Go/No-Go：No-Go。没有足够自有标注样本证明连续 15 字阈值误报可控，因此不实现运行时 `outline_copy` advisory，不扩展 `rules.Lint` 输入、不阻断 Commit、不清洗 imported/user 原文、不复制外部 JavaScript。

## AB4 作者记忆边界结论

当前不引入独立 Author Memory 运行时。现有全局 `~/.ainovel/rules/*.md` 已覆盖跨书复用输入，但它们经过每本书的 UserRules 归一化，不能等同于带确认、回执、冲突和撤回的作者记忆。作者记忆若未来实现，必须是跨项目候选输入，经过“明确记住 → 回显 → 用户确认 → 转为 UserRules Candidate”，不得直接注入 Writer、覆盖本书硬约束或混入 Book Facts/Run Intent。

新增 `docs/author-memory-boundary.md`，并用边界矩阵测试锁定 UserRules、Author Memory、Book Facts、Run Intent 与未确认猜测的分离。当前 Go/No-Go：No-Go，不新增存储协议、CLI、TUI 或生产类型。

## 其他候选

### 上游差异

优先比较：

- `LatestCompleted()` 与当前完成章语义；
- 分层完成状态补偿；
- `ChapterRecordStore.Prepare` 与当前 Rewrite/Import 候选校验；
- 返工伏笔恢复与 `RestoreOwnPlants + ApplyForeshadowUpdates`。

只做行为等价测试；不直接 merge/cherry-pick。

### 字数口径

对照 `oh-story` 的 `visible_chars_v1` 与当前：

```text
domain.WordCount
chapter_target_chars
120% generated 上限
imported/user 来源政策
```

若口径一致，只记录测试/文档结论；不引入第二份长度状态。

### 作者记忆

只在明确“记住”并经用户确认时考虑；候选能力包括回执、相关性上限、冲突替代和撤回。不能让模型推断自动成为长期偏好，不能覆盖本书硬约束。

### 细纲照搬

可考虑：

```text
同章 outline + prose
→ 连续文字重合证据
→ advisory
→ Editor 语义判断
```

前提是 ChapterContract/大纲存在稳定可比文本；不直接硬阻断 Commit，不复制外部 JS。

### 设定词典

只考虑术语首现线索不足、ReaderKnown 超前或计划揭示未兑现等 advisory；优先复用 Knowledge/ReaderKnown，不创建平行认知状态机。

## AB7 最终收口结论

截至当前基线，外部更新审查已经完成：AB1/AB2/AB3 选择性吸收并通过回归；AB4/AB5/AB6 明确记录 No-Go，不新增作者记忆运行时、细纲照搬 advisory 或设定术语事实源。README 的 Docker 镜像地址已修正为实际发布仓库 `ghcr.io/r6c/ainovel-cli`，Go module 路径保持 `github.com/voocel/ainovel-cli`。

全量测试、`go vet`、关键 Race、文档链接、许可证边界、敏感信息和生产范围检查均通过。当前没有新的 S1/S2 运行时问题；下一步可由用户另行决定是否进行发行包维护或新功能规划。

## AB7 最终 Go / No-Go

截至当前基线，五个外部仓库均已完成更新与许可证审查。`voocel/ainovel-cli` 只选择性吸收了完成章语义和分层完结补偿；`MuMuAINovel` 的推理隔离原则已用自有实现完成验证；`oh-story` 的字数/作者记忆/细纲照搬只作为设计参考；`chinese-novelist-skill` 的设定词典和文风基准不引入；`lieflat-less-ai-tone` 不安装、不建立第二条管线。

最终 No-Go：不整体合并外部仓库，不安装外部 Skill，不复制 GPL-3.0 或许可证不明确的代码/文本，不新增作者记忆事实源、细纲照搬规则或设定术语事实源。核心代码、全量测试、`go vet`、关键 Race、文档链接、许可证边界和敏感信息检查均通过。

## 已通过的稳定边界

- Knowledge/Foreshadow 生命周期分别由专用纯 Apply/Replay 函数裁决。
- PendingCommit v2 区分首次冻结校验、密封和恢复幂等重放。
- Import 使用全书事实重放和 provenance 写权限保护。
- Context 使用 Knowledge 净化视图，隐藏 Truth 不泄露。
- Linux/无头不依赖浏览器、GUI、扫榜或反爬。
- `v0.1.2` 的 Release、Docker、多平台资产和安装链已验收。

## 本轮范围边界

本轮规划不做：

- 整体合并外部仓库；
- 安装任何外部 Skill；
- 复制 GPL-3.0 代码或未明确许可文本；
- 引入第二个去 AI 味管线；
- 引入数据库、Web 工作台、Chrome/CDP 或扫榜；
- 在没有确定性证据前修改生产逻辑；
- 真实 Provider 调用（除非用户单独确认真实评测阶段）。

## 当前结论

项目没有跑偏。AB0—AB7 已完成；当前没有新的 S1/S2 运行时问题。外部仓库均已完成选择性评估，已吸收内容通过回归，No-Go 候选保持隔离。后续如继续，应另行创建新计划，不把历史阶段文本当作当前任务。

## AC：发布基线一致性（2026-08-27）

已确认的事实：

- `HEAD=5ed53d0`，`origin/main=286a16a`，本地 `main` 超前 9 个提交；
- `v0.1.2` 指向 `70a806b`，不是当前 HEAD；
- Release workflow 触发标签使用 `v*`，但 `gen-changelog.sh` 当前通过 `git describe --tags HEAD` 猜当前标签；
- 同一提交存在旧 RC/正式标签时，默认描述可能选中旧标签，导致 Release notes 或 GoReleaser 基线歧义。

本轮已用显式 `RELEASE_TAG`、`RELEASE_SHA` 和发布说明元数据修复该问题；先完成失败测试，再实现脚本与 workflow 接入。没有移动既有标签或推送远端。

AC 实施结果：

- `check-release-baseline.sh` 校验显式发布标签、当前 checkout、标签指向和发布说明头部；
- `gen-changelog.sh` 服从显式标签/提交，AI 成功、fallback、空提交范围均写入同一元数据头；
- Release workflow 在 GoReleaser 前运行 fail-closed 门禁；
- 同一提交多标签不再由脚本自行猜测当前发布标签。
- AC1—AC6 已完成：门禁脚本、发布说明元数据、Release workflow 和普通 CI 契约均已接入；下一步候选为 Import 评测证据、模型入口 Usage、Context 决策可见性。

后续候选规划：

1. Import 评测证据可复核性：统一使用已提交 Runner，逐条原子保存脱敏结果和 Usage；
2. 模型入口 Usage 统一契约：用 fake model 覆盖八类入口、流式 Done 和预算哨兵；
3. Context 决策可见性：仅在真实失败样本出现后，以测试/诊断模式提供排除原因，不改变公共 JSON。

当前均为后续候选，不在 AC 收口中实现。

## AD：lieflat-less-ai-tone 选择性吸收结果

截至 2026-08-28，已读取外部仓库 main 的 `SKILL.md`、`RESEARCH.en.md`、README 与 MIT License。没有安装 Skill、运行 Python 脚本或复制外部代码/长文本。

选择性吸收进入现有唯一 `assets/references/anti-ai-tone.md` 的内容：密集顿号只指无功能空清单；翻译腔只限五类可定位结构；已有具体数据不得被概括表述覆盖；新增统计/算子先检查至少 20 个实际命中实例。它们均为 Writer/Editor 的语义参考，不是 AI 来源检测、自动改写或 Commit 硬门禁。

已通过资源契约和全量门禁。由于本轮没有把外部统计语料或阈值作为本项目证据，因此没有新增 `rules.Lint` 规则、Python 依赖或第二条去 AI 味管线。
