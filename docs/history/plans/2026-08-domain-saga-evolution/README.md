# 2026-08 领域事实与提交 Saga 演进归档

本目录保存 `ainovel-cli` 从伏笔生命周期升级到 PendingCommit 完整性加固的完整文件规划历史。

## 覆盖里程碑

| 里程碑 | 内容 | 主要提交 |
|---|---|---|
| A | 伏笔生命周期：reinforce / partial_payoff / LastAdvancedAt | `13a775b` |
| B | 作者真相与角色知情 | `6a7b7f7` |
| C1 | 读者揭示与信息差 | `03bf271` |
| C2a | 角色错误信念与纠正 | `3ee475c` |
| D | Import 发布前全书事实重放门禁 | `cafb752` |
| E1 | Knowledge 纯 Apply/Replay 规则收敛 | `a041b5c` |
| E2 | Foreshadow 纯 Apply/Replay 规则收敛 | `429bb4a` |
| F | PendingCommit 冻结载荷完整性 | `f5e91de` |
| G | 稳定文档与规划历史归档启动基线 | 本归档末尾 |

## 文件

- [`task_plan.md`](task_plan.md)：阶段计划、范围边界、验收命令与错误表。
- [`findings.md`](findings.md)：架构盘点、审查发现、领域语义与解决状态。
- [`progress.md`](progress.md)：逐个 TDD 红→绿切片、工具错误与验证记录。

## 使用方式

这些文件是历史审计材料，不是当前执行计划。当前稳定术语与代码导航见仓库根目录 [`CONTEXT.md`](../../../../CONTEXT.md)，当前活跃计划仍见根目录 `task_plan.md`。

历史文件中会同时出现 `planned`、`in_progress`、`complete`、旧工作区状态等文字；它们只表示当时的会话快照，不应覆盖当前仓库事实。
