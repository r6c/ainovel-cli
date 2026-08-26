# Context Selection Policy 映射

> 候选 2 阶段 185 产物。该文档描述当前 `novel_context` 的输入、选择、净化、裁剪与输出边界；不定义新的事实源或公共接口。

## 1. 总编排入口

```text
internal/tools/novel_context.go
  ContextTool.Execute
    ├─ chapter > 0
    │   ├─ buildBaseContext
    │   ├─ prepareChapterContext
    │   ├─ buildChapterContext
    │   └─ 读取 rule_violations
    └─ chapter == 0
        ├─ buildProgressStatus
        └─ buildArchitectContext

  buildSimulationProfile
  buildUserRules
  trimByBudget
  buildLoadingSummary
  json.Marshal
```

`ContextTool.Execute` 是适配器入口，负责错误汇总、预算裁剪和最终 JSON 输出；它不应重新定义领域生命周期。

## 2. 输入矩阵

| 输入 | 读取位置 | 消费者 | 选择/约束 |
|---|---|---|---|
| Book/ Premise/ Outline | `store.Book`、`store.Outline` | Writer、Architect | 基础设定；章节路径不加载全量原文之外的额外来源 |
| Progress/RunMeta | `store.Progress`、`store.RunMeta` | 两种模式 | 当前阶段、章节、规划层级、预算窗口 |
| Chapter Outline | `Outline.GetChapterOutline` | Writer | 角色匹配、当前章合同与下一章衔接 |
| ChapterPlan/Contract | `Drafts.LoadChapterPlan` | Writer | 当前章 required/forbidden/emotion/payoff/hook 约束 |
| Chapter summaries | `Summaries` | Writer、Architect | 按章节窗口或卷/弧层级裁剪 |
| Timeline | `World.LoadRecentTimeline` | Writer | 最近时间线窗口 |
| Foreshadow | `World.LoadActiveForeshadow` | Writer、Architect | 活跃伏笔；Recall 触发时与 story threads 协同 |
| Knowledge | `World.LoadKnowledgeState` | Writer | `selectKnowledgeForCurrentOutline` 净化，最多 8 条，过滤未来信息 |
| Relationships/StateChanges | `World` | Writer | 最近变化窗口；相关章节选择使用摘要维度 |
| Characters/Cast | `Characters`、`Cast` | Writer、Architect | Tier 过滤、快照优先、最近活跃配角上限 |
| Style/AuthorRevisionStyle | `World`、`Drafts` | Writer、Editor | 当前风格规则优先；无规则时回退 anchors/voice samples |
| UserRules | `UserRules.Load` | Writer、Architect | 只注入 `structured + preferences`，不注入 sources/conflicts |
| References | `ContextTool.refs` | Writer、Architect | 按章节/模式选择；平台 rubric 仅显式 `fanqie` |
| SimulationProfile | `Simulation.Load` | Writer、Architect | 只注入 compact profile，不注入 source reports |
| Rule violations | `World.LoadRuleViolations` | Writer、Editor | 机械事实；由 Editor 语义判断，不在 Context 中裁决 |
| Reviews/Rewrite brief | `World.LoadReviewsAffectingChapter` | Rewrite Writer | 仅 rewrite 章节注入；正文按需读取 |
| Budget | `trimByBudget` | Writer、Architect | 低优先级字段依次裁剪，记录 `_trimmed` |

## 3. 选择与净化层

### 章节路径

```text
prepareChapterContext
  → 读取状态和当前 OutlineEntry
  → 载入 Knowledge/Foreshadow/Relationship/StateChange
  → selectStoryThreads
  → story thread 过稀时回退

buildChapterEpisodicMemory
  → selectKnowledgeForCurrentOutline
  → 活跃伏笔或 selected memory
  → recent cast
  → related chapters
  → layered position

buildChapterReferencePack
  → AuthorRevisionStyle 或 style anchors/voice samples
  → writerReferences

buildChapterSelectedMemory
  → story threads
  → review lessons
```

### Architect 路径

```text
buildArchitectPlanning
  → layered outline / flat outline
  → compass
  → volume/arc summaries
  → completion signals

buildArchitectFoundation
  → Book/Premise/Characters/World/Foreshadow
  → foundation status
  → writer feedback

buildArchitectReferences
  → style rules
  → architectReferences
```

### 净化边界

- Knowledge 不直接序列化完整 `KnowledgeEntry`；使用 `knowledgeBoundary`。
- 当前角色或读者已知才可输出 Truth。
- active belief 可以在 Truth 隐藏时单独输出。
- 不输出当前章或未来才发生的 learn/reveal/correction。
- 平台 rubric 必须由显式 UserRules 选择。
- SimulationProfile 使用 compact 投影，禁止 source reports。
- UserRules 只输出创作所需字段。

## 4. 预算与序列化

```text
候选数据
→ 优先级裁剪
→ `_trimmed` 记录
→ `_loading_summary`
→ JSON envelope
```

章节预算为 100 KiB，Architect 预算为 60 KiB。低优先级顺序从 `references` 开始，随后是 voice/style/previous tail/timeline/state/foreshadow/knowledge/relationship 等；核心合同、Progress 和必要结构不主动删除。

## 5. 测试覆盖映射

| Policy 部分 | 当前测试文件 |
|---|---|
| Knowledge 净化与信息边界 | `context_knowledge_test.go`、`novel_context_reader_boundary_test.go` |
| Recall/伏笔/Review | `context_recall_test.go` |
| Budget/裁剪 | `context_budget_test.go` |
| References/平台 | `context_references_test.go` |
| Modes/UserRules/Style | `context_modes_test.go` |
| Envelope/规则事实 | `context_envelope_test.go` |
| 错误与降级 | `context_errors_test.go` |
| SimulationProfile | `novel_context_simulation_test.go` |

## 6. 阶段 185 结论

- 选择、净化、预算和序列化均已有可直接运行的测试 seam。
- `ContextTool.Execute` 仍适合作为适配器，不需要替换。
- 若深化，优先集中“选择与净化”纯逻辑；不要把 IO、预算和 JSON envelope 混进新模块。
- 是否生产深化由阶段 186 deletion test 决定；不能仅以文件行数为理由拆分。
