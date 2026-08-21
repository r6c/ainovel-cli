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
里程碑 B：最小知识事实闭环——planned
当前门禁：阶段 8，先隔离/提交伏笔批次
```

### 下一步

在用户授权或自行建立 Git 边界后，加载 TDD 技能并从阶段 9 开始：

```text
只写 WorldStore establish→learn 的首个失败测试
```

门禁完成前不修改知识状态生产代码。

### 规划一致性校验

- `task_plan.md` 中仅阶段 8—15 为 `pending`，旧阶段 0—7 保持 `complete`。
- `git diff --stat -- assets internal` 仍为原伏笔批次的 18 个文件、812+ / 51-，本次规划没有新增生产代码 diff。
- 搜索确认仓库尚无 `KnowledgeEntry`、`UpdateKnowledge` 或 `knowledge_state` Go 实现。
- `git diff --check` 通过。
- 规划已停在阶段 8 门禁，没有越权创建提交、stash、分支或知识状态代码。
