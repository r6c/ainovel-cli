# ainovel-cli 当前稳定发现

## 当前基线

- Git 基线：`a20fad7 文档：继续归档并收敛稳定工作记忆`
- 当前工作区：仅有本轮规划文件和历史归档变更；未修改生产代码、测试代码或发布配置。
- 项目定位：本地文件系统驱动、可恢复、可审计的 AI 小说创作运行时。
- 核心边界：模型负责开放语义；代码负责状态、约束、事务、恢复和验证。

## 本轮规划决定（2026-08-26）

用户要求继续处理三项候选：

1. 稳定工作记忆继续归档；
2. Context selection policy 二次深化；
3. Import 认知 A/B 解释与样本扩展。

三项不合并为一个大提交：

```text
G2 稳定工作记忆归档
  ↓
C2 Context selection policy 二次深化

U2 Import 认知 A/B 解释与样本扩展
  ↘ 独立、可暂停、不阻塞发布
```

当前 GoReleaser snapshot 仍是独立发布事项，不纳入本轮研究/重构范围。

## 归档结果

本轮规划前的根工作记忆已完整复制到：

```text
docs/history/plans/2026-08-pre-release-candidates/
```

归档包含：

- `task_plan-before-20260826.md`
- `findings-before-20260826.md`
- `progress-before-20260826.md`
- `README.md`

归档保留旧阶段、工具错误和当时状态；这些内容不覆盖根目录当前计划。

## Context 二次深化边界

已有第一轮深化：

```text
internal/tools/context_knowledge_policy.go
selectKnowledgeBoundaries
```

它负责 Knowledge 的选择、时间过滤、ReaderKnown/CharacterKnown 净化和 8 条上限；`ContextTool` 继续负责 IO、错误降级、预算和 JSON Envelope。

本轮只通过输入/决策/输出矩阵和 deletion/decision trace 测试判断是否值得继续深化。即使深化，也不创建：

- `ContextService`
- Repository
- 通用策略框架
- 第二个 JSON Envelope
- 新事实源

必须保持：

- 隐藏 Truth 不泄露；
- 当前章/未来信息不提前进入；
- ReaderKnown 与 CharacterKnown 分离；
- active belief 净化；
- 8 条上限；
- `_trimmed` 预算可观测性。

## Import 认知 A/B 当前证据

已经完成的 12 条样本、baseline/calibrated 三轮 A/B 结果：

```text
36/36 结果有效
Provider 错误：0
超时：0
```

当前结论：

- calibrated 提高 `learn` recall；
- calibrated 降低整体 precision；
- `reveal_to_reader` precision 下降；
- 动作集合完全匹配下降；
- 负例未出现新增明显误报；
- 当前 Prompt 作为折中版本保留，不继续堆规则。

本轮 U2 不重复 12 条样本上的 Prompt 调参，而是：

1. 先解释已有混淆矩阵；
2. 新增 12 条自建样本，扩展到 24 条；
3. 按题材、视角和认知边界分层；
4. 使用可断点 Runner 进行扩展评测；
5. 只有出现稳定、可解释收益时才提出新的 Prompt 修改。

不使用第三方小说原文，不保存完整模型响应或 Provider 凭证。

## U2 阶段 209—210 进度

阶段 209 已完成：新增 `evals/import-knowledge/explanation.md`，仅基于已有聚合统计解释 baseline/calibrated 的动作级权衡，不虚构逐样本预测。

阶段 210 已完成：校准集从 12 条扩展到 24 条，新增样本全部为本项目自建中文片段，覆盖明确角色接受、读者揭示、同一 Truth 多动作、稳定错误信念、未经核验转述、明确不相信和 partial payoff 边界。`labels.json` 现在对 `believe` 同时要求角色与内容。

阶段 211 尚未开始真实 Provider 扩展评测；扩展样本仅通过确定性数据契约和仓库门禁，不能宣称已完成 A/B。

## GoReleaser 环境记录

本轮规划前尝试固定 GoReleaser v2.17.1：

- 直接下载资产名最初猜错，返回 404；随后通过 GitHub API 确认真实资产名；
- 按真实资产下载时在宿主 120 秒窗口内阻塞；
- 本机没有留下 Goreleaser 进程或有效工具文件；
- Go module 安装同版本也在下载依赖时超时；
- 未执行 snapshot，未创建标签或发布。

该事项保留为独立发布任务，不影响本轮规划。

## 稳定架构边界

正式事实与规则仍为：

```text
ChapterRecord.Facts
→ Revision Projector
→ 当前投影

Knowledge/Foreshadow 生命周期
→ 各自专用纯 Apply 函数

Import
→ 统一 ChapterFacts 映射
→ 全书事实重放
→ 发布前门禁

PendingCommit
→ 首次冻结校验 + v2 密封
→ 恢复时密封校验 + 纯载荷校验 + 幂等重放
```

不引入：

- 数据库；
- 通用状态机；
- CRUD Service；
- 浏览器自动化；
- 扫榜；
- 并行写相邻章节；
- 第二套 Import Saga；
- 第二个去 AI 味 Skill。

## 阶段 203—204 文档收口

稳定文档导航核对结果：

- `CONTEXT.md`、README、架构、发布验收、发布说明、升级说明和 CHANGELOG 的相对链接均有效；
- `docs/release-acceptance.md` 的验收小节已整理为连续的 `3.1`—`3.14`；
- `CONTEXT.md` 的 AI 味语义判据小节已修正为 `11.1`，与上级 Prose Lint 章节一致；
- 当前根计划、进度和发现文件只保留稳定状态，详细过程已进入日期归档；
- 归档目录未发现 Provider 凭证、私钥或敏感认证内容。

阶段 204 归档门禁完成后，G2 收口，不修改生产代码。

## C2 阶段 205—208 收口

阶段 205 已建立 `docs/context-policy-decision-matrix.md`，固定 Context 的输入、候选资格、排除、净化、排序/上限、预算和 Envelope 边界。

阶段 206 deletion test 在临时副本中验证：删除 Knowledge 选择/净化会破坏隐藏 Truth 和 Reader/Character 边界；删除预算裁剪会破坏上限与 `_trimmed`；删除 Envelope 装配会破坏公共分区。Budget 完整编排回归通过，删除实验失败来自预期行为缺失，而非环境问题。

阶段 207 决策：现有 `context_knowledge_policy.go` 已提供足够的纯策略边界；没有证据表明新增 Context Service、Repository、通用策略框架或生产决策 trace 能带来杠杆收益，因此不做无条件生产重构。

阶段 208 回归通过：Context、Host、Import、全量测试、vet、race、Markdown 链接和 diff check 均通过；ReaderKnown/CharacterKnown、隐藏 Truth、未来过滤、8 条上限、预算裁剪和 Envelope 行为未变。

C2 收口后，下一项按独立路线进入 U2 阶段 209；U2 仍不阻塞 GoReleaser 发布验收。

## 历史说明

此前 A—X、候选 2/3/4、真实 Provider 验收和 TDD 过程已归档。根目录不再重复保留全部过程日志；如需恢复历史，读取：

```text
docs/history/plans/2026-08-domain-saga-evolution/
docs/history/plans/2026-08-pre-release-candidates/
```
