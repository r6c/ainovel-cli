# 伏笔生命周期闭环修复进度

## 会话信息

- 任务：根据代码审查结果生成可执行修复计划
- 当前状态：全部阶段完成
- 执行原则：严格 TDD，红→绿，单切片推进
- 项目根：`ainovel-cli`

## 2026-08-21：规划初始化

### 已完成

1. 加载 `planning-with-files-zh` 技能。
2. 检查项目根：此前不存在 `task_plan.md`、`findings.md`、`progress.md`。
3. 运行 `session-catchup.py`：没有未同步会话内容。
4. 读取 `git diff --stat`，确认当前已有 12 个未提交修改文件。
5. 将审查结论拆成 7 个实施阶段。
6. 固化生命周期决策：`resolved` 为终态，重复 `resolve` 暂保持幂等。
7. 创建三个规划文件。

### 当前未提交基线

```text
assets/load_test.go
assets/prompts/import-analyze.md
assets/prompts/writer.md
assets/testdata/writer-golden.md
internal/chapterfacts/facts.go
internal/domain/review.go
internal/host/imp/contracts.go
internal/host/imp/contracts_test.go
internal/store/world.go
internal/store/world_test.go
internal/tools/commit_chapter_test.go
internal/tools/commit_validation.go
```

规划文件创建后还会新增：

```text
task_plan.md
findings.md
progress.md
```

### 审查前验证记录

以下命令已通过，但不代表新动作的全链路已覆盖：

```text
go test ./...
go vet ./...
git diff --check
```

审查随后发现现有测试未覆盖 `revision.Projector` 和 Rewrite 集成接缝。

### 当前发现计数

- S1：2
- S2：2
- S3：2
- 额外消费端一致性问题：1（久挂基准仍使用 PlantedAt）

### 下一步

开始阶段 1：

1. 重新读取 `task_plan.md` 和 `findings.md`。
2. 在 `internal/revision/revision_test.go` 只写首个失败测试。
3. 运行单测确认红灯落在 Projector 不支持 `reinforce/partial_payoff`。
4. 不提前修改 Commit 校验或诊断逻辑。

### 阶段 1 红灯

新增公开投影测试：`plant@1 → reinforce@2 → partial_payoff@3 → resolve@4`。

失败结果：

```text
project foreshadow lifecycle: 第 2 章伏笔操作非法: "reinforce"
```

红灯准确位于 `revision.Projector` 的旧动作白名单。

### 阶段 1 绿灯

只扩展 `revision.projectWorld` 现有 switch：

- `advance` 写入 `LastAdvancedAt`
- `reinforce → reinforced` 并写入 `LastAdvancedAt`
- `partial_payoff → partially_paid` 并写入 `LastAdvancedAt`
- `resolve` 保留最近推进章

新增测试、旧 Projector 测试和整个 revision 包均通过。终态约束按计划留到阶段 4。

### 阶段 2 集成结果

新增 `CommitChapterTool.Execute` 重写场景：合法的 `plant@1 → reinforce@2` 历史记录在重写第 2 章后可全量重建；账本仍为 `reinforced@2`、返工队列排空、PendingCommit 清除。测试直接通过，无需额外生产修改。

### 阶段 3 红灯

同一 payload 执行 `resolve → reinforce` 时，`Execute` 返回错误，但 `LoadPendingCommit` 得到 `Stage=started`。这证明前置校验只读取初始账本状态，没有模拟 payload 内动作顺序。

### 阶段 3 绿灯

`validateCommitArgs` 现在按 payload 顺序维护临时 `plantedAt/status`，同时覆盖既有条目和本章新种条目。以下序列均在 PendingCommit 前拒绝：

- `resolve → reinforce`
- `resolve → partial_payoff`
- `plant → resolve → reinforce`

### 阶段 4 完成

`resolved` 已在三个接缝中成为终态：

- WorldStore 拒绝 `advance/reinforce/partial_payoff`
- Commit 在 PendingCommit 前拒绝以上转换
- Projector 全量重建拒绝以上历史记录

