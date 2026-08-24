# 伏笔生命周期审查发现

## 总体判断

当前方向没有跑偏：改动仍以现有 `domain + WorldStore + CommitChapterTool` 为核心，没有引入新服务、数据库、状态机框架或外部项目运行架构。

审查发现的“正常提交—恢复—章节重写—全量投影—诊断消费”缺口现已全部闭合，并通过全量测试、静态检查与范围审计。

## 解决状态

| 发现 | 状态 | 解决结果 |
|---|---|---|
| S1-1 Projector 不认识新动作 | 已解决 | 支持五动作，重建 `LastAdvancedAt`，并有 Rewrite 集成回归 |
| S1-2 payload 内状态不推进 | 已解决 | 前置校验顺序维护临时状态，非法序列不创建 PendingCommit |
| S2-1 resolved→advance 重新打开 | 已解决 | Store、Commit、Projector 三个接缝统一将 resolved 视为终态 |
| S2-2 Projector 丢失 LastAdvancedAt | 已解决 | `advance/reinforce/partial_payoff` 全量重建最近推进章 |
| S3-1 诊断漏计新活跃状态 | 已解决 | `status != resolved` 均计入打开伏笔 |
| S3-2 Saga 使用第 0 章夹具 | 已解决 | 改为合法 `plant@1 → reinforce@2` |
| 久挂按 PlantedAt 计算 | 已解决 | Diagnostics 与 Context 均优先使用 `LastAdvancedAt` |
| 审计补充：重复 plant 重置临时状态 | 已解决 | 只有未知 ID 的 plant 才建立临时 planted 状态 |

## 当前已有能力

- `ForeshadowEntry` 新增 `LastAdvancedAt`
- 新增状态：`reinforced`、`partially_paid`
- ChapterFacts Schema 和 Validate 接受五动作
- WorldStore 支持 `reinforce`、`partial_payoff`
- `advance/reinforce/partial_payoff` 写入最近推进章节
- Commit 正常入口支持 `reinforce`
- 已回收条目不能 `reinforce/partial_payoff`
- 非法既有状态转换可在 PendingCommit 前拒绝
- Import Schema 和 Prompt 已同步五动作
- Writer Prompt 已说明动作使用时机
- Markdown 显示部分兑现和最近推进章节
- 强化动作的 PendingCommit 重放现有测试通过

## 阻塞发现

### S1-1：revision.Projector 不认识新动作

位置：`internal/revision/projector.go:104-131`

Projector 只支持：

```text
plant / advance / resolve
```

含 `reinforce` 或 `partial_payoff` 的 ChapterRecord 在章节重写时会导致 Projector 返回“伏笔操作非法”。重写流程在 Projector 前已经覆盖终稿和 ChapterRecord，因此失败会留下 `CommitStageStarted` 的 PendingCommit，并在恢复时确定性重现。

必要修复：Projector 与 WorldStore 的动作、终态保护和 `LastAdvancedAt` 语义保持一致。

### S1-2：提交前校验不推进同一 payload 的临时状态

位置：`internal/tools/commit_validation.go:25-52`

当前校验只读取初始 ledger 状态，不在循环中更新临时状态。以下非法序列会通过前置校验：

```text
resolve → reinforce
resolve → partial_payoff
plant → resolve → reinforce
```

随后 PendingCommit 已创建，WorldStore 在处理中途失败，形成不可恢复的 started 状态。

必要修复：按动作顺序维护临时可见状态，并在每一步先校验、后推进。

## 重要一致性发现

### S2-1：`resolved → advance` 会重新打开伏笔

位置：

- `internal/store/world.go:201-204`
- `internal/tools/commit_validation.go:36-50`

当前只禁止已回收条目执行两个新动作，却允许 `advance`。这会得到矛盾条目：

```json
{
  "status": "advanced",
  "resolved_at": 10
}
```

Writer Prompt 已写“不对已回收伏笔再次推进”，因此代码和契约不一致。

建议决策：`resolved` 为终态，禁止 `advance/reinforce/partial_payoff`；重复 `resolve` 保持幂等。

### S2-2：Projector 重建会丢失 `LastAdvancedAt`

位置：`internal/revision/projector.go:119-129`

旧 `advance` 只更新状态，没有写最近推进章节。即使没有新动作，任何重写触发全量投影后都会丢失该字段。

