# ainovel-cli 当前演进计划

## 当前状态

- 总体状态：`complete`
- 当前基线：AC 发布基线一致性门禁待提交；稳定版本仍为 `v0.1.2`。
- 工作区：AC1—AC6 已完成；本轮脚本、Workflow、测试和文档变更待提交，生产代码无未提交变更。
- 产品边界：本地文件系统驱动、可恢复、可审计的 AI 小说创作运行时。

## 稳定架构边界

1. 模型负责开放语义；代码负责状态、约束、事务、恢复和验证。
2. `ChapterRecord.Facts` 是派生事实重建输入；JSON/Markdown 状态文件是可重建投影，不是第二事实源。
3. Knowledge 与 Foreshadow 生命周期分别由专用纯 Apply/Replay 函数裁决。
4. Commit 首次冻结执行纯载荷校验与当前状态试运行；恢复执行密封校验、纯载荷校验和幂等重放。
5. Import 通过统一 ChapterFacts 映射和全书重放门禁发布；`generated/imported/user` 正文来源权限不同。
6. Context 负责读取、选择、净化、预算裁剪和 Envelope；Writer 不直接消费完整 KnowledgeEntry。
7. 不引入数据库、通用状态机、CRUD Service、浏览器自动化、扫榜或并行相邻章节写作。
8. 外部仓库只提供设计参考和行为假设，不直接合并代码、Prompt、Skill 或运行架构。

## 历史归档

本轮规划前的根工作记忆已归档至：

```text
docs/history/plans/2026-08-external-updates/
```

此前演进过程仍在：

```text
docs/history/plans/2026-08-domain-saga-evolution/
docs/history/plans/2026-08-pre-release-candidates/
```

历史快照中的旧状态不覆盖本文件。

## 本轮规划总览

```text
AB0：外部更新证据与许可证登记       complete
        ↓
AB1：推理内容与最终内容隔离专项     complete
        ↓
AB2：上游 ainovel-cli 差异审查       complete
        ↓
AB3：字数口径与完成收口复核          complete
        ↓
AB4：作者记忆边界设计                complete
        ↓
AB5：细纲照搬检测候选                complete
        ↓
AB6：设定词典候选                    complete
        ↓
AB7：范围、文档与 Go/No-Go 收口      complete
```

AB4—AB6 是有条件候选：只有前一阶段的行为测试或证据证明有明确增量，才进入实现。AB1—AB3 优先。所有阶段均不阻塞已发布的 `v0.1.2`，除非发现真实 P0/P1。

# 阶段 AB0：外部更新证据与许可证登记

状态：`complete`

完成内容：

- 核对五个外部仓库的默认分支、最新提交/版本、最近变更和许可证字段；
- 对比 `voocel/ainovel-cli` 与当前 fork `r6c/ainovel-cli` 的关系；
- 将旧根规划归档；
- 不安装外部 Skill，不执行外部脚本，不复制外部代码或长 Prompt。

外部证据与吸收决策保存在 `findings.md`，不写入本计划作为指令性网页内容。

# 阶段 AB1：推理内容与最终内容隔离专项

状态：`complete`

目标：确认当前 Go/agentcore/Provider 转换链是否会把 `reasoning_content`、`reasoning_details`、`<think>` 等内容混入 JSON、章节正文、Tool 参数或用户可见共创文本。

首个 TDD 切片：

- 非流式消息同时含 final content 与 reasoning content；
- 流式 reasoning delta、正文 delta、tool-call delta、Done/Usage；
- 只有 reasoning、没有 final content；
- final content 中夹带 think block；
- reasoning 中含 JSON 片段；
- Cocreate、结构化调用、普通章节写作分别观察可见内容。

成功标准：

- reasoning 只进入内部调试/可选思考字段；
- JSON 解析只消费 final content；
- 章节正文不出现思考内容；
- Tool 参数不被 reasoning 污染；
- StreamEvent 顺序与 Usage 记录不回归；
- 不打印完整思考内容。

实现约束：优先在现有 Provider/agentcore 转换接缝修复；不得复制 MuMu 的 Python 代码；不新增第二套模型客户端。

已完成结果：

- `agentcore.Message.TextContent()` 排除 `ContentThinking`，结构化 JSON 解码回归通过；
- `coCreateStream` 不再把 thinking-only 响应当作用户回复；
- 共创最终回复、流式预览和未闭合 `<think>/<thinking>` 块均做最终内容清理；
- 普通 Worker 与 Cocreate 会话日志不保存完整 reasoning，只保留可见响应和 thinking 长度；
- thinking 中的 JSON 不会成为 Tool 参数；
- `Failure.Raw`、结构化自愈历史和 Import 失败工件不再携带内嵌 thinking；
- TUI thinking 进度、Usage 和流式事件顺序保持兼容。