同章重复 `resolve` 仍保持 Saga 幂等。

### 阶段 5 完成

- Diagnostics 将所有 `status != resolved` 的伏笔计为打开。
- `ForeshadowStale` 和统计均优先使用 `LastAdvancedAt`。
- Context 久挂召回按最近推进章筛选和排序。
- 久挂提示从“未回收”修正为“未推进”。

### 阶段 6 完成

- Saga 重放夹具改为合法的第 1 章种植、第 2 章强化。
- timeline、state change、payload、PendingCommit、checkpoint 和终稿断言统一为第 2 章。
- `ForeshadowUpdate.Action` 注释同步五动作。
- 搜索确认无 `UpdateForeshadow(0)` 和旧三动作注释残留。

## 测试日志

| 阶段 | 命令 | 结果 |
|---|---|---|
| 审查前全量 | `go test ./...` | 通过，但缺少关键接缝覆盖 |
| 审查前静态检查 | `go vet ./...` | 通过 |
| 审查前 Diff 检查 | `git diff --check` | 通过 |
| 阶段 1 红灯 | `go test ./internal/revision -run '^TestProjectorRebuildsForeshadowLifecycleFromChapterRecords$' -count=1` | RED：第 2 章 `reinforce` 被 Projector 判为非法动作 |
| 阶段 1 局部绿灯 | `go test ./internal/revision -run '^(TestProjectorRebuildsForeshadowLifecycleFromChapterRecords|TestProjectorRebuildsWorldStateFromChapterRecords)$' -count=1` | 通过 |
| 阶段 1 包验证 | `go test ./internal/revision -count=1` | 通过 |
| 阶段 2 集成 | `go test ./internal/tools -run '^TestCommitChapterRewriteRebuildsForeshadowLifecycle$' -count=1` | 通过；阶段 1 修复已闭合重写接缝 |
| 阶段 3 红灯 | `go test ./internal/tools -run '^TestCommitChapterRejectsReinforceAfterResolveInSamePayloadBeforePending$' -count=1` | RED：Store 拒绝序列时 PendingCommit 已处于 started |
| 阶段 3 绿灯 | `go test ./internal/tools -run '^(TestCommitChapterRejectsActionsAfterResolveInSamePayloadBeforePending|TestCommitChapterRejectsReinforcingResolvedForeshadowBeforePending|TestCommitChapterRejectsUnknownForeshadowReferenceBeforePending|TestCommitChapterRewriteRejectsForwardForeshadowReference)$' -count=1` | 通过
| 阶段 4 Store 红灯 | `go test ./internal/store -run '^TestForeshadow_RejectsAdvancingResolvedEntry$' -count=1` | RED：`advance` 成功重新打开 resolved 条目 |
| 阶段 4 Store 绿灯 | `go test ./internal/store -run '^TestForeshadow_' -count=1` | 通过 |
| 阶段 4 Commit 红灯 | `go test ./internal/tools -run '^TestCommitChapterRejectsAdvancingResolvedForeshadowBeforePending$' -count=1` | RED：Store 拒绝时 PendingCommit 已处于 started |
| 阶段 4 Commit 绿灯 | `go test ./internal/tools -run '^(TestCommitChapterRejectsAdvancingResolvedForeshadowBeforePending|TestCommitChapterRejectsReinforcingResolvedForeshadowBeforePending|TestCommitChapterRejectsActionsAfterResolveInSamePayloadBeforePending)$' -count=1` | 通过 |
| 阶段 4 Projector 红灯 | `go test ./internal/revision -run '^TestProjectorRejectsAdvancingResolvedForeshadow$' -count=1` | RED：全量重建接受 resolved→advance |
| 阶段 4 Projector 绿灯 | `go test ./internal/revision -run '^TestProjector.*Foreshadow' -count=1`；`go test ./internal/revision -count=1` | 通过；三种重新打开路径均被拒绝 |
| 阶段 4 resolve 重放 | `go test ./internal/store -run '^TestForeshadow_' -count=1` | 通过；同章重复 resolve 保持幂等 |
| 阶段 5 活跃计数红灯 | `go test ./internal/diag -run '^TestAnalyzeCountsAllUnresolvedForeshadowAsOpen$' -count=1` | RED：期望 4，实际只统计 2 |
| 阶段 5 活跃计数绿灯 | 单测及 `go test ./internal/diag -count=1` | 通过 |
| 阶段 5 停滞语义红灯 | `go test ./internal/diag -run '^TestAnalyzeMeasuresForeshadowStalenessFromLastAdvance$' -count=1` | RED：长期未推进的 advanced 条目未被识别，实际 0 |
| 阶段 5 诊断绿灯 | 两条定向测试及 `go test ./internal/diag -count=1` | 通过 |
| 阶段 5 Context 红灯 | `go test ./internal/tools -run '^TestContextToolSelectedMemorySurfacesAgingForeshadow$' -count=1` | RED：刚强化条目按 PlantedAt 被误召回，文案仍为“未回收” |
| 阶段 5 Context 绿灯 | 定向用例及 `go test ./internal/tools -run '^TestContextTool' -count=1` | 通过
| 阶段 6 合法夹具 | `go test ./internal/tools -run '^TestCommitChapterReplayAfterPartialCommitDoesNotDuplicateWorldState$' -count=1` | 通过；场景已平移为 plant@1→reinforce@2 |
| 阶段 7 关键包 | revision / store foreshadow / tools foreshadow+context / diag / assets / import | 六组全部通过 |
| 阶段 7 审计补充红灯 | `resolve → repeat plant → reinforce` | RED：重复 plant 重置临时状态，Store 失败后留下 PendingCommit |
| 阶段 7 审计补充绿灯 | 顺序状态表及相关提交前校验测试 | 通过；既有 ID 的重复 plant 不再重置临时状态 |
| 阶段 7 全量测试 | `go test ./...` | 通过 |
| 阶段 7 静态检查 | `go vet ./...` | 通过 |
| 阶段 7 Diff 检查 | `git diff --check` | 通过 |
| 阶段 7 范围审计 | `git status --short`、`git diff --stat`、计划外类型扫描 | 通过；18 个受跟踪文件，812+ / 51-，无新 Service/接口/状态机/格式版本 |

