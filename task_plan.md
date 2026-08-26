# ainovel-cli 当前演进计划

## 当前状态

- 总体状态：`complete`
- 当前路线：候选 2 Context 选择策略已收敛；候选 3 Provider 评测按需暂停
- 当前阶段：候选 2——Context selection policy（complete）
- 基线提交：`5437350 重构：完成测试资产按接缝整理`
- 当前工作区：仅规划/发现/进度记录有未提交变更；未开始生产代码修改。
- 已完成发布候选稳定化：X（阶段 174—179 complete）
- 当前不创建版本标签；候选 2/3/4 先完成规划与低风险准备。

## 已完成里程碑摘要

A—X 已完成，完整过程见：

```text
docs/history/plans/2026-08-domain-saga-evolution/
```

最近完成：

- V：跨进程 PendingCommit 恢复验收
- W：本地拆文方法画像真实验收
- X：发布候选稳定化与交付检查

已知但不阻塞主线的限制：

- Import 认知动作完整三轮 precision/recall 受 Provider 长连接/HTTP 502 阻塞；已有定向 Prompt 修订和真实两章回归证据。
- 真实 Architect 扩弧后的第 3 章 Context 端到端验收受 Provider 阻塞；代码层 Context 边界已有测试。
- 当前尚未创建版本标签或 GitHub Release。

## 稳定架构边界

1. `ChapterRecord.Facts` 是派生事实的重建输入；正文与事实变更必须经过 Commit/Revision 接纳。
2. `knowledge_state.json`、`foreshadow_ledger.json` 等是可由 ChapterRecords 全量重建的当前投影，不是第二事实源。
3. Knowledge 与 Foreshadow 生命周期分别由专用纯 Apply 函数唯一裁决。
4. Import 在分析、综合、发布前按章重放正式 ChapterFacts；非法尾部失效后回到 Analyze。
5. PendingCommit 首次冻结前执行纯载荷 + 当前状态校验；恢复执行密封 + 纯载荷校验后幂等重放。
6. Writer 只消费净化后的 `knowledge_boundaries`；当前角色与读者都未知的 Truth 不能泄露。
7. 不引入数据库、通用状态机、CRUD Service、浏览器自动化、扫榜或并行相邻章节写作。

## 候选依赖总览

```text
候选 4：测试按接缝拆分
        │ 提供更清晰的测试面与失败定位
        ▼
候选 2：Context selection policy 深化
        │ 复用拆分后的 context_knowledge/context_budget/context_recall 测试面
        ▼
候选 2b：Context 深化实现（若 deletion test 证明值得）

候选 3：Import 认知 A/B 完整评测
        └── 独立并行、可暂停，不阻塞候选 2/4 或 Release Candidate
```

> 候选 4 是结构准备，候选 2 是模块深化，候选 3 是真实 Provider 评测。三者不合并成一个大提交。

# 候选 4：按接缝拆分测试文件

## 目标

改善测试的 locality 和接口可导航性，不改变生产行为，不重写测试语义，不在拆分过程中顺手修产品问题。

当前热点：

```text
internal/tools/commit_chapter_test.go 约 3645 行
internal/tools/novel_context_test.go 约 1516 行
internal/store/world_test.go          约 1444 行
internal/host/engine_test.go          约 1205 行
```

### 阶段 180：测试资产清单与归属表

状态：`complete`

产物：`docs/test-asset-map.md`。

基线快照：

```text
commit_chapter_test.go：74 个 Test 函数
novel_context_test.go：31 个 Test 函数
analyze_test.go：23 个 Test 函数
runner_test.go：14 个 Test 函数
contracts_test.go：5 个 Test 函数
```

原始 `internal/tools`、`internal/host/imp` 测试、全量 `go test ./...`、`go vet ./...` 和 `git diff --check` 均通过。阶段 180 没有移动文件、修改测试或修改生产代码。

为每个测试函数建立静态归属表：

- 文件/测试名
- 领域 seam
- 使用的公共接口
- 是否依赖共享 fixture
- 是否包含跨 seam 集成行为
- 预期新文件

首批只盘点 `internal/tools` 与 `internal/host/imp`，不一开始处理全仓。

门禁：不移动文件，不改测试行为；输出一份可审阅的拆分映射。

### 阶段 181：Commit 测试按 seam 拆分

状态：`complete`

