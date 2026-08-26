# Context Selection Policy 决策矩阵

本文件记录 `novel_context` 当前的输入、候选选择、排除、净化、排序、预算和输出边界。

## 总体管线

```text
Store / Bundle / 当前章节
→ 读取与错误降级
→ 候选选择
→ 时间与角色边界过滤
→ 脱敏/净化
→ 排序与数量上限
→ Envelope 装配
→ Budget 裁剪
→ JSON 输出
```

`ContextTool.Execute` 是唯一公共适配器；本文件不是新的事实源或策略接口。

## 输入与输出总表

| 输入类别 | 来源 | 选择依据 | 排除条件 | 输出位置 | 预算行为 |
|---|---|---|---|---|---|
| 当前章节大纲 | `Outline` | `chapter=N` 的当前 `OutlineEntry` | 章节不存在时按既有错误降级 | `working_memory.current_chapter_outline` | 核心字段，非低优先级裁剪项 |
| 章节计划/合同 | `Drafts` | 当前章计划存在且有有效合同内容 | 空计划或无合同内容不输出合同对象 | `working_memory.chapter_plan` / `chapter_contract` | 随工作记忆处理 |
| Knowledge | `knowledge_state` | 当前大纲涉及角色；或读者已知 | Truth 尚未建立、当前章/未来建立、角色/读者均未知、超 8 条 | `episodic_memory.knowledge_boundaries` | 可裁剪，记录 `_trimmed` |
| Foreshadow | `foreshadow_ledger` | 当前大纲关键词、ID、描述词；不足时久挂回填 | 已回收、无相关性且未达到久挂阈值、超过召回上限 | `episodic_memory.foreshadow_ledger` / `selected_memory.story_threads` | 依现有 Envelope 优先级裁剪 |
| 时间线/摘要 | `Timeline`、`Summaries` | 当前章前的近期窗口、分层摘要 | 当前章之后、超窗口且无其他关联 | `episodic_memory` / `working_memory` | 依现有优先级裁剪 |
| 角色状态 | `Characters`、状态变化投影 | 当前章大纲角色匹配、最近变化 | 无关角色、未来变化 | `episodic_memory` | 依现有优先级裁剪 |
| 关系状态 | `relationship_state` | 当前章至少两名匹配角色及其历史关系 | 无关角色对、未来变化 | `episodic_memory.relationship_state` | 依现有优先级裁剪 |
| Review 经验 | `ReviewEntry` | 当前章相关历史 Review、影响当前章的弧评审 | 非 warning/error 类问题、超回填上限 | `selected_memory.review_lessons` | 依 `selected_memory` 优先级裁剪 |
| UserRules | `meta/user_rules.json` | 始终注入稳定结构 | `sources/conflicts` 不进入模型 Context | `working_memory.user_rules` | 按工作记忆优先级处理 |
| SimulationProfile | `simulation_profile.json` | 有画像且当前模式允许 | 无画像、损坏可选投影 | `working_memory` 或 `planning_memory` | 可随对应记忆区裁剪 |
| Platform Rubric | `References` + `structured.platform` | 仅显式 `fanqie` | 未指定或未知平台 | `reference_pack.references.platform_rubric` | 随 References 裁剪 |
| Rule Violations | `rule_violations` | 当前章已有机械事实 | 没有违规时不输出 | 顶层 `rule_violations` | 作为事实输入，不自行裁决 |

## Knowledge 决策矩阵

| 条件 | 候选资格 | 净化输出 |
|---|---:|---|
| `EstablishedAt >= chapter` | 否 | 不输出，避免未来 Truth 泄露 |
| 当前角色在 `KnownBy` 且 `LearnedAt < chapter` | 是 | 输出 Truth、KnownBy、首次获知章 |
| `ReaderRevealedAt > 0 && ReaderRevealedAt < chapter` | 是 | 输出 Truth、ReaderRevealedAt；不自动增加角色 KnownBy |
| 当前角色只有 active belief，读者未知 | 是 | 输出 ID、active belief、形成章；不输出 Truth |
| 当前角色 belief 已在本章或未来纠正 | 当前认知视图否 | 不输出未来 `CorrectedAt` 或已纠正 belief |
| 只有无关角色知道，读者未知 | 否 | 不输出 |
| 角色/读者均未知但存在作者 Truth | 否 | 不输出 |
| 多于 8 条合格 Knowledge | 前 8 条 | 按当前投影倒序取最近条目 |