## 次要发现

### S3-1：诊断漏计新活跃状态

位置：`internal/diag/diag.go:118-126`

`ForeshadowOpen` 只统计 `planted/advanced`，漏掉：

```text
reinforced / partially_paid
```

建议与 `LoadActiveForeshadow` 对齐，以 `status != resolved` 定义活跃。

### S3-2：Saga 重放测试使用第 0 章伏笔

位置：`internal/tools/commit_chapter_test.go:657-660`

`PlantedAt == 0` 在领域中还承担“缺失值”语义，不能作为合法章节。该夹具应改成 `plant@1 → reinforce@2` 或使用合法 ledger seed。

## 额外消费端发现

### 久挂召回仍按埋设章计算

位置：

- `internal/tools/novel_context.go:748-780`
- `internal/diag/rules_planning.go:18-25`

当前久挂逻辑仍使用：

```text
currentChapter - PlantedAt
```

新增 `LastAdvancedAt` 后，更准确的“多久未推进”应使用：

```text
LastAdvancedAt > 0 ? LastAdvancedAt : PlantedAt
```

否则刚刚强化的老伏笔仍会立刻被标记为久挂。

该问题不阻塞存储正确性，但属于本批新增字段的必要消费闭环，计划在诊断阶段独立 TDD 处理。

## 架构边界核对

### 符合既定方向

- Domain/Store 仍是事实源
- Markdown 仍是投影
- Prompt 只说明开放语义，代码负责约束
- Import 只是同步现有事实契约
- 没有移植 Skill/Hook 架构
- 没有引入数据库或 CRUD Service
- 没有扩展到其他人物/知识状态

### 不应在本计划中顺带实现

- 完整伏笔状态机抽象
- `noticed / abandoned / contradicted`
- 预计回收窗口和读者可见度
- 作者真相/读者已知模型
- 角色心理状态
- GenrePack / PlatformPack / BenchmarkPack
- Web 作者工作台

## 测试矩阵

| 接缝 | 当前覆盖 | 本计划补充 |
|---|---|---|
| WorldStore 正常生命周期 | 已覆盖 | 增加 resolved→advance 终态测试 |
| Commit 正常提交 | 已覆盖 reinforce | 增加有序 payload 非法序列 |
| PendingCommit 重放 | 已覆盖 reinforce | 修正合法章节夹具 |
| ChapterFacts Schema | 已覆盖 | 保持 |
| Import Schema | 已覆盖 | 保持 |
| Markdown 投影 | 已覆盖 | 保持 |
| Writer Prompt | 已覆盖 | 保持 |
| revision.Projector | 未覆盖新动作 | 增加完整生命周期重建 |
| Commit Rewrite | 未覆盖新动作 | 增加返工集成测试 |
| Diagnostics | 未覆盖新状态 | 增加活跃计数与久挂测试 |

---

# 下一阶段路线去重盘点（2026-08-21）

## 输入建议的处理原则

跨项目分析只作为候选能力来源，不作为直接实现指令。下一阶段继续遵守：

```text
模型负责开放语义
代码负责状态、约束、事务、恢复和验证
Domain/Store 是事实源
Prompt、CLI、TUI 是适配器或投影
```

不复制 GPL 或无明确许可证项目的代码、提示词和文档；即使参考 MIT 项目，也优先按本仓库接口 clean-room 实现。

## 初步能力去重

| 候选能力 | 当前仓库初步现状 | 规划含义 |
|---|---|---|
| 伏笔生命周期 | 本轮已完成五动作、终态、Saga、Projector、诊断和 Context 闭环 | 不再继续扩状态；先稳定当前未提交批次 |
| 角色心理状态 | 未发现专用心理事实类型，但已有通用 `StateChange` | 先核对通用状态能否承载，避免新增重复账本 |
| 角色知识/读者已知 | 首轮搜索未发现 `KnowledgeFact/KnownBy/ReaderKnows/Belief` 等模型 | 很可能是下一项真正的领域缺口 |
| 关系变化 | 已有 `RelationshipChange`、Store 更新和 Commit 写入 | 不作为“新能力”重新规划，只评估是否缺重建/上下文闭环 |
| 章节情绪收益 | `ChapterContract` 已有 `EmotionTarget`、`PayoffPoints`，Review 已有 contract assessment | 优先补评测或消费，不新建重复 `ChapterBeat` 类型 |
| 人工修改后状态同步 | 已有 revision Scan、Analyze、Accept、ChapterRecord、Projector 和 stale prepared analysis 防护 | 原建议中的“检测哈希→patch→重建”大部分已存在，先做差距审计 |
| 确定性 Prose Lint | 已有 `rules.Lint`，提交和 Projector 均执行 | 适合后续纵向增强规则，不另起一套 Lint Pipeline |
| 可插拔知识包 | 当前尚未盘点资源加载与覆盖协议 | 应晚于领域事实模型，且不能先设计五种 Pack 抽象 |
| 初始化访谈/用户偏好 | 当前尚未盘点 bootstrap/startup | 优先级低于长篇一致性事实模型 |