首个切片已完成：新增 `internal/tools/commit_payload_test.go`，移动篇幅、Markdown、Lint、Schema 与嵌套字段测试；原测试名保持不变，未修改生产代码。新文件补充 `domain` import，原文件移除不再使用的 `llmcontract` import。

验证：

```text
go test ./internal/tools -run 'TestChapterTargetMax|TestCommitChapterRejectsPersistedChapterTarget|TestCommitChapterRejectsChapterOverTarget|TestCommitChapterDoesNotBlockChapterBelowTarget|TestCommitChapterRewriteRejectsChapterOverTarget|TestCommitChapterRejectsMarkdownResidue|TestCommitChapterPersistsDuplicateParagraph|TestCommitChapterSchema' -count=1
go test ./internal/tools -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

阶段 181 已完成：Commit 测试按接缝拆分完成，后续阶段进入 Context/Import 测试资产整理。

完成的 Commit 测试文件：

```text
commit_chapter_test.go       2 个基础/兼容测试
commit_payload_test.go      10 个 payload/schema/篇幅/格式测试
commit_knowledge_test.go    20 个 Knowledge/Belief 测试
commit_foreshadow_test.go    9 个 Foreshadow 测试
commit_rewrite_test.go      15 个 Rewrite/完成态测试
commit_integrity_test.go    16 个 PendingCommit 密封/恢复测试
commit_completion_test.go   2 个嵌套字段/Cast Ledger 测试
commit_process_recovery_test.go 1 个跨进程恢复测试
合计                         75 个测试
```

原测试名称保持不变，未修改生产代码。定向测试、`internal/tools`、全量测试、vet 与 diff check 全部通过。

第三个切片已完成：新增 `internal/tools/commit_foreshadow_test.go`，移动 9 个 Foreshadow 生命周期、Rewrite 重建与 Saga 重放测试；原测试名保持不变，未修改生产代码。仅清理新文件中未使用的测试 import。

验证：

```text
go test ./internal/tools -run 'TestCommitChapter.*Foreshadow|TestCommitChapterReinforcesForeshadow|TestCommitChapterReplayKeepsSameChapterAdvancedThenResolvedForeshadow' -count=1
go test ./internal/tools -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

第二个切片已完成：新增 `internal/tools/commit_knowledge_test.go`，移动 20 个 Knowledge/Belief/ReaderKnown/Knowledge Replay 测试；原测试名保持不变，未修改生产代码。仅调整新文件的测试 import 依赖。

验证：