## 错误与处理

| 时间 | 错误 | 处理 |
|---|---|---|
| 前序实现阶段 | `rg: command not found` | 改用结构化搜索工具或 `grep`，不再重复调用 `rg` |
| 审查阶段 | 全量测试未暴露 Projector 缺口 | 增加 Projector 与 Rewrite 集成测试计划 |
| 阶段 6 | Saga 夹具批量替换中的断言文本不唯一 | 结构化编辑原子拒绝；改用函数级唯一上下文逐段替换 |
| 阶段 7 | `check-complete.sh` 返回 0 但报告未找到项目根计划 | 不重复调用；搜索确认 task_plan 无 pending/in_progress/未勾选项 |
| 路线规划 | 搜索大纲角色字段的正则未转义 `[]` | 改为直接读取 `internal/domain/story.go`，未重复失败查询 |
| 阶段 12 | 猜测读取不存在的 `internal/llmcontract/jsonschema.go` | 改读搜索确认的 `validate.go` 与 `contract.go`，不重复错误路径 |
| 阶段 12 | Import Knowledge 测试仅向静态 Schema 塞未知字段，因 JSON Schema 默认允许 additionalProperties 而误绿 | 改为先直接断言 `knowledge_updates` 属性和动作枚举存在，再验证合法 payload |
| 阶段 12 | `go test ./internal/host/imp -count=1` 超过宿主 120 秒超时，未返回具体失败 | 不原样重跑；拆为 contracts/analyze/publish 定向测试定位 |
| 阶段 14 | 对单个 `assets/load_test.go` 路径调用目录搜索，返回 ENOTDIR | 改为 `read_file` 直接读取，不重复错误调用 |
| 阶段 15 | 读取不存在的 `internal/store/chapter_records_test.go` | 目录确认无该文件；新建最小公开 Load 兼容测试，不重复错误读取 |
| 阶段 15 | 对单个 `commit_chapter.go` 文件路径调用目录搜索，返回 ENOTDIR | 已由随后 read_file 获得调用顺序；不重复该搜索 |
| 阶段 15 | migration 两处 `validateRecordSet` 批量替换因非唯一匹配被拒绝 | 文件未部分修改；改用带上下文精确替换 |