## 当前初步优先级

在未完成数据粒度核对前，暂定：

1. 先冻结并提交/妥善保存当前伏笔批次，避免在 18 个未提交文件上继续叠加新领域改动。
2. 审计 `StateChange`、`RelationshipChange`、revision Projector 和 ChapterFacts 的闭环完整度。
3. 若通用状态已足够承载心理变化，下一真正缺口优先考虑“知识/揭示状态”，而不是重复新增心理账本。
4. Prose Lint、知识包、访谈模式拆成独立后续里程碑，不与下一领域切片混做。

## 数据粒度核对结果

### `StateChange` 已可承载角色心理，不宜立刻新建心理账本

现有结构：

```go
type StateChange struct {
    Chapter  int
    Entity   string
    Field    string
    OldValue string
    NewValue string
    Reason   string
}
```

`Field` 是开放字符串，已经能表达：

```text
emotion / fear / trust / resolve / motivation / mental_state
```

而且它已经贯通：

```text
ChapterFacts
→ CommitChapterTool
→ state_changes JSONL
→ PendingCommit 幂等重放
→ ChapterRecord
→ revision.Projector 全量重建
→ novel_context.recent_state_changes / related_chapters
```

所以“角色心理状态”当前更像字段词汇和 Writer/Editor 纪律不足，不是缺少新的事实存储。若未来确需查询“当前心理快照”，应先证明 append-log + 最近变化召回不足，再决定是否增加投影；不能先新增 `PsychologicalState` 类型。

### 关系变化已经闭环

`RelationshipEntry` 已由 `ChapterFacts.RelationshipChanges` 提交，`WorldStore.UpdateRelationships` 保存当前关系投影，Projector 按 ChapterRecord 重建，Context 注入 `relationship_state`。因此跨项目建议中的“增加角色关系变化”已基本落地。

剩余可能问题是：

- 关系只有当前描述，没有 old/new/reason 历史；
- 是否需要历史应由具体回归场景驱动，而非抽象补全。

### 人工修订后的派生事实重建已较成熟

现有 revision 流程已经具备：

```text
正文变化扫描
→ 基于 accepted content hash 判断变化
→ LLM 重新分析 ChapterFacts / StyleDelta
→ 防 prepared analysis 过期
→ 用户接纳
→ ChapterRecord 更新
→ Projector 重建 timeline / foreshadow / relationships / state changes / cast / progress / style / lint
```

因此原建议中的“检测哈希→生成 patch→重建后续上下文”大部分已经存在。下一步不应再规划新的 stale 系统，而应针对缺失的领域事实（例如知识状态）让 revision 分析和 Projector 自然覆盖。

### 情绪收益已有计划与评审接缝

`ChapterContract` 已承载 `EmotionTarget`、`PayoffPoints`、`HookGoal`，Writer Context 会注入合同，Editor 的 ReviewEntry 记录 contract status/misses。暂不需要另建 `ReaderContract` 或 `ChapterBeat`；后续可优先增强 rubric 或诊断统计。

### Prose Lint 已有统一执行点

`rules.Lint` 在正常章节提交和 revision Projector 中都会运行，另有全书 `stylestat` 注入 Context。后续去 AI 味机械检测应纵向增加规则及结构化 Violation，而不是新增第二条 Lint Pipeline。

## 下一领域候选：知识与揭示状态

当前代码没有表达下列区别：

```text
作者认定的客观真相
角色 A 知道什么
角色 B 相信什么（可能为假）
读者目前知道什么
该事实计划何时揭示
```