本阶段没有修改 agentcore 上游库，没有引入模型客户端或新的持久化事实源。

# 阶段 AB2：上游 ainovel-cli 差异审查

状态：`complete`

目标：对 `voocel/ainovel-cli` 的近期修复做行为级比较，不直接 merge/cherry-pick。

完成结果：

- `Progress.LatestCompleted()` 已统一接入 Flow、Host Resume、Host Snapshot 和 Store 一致性检查；按完成列表顺序工作的全局评审逻辑保持不变。
- 上游分层完结补偿已按当前 `layeredComplete` 规则 clean-room 实现为 `ReconcileLayeredCompletion`，Engine 在路由前的明确恢复窗口调用；卷末三件套缺失、骨架弧和开放长线语义保持原规则。
- 上游 `ChapterRecordStore.Prepare` 不吸收：当前 Rewrite 在 PendingCommit 前已有候选记录集验证，Revision 在 PendingRevision 前已有候选记录验证，新增 API 只会扩大表面积。
- 返工伏笔恢复已有 `RestoreOwnPlants + ApplyForeshadowUpdates` 与完整生命周期回归覆盖，无需复制上游修复。

不直接 merge/cherry-pick 上游。

优先审查：

1. `LatestCompleted()` 是否比当前数组末尾语义更安全；
2. 分层完成状态补偿是否覆盖当前终态恢复之外的窗口；
3. `ChapterRecordStore.Prepare` 是否能替代或补强当前 Rewrite/Import 候选记录验证；
4. 返工伏笔恢复是否存在当前 `RestoreOwnPlants + ApplyForeshadowUpdates` 未覆盖的变体。

每项必须先有当前行为测试或差异样例；无差异则只记录结论，不改代码。不得引入上游不同的状态模型或流程架构。

# 阶段 AB3：字数口径与完成收口复核

状态：`complete`

目标：对照 `oh-story` v0.7.7 的 `visible_chars_v1` 与当前 `domain.WordCount`、`chapter_target_chars`、120% 上限、生成/导入来源政策和完成 checkpoint。

完成结果：

- 当前唯一章节字数口径是 `domain.WordCount`：先执行 `NormalizeChapterContent`（去 BOM、统一 CRLF/CR 为 LF），再按 Unicode rune 计数。
- DraftStore、`draft_chapter`、Commit 和 Revision Projector 均使用该口径；`generated/imported/user` 只决定篇幅门禁是否适用，不改变计数方式。
- `generated` 正文继续只在超过明确单章目标的 120% 时于 PendingCommit 前拒绝；`imported/user` 正文保留原文，不受该生成门禁约束；不设置机械下限。
- Checkpoint 只记录工件摘要，不定义第二套字数计算；无需引入 `visible_chars_v1`、Length Service 或一次压缩状态。
- 新增 BOM/CRLF 跨层回归测试，确认 Draft、Commit 和 Projector 的结果一致。

验证：

- 标题、换行、Markdown 标记的计数口径；
- generated/imported/user 三种来源；
- 普通提交与 Rewrite；
- Progress、Projector、Commit 是否使用同一口径；
- 超长是否应最多压缩一次；
- 偏短是否继续交给 Editor，不设置机械下限；
- 缺少合法目标时是否保持当前保守行为。

若口径已经一致，只增加文档/回归结论；不新增长度 Service、第二份合同字段或自动灌水流程。

# 阶段 AB4：作者记忆边界设计

状态：`complete`

目标：评估 `oh-story` 跨会话作者记忆与当前 UserRules 的边界。

先设计，不直接实现完整目录协议。必须明确：

```text
UserRules       = 当前书/当前运行可执行规则
Author Memory   = 跨项目、用户明确要求长期记住的偏好
Book Facts      = 本书事实
Run Intent      = 本次运行意图
```

首批只考虑：

- 明确“记住”意图；
- 用户确认；
- 写入回执；
- 相关性查询上限；
- 冲突替代；
- 撤回/清除。

不得把模型推断自动写成长期偏好；不得复制 `.story/作者记忆/` 的文件结构或实现代码；不得让作者记忆覆盖本书硬约束。

完成结果：

- 当前不实现独立 Author Memory；全局 rules 继续承担跨书复用输入，本书唯一可执行规则仍是 `meta/user_rules.json`。
- 明确区分 `UserRules`、Author Memory、Book Facts 和 Run Intent；未确认的跨项目偏好不得持久化。
- 若未来实现，只能经过“明确记住 → 回显 → 用户确认 → 转为 UserRules Candidate”的路径，并提供单条撤回，不覆盖本书事实和硬约束。
- 新增 `docs/author-memory-boundary.md` 与边界矩阵测试；没有新增运行时类型、存储格式或接口。

# 阶段 AB5：细纲照搬检测候选