## 文件变更日志

| 文件 | 操作 | 目的 |
|---|---|---|
| `task_plan.md` | 新建 | 保存阶段、范围、TDD 接缝和完成定义 |
| `findings.md` | 新建 | 保存审查证据、严重度和测试矩阵 |
| `progress.md` | 新建 | 保存会话、基线、验证和下一步 |
| `internal/revision/projector.go` | 修改 | 支持五动作、LastAdvancedAt 与 resolved 终态 |
| `internal/revision/revision_test.go` | 修改 | 覆盖完整生命周期和终态重建 |
| `internal/diag/diag.go` | 修改 | 统计全部未回收伏笔并按最近推进章判断停滞 |
| `internal/diag/rules_planning.go` | 修改 | 停滞规则使用 LastAdvancedAt |
| `internal/diag/diag_test.go` | 新建 | 覆盖活跃计数和停滞语义 |
| `internal/tools/novel_context.go` | 修改 | 久挂召回按最近推进章筛选、排序和提示 |
| `internal/tools/novel_context_test.go` | 修改 | 覆盖近期强化伏笔不被误召回 |
| `internal/tools/commit_validation.go` | 修改 | 顺序模拟 payload 状态并保护 resolved 终态 |
| `internal/tools/commit_chapter_test.go` | 修改 | 覆盖重写、顺序校验、终态和合法 Saga 夹具 |
| `internal/store/world.go` | 修改 | 禁止 resolved→advance |
| `internal/store/world_test.go` | 修改 | 覆盖终态和同章 resolve 重放幂等 |

## 最终范围审计

- 受跟踪修改文件：18 个，均属于伏笔事实契约、存储、提交、重建、诊断、上下文或对应测试/Prompt。
- 新文件：`internal/diag/diag_test.go` 和 3 份规划文件。
- 没有新增 Service、接口、状态机抽象、数据库或格式版本。
- 搜索确认没有 `UpdateForeshadow(0)` 和旧三动作注释残留。
- 尚未创建 Git 提交。

---

## 2026-08-21：跨项目建议结合仓库现状重排路线

### 本次目标

不是继续直接编码，而是把外部项目建议与 `ainovel-cli` 的真实能力去重，追加下一阶段可执行计划。

### 已完成盘点

1. 恢复现有规划上下文，确认伏笔阶段 0—7 已完成。
2. 核对当前工作区：18 个受跟踪文件仍未提交，累计 812+ / 51-。
3. 核对领域与运行链路：
   - `StateChange` 已可承载角色心理字段；
   - `RelationshipChanges` 已贯通 Commit、Store、Projector、Context；
   - revision 已具备正文 hash、重新分析、stale 防护和派生状态重建；
   - `ChapterContract` 已有 EmotionTarget/PayoffPoints/HookGoal；
   - `rules.Lint` 已有 Commit + Revision 的统一执行点；
   - `assets.Load` 已有内置/全局/本书三层覆盖和题材 references；
   - `userrules` 已承载全局偏好；
   - TUI 已有 quick + cocreate。
4. 确认下一真正缺口是“作者真相与角色获知状态”。
5. 将主计划升级为里程碑路线，追加阶段 8—15。
6. 第一批知识状态范围收敛为 `establish/learn`，明确排除 belief、reader reveal、forget、reveal plan。
7. 确认旧 ChapterRecord 可通过新增零值切片字段兼容，默认不提升格式版本。
8. 确认 Context 可复用 `matchOutlineCharacters`、Envelope、RecallItem 和预算裁剪，不新增检索服务。