时间线只能记录“发生了什么”，StateChange 只能记录某实体字段变化，伏笔只能记录线索生命周期；三者都不能可靠回答“谁在第几章知道/相信了哪个事实”。

因此下一领域阶段应优先做**最小知识事实切片**，但在动代码前还需核对：

- ChapterFacts 严格 Schema 的扩展成本；
- revision Analyze/Projector 的重建路径；
- Import Schema 是否必须同步；
- Writer Context 的最小消费方式；
- 是否需要格式版本（优先避免）；
- 第一切片应只覆盖角色获知，不一次加入 belief、reader reveal 和 reveal plan。

## 产品与资源层去重结果

### 可插拔资源并非空白

`assets.Load` 已有：

```text
内置资源
< 全局 ~/.ainovel/style
< 本书 <output>/style
```

并支持：

- voice/anti-ai-tone 追加覆盖；
- styles 同名替换和新增；
- `references/genres/<style>/style-references.md`；
- `arc-templates.md`；
- 核心 Prompt A/B 覆盖；
- 按任务注入 Reference，而非所有资料常驻上下文。

所以原建议中的 `GenrePack/StylePack/QualityPack` 不应作为新框架一次性引入。近期更合理的是在现有 `References + LoadOptions + novel_context` 上增加一个具体题材或具体 rubric，待两个以上真实资源类型出现重复元数据需求后再抽象 Pack 协议。

### 全局偏好已经存在

仓库已有全局规则/偏好目录和 `userrules` 结构化归一化，Context 会向 Architect/Writer 注入 `structured + preferences`。因此不应再平行新增：

```text
~/.config/ainovel/preferences.json
```

若要增强跨书偏好，应扩展现有全局 `.md` 规则语法、规范化结果和 UI 编辑入口，保持单一事实来源。

### 启动流程已有 quick + cocreate

当前 TUI 已有：

```text
快速开始
共创规划
```

并且 cocreate 有持久会话日志、建议候选和正式启动前准备流程。所谓“三层访谈”更适合作为 cocreate 的阶段化对话策略或建议模板，而不是立刻新增第三个 `interview` 模式。只有用户测试证明 quick/cocreate 无法容纳该流程时，才考虑显式新模式。

### Prose Lint 是“规则不足”，不是“管线缺失”

当前 `rules.Lint` 只有：

- `markdown_residue`
- `non_cjk_fragments`

但它已经在 Commit 和 Revision Projector 两处统一执行，且结构化返回 `Violation`。后续可按真实误差率逐条增加：重复段落、异常标点、章节截断、细纲照搬等；不能另建 anti-AI lint 服务。

## 对原五阶段路线的修订

| 原建议阶段 | 结合仓库后的处理 |
|---|---|
| 第一阶段领域事实 | 伏笔已完成；心理/关系已有通用承载；下一项聚焦最小知识状态 |
| 第二阶段 Pack | 延后抽象；先扩展现有 References/覆盖层的一个具体资源 |
| 第三阶段 Prose Lint | 保留，但作为 `rules.Lint` 增量规则里程碑 |
| 第四阶段人机共创 | revision 和 cocreate 已较成熟；只补具体 UX 缺口 |
| 第五阶段扫榜拆文 | 保持独立命令/产物，不进入当前核心 Engine 计划 |

## 兼容性判断

给 `ChapterFacts` 新增切片字段时：

- 旧 ChapterRecord JSON 会解码为零值，通常无需提升 `ChapterRecordVersion`；
- 严格模型 Schema 会要求新数组显式存在，因此 Writer、revision、import 契约必须同步；
- Commit Saga 的 payload 冻结和 ChapterRecord 重建要求新事实操作幂等；
- Import 发布必须把新字段传入 `commit_chapter`，否则导入书会丢失该事实；
- 不能只加 Prompt 或只加 Store。

## Knowledge Context 消费决策

`OutlineEntry` 没有显式角色列表，但 Context 已有：

```text
大纲 Title + CoreEvent + Scenes
→ matchOutlineCharacters
→ 当前章涉及角色
```

因此知识状态第一版无需给大纲加角色字段，也无需新建检索服务。可复用现有匹配逻辑：

