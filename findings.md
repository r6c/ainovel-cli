# ainovel-cli 稳定发现

## 总体结论

项目没有跑偏，仍是“事实层确定、语义层自主”的本地 AI 小说创作运行时：模型负责开放创作与语义判断，代码负责状态、约束、事务、恢复和验证。

完整演进与 TDD 发现已归档：

- [`docs/history/plans/2026-08-domain-saga-evolution/findings.md`](docs/history/plans/2026-08-domain-saga-evolution/findings.md)

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

普通提交和 Rewrite 共用持久化 Saga。密封 v1 包含：

- PayloadDigest：compact JSON payload 的 SHA-256
- DraftDigest：正文快照 UTF-8 的 SHA-256
- IntentDigest：Chapter、Rewrite、RewriteMode 的 SHA-256

首次冻结前执行纯载荷和当前状态语义校验；恢复验证密封与纯载荷后按 Stage 幂等重放，不根据已部分应用的当前投影重新裁决历史意图。

旧 `started/state_applied` 工件在纯载荷通过后先升级密封；`progress_marked/signal_saved` 只收尾结果。完整性失败保留工件并返回 `ErrPendingCommitIntegrity`。

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

## 保持不动的架构边界

- Engine/Route 继续按 Store 事实确定性派发。
- Architect、Writer、Editor 保持三个自主 Worker；Arbiter 仅做边界清晰的语义裁定。
- 章节正文串行提交，不并行生成有叙事依赖的相邻章节。
- 文件 Store 继续作为单机事实层，不引入数据库。
- Prompt 解释语义，代码执行引用、字段、生命周期、事务和恢复约束。
- CLI/TUI/Web（若未来出现）只能是 Adapter 或投影，不能成为新事实源。

## Prose Lint 当前边界

`rules.Lint` 是内置、始终执行的产品底线检查；`rules.Check` 是用户结构化规则检查。两者都只返回 `Violation` 事实，不阻断流程或创建新 verdict。

重复段落首版已完成：按非空正文行识别段落，仅检测 TrimSpace 后完全相同且至少 24 个 Unicode 字符的内容。短句、标题和相似段落留给 Editor 语义判断；不做跨章累计或模糊相似度。Target 最多保留前 48 字加省略号，Commit 与 Revision Projector 共用同一 Lint，Editor 将 warning 映射到现有 aesthetic 维度。

## 后续优先级

核心领域与事务已完成一轮加固。下一项建议转向低风险质量增量：

1. Knowledge 最小诊断与导出。
2. 具体平台 Rubric 试点。
3. 扩展现有 cocreate 访谈。
4. 扫榜与拆文独立命令。

暂不增加 `doubt/suspect/forget/reader belief` 等认知动作，除非出现明确产品需求。
