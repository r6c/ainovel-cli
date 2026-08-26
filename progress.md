# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-26
- 基线：`45de770 修复：使导入分析提示词缓存失效`
- 当前里程碑：G2——稳定工作记忆继续归档与收敛
- 当前阶段：G2 已完成，下一步为 C2 阶段 205
- 本轮不修改生产代码、不调用 Provider、不执行 GoReleaser。
- 阶段 203 已完成：稳定文档相对链接全部可解析；发布验收清单编号已整理为 3.1—3.14；CONTEXT.md 的 AI 味语义判据小节编号已修正为 11.1。

## 已完成主线摘要

A—X、V、W、O、P1、R、S、T、U，以及候选 2/3/4 的详细过程均已归档至：

```text
docs/history/plans/2026-08-domain-saga-evolution/
docs/history/plans/2026-08-pre-release-candidates/
```

当前已完成的主能力包括：

- Knowledge/Foreshadow 生命周期和专用纯 Apply/Replay；
- ReaderKnown、CharacterKnown、Belief 和信息边界；
- Import 全书事实重放、provenance 和零污染发布；
- PendingCommit v2 密封、跨阶段/跨进程恢复；
- Headless/Cocreate/Revision/Import/Deconstruct 真实验收；
- Linux/无头、Docker、UsageTracker 和 BudgetSentinel；
- 确定性 Prose Lint、诊断、脱敏导出和本地拆文；
- Commit、Context、Import 测试按 seam 拆分。

## 当前稳定限制

- Import 认知动作三轮 baseline/calibrated A/B 已完成 36/36 有效结果；calibrated 提升 `learn` recall，但整体 precision、`reveal_to_reader` precision 和动作集合完全匹配下降，因此当前 Prompt 作为折中版本保留，不继续堆规则。
- 真实 Architect 扩弧后的第 3 章 Context 端到端验收仍未完成；代码层 ReaderKnown/CharacterKnown 边界已有确定性测试。
- GoReleaser snapshot 尚未完成，未创建版本标签或 GitHub Release。
- Release workflow 的 AI 发布说明生成仍是可选发布治理议题，不属于主程序运行时依赖。

## 本轮 G2 计划

### 阶段 201：归档当前快照——complete

本轮规划前的 `task_plan.md`、`findings.md`、`progress.md` 已归档至：

```text
docs/history/plans/2026-08-pre-release-candidates/
```

### 阶段 202：精简稳定工作记忆——in_progress

根目录文件将只保留当前稳定事实、当前路线、已知限制和下一步入口；完整过程继续留在日期归档中。

### 阶段 203：稳定文档导航核对——pending

检查 `CONTEXT.md`、`README.md`、架构、发布验收和升级文档的链接及当前状态描述。

### 阶段 204：归档门禁与收口——pending

检查归档完整性、相对链接、敏感信息和过期 active 状态；之后再进入 Context 二次深化或 Import 样本扩展。

## C2 计划：Context selection policy 二次深化

### 阶段 205：输入/决策/输出矩阵——pending

使用已拆分的 Context 测试面记录：

```text
事实输入
→ 候选资格
→ 排除原因
→ 净化字段
→ 排序/上限
→ 预算裁剪
→ Envelope 输出
```

### 阶段 206：删除与决策追踪测试——pending

验证删除选择、角色匹配、排序、trace 和预算逻辑是否会造成泄漏、复杂度转移或诊断退化；不以文件大小单独驱动重构。

### 阶段 207：条件性策略深化——pending

只有阶段 206 证明存在真实缺口时才改生产结构；不创建 ContextService/Repository，不改变公开 Envelope。

### 阶段 208：Context 回归与收口——pending

全量测试、vet、race、diff check；若无需进一步深化，则以测试/文档结论收口。

## U2 计划：Import 认知 A/B 解释与样本扩展

### 阶段 209：解释已有 A/B——pending

不调用 Provider，按混淆矩阵和语义类别解释 baseline/calibrated 的权衡。

### 阶段 210：样本 12→24——pending

新增 12 条完全自建样本，扩展明确 learn、明确 reveal、同 Truth 多动作、未证实转述、belief 和不同题材/视角反例。

### 阶段 211：可断点扩展评测——pending

复用既有 Runner；每条结果原子落盘，Prompt 身份隔离，错误可续跑，不保存原始响应或凭证。Provider 阻塞达到停止条件即暂停。

### 阶段 212：合并统计与 Go/No-Go——pending

只有扩大样本后仍出现稳定、可解释收益，才提出下一轮 Prompt 修改；否则保留当前折中版本。

### 阶段 213：U2 文档收口——pending

只提交脱敏标签、聚合统计和解释。

## 发布事项

GoReleaser 发布验收独立于本轮：

```text
固定 GoReleaser v2
→ goreleaser check
→ goreleaser release --snapshot --clean
→ 六平台产物验收
→ 再决定 v0.1.0-rc.1
```

本轮固定 v2.17.1 的下载尝试因网络/依赖下载在工具时限内未完成；未安装系统工具、未修改 PATH、未创建标签、未发布远端。

## 近期错误记录

| 事项 | 处理 |
|---|---|
| 归档快照前根计划包含大量历史状态 | 已完整复制后压缩根工作记忆 |
| GoReleaser 直接资产名最初猜错 | 通过 GitHub API 确认真实资产名；后续下载仍受网络阻塞 |
| GoReleaser module 安装下载依赖超时 | 未重复无界下载，保留为独立发布门禁 |
| 历史文件中的旧 Provider 状态可能污染当前导航 | 历史保留在归档，根进度仅保留稳定当前状态 |

## 下一步

先完成 G2 阶段 203—204 的稳定文档导航和归档门禁；随后进入 C2 阶段 205 的 Context 输入/决策/输出矩阵。U2 独立、可暂停，不阻塞发布候选。
