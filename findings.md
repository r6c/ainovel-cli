# ainovel-cli 当前稳定发现

## 当前基线

- Git 基线：`496c0bb 兼容：收口上游完成状态与分层完结`
- 稳定版本：`v0.1.2`
- 工作区：AB3 包含章节字数口径统一的生产/测试/文档变更；提交前门禁待执行。
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

项目没有跑偏。当前最值得先验证的是 Provider/agentcore 层的 reasoning 内容隔离；上游差异、字数、作者记忆、细纲照搬和设定词典按证据逐项推进，不合并成大重构。