### 当前状态

```text
里程碑 A：伏笔生命周期闭环——complete
里程碑 B：最小知识事实闭环——complete
当前状态：阶段 8—15 全部完成
```

### 下一步

在用户授权或自行建立 Git 边界后，加载 TDD 技能并从阶段 9 开始：

```text
只写 WorldStore establish→learn 的首个失败测试
```

阶段 8 已完成；开始阶段 9 时只修改 Knowledge Store 首个 TDD 切片。

### 阶段 9 首个红灯

公共 Store 场景：

```text
establish k_shadow@1 → 林墨 learn@2
```

命令：

```text
go test ./internal/store -run '^TestKnowledge_CharacterLearnsEstablishedTruth$' -count=1
```

结果为编译红灯：缺少 `domain.KnowledgeUpdate`、`WorldStore.UpdateKnowledge` 和 `LoadKnowledgeState`，符合预期。

### 阶段 9 首个绿灯

只增加 Knowledge 领域 DTO 和 WorldStore JSON 当前投影，`establish@1 → 林墨 learn@2` 已通过。尚未接入 ChapterFacts、Commit、Projector、Import 或 Context。

### 阶段 9 完成

Knowledge Store 已通过以下公共行为：

- `establish@1 → learn@2`
- 相同 ID、不同 Truth 拒绝且原事实不变
- 相同 ID、相同 Truth 幂等
- 未知事实不能 learn
- 同角色重复 learn 幂等并保留首次 LearnedAt
- establish 要求 ID/Truth，learn 要求角色名
- `knowledge_state.json` 与 `knowledge_state.md` 同步投影

验证：

```text
go test ./internal/store -run '^TestKnowledge_' -count=1
go test ./internal/store -count=1
```

均通过。阶段 10 开始接入共享 ChapterFacts 与 Commit Saga。

### 阶段 10 当前进度

已完成：

- `ChapterFacts.KnowledgeUpdates` 零值兼容字段；
- Commit 严格 Schema 的 `knowledge_updates` 契约；
- `chapterfacts.Validate` 的 ID/action/Truth/Character 校验；
- Commit 正常 establish 和 learn；
- 同 payload `establish → learn`；
- 未知 learn 与冲突 Truth 在 PendingCommit 前拒绝。

下一切片：构造 `CommitStageStarted` 重放，确认 Knowledge 状态幂等且 PendingCommit 清除。

### 阶段 10 完成

Knowledge 已贯通共享 ChapterFacts 与 Commit Saga：

- 严格 Schema、Validate；
- establish、learn、同 payload 顺序；
- 未知引用和 Truth 冲突在 PendingCommit 前拒绝；
- `CommitStageStarted` 重放幂等且 PendingCommit 清除。

验证：`go test ./internal/chapterfacts -count=1` 与 `go test ./internal/tools -count=1` 均通过。

### 阶段 11 严格 Schema 回归

Knowledge Projector 定向测试已通过，但整个 revision 包出现预期契约红灯：

```text
$.facts.knowledge_updates 是必填字段
```

原因是旧修订模型测试夹具仍按旧严格 Schema 返回 facts。处理原则：只同步夹具为空数组；不放宽 Schema、不把字段改为可选。

### 阶段 11 完成

Knowledge 已贯通 Revision：

- Projector 重建 `establish@1 → learn@3`；
- 冲突 Truth 历史拒绝；
- 重复 learn 去重并保留首次章节；
- revision 严格 Schema 夹具同步；
- 含 Knowledge 的 Commit Rewrite 正常重建并清理队列/PendingCommit。

`go test ./internal/revision -count=1` 与 Knowledge 工具测试均通过。

### 阶段 12 完成

Knowledge 已贯通 Import：DTO、严格 Schema、跨批次 ledger、首批连续性校验、publish 参数映射和真实两章发布均通过。

整个包验证：

```text
go test ./internal/host/imp -count=1 -timeout=5m
```