```text
go test ./internal/tools -run 'TestCommitChapter.*Knowledge|TestCommitChapter.*Belief|TestCommitChapter.*Learning|TestCommitChapter.*Reveal|TestCommitChapterReplayDoesNotDuplicateKnowledgeState' -count=1
go test ./internal/tools -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

候选文件：

```text
internal/tools/commit_pending_test.go
internal/tools/commit_integrity_test.go
internal/tools/commit_knowledge_test.go
internal/tools/commit_foreshadow_test.go
internal/tools/commit_rewrite_test.go
```

规则：

- 保留原测试名，避免测试历史与筛选命令漂移。
- 共享 helper 只在确实跨 seam 时留在原文件或迁入同包 `test_helpers_test.go`。
- 只移动测试，不改变生产代码。
- 每次移动后先跑原测试集合，再跑全量。

成功标准：`git diff` 只显示文件移动/测试内容位置变化；测试数量、名称和行为不变。

### 阶段 182：Context 测试按消费 seam 拆分

状态：`complete`

阶段 182 已完成：Context 测试按消费 seam 拆分为 Knowledge、Recall、Budget、References、Modes、Envelope、Errors；保留 `novel_context_simulation_test.go` 与 `novel_context_reader_boundary_test.go` 两个独立文件，原文件仅保留共享 helper。与拆分前 HEAD 对比仍为 33 个唯一测试，未丢失、重复或新增；定向测试、`internal/tools`、全量测试、vet、diff check 全部通过。生产代码无差异。候选 4 下一步进入阶段 183：Import 测试资产按阶段拆分。

第二个 Recall 切片已完成：新增 `internal/tools/context_recall_test.go`，移动 8 个 Recall/伏笔/Review 记忆测试；原测试名保持不变。完整 Context 测试集合仍为 33 个唯一测试，未丢失、重复或新增；定向测试、`internal/tools`、全量测试与 vet 已通过。下一切片处理 Budget/References/Envelope，保留已有独立 Simulation 与 Reader Boundary 文件。

第三个切片已完成：新增 `context_budget_test.go` 与 `context_references_test.go`，移动 7 个 Budget/Platform Rubric/References 测试；原测试名保持不变。完整 Context 测试集合仍为 33 个唯一测试，未丢失、重复或新增；Budget/References 定向测试、`internal/tools`、全量测试与 vet 已通过。下一切片处理 Envelope/错误降级与基础模式测试。

候选文件：

```text
internal/tools/context_knowledge_test.go
internal/tools/context_recall_test.go
internal/tools/context_budget_test.go
internal/tools/context_references_test.go
internal/tools/context_simulation_test.go
```

重点保持：

- JSON 结构级泄漏测试；
- ReaderKnown/CharacterKnown 边界；
- 预算裁剪与 `_trimmed`；
- Platform Rubric 条件注入；
- SimulationProfile compact 注入；
- 相关章节和伏笔召回。

### 阶段 183：Import 测试资产按阶段拆分

状态：`complete`

候选文件：

```text
internal/host/imp/contracts_test.go
internal/host/imp/analyze_facts_test.go
internal/host/imp/analyze_knowledge_test.go
internal/host/imp/publish_provenance_test.go
internal/host/imp/recovery_test.go
```

只在阶段 180 归属表确认测试边界后执行；不凭文件名猜测移动范围。

### 阶段 184：拆分回归与范围审计

状态：`complete`

运行：

```bash
go test ./internal/tools ./internal/host/imp -count=1 -timeout=5m
go test ./... -timeout=5m
go vet ./...
git diff --check
```

确认：

- 测试函数没有重复或丢失；
- `-run` 过滤路径仍可用；
- 没有新增测试框架；
- 没有生产代码改动；
- 文件移动不会改变 package 初始化或 fixture 生命周期。

建议独立提交：

```text
重构：按接缝整理测试资产
```

### 阶段 183—184 完成结论

阶段 183 已将 Import 测试按 Contracts、Knowledge、事实校验、Provenance、Runner/Recovery 等 seam 拆分；已有独立的 Segment、Source、State、Synthesis、Workspace、Call 测试文件保持不动。阶段 184 完成 103 个唯一测试的集合审计和全量回归，生产代码未修改。

# 候选 2：Context selection policy 深化

## 目标

在候选 4 提供清晰测试面后，评估是否值得把现有 Context 装配深化为一个更深、局部性更好的模块：

```text
事实读取
→ 相关性/角色选择
→ Knowledge 净化
→ References/SimulationProfile 选择
→ 预算裁剪
→ JSON envelope 序列化
```

不新增 Service、Repository、通用 Context 框架或第二个事实源。

### 阶段 185：Context 依赖与选择矩阵

状态：`complete`

列出当前 `ContextTool` 的所有输入与输出：

- Store 当前投影；
- Outline/ChapterContract；
- UserRules；
- References；
- SimulationProfile；
- Revision style；
- Budget；
- Writer/Architect envelope。

将测试按选择、净化、预算、序列化分组，确认真实 seam。

门禁：只读分析，不先引入接口。

### 阶段 186：deletion test 与候选模块形状

状态：`complete`

对以下逻辑做 deletion test：

- 删除 Knowledge 选择是否会把复杂度错误地转移到调用方；
- 删除净化步骤是否直接造成 Truth 泄漏；
- 删除预算裁剪是否破坏 Worker 合同；
- 删除 builder 是否只是把复杂度搬到另一个文件。

只有证明“删除会集中复杂度”，才考虑形成更深模块；否则只保留测试拆分，不做生产重构。

### 阶段 187：Context selection policy 的最小实现（条件阶段）

状态：`complete`

结论：仅将 Knowledge 选择/净化规则提取为同包纯策略函数；`ContextTool` 继续负责 Store IO、角色匹配、错误降级、预算和 JSON envelope。

仅当阶段 186 证明值得深化时执行：

- 将选择/净化/有界输出集中在一个明确测试面；
- 保留现有 `ContextTool.Execute` 作为适配器；
- 保持 JSON envelope 兼容；
- 不改变 ReaderKnown/CharacterKnown、未来信息过滤、8 条上限和预算裁剪语义。

首个红灯必须来自现有 JSON 结构级泄漏或边界回归测试；不以“文件太大”作为理由。

### 阶段 188：Context 全量回归与提交

状态：`complete`

结论：Knowledge 纯选择/净化策略行为等价；未改变 JSON envelope、Budget、References、ReaderKnown/CharacterKnown 或错误降级语义。

运行：

```bash
go test ./internal/tools -run 'Context|NovelContext' -count=1 -timeout=5m
go test ./internal/host ./internal/host/imp -count=1 -timeout=5m
go test ./... -timeout=5m
go vet ./...
git diff --check
```

若阶段 186 判定“不值得生产重构”，则以测试/文档结论收口，不创建空壳接口。

建议提交信息（仅在确有生产深化时）：

```text
重构：深化 Context 选择与净化策略
```

# 候选 3：Import 认知 A/B 完整评测

## 目标

补齐 `establish/believe/learn/reveal_to_reader` 的动作级 precision/recall 与负例统计；不把 Provider 不稳定误判成 Prompt 质量，不把评测 runner 变成运行时框架。

### 阶段 189：可断点评测 runner 设计

状态：`planned`

要求：

- 每条样本独立调用；
- 每条完成立即落盘；
- 支持断点续跑；
- 缺结果必须失败，不得 skip 伪成功；
- 记录 Provider 错误、超时、Usage；
- 不保存完整模型响应或凭证；
- 基线/修订 Prompt 版本指纹明确。

先用 fake model 测试 runner 状态机，不调用 Provider。

### 阶段 190：有限稳定通道探针

状态：`planned`

只跑：

```text
ik03/ik04/ik05/ik07/ik10/ik11/ik12
baseline/calibrated 各一次
```

停止条件：

- 单条超时 180 秒；
- 连续 2 条 Provider 阻塞；
- 结果缺失；
- 结构化自愈超过固定次数。

只在至少 6/7 个样本形成有效双侧结果时进入完整评测。

### 阶段 191：完整三轮 A/B

状态：`planned`

执行：

```text
12 条样本
× baseline/calibrated
× 3 轮顺序变化
```

指标：

- 各动作 precision/recall；
- 同 Truth 多动作完整集合准确率；
- 负例误报；
- Knowledge ID 复用稳定性；
- 三轮一致性；
- Provider 错误率/Usage。

若 Provider 不稳定，阶段保持 `blocked_provider`，不修改生产 Prompt。

### 阶段 192：评测报告与 Go/No-Go

状态：`planned`

只有满足以下条件才允许合入进一步 Prompt 修改：

- `learn/reveal_to_reader` 召回改善；
- 负例误报不显著增加；
- `believe` 不退化；
- 完整动作集合准确率改善；
- 结果可由脱敏报告复核。

否则保留当前已提交的最小 Prompt 修订，不继续堆规则。

建议独立提交：

```text
评测：完成导入认知动作 A/B 验证
```

# 三条路线共同门禁

## 阶段 193：发布候选不回归

状态：`planned`

候选 2/3/4 任一变更都必须继续通过：

```bash
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/tools ./internal/host/imp ./internal/host/sim -timeout=10m
git diff --check
```

候选 3 的真实 Provider 失败不能通过修改生产代码规避；候选 4 的测试拆分不能改变测试语义；候选 2 的 Context 深化不能改变 JSON envelope 或知识泄漏边界。

## 阶段 194：文档与路线收口

状态：`planned`

同步：

- `CONTEXT.md`；
- `docs/architecture.md`；
- `docs/release-acceptance.md`；
- `findings.md`；
- `progress.md`。

最终不保留过期 `in_progress/pending` 描述在当前根工作记忆中；历史过程归档到 `docs/history/plans/`。

## 明确不做

- 不在候选 4 中重构生产代码；
- 不在候选 2 中创建通用 Context Service/Repository；
- 不在候选 3 中引入第二套 Import Schema 或运行时评测框架；
- 不因为 Provider 阻塞伪造模型质量结论；
- 不新增 Knowledge/Foreshadow 动作；
- 不创建扫榜、浏览器、Chrome/CDP、数据库或 Web 事实源；
- 不在 Release Candidate 前混入无关功能。

## 当前下一步

从阶段 180 开始：先建立候选 4 的测试资产归属表，不修改生产代码。