状态：`complete`

目标：评估 `oh-story` 的 outline-copy 检测是否适合当前 ChapterContract/大纲文本。

先做输入可得性和误报样本：

- 细纲与正文是否有稳定的同章配对；
- 复沓锚句、专名、系统提示、固定台词如何豁免；
- 15 字连续重合阈值是否适合中文网文；
- Import、Rewrite、generated 三种来源如何处理。

若实现，只能先作为 `rules.Lint`/Editor advisory；不直接阻断 Commit，不复制外部 JS，不做语义相似度或跨章模糊匹配。

当前实施阶段：

1. 建立 `OutlineEntry` 与 `ChapterRecord.Content` 的同章输入可得性矩阵；
2. 用自有正反例锁定正常复述、专名、固定台词、系统提示、锚句和真正疑似照搬的边界；
3. 仅在确定性连续重合规则的误报可控时，才考虑实现 advisory；
4. 若输入或误报证据不足，以文档/测试结论收口，不创建新规则。

当前结论：输入可得，但检测规则暂不实现；先完成误报基线。任何后续实现必须保持 `generated/imported/user` 来源边界，Import 原文不因该 advisory 被清洗或阻断。

# 阶段 AB6：设定词典候选

状态：`pending`

目标：评估 `chinese-novelist-skill` 的设定词典是否能复用现有 Knowledge/ReaderKnown，而不是新建平行事实源。

先定义边界：

```text
Knowledge       = 作者 Truth、角色 KnownBy、ReaderRevealedAt、Belief
Setting Term    = 术语首现、类别线索、读者可见含义、计划揭示提示
```

首批只考虑：

- 新术语首现缺少类别/可感知线索时生成 advisory；
- 已知术语超出 ReaderKnown 时提示；
- 计划揭示临近但未兑现时提示。

只有能复用现有事实和投影，且误报可控，才考虑实现。不创建新的认知状态机或 `TermGlossaryService`。

完成结果：

- 已确认 `OutlineEntry` 与 `ChapterRecord.Content` 具备稳定同章配对输入。
- 已建立 `docs/outline-copy-boundary.md`，锁定标题、专名、固定台词、系统提示、任务清单和事实锚点等误报边界。
- 当前 No-Go：不实现运行时 `outline_copy` advisory；不扩展 `rules.Lint` 输入、不阻断 Commit、不清洗 imported/user 原文、不复制外部 JavaScript。

# 阶段 AB7：范围、文档与 Go/No-Go 收口

状态：`complete`

完成条件：

- `go test ./...`、`go vet ./...`、关键 Race 通过；
- 所有外部来源、许可证和吸收/不吸收决定可追溯；
- 没有外部代码、长 Prompt、凭证或临时脚本进入仓库；
- 稳定文档与当前阶段一致；
- 逐阶段决定：实现、仅保留测试/文档，或明确不引入。

建议提交信息按实际内容使用中文，例如：

```text
兼容：隔离推理内容与最终输出
审查：记录上游差异与吸收决策
评估：复核字数口径与章节完成收口
```

## 本轮不做

- 不安装 `chinese-novelist-skill`、`oh-story` 或 `lieflat-less-ai-tone`；
- 不复制任何外部 Skill、Prompt、脚本或 GPL-3.0 代码；
- 不 merge/cherry-pick `voocel/ainovel-cli`；
- 不引入 PostgreSQL、FastAPI、React、Web 工作台；
- 不恢复扫榜、Chrome/CDP、浏览器登录态或网络抓取；
- 不引入第二个去 AI 味管线；
- 不调用真实 Provider，除非用户明确把某一阶段推进到真实评测；
- 不新增通用状态机、ContextService、Repository 或新的事实源。

## 当前下一步

AC 发布基线一致性门禁已完成；下一步另行规划 Import 评测证据可复核性、模型入口 Usage 契约和 Context 决策可见性。

# 里程碑 AC：发布基线一致性门禁

状态：`in_progress`

目标：阻止以下情况进入 Release workflow：

```text
测试/构建提交 ≠ 触发标签提交
发布说明对应的标签 ≠ 当前发布标签
同一提交的旧 RC 标签被 GoReleaser/脚本误选
```

## 阶段 AC1：基线失败测试

状态：`complete`

- 模拟两个标签指向同一提交；
- 显式指定当前发布标签与提交；
- 要求基线检查拒绝标签/提交/发布说明不一致；
- 不推送、不创建新标签。

## 阶段 AC2：最小基线脚本

状态：`complete`

新增一个 CI 可复用的 shell 检查：

- 校验发布标签解析到 `RELEASE_SHA`；
- 校验当前 checkout 提交与 `RELEASE_SHA` 一致；
- 校验发布说明包含当前标签和提交元数据；
- 不依赖标签排序猜测；
- 失败时 fail-closed。