通过。此前默认宿主超时没有复现。

### 阶段 13 完成

Knowledge Context 已实现：

- 复用大纲文本与 `matchOutlineCharacters` 选择当前角色；
- 只注入这些角色已知的 Truth，不泄露无关角色独占真相；
- 过滤当前章及未来才建立/获知的信息；
- 最近优先、最多 8 条；
- 接入 `trimByBudget` 和 `_trimmed`。

`go test ./internal/tools -count=1` 通过。

### 阶段 14 完成

Writer、Editor、Revision 和 Import Prompt 已同步知识语义纪律，Writer golden 保持一致。`go test ./assets -count=1` 通过。

### 阶段 15 全量回归

`go vet ./...` 与 `git diff --check` 通过；`go test ./... -timeout=5m` 发现 4 个 Host Engine 流程未落章：

- `TestEngine_ReviewPermitWritesExactlyOneNewChapter`
- `TestEngine_WritesBookToCompletion`
- `TestEngine_PauseWithEditorDispatchWaitsForRewriteQueue`
- `TestEngine_TargetChapterHoldStopsAtRequestedChapter`

其他包全部通过。症状 `CompletedChapters=[]`/返工队列未排空，初步判断 Host 的模拟 Writer 工具参数仍缺严格必填 `knowledge_updates`。下一步单测定位后只同步共享夹具。

### 阶段 15 链路审计发现

Import 的逐章事实 Schema 已新增必填 `knowledge_updates`，但 `analysisSchemaVersion` 仍为 2。由于该版本参与分析工件 InputDigest，未提升会让旧 Schema 产生的缓存可能继续被视为新鲜。下一切片先构造 v2 digest 工件证明当前误复用，再提升到 v3。

关于章节重写：暂不照搬伏笔的 `RestoreOwnPlants`。用户修订可能明确删除或改写作者真相，强制保留旧 establish 会让知识事实不可撤销；当前 revision 会重新分析完整正文，Projector 对未知 learn/冲突 Truth 有确定性保护。除非出现具体失败场景，不新增 RestoreOwnKnowledge。

### 阶段 15 完成

最终链路审计额外发现并修复：

1. Import 新严格 Schema 未提升分析缓存版本：`analysisSchemaVersion 2 → 3`，旧 v2 分析工件失效。
2. 重写真相建立章可能先覆盖 ChapterRecord、再因后续 learn 失去前置事实而锁死：现于 Rewrite PendingCommit 前用候选记录集执行 `revision.ValidateRecordSet`。
3. 无 ChapterRecord 的旧书/旧测试夹具继续沿用原返工路径；有完整记录时执行依赖验证。
4. `TestLoadEmpty`、工具说明和 WorldStore 注释同步 Knowledge。

最终验证：

```text
go test ./... -timeout=5m
go vet ./...
git diff --check
```

全部通过。

范围审计：

```text
30 files changed
1574 insertions(+)
29 deletions(-)
```

未新增 Service、Repository、数据库、Web 事实源或通用状态机；未实现 belief、reader reveal、forget、reveal plan。Knowledge 批次尚未创建 Git 提交。

### 下一步规划：里程碑 C1

当前 Knowledge 批次仍未提交（31 个文件、1675+ / 38-），所以新计划首先设置阶段 16 提交/隔离门禁。

下一领域切片选择“读者揭示状态”，只规划：

```text
reveal_to_reader
ReaderRevealedAt
```

错误信念、撤销揭示、多读者模型继续延期。计划已拆为阶段 16—23：隔离、Store、ChapterFacts/Commit、Revision/Projector、Import、Context、Prompt、全量验证。

本次只更新规划文件，没有修改 reader reveal 生产代码。

### 阶段 16 完成

提交前全量测试、vet、diff check 均通过；已使用中文提交信息建立 Knowledge 批次边界：

```text
6a7b7f7 功能：追踪作者真相与角色知情状态
```

