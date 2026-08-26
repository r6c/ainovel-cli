# ainovel-cli 当前演进计划

## 当前状态

- 总体状态：`planned`
- 当前基线：`a20fad7 文档：继续归档并收敛稳定工作记忆`
- 工作区：C2 已完成；当前无未提交生产代码变更，本轮不调用 Provider、不执行 GoReleaser。
- 已完成主线：领域事实、Commit Saga、Import、Context、Linux/无头、真实 Headless/Cocreate/Revision/Import/Deconstruct、测试资产整理。
- 当前发布事项：GoReleaser snapshot 尚未验收，未创建版本标签。

## 稳定架构边界

1. `ChapterRecord.Facts` 是派生事实重建输入；JSON/Markdown 当前状态文件是可重建投影，不是第二事实源。
2. Knowledge 与 Foreshadow 生命周期分别由专用纯 Apply 函数裁决。
3. Commit 首次冻结执行纯载荷与当前状态校验；恢复执行密封、纯载荷校验和幂等重放。
4. Import 通过统一 ChapterFacts 映射和全书重放门禁发布；`generated/imported/user` 正文来源权限不同。
5. Context 负责读取、选择、净化、预算裁剪和 Envelope；Writer 不直接消费完整 KnowledgeEntry。
6. 不引入数据库、通用状态机、CRUD Service、浏览器自动化、扫榜或并行相邻章节写作。

## 已完成里程碑摘要

A—X、V、W、O、P1、R、S、T、U 的已完成部分、候选 2/3/4 与真实验收过程见：

```text
docs/history/plans/2026-08-domain-saga-evolution/
docs/history/plans/2026-08-pre-release-candidates/
```

## 本轮规划总览

```text
G2：稳定工作记忆继续归档与收敛
        ↓ 提供稳定导航和历史隔离
C2：Context selection policy 二次深化
        ↘ 仅在 deletion/decision test 证明值得时改生产结构
U2：Import 认知 A/B 解释与样本扩展（独立、可暂停、不阻塞发布）
```

U2 不依赖 C2；二者不得合并成一个大提交。GoReleaser 发布验收另列为发布事项，不在本轮自动执行。

# 里程碑 G2：稳定工作记忆继续归档与收敛

目标：让根目录文件只承担当前导航，把完整过程、旧状态和工具错误继续放进带日期的历史目录。

## 阶段 201：归档当前快照

状态：`complete`

已将本轮规划前的 `task_plan.md`、`findings.md`、`progress.md` 归档至：

```text
docs/history/plans/2026-08-pre-release-candidates/
```

历史快照只作为当时记录，不覆盖当前根计划。

## 阶段 202：精简稳定工作记忆

状态：`complete`

根目录只保留：

- 当前基线和工作区状态；
- 稳定架构边界；
- 当前活跃里程碑；
- 已知限制；
- 下一步入口。

不在根文件继续堆叠逐工具调用日志。完整过程仍留在历史归档。

## 阶段 203：稳定文档导航核对

状态：`complete`

核对 `CONTEXT.md`、`README.md`、`docs/architecture.md`、`docs/release-acceptance.md` 的链接和当前状态描述；只修正文档事实漂移，不修改运行时。

## 阶段 204：归档门禁与收口

状态：`complete`

验证：

```text
根计划无过期 active 状态
历史归档可定位
Markdown 相对链接有效
敏感信息和 Provider 凭证未进入归档
```

建议提交信息：

```text
文档：继续归档并收敛稳定工作记忆
```

# 里程碑 C2：Context selection policy 二次深化

目标：在已有 `context_knowledge_policy.go` 的基础上，验证并改善“选择决策可解释性、策略局部性和回归可观察性”，不创建 `ContextService`、Repository 或第二个 Envelope。

## 阶段 205：输入/决策/输出矩阵

状态：`complete`

基于已拆分的 Context 测试资产，补充决策矩阵：

```text
输入事实
→ 候选资格
→ 排除原因
→ 净化字段
→ 排序/数量上限
→ 预算裁剪
→ Envelope 输出
```

覆盖 Knowledge、Foreshadow、Recall、References、SimulationProfile 和 UserRules；先只写测试/文档，不改生产结构。

## 阶段 206：删除与决策追踪测试

状态：`complete`

通过 deletion test 和行为 trace 判断当前策略是否仍有真实深化空间：

- 删除候选过滤是否泄露未来/隐藏 Truth；
- 删除角色匹配是否扩大无关知识；
- 删除排序是否破坏最近优先和 8 条上限；
- 删除策略 trace 是否使误选无法诊断；
- 预算裁剪是否仍能记录 `_trimmed`。

如果删除只会造成测试断言搬家，不进行生产重构。

## 阶段 207：条件性策略深化与收口

状态：`complete`

仅当阶段 206 证明值得时，才在现有 `internal/tools` 包内做小范围深化：