1. 若大纲能识别角色，只选择这些角色相关的 KnownBy 条目；
2. 若无法识别角色，仅注入近期 `learn` 变更或干脆不注入，不能回退为全量作者真相；
3. 使用现有 Context envelope、预算裁剪和 `_trimmed` 可观测机制；
4. `RecallItem` 可用于选中摘要，但正式知识状态仍以 `knowledge_state.json` 为事实投影。

这能避免新增“知识检索服务”，并降低作者真相意外泄漏给 Writer 的风险。

## 最小知识模型为何不直接使用 `StateChange`

虽然 `StateChange` 能记录：

```text
Entity=林墨, Field=knows:f_secret, NewValue=true
```

但它无法单独保证：

- `f_secret` 对应的客观 Truth 是什么；
- learn 是否引用已建立事实；
- 同一 Truth ID 是否被冲突定义；
- 当前 KnownBy 投影如何查询；
- 作者真相与角色状态的边界。

所以知识状态是一个真实的新领域事实，而角色心理仍可留在通用 StateChange 中。

---

# 最小知识事实里程碑完成结论（2026-08-21）

## 已实现范围

仅实现：

```text
establish：建立作者认定的客观真相
learn：某角色明确获知已有真相
```

没有实现 belief、reader reveal、forget、reveal plan。

## 完整链路

```text
ChapterFacts 严格 Schema / Validate
→ Commit 提交前临时可见性与 Truth 冲突校验
→ PendingCommit 幂等重放
→ knowledge_state.json / knowledge_state.md
→ ChapterRecord
→ Revision Analyze / Rewrite
→ Projector 全量重建
→ Import Schema / ledger / publish
→ Context 角色相关选择与预算裁剪
→ Writer / Editor 语义纪律
```

## 关键不变量

- 同一 ID 不可建立不同 Truth。
- 相同 ID + 相同 Truth 重放幂等。
- learn 必须引用已建立 Truth。
- 同角色重复 learn 不复制，保留首次 LearnedAt。
- `establish` 不代表任何角色自动知情。
- 重写不得在保留后续 learn 的同时删除其唯一 establish；该错误在 Rewrite PendingCommit 前拒绝。
- Writer Context 只看到当前大纲涉及角色已经知道的历史真相，不看到无关角色独占真相或未来信息。
- Context 最多注入 8 条 Knowledge，并可被预算裁剪、记录 `_trimmed`。

## 兼容性与缓存

- `ChapterRecordVersion` 保持 1；旧记录缺少 `knowledge_updates` 时解码为空切片。
- Import `analysisSchemaVersion` 从 2 提升到 3，只失效逐章分析缓存，不提升工作区版本。
- 没有增加数据库、迁移服务或第二事实源。

## 审计判断

该实现保持了既定架构边界：Knowledge 是 ChapterRecord 派生的当前投影，Commit/Revision/Import 共享同一事实契约，Context 只是有界消费端。没有把外部项目的 Skill/Hook、CRUD Service 或数据库架构移植进来。

---

# 下一里程碑决策：C1 读者揭示状态

## 决策

下一步不同时实现“错误信念 + 读者已知”，而是只扩展：

```text
reveal_to_reader
ReaderRevealedAt
```

## 原因

1. 读者揭示是现有 Truth 的单调属性：未知 → 已知，容易保持 Saga 幂等和 Projector 重放。
2. 错误信念不是 Truth 的简单属性，需要额外描述“角色相信的内容”，以及纠正、替换、撤销和与客观 Truth 的关系；过早加入会形成新的认知状态机。
3. 读者揭示能直接解决悬疑、马甲、误会和戏剧性反讽中的上下文问题，增量价值明确。
4. 当前 Knowledge 批次仍未提交，必须先隔离后再扩展。

## 语义边界

- `reveal_to_reader` 表示正文已让读者明确知道完整 Truth。
- reveal 不自动修改任何角色 KnownBy。
- learn 不自动修改 ReaderRevealedAt。
- 暗示、伏笔强化、部分兑现不等于完整 reveal。
- 第一版不支持“部分读者知道”“不可靠叙述者误导读者”或撤销揭示。

## Context 风险控制

未来 Context 只能注入两类 Truth：

1. 当前大纲涉及角色已经知道；
2. 读者在当前章之前已经知道。

作者已建立、但当前角色与读者都未知的 Truth 继续隐藏。否则新增 ReaderRevealedAt 反而会成为向 Writer 泄漏全量作者真相的通道。