Git 提交者身份仍由本机自动配置，未擅自 amend 或修改用户 Git 配置。C1 从干净工作区开始。

### 阶段 17 首个红灯

公共 Store 场景 `establish@1 → reveal_to_reader@3` 编译红灯：`KnowledgeEntry` 缺少 `ReaderRevealedAt`，符合预期。首个绿灯只增加字段与 Store 动作分支。

### 阶段 17 完成

Reader reveal 已在 Store 闭环：正常 reveal、未知引用拒绝、重复 reveal 保留首次章节、同 payload establish→reveal、JSON/Markdown 投影；整个 Store 包通过。

### 阶段 18 完成

Reader reveal 已贯通 ChapterFacts 与 Commit：三动作严格枚举、正常提交、同 payload establish→reveal、未知引用在 Pending 前拒绝、started 重放保留首次章节；整个 tools 包通过。

### 阶段 19 完成

Projector/Rewrite 已覆盖 reader reveal：正常重建、未知历史拒绝、重复保留首次章节、删除 reveal 可清零、删除 establish 且后续 reveal 时在 Pending 前拒绝；revision 与 tools 包通过。

### 阶段 20 完成

Import 已贯通 reader reveal：三动作严格 Schema、首批引用校验、跨批次 reader known ledger、真实发布和 analysisSchemaVersion 3→4 缓存失效；整个 Import 包通过。

### 阶段 21 完成

Context 已表达 ReaderKnown/CharacterKnown 信息差：读者已知但当前角色未知的 Truth 可见，读者未知且仅无关角色知道的 Truth 隐藏，当前章才 reveal 的 Truth 不提前暴露；既有 8 条上限和预算裁剪继续生效，整个 tools 包通过。

### 阶段 22 完成

Writer、Editor、Revision、Import Prompt 已同步 reveal_to_reader 纪律：完整揭示、角色/读者知情独立、提前泄底/重复揭秘检查、partial_payoff 不等于完整 reveal；Writer golden 与 assets 包通过。

测试编写时曾漏掉一个 Go 循环的 `{`，首次运行是语法红灯；修正测试后获得预期的 Prompt 行为红灯，再完成最小资源修改。

### 阶段 23 / 里程碑 C1 完成

协议审计额外收紧了 `reveal_to_reader`：只允许 ID，Truth/Character 在 ChapterFacts、Store 和 Import 三层都会拒绝，避免多余字段被静默忽略。

最终验证：

```text
go test ./... -timeout=5m
go vet ./...
git diff --check
```

全部通过。范围审计为 24 个文件、732+ / 101-；未新增 Service、Repository、状态机、数据库或格式迁移，未实现 belief、reader_belief、unreveal、conceal 或多读者模型。

### C1 规划校验

- 阶段 16—23 均已写入且仅这些新阶段为 pending。
- 后续候选已改为 C2 角色错误信念，不与 C1 读者揭示重复。
- 搜索确认 `internal/` 尚无 `ReaderRevealedAt` 或 `reveal_to_reader` 实现。
- `git diff --check` 通过。
- 生产代码仍是尚未提交的 Knowledge 批次，本次没有新增 C1 代码。

### 规划一致性校验

- `task_plan.md` 中仅阶段 8—15 为 `pending`，旧阶段 0—7 保持 `complete`。
- `git diff --stat -- assets internal` 仍为原伏笔批次的 18 个文件、812+ / 51-，本次规划没有新增生产代码 diff。
- 搜索确认仓库尚无 `KnowledgeEntry`、`UpdateKnowledge` 或 `knowledge_state` Go 实现。
- `git diff --check` 通过。
- 规划当时停在阶段 8 门禁，没有越权创建知识状态代码。

### 阶段 8 完成

已获用户授权并创建独立提交：

```text
13a775b feat: complete foreshadow lifecycle tracking
```

提交包含伏笔生命周期批次、对应测试和三份规划文件。Git 提示提交者身份由本机用户名/主机名自动配置；未擅自 amend 或修改用户 Git 配置。下一批从该提交边界开始。