- 保留 `ContextTool.Execute` 公共入口；
- 保留 `context_knowledge_policy` 作为纯策略边界；
- 如确有需要，增加结构化决策 trace，仅供诊断/测试，不改变公开 Envelope；
- 不把 trace 变成作者事实，不把策略变成新 Service；
- 保持 ReaderKnown/CharacterKnown、未来过滤、8 条上限和预算语义完全兼容。

如果 deletion/trace 测试证明现有结构足够，则本里程碑只更新测试和文档，明确“不再深化”。

## 阶段 208：Context 二次深化回归

状态：`complete`

运行：

```bash
go test ./internal/tools -run 'Context|NovelContext' -count=1 -timeout=5m
go test ./internal/host ./internal/host/imp -count=1 -timeout=5m
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/tools ./internal/host/imp -count=1 -timeout=10m
git diff --check
```

建议提交信息（仅有实际生产深化时）：

```text
重构：增强 Context 选择策略可解释性
```

# 里程碑 U2：Import 认知 A/B 解释与样本扩展

目标：在不继续堆 Prompt 规则的前提下，扩大自有样本并解释现有 A/B 权衡；评测线独立、可断点、可暂停，不阻塞 Release Candidate。

当前基线：

```text
12 条自建样本
baseline/calibrated × 3 轮
36/36 结果有效
```

已知结论：calibrated 提高 `learn` recall，但整体 precision、`reveal_to_reader` precision 和动作集合完全匹配下降；当前 Prompt 暂不继续修改。

## 阶段 209：评测结果解释

状态：`planned`

对现有 `ab-summary.json` 做脱敏再分析，输出：

- 按样本的动作集合差异；
- 按语义类别的混淆矩阵；
- `establish` 与“被转述说法”的边界错误；
- `learn` 漏报/补报类型；
- `reveal_to_reader` 误报类型；
- `believe` 误报是否集中于角色误解叙述；
- baseline/calibrated 的三轮一致性；
- Provider 错误率与 Usage 分布。

不重新调用 Provider，不修改生产 Prompt。

## 阶段 210：扩展自建样本

状态：`planned`

从 12 条扩展到 24 条，新增 12 条独立样本，按现有混淆分层：

- 明确角色接受 Truth；
- 角色听到但不相信；
- 读者明确获知但角色未知；
- 同一 Truth 多动作；
- 转述、谎言、未经证实广播；
- 稳定错误信念与客观 Truth 分离；
- 普通世界设定不等于 reader reveal；
- 不同题材/视角/叙述可靠性反例。

样本、金标、类别说明和评测结果严格分离；只使用自有文本，不引入第三方小说原文。

## 阶段 211：可断点有限扩展评测

状态：`planned`

使用既有 Runner，执行新增 12 条的：

```text
baseline/calibrated × 3 轮
```

要求：

- 每条结果立即落盘；
- Prompt 名称/摘要变化时不得复用旧结果；
- 缺结果 fail-loud；
- 单条 Provider 错误记录并可续跑；
- 不保存完整模型响应、正文或凭证。

停止条件：连续两条 Provider 阻塞、单条超过 180 秒、结构化自愈超过上限。

## 阶段 212：合并统计与 Go/No-Go

状态：`planned`

将新增 12 条与原 12 条分别统计并合并，报告：

- precision/recall；
- exact action set；
- 多动作顺序；
- 负例误报；
- `believe` 与 `reveal_to_reader` 混淆；
- 题材/视角分层差异；
- Provider 可用性和成本。

只有在扩大样本后仍出现稳定、可解释的收益，才允许提出下一轮 Prompt 修改；否则保留当前折中 Prompt。

## 阶段 213：U2 文档收口

状态：`planned`

更新：

- `evals/import-knowledge/report.md`；
- `evals/import-knowledge/README.md`；
- `CONTEXT.md`；
- `findings.md`；
- `progress.md`。

只提交脱敏样本标签、聚合统计和解释，不提交完整模型响应、Provider 配置或原始临时工作区。

建议提交信息：

```text
评测：扩展并解释导入认知动作基线
```

# 共同门禁与范围边界

## 阶段 214：最终验证

状态：`planned`

运行：

```bash
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/tools ./internal/host/imp ./internal/eval -count=1 -timeout=10m
git diff --check
```

## 不进入本轮

- 不创建 GoReleaser Tag/Release；
- 不修改运行时 Import Prompt，除非 U2 Go/No-Go 明确授权；
- 不创建 ContextService、Repository 或通用策略框架；
- 不改变 Context JSON Envelope；
- 不新增 Knowledge/Foreshadow 动作；
- 不引入第二个去 AI 味 Skill；
- 不恢复扫榜、Chrome/CDP、浏览器登录态；
- 不引入数据库、Web 事实源或新的 E2E 框架；
- 不把 Provider 阻塞伪装成质量结论。

## 后续顺序

```text
G2：稳定工作记忆归档
→ C2：Context selection policy 二次深化
→ U2：Import A/B 解释与样本扩展
→ GoReleaser snapshot
→ 再决定 v0.1.0-rc.1
```

当前入口：U2 阶段 209——解释已有 Import 认知 A/B。
