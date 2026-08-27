# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-26
- 基线：`c8a4212 评测：扩展导入认知动作校准样本`
- 当前里程碑：发布候选产物验收
- 当前阶段：GoReleaser v2.17.1 Snapshot 已完成，六平台产物已验收，发布记录收口中
- 阶段 209 已完成：新增 `evals/import-knowledge/explanation.md`，仅基于现有聚合数据解释 A/B 权衡；不从缺失的逐样本结果反推具体错误。
- 阶段 210 已完成：样本从 12 条扩展到 24 条，新增 12 条完全自建边界样本；`believe` 金标现在要求角色与内容；确定性测试、Import、全量、vet、race 和脱敏检查通过。
- 阶段 205 已完成：新增 `docs/context-policy-decision-matrix.md`，覆盖 Context 输入、候选资格、排除原因、净化、排序/上限、预算裁剪和 Envelope 输出；未修改生产代码。
- 阶段 206 已完成：临时副本删除实验确认 Knowledge 选择/净化、预算裁剪和 Envelope 装配均不可删除；阶段 207 不做无证据的生产重构，不新增 Context Service/Repository/决策 trace。
- 本轮不调用 Provider、不创建 Tag/Release；GoReleaser Snapshot 已完成，主工作区仍需最终门禁。
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
- GoReleaser v2.17.1 Snapshot 已完成，六平台归档、checksum、版本注入和包内容已验收；未创建版本标签或 GitHub Release。
- Release workflow 的 AI 发布说明生成仍是可选发布治理议题，不属于主程序运行时依赖。

## 本轮 G2 计划

### 阶段 201：归档当前快照——complete

本轮规划前的 `task_plan.md`、`findings.md`、`progress.md` 已归档至：

```text
docs/history/plans/2026-08-pre-release-candidates/
```

### 阶段 202：精简稳定工作记忆——complete

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

### 阶段 209：解释已有 A/B——complete

已生成 `evals/import-knowledge/explanation.md`，仅基于现有聚合统计解释 baseline/calibrated 的权衡；没有从缺失的逐样本数据反推具体预测。

### 阶段 210：样本 12→24——complete

新增 12 条完全自建样本，扩展明确 learn、明确 reveal、同 Truth 多动作、未证实转述、belief 和不同题材/视角反例。

### 阶段 211：可断点扩展评测——partial_evidence

新增样本曾完成 72 次真实调用，但本轮协调器未直接调用已提交 Runner，且逐样本结果已清理；结果只保留有限动作级聚合，不宣称 Runner 接入已完成。

### 阶段 212：合并统计与 Go/No-Go——partial_evidence

已形成动作级聚合，但 exact-match、逐样本一致性和新增调用成本不可独立复核；当前仅决定保留 calibrated Prompt，不授权继续修改。

### 阶段 213：U2 文档收口——complete

已提交扩展汇总、报告和证据限制；明确新增结果的逐样本 exact-match、顺序、一致性和成本不可独立复核。

### 阶段 214：最终验证——complete

评测资产、Import、全量测试、vet、race、JSON 结构、Markdown 链接和脱敏检查通过。U2 仍因阶段 211/212 的证据保存限制保持 `partial_evidence`。

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

## U2 当前收口

阶段 211 曾完成 72 次新增样本真实调用，但一次性协调器未直接使用已提交 Runner，且逐样本结果已清理；当前仅保留动作级聚合，exact-match、逐样本一致性和新增调用成本不可独立复核。阶段 212 因此保持 `partial_evidence`，Go/No-Go 为保留 calibrated Prompt、不继续追加规则。阶段 213/214 文档与门禁已完成。

## 下一步

U2 保持部分证据，不继续修改 Prompt；下一独立事项仍为 GoReleaser snapshot 与发行包验收。


## AA：发布后稳定性观察

- 预检确认 `v0.1.1` Release 位于 `r6c/ainovel-cli`，CI/Release/Docker 均成功，资产 7 项。
- 发现 P1：`scripts/install.sh` 仍请求旧仓库 `voocel/ainovel-cli`，指定 `v0.1.1` 时返回 404。
- 已在工作区修正安装脚本为 `r6c/ainovel-cli`，并用实际 `v0.1.1` Darwin arm64、Linux arm64、Windows arm64 资产完成 checksum 验证；安装后二进制的版本/帮助命令通过。
- 该修复尚未进入 `v0.1.1` 已发布资产，必须通过新的 `v0.1.2` 补丁版本远端回归。


### AA 补丁版本收口（2026-08-27）

安装脚本仓库地址已修正；`v0.1.2` 待远端工作流回归。