## C1 实施结论

`reveal_to_reader` 已完成全链路实现。它是单调、幂等的首次完整揭示记录：只引用已有知识 ID，不修改角色 KnownBy，也不接受 Truth/Character 作为附加输入。Context 只在 `ReaderRevealedAt < currentChapter` 时将读者已知 Truth 暴露给 Writer，因此当前章揭示不会提前泄漏。

C1 没有把伏笔 `partial_payoff` 自动映射为读者完整获知：部分兑现可能只揭开局部答案，完整 Truth 仍需独立 `reveal_to_reader`。错误信念、撤销揭示和多读者模型继续留给后续独立里程碑。

---

# C1 后续规划盘点

## 当前状态修正

- C1 已由提交 `03bf271 功能：追踪读者揭示与信息差状态` 隔离。
- 工作区干净。
- 规划头部仍残留“Knowledge 批次未提交”的旧描述，需要在下一计划中修正。

## C2 错误信念的关键风险

错误信念不能直接复用通用 `StateChange`：

```text
Entity=林墨, Field=believes:k_shadow, NewValue=黑影是仇人
```

仍无法保证 `k_shadow` 已建立、同一角色是否有多个冲突信念、何时纠正，也无法由 Projector 稳定重建。

但错误信念也不能简单把 `Belief` 加到 `KnowledgeEntry` 后将整条 Entry 注入 Writer Context。若当前角色只相信错误内容，而读者和角色都不知道客观 Truth，序列化整个 `KnowledgeEntry` 会泄露隐藏 Truth。

因此若选择 C2，必须坚持：

1. Store/ChapterRecord 中保留客观 Truth 与信念事实；
2. Writer Context 使用经过净化的认知边界投影；
3. 当前角色只看到自己的 belief 内容；
4. 客观 Truth 只有在当前角色已知或读者已知时才进入 Context；
5. 不通过 Prompt 约束来掩盖 Truth 泄漏。

这意味着 C2 的最小可交付范围应单独设计，而不能把 belief/correction 与更多认知状态一次性加入。

## 与 Prose Lint 的优先级比较

当前 `rules.Lint` 已有两个确定性检查：

- `markdown_residue`
- `non_cjk_fragments`

Commit 与 Revision Projector 都会运行同一个 `rules.Lint`，因此未来增加重复段落等规则的接缝已经成熟，适合后续单独的小里程碑。

但错误信念是已完成 Knowledge 主链中仍缺失的最后一个核心认知维度，而且会直接影响角色一致性。下一步仍优先做 C2a，但必须缩为：

```text
believe：角色形成一个与客观 Truth 不同的明确认知
learn：沿用现有动作；角色获知客观 Truth 时纠正其活跃错误信念
```

不新增 `correct_belief` 动作。原因是现有 `learn` 已有明确语义：角色确实获知客观 Truth；此时该角色对同一 Knowledge ID 的错误信念必然不再是当前认知。这样可避免引入第二种“纠正但是否知道真相”的模糊动作。

第一版仍不支持：

- 一个角色对同一 Truth 同时持有多个错误信念；
- 错误信念内容的中途改写；
- 纠正后再次相信错误内容；
- 不可靠读者信念；
- `forget`、`doubt`、`suspect`。

## C2a 实施结论

C2a 已按最小范围闭环：新增 `KnowledgeBelief` 与 `believe`，复用 `learn` 标记 `CorrectedAt`。Store、ChapterFacts、Commit Saga、Projector、Rewrite、Import、Context 与四份 Prompt 均同步。

实现期间发现两类重要边界：

1. `CommitStageStarted` 恢复不能用已部分应用后的当前投影重复校验冻结 payload，否则合法历史 belief 会被误判；语义校验现只发生在首次创建 PendingCommit 前。
2. Context 不能序列化完整 KnowledgeEntry/KnowledgeBelief；净化 DTO 既隐藏当前角色与读者均未知的 Truth，也去除本章/未来 `CorrectedAt`，避免提前泄露认知纠正。

最终审计还补齐了直接 Store 入口的四动作字段矩阵，避免多余 Truth/Character/Belief 被静默忽略。Import 分析缓存版本已从 4 提升到 5。全量测试、vet 与 diff check 均通过。