## 阶段 AC3：Release workflow 接入

状态：`complete`

- 通过 `github.ref_name` 和 `github.sha` 显式传入当前发布身份；
- `gen-changelog.sh` 使用显式当前标签，不再依赖 `git describe HEAD` 猜当前标签；
- GoReleaser 前执行基线门禁；
- 保留无 Secret 时的确定性 fallback。

## 阶段 AC4：发布说明可重复性收口

状态：`complete`

- 发布说明写入 tag/commit 元数据；
- AI 生成仅改变正文，不改变基线头；
- 生成失败仍可回退确定性提交列表；
- 不把外部 AI 变成发布成功的必要依赖。

## 阶段 AC5：候选 2—4 规划

状态：`complete`

### 候选 AC5-A：Import 评测证据可复核性

状态：`planned`

先修正评测执行与证据保存协议，再重新调用 Provider：

- 真实调用必须经过已提交的 `ImportKnowledgeRunner`；
- 每条结果原子落盘，保存 sample/round/arm、Prompt 名称与摘要、动作签名和 Usage；
- 不保存完整模型响应、原文或凭证；
- 统计脚本必须能从脱敏工件重新计算 precision/recall、exact set/order 和一致性；
- 任一 Provider 错误只影响该样本，不把未完成轮次算成有效结果；
- 连续阻塞达到停止条件时暂停，不修改 Prompt。

### 候选 AC5-B：模型入口 Usage 统一契约

状态：`planned`

为 Architect、Writer、Editor、Arbiter、Import、Revision、Cocreate、Deconstruct 建立 fake-model 用量契约：

- `per_agent` 与 `per_model` 都有记录；
- input/output token 和 cost 不伪造为零；
- 流式 Done Usage 只记录一次；
- BudgetSentinel 能看到累计费用；
- 模型缺 Usage 时按现有缺失策略记录，不静默宣称零成本；
- 不新增 Usage 系统或入口抽象。

### 候选 AC5-C：Context 决策可见性

状态：`planned`

只在真实失败样本出现后深化。第一步先加测试专用决策报告，不改变公共 JSON：

- 记录候选来源、入选/排除原因、排序和裁剪原因；
- 默认不输出 Truth、Belief 或正文；
- 仅在测试/诊断模式启用；
- 不新增 `ContextService`、Repository 或通用 Trace 框架；
- 如果现有边界测试已足够，保持 No-Go。

候选执行顺序：

```text
AC5-A Import 证据可复核性
→ AC5-B 模型入口 Usage 契约
→ AC5-C Context 决策可见性（有真实失败再做）
```

三项均不阻塞已经发布的 `v0.1.2`，也不自动创建新版本标签。

## 阶段 AC6：全量验证与收口

状态：`complete`

通过：

```bash
sh -n .github/scripts/check-release-baseline.sh
sh -n scripts/release_baseline_test.sh
sh scripts/release_baseline_test.sh
bash -n .github/scripts/gen-changelog.sh

go test ./... -timeout=5m
go vet ./...
git diff --check
```

本里程碑不创建标签、不推送、不发布远端；完成后另行决定版本基线。

## 当前修复：/start 基础设定审查循环

状态：`complete`

最近修复：

- Architect 无 chapter 的 `novel_context` 与基础设定写工具串行；章节 Context 保持并发。
- stale fingerprint 返回结构化 `foundation_ready=false/current_fingerprint/next_action`，避免 Agent Core 丢弃可修正结果。
- Architect Prompt 明确消费 `current_fingerprint`，不得重提旧值。


目标：修复 Architect 在 `/start <大纲文件>` 初始规划中反复收到旧 foundation fingerprint，导致 `audit_foundation` 持续返回 tool conflict 的循环。

已确认的最小方向：

- `FoundationFingerprint` 只读且连续读取稳定；
- Architect 的无 `chapter` `novel_context` 必须与 `save_foundation`/`audit_foundation` 串行；章节上下文继续允许并发；
- 过期 fingerprint 不应以 `result + error` 返回，因为 Agent Core 会丢弃 result；应返回 `foundation_ready=false`、`current_fingerprint` 和 `next_action` 的普通工具结果；
- 不放宽 fingerprint 校验，不修改用户上传的大纲，不新增状态机或重试服务。

验证门禁：

- `novel_context → audit_foundation` 新鲜 fingerprint 公共链；
- stale fingerprint 结构化修正结果；
- Architect/章节 Context 调度安全；
- `go test ./...`、`go vet ./...`、关键 Race、`git diff --check`。

结果：已通过；`novel_context` 的 Architect 无 chapter 请求与基础设定写工具串行，stale fingerprint 作为 `foundation_ready=false` 的结构化可修正结果返回；章节上下文仍保持并发安全。
