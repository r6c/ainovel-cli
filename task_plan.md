# ainovel-cli 当前演进计划

## 当前状态

- 总体状态：`complete`
- 当前里程碑：G——稳定文档与规划历史归档
- 当前阶段：阶段 64—68 全部完成
- 基线提交：`f5e91de 加固：校验章节提交冻结载荷的完整性`
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

## G 之后的候选顺序

1. H：确定性重复段落 Prose Lint。
2. I：Knowledge 最小诊断与导出投影。
3. J：一个具体平台 Rubric 试点。
4. K：在现有 cocreate 上增加阶段化访谈。
5. L：扫榜与拆文独立命令。

## 本批明确不做

- 修改 Go 运行时行为
- 新领域动作或状态
- 数据库、Web 事实源、Service/Repository
- 自动迁移或删除历史规划
- Prose Lint、Rubric、TUI 新功能

## 错误记录

| 错误 | 处理 |
|---|---|
| 搜索 Projector 声明时使用未闭合括号正则 | 停止重复该查询，改用字面搜索确认 `ValidateRecordSet` 与 `Projector.Apply` |