Knowledge 纯策略位于：

```text
internal/tools/context_knowledge_policy.go
selectKnowledgeBoundaries
```

该策略不执行 IO、预算裁剪或 JSON 序列化。

## Foreshadow 决策矩阵

| 步骤 | 条件 | 输出 |
|---|---|---|
| 相关性召回 | ID 或描述词命中当前章节 focus | `story_thread`，按当前扫描顺序加入 |
| 久挂回填 | 活跃且 `chapter-lastAdvanced >= threshold` | 最久未推进优先回填 |
| 去重 | 相同 Key/Kind/Summary 已选 | 不重复加入 |
| 数量上限 | 达到 `maxThreads` | 停止继续召回 |
| 无触发 | 伏笔数量不足或无当前触发 | 保留完整伏笔账本的既有路径 |

`resolved` 条目不会作为活跃伏笔召回；停滞时间优先使用：

```text
LastAdvancedAt > 0 ? LastAdvancedAt : PlantedAt
```

## References / Simulation / UserRules

### References

```text
基础参考
→ 题材与风格参考
→ Anti-AI 参考
→ 显式平台 Rubric
```

平台 Rubric 只有在：

```text
structured.platform == "fanqie"
```

时进入 Context。未指定平台时，代码层不注入 Rubric 文本。

### SimulationProfile

只注入 compact profile：

```text
SimulationProfile
→ 抽象方法画像
→ 不注入 source_reports
→ 不注入本地原始语料
```

### UserRules

只注入：

```text
structured
preferences
```

不注入：

```text
sources
conflicts
```

避免把规则来源诊断细节暴露给 Writer。

## 错误降级矩阵

| 数据 | 不存在 | 损坏/读取失败 |
|---|---|---|
| 可选 SimulationProfile | 静默不输出 | warning，继续构建 Context |
| 可选 Review/Recall 数据 | 静默不输出 | warning，继续构建 Context |
| Knowledge 当前投影 | 按空投影处理 | 核心读取错误，按现有 `contextReads` 处理 |
| 当前大纲/核心 Book | 依现有核心要求返回错误 | 返回错误，不伪造上下文 |
| UserRules | 使用内置默认快照 | 核心错误，拒绝不一致结果 |

## Budget 与 Envelope

候选被选中并净化后，才进入现有 Envelope：

```text
chapter > 0
→ working_memory / episodic_memory / reference_pack / selected_memory

chapter == 0
→ planning_memory / foundation_memory / reference_pack
```

预算裁剪规则：

- Writer 上限：100KB；
- Architect 上限：60KB；
- 低优先级字段按既有顺序移除；
- 每次移除记录到 `_trimmed`；
- 不让 `knowledge_boundaries` 或 `platform_rubric` 绕过预算；
- 不裁剪当前章节大纲等核心契约字段，除非既有裁剪策略明确允许。

## 现有测试覆盖

| Seam | 测试文件 |
|---|---|
| Knowledge/Belief/ReaderKnown | `context_knowledge_test.go`、`novel_context_reader_boundary_test.go` |
| Recall/Foreshadow/Review | `context_recall_test.go` |
| Budget | `context_budget_test.go` |
| References/Platform | `context_references_test.go` |
| Modes/UserRules/Style | `context_modes_test.go` |
| Envelope/RuleViolations | `context_envelope_test.go` |
| Errors/降级 | `context_errors_test.go` |
| Simulation 隔离 | `novel_context_simulation_test.go` |

## 稳定边界

本矩阵不意味着新增公共策略接口。后续如进行决策 trace，只能满足：

- 供测试/诊断解释选择与排除；
- 不进入作者事实；
- 不改变现有 JSON Envelope；
- 不输出完整隐藏 Truth；
- 不替代 `ContextTool.Execute`；
- 不创建 Context Service/Repository。