---

# 全项目 Review 后续：Import 全书事实门禁（2026-08-24）

## 总体判断

全项目 Review 结论是宏观架构未跑偏，但发现一个 P0 恢复缺口：Import 的局部批次验证不能证明跨批次 ChapterFacts 生命周期合法。

## 证据

`internal/host/imp/analyze.go` 的 `validateBatch` 每批从空的 Truth/KnownBy/Belief map 开始。`start > 0` 时未知引用被有意放行，前批次状态仅以 Prompt ledger 形式提供给模型。因此下列跨批次错误不能被确定性拒绝：

```text
批次 1：establish k → 林墨 learn k
批次 2：林墨 believe k
```

当前 runner 在 synthesis 前只检查分析工件数量，在 publish 前先执行 `publishFoundation`，再逐章走 Commit。非法事实会在正式 Store 已被部分修改后才失败。工作区分析和 synthesis 工件仍保持新鲜，`NextAction` 继续返回 `ActionPublish`，重跑会卡在同一章。

## 可复用接缝

- `revision.ValidateRecordSet(records)` 已是纯全书 ChapterFacts 重放接缝，当前被 Rewrite 和 Migration 复用；Import 成为第二个适配器是合理深化，不需要复制生命周期规则。
- `discardAnalysesAfter(w, keep, total)` 已能删除某章后的分析尾部；只需补首个失败章定位与 synthesis/story-resolution 失效。
- `LoadState → Facts → NextAction` 已按连续新鲜分析数推导动作；删除非法尾部后会自然回到 `ActionAnalyze`，无需新 Action/Stage。
- Import 只允许导入空书，因此 publish 前失败可以证明正式 Store 零污染，不需要回滚既有作品。

## 设计取舍

1. 本批只解决 P0 门禁，不顺带实施 Knowledge/Foreshadow pure apply/replay 深化；后者单列下一架构里程碑，避免把故障修复扩大成跨包重构。
2. `validateBatch` 保留批次形状、章号和本批字段校验；全书领域不变量交给 `revision.ValidateRecordSet`。
3. 必须建立唯一 ImportedChapterFacts→ChapterFacts 映射，并让门禁与 publish 参数共享，防止验证和发布语义漂移。
4. 候选批次和 salvage prefix 都要在落盘前与 prior facts 合并重放。
5. synthesis/publish 仍做防御性复验，用于旧缓存、手工修改或历史 Bug 工件。
6. 全书重放失败时，正常路径一次 O(n) 验证；只在异常路径逐前缀定位首个失败章，允许 O(n²)。
7. 分析 Schema 版本从 5 提升到 6；workspace 与 ChapterRecord 版本保持不变。
8. range digest 本身绑定 facts 输入；阶段 36 将先用测试证明它自然失鲜，若证明成立则不做冗余删除。

## 延后事项

- Knowledge/Foreshadow 生命周期规则去重；
- PendingCommit payload/draft digest；
- CONTEXT.md 与历史计划归档；
- Knowledge 诊断/TUI/导出；
- Prose Lint、平台 Rubric 和 cocreate 访谈。

这些问题有价值，但均不应混入 P0 Import 修复。

## 里程碑 D 实施结论

P0 缺口已闭合：Import 现在把模型批次的局部 Schema/章号校验与正式 ChapterRecord 全书重放分层处理。

```text
validateBatch
→ importedChapterFacts 唯一映射
→ revision.ValidateRecordSet
```

正常 AnalyzeNext 的候选批次和长度截断 salvage prefix 都必须与既有前缀一起重放，只有通过后才写分析工件。Synthesis 在任何模型调用前复验，Publish 在 resolveStory/Foundation/Hold/Commit 前复验。

旧工件或手工修改在第 N 章首次非法时，系统保留 1..N-1，删除 N 章及后续分析，并失效 synthesis/story-resolution；现有 LoadState/NextAction 自动回到 ActionAnalyze。Range digest 只消费 compact narrative evidence，不消费 Knowledge/Foreshadow tracking 字段，因此跟踪事实修正不要求主动删除 range cache，其 InputDigest 语义已有回归测试锁定。

分析缓存版本已从 5 提升为 6；workspace 和 ChapterRecord 版本不变。全量测试、vet、关键包 race、diff check 与范围扫描全部通过。
