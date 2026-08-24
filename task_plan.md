# ainovel-cli 分阶段能力演进计划

## 规划总览

- 总体状态：`complete`
- 当前执行阶段：阶段 40—47 全部完成
- 已完成领域里程碑：A 伏笔生命周期；B 作者真相 + 角色获知；C1 读者揭示 + 信息差；C2a 最小角色错误信念；D Import 全书事实重放门禁
- 下一里程碑：E1 Knowledge 纯 Apply/Replay 规则收敛
- 规划原则：不增加认知动作；把现有规则收敛为专用纯函数，Store/Commit/Projector 作为 IO、事务和重建适配器
- 当前工作区：干净；基线提交 `cafb752 修复：导入发布前重放并验证全书事实`

## 路线决策摘要

结合当前仓库，原跨项目建议修订为：

1. 伏笔生命周期已经完成，不继续扩 `noticed/abandoned/contradicted`。
2. 角色心理先使用现有开放式 `StateChange.Field`，不新建心理账本。
3. 人物关系、章节修订重建、情绪合同、全局偏好、资源覆盖、quick/cocreate 和基础 Lint 都已有主链，不重复造系统。
4. 下一真实领域缺口是“作者真相与角色知道什么”。
5. 第一批只实现 `establish/learn`，暂不实现错误信念、读者已知、遗忘、揭示计划。
6. 后续 Prose Lint 在 `rules.Lint` 上纵向增加规则；资源包在现有 `assets.Load/References` 上先做具体内容，至少出现两类重复需求后再抽象 Pack。

---

## 已完成里程碑 A：伏笔生命周期闭环

### 目标

在不改变现有 `domain + WorldStore + CommitChapterTool + revision.Projector` 架构、不新增服务或状态机抽象的前提下，修复伏笔生命周期升级在章节重写、Saga 前置校验、诊断消费和测试夹具中的闭环问题。

完成后，新动作必须在以下完整链路中语义一致：

```text
章节事实 Schema
→ CommitChapterTool 提交前校验
→ PendingCommit Saga
→ WorldStore 增量写入
→ ChapterRecord 持久化
→ revision.Projector 全量重建
→ Markdown / Context / Diagnostics 投影
```

## 里程碑 A 状态

- 状态：`complete`
- 当前阶段：阶段 0—7 全部完成
- 工作方式：严格 TDD，单个公开接缝、单个失败测试、最小实现、局部复跑
- 当前工作区：已有上一批 12 个文件的未提交改动；后续不得覆盖或回退这些改动
- 基线验证：`go test ./...`、`go vet ./...`、`git diff --check` 已在审查前通过，但审查发现未覆盖的行为缺口

## 已确认公共接缝

1. `revision.NewProjector(store).Apply(records)`：章节记录全量重建公开接缝
2. `CommitChapterTool.Execute(ctx, args)`：正常提交、重写与 PendingCommit 安全边界
3. `WorldStore.UpdateForeshadow(chapter, updates)`：伏笔账本公开存储行为
4. `diag` 公开统计/报告行为：活跃伏笔计数和停滞判断

测试只通过这些既有接缝观察结果，不测试私有 switch 或内部 map。

## 范围

### 本计划内

- `revision.Projector` 支持 `reinforce`、`partial_payoff`
- Projector 重建 `LastAdvancedAt`
- 含新动作的章节可以安全重写并完成 Saga
- 提交前校验按同一 payload 的动作顺序推进临时状态
- 明确并实现 `resolved` 终态约束
- 诊断将全部未回收状态计为活跃伏笔
- 停滞语义改用最近推进章节（若有）
- 修正 Saga 测试中的第 0 章非法夹具
- 同步注释和最小必要测试

### 本计划外

- 不新增 `noticed / abandoned / contradicted` 等状态
- 不新增预计回收窗口、读者可见度、相关人物等字段
- 不新增 Lifecycle Service、Repository、状态机接口或格式版本
- 不引入数据库、Web 工作台、题材包或平台包
- 不重构 Commit Saga、Store 或 Projector 的整体结构
- 不创建 Git 提交，除非用户另行要求

## 生命周期决策

本计划采用以下最小语义：

```text
plant           → planted
advance         → advanced
reinforce       → reinforced
partial_payoff  → partially_paid
resolve         → resolved
```

`resolved` 为终态：

- 禁止 `resolved → advance`
- 禁止 `resolved → reinforce`
- 禁止 `resolved → partial_payoff`
- 重复 `resolve` 保持现有赋值式幂等行为，除非失败测试证明现有协议要求拒绝

`LastAdvancedAt` 语义：

- `advance / reinforce / partial_payoff` 更新为当前章节
- `resolve` 保留最近推进章节
- `plant` 不填写该字段

## 实施阶段

### 阶段 0：冻结基线与审查结论

状态：`complete`

- [x] 确认当前 12 个修改文件均属于前两批伏笔升级
- [x] 全量测试、vet、diff check 已通过
- [x] 完成只读代码审查
- [x] 记录 2 个 S1、2 个 S2、2 个 S3

验收：`findings.md` 与 `progress.md` 已记录基线。

---

### 阶段 1：Projector 支持完整生命周期

状态：`complete`

首个 TDD 切片：

1. 在 `internal/revision/revision_test.go` 增加公开投影失败测试：
   ```text
   plant@1 → reinforce@2 → partial_payoff@3 → resolve@4
   ```
2. 调用 `revision.NewProjector(st).Apply(records)`。
3. 断言：
   - `status == resolved`
   - `last_advanced_at == 3`
   - `resolved_at == 4`
   - `planted_at` 和描述保留
4. 运行单测，确认红灯落在 `reinforce` 或 `partial_payoff` 非法动作。
5. 只扩展 `projectWorld` 现有 switch，不新增抽象。
6. 复跑 Projector 与 revision 包测试。

额外约束：Projector 必须与 WorldStore 一样拒绝终态重新打开。

验收：

```bash
go test ./internal/revision -run 'Projector.*Foreshadow|ProjectorRebuildsWorldState' -count=1
go test ./internal/revision -count=1
```

---

### 阶段 2：重写 Commit Saga 集成闭环

状态：`complete`

TDD 切片：

1. 通过合法章节提交/记录种下并推进伏笔，使历史 ChapterRecord 包含新动作。
2. 将已完成章节放入返工队列并保存发生变化的草稿。
3. 通过 `CommitChapterTool.Execute` 执行重写。
4. 断言：
   - 重写成功
   - Projector 重建的新状态正确
   - PendingCommit 被清除
   - 返工队列正常 drain
5. 如失败，只修复 Projector/重写接缝，不改 Saga 阶段设计。

验收：

```bash
go test ./internal/tools -run 'Rewrite.*Foreshadow|Foreshadow.*Rewrite' -count=1
```

---

### 阶段 3：同一 payload 的顺序状态校验

状态：`complete`

首个 TDD 切片：

1. 通过 `CommitChapterTool.Execute` 提交：
   ```json
   [
     {"id":"f1","action":"resolve"},
     {"id":"f1","action":"reinforce"}
   ]
   ```
2. 断言请求在创建 PendingCommit 前失败。
3. 再覆盖：
   ```text
   resolve → partial_payoff
   plant → resolve → reinforce
   ```
4. 在 `validateCommitArgs` 现有循环中维护临时可见状态：
   - 初始化为账本状态
   - `plant` 建立本章可见状态
   - 每个动作校验后推进临时状态
5. 不新增独立状态机或服务。

验收：

```bash
go test ./internal/tools -run 'Foreshadow.*BeforePending|Foreshadow.*Sequence' -count=1
```

---

### 阶段 4：锁定 `resolved` 终态

状态：`complete`

TDD 切片顺序：

1. Store 公共行为：`resolved → advance` 必须失败且账本不变。
2. Commit 公共行为：同一非法转换必须在 PendingCommit 前失败。
3. Projector 公共行为：历史记录中 `resolved → advance/reinforce/partial_payoff` 必须拒绝重建。
4. 保留重复 `resolve` 的现有幂等行为，并加回归测试（若已有覆盖则复用）。

最小实现位置：

- `WorldStore.UpdateForeshadow` 的 `advance` 分支
- `CommitChapterTool.validateCommitArgs` 临时状态校验
- `revision.projectWorld` 的对应分支

验收：三个接缝对终态语义一致。

---

### 阶段 5：诊断与久挂语义对齐

状态：`complete`

切片 5A——活跃计数：

1. 新增诊断测试，输入 `planted / advanced / reinforced / partially_paid / resolved`。
2. 断言前四种计入 `ForeshadowOpen`，`resolved` 不计入。
3. 实现优先使用 `status != resolved`，与 `LoadActiveForeshadow` 一致。

切片 5B——停滞基准：

1. 新增测试：老伏笔近期 `LastAdvancedAt` 有推进时不应被判停滞。
2. 基准章节采用：
   ```text
   LastAdvancedAt > 0 ? LastAdvancedAt : PlantedAt
   ```
3. 同步 `StaleForeshadow` 和 `agingForeshadow`；摘要中的“已 N 章未回收”改为“已 N 章未推进”或等价准确文案。
4. 不改变阈值算法。

验收：诊断、Context 召回和 Store 活跃定义一致。

---

### 阶段 6：修正测试夹具与注释

状态：`complete`

- [x] 把 Saga 重放测试的 `plant@0` 改成合法章节场景，例如 `plant@1 → reinforce@2`
- [x] 确保 PendingCommit、timeline、state change 的章节号一致
- [x] 更新 `ForeshadowUpdate.Action` 注释为五动作
- [x] 检查错误信息是否仍准确描述“推进或回收”
- [x] 不做无关重命名或测试重构

验收：测试不依赖 `PlantedAt == 0` 这一“缺失值”状态。

---

### 阶段 7：全量验证与范围审计

状态：`complete`

按顺序运行：

```bash
gofmt -w <本轮修改的 Go 文件>
go test ./internal/revision -count=1
go test ./internal/store -run '^TestForeshadow_' -count=1
go test ./internal/tools -run 'Foreshadow|ReplayAfterPartialCommit' -count=1
go test ./internal/diag -count=1
go test ./assets -count=1
go test ./internal/host/imp -count=1
go test ./...
go vet ./...
git diff --check
git status --short
git diff --stat
```

最终范围审计：

- [x] 没有新增抽象层
- [x] 没有新增格式版本或迁移
- [x] 没有修改无关领域
- [x] 所有新动作均覆盖正常提交、恢复、重写、投影和诊断
- [x] 没有遗留第 0 章测试数据
- [x] 当前工作区变更清单已记录在 `progress.md`

---

## 里程碑 B：最小知识事实闭环

### 目标

让系统能可靠区分：

```text
作者认定的客观真相
≠
某个角色已经知道该真相
```

并将知识事实贯通：

```text
ChapterFacts 严格 Schema
→ CommitChapterTool 提交前校验
→ PendingCommit 幂等应用
→ WorldStore 当前知识投影
→ ChapterRecord
→ revision.Projector 全量重建
→ Import 发布
→ novel_context 有界消费
```

### 最小领域模型草案

第一批建议只增加事实类型，不增加 Service 或状态机接口：

```go
type KnowledgeEntry struct {
    ID            string
    Truth         string
    EstablishedAt int
    KnownBy       []KnowledgeHolder
}

type KnowledgeHolder struct {
    Character string
    LearnedAt int
}

type KnowledgeUpdate struct {
    ID        string
    Action    string // establish / learn
    Truth     string // establish 必填
    Character string // learn 必填
}
```

动作语义：

```text
establish：建立作者真相，不自动让任何角色或读者知道
learn：已有角色在本章获知一个已建立事实
```

约束：

- `establish` 必须有稳定 ID 和非空 Truth。
- 相同 ID、相同 Truth 的重复 establish 幂等。
- 相同 ID、不同 Truth 的重复 establish 拒绝，避免静默篡改作者真相。
- `learn` 必须引用已建立事实并提供角色名。
- 同一角色重复 learn 幂等，保留首次 `LearnedAt`。
- 同一 payload 中 `establish → learn` 合法。
- 不支持 `forget / believe / disbelieve / reveal_to_reader / schedule_reveal`。
- 新字段对旧 ChapterRecord 为零值兼容，默认不提升 `ChapterRecordVersion`。

### 已确认公共接缝

1. `WorldStore.UpdateKnowledge(chapter, updates)`：公开存储行为。
2. `CommitChapterTool.Execute(ctx, args)`：Schema、提交前引用校验和 Saga 安全性。
3. `revision.NewProjector(store).Apply(records)`：章节重写后的全量知识重建。
4. Import 的 `analysisContract → publish commit_chapter`：导入事实不丢失。
5. `ContextTool.Execute(ctx, args)`：Writer 能看到与当前任务相关的知识边界。

### 阶段 8：隔离当前伏笔批次

状态：`complete`

执行门禁：在开始知识状态代码前，必须任选一种方式建立独立边界：

1. 用户授权创建 Git 提交；或
2. 用户自行提交；或
3. 明确建立可恢复的独立分支/stash（不得丢失当前 18 个文件改动和规划文件）。

门禁验收：

- [x] 当前伏笔批次有独立提交：`13a775b feat: complete foreshadow lifecycle tracking`
- [x] `go test ./...` 通过（提交前最终验证）
- [x] `go vet ./...` 通过（提交前最终验证）
- [x] `git diff --check` 通过
- [x] 下一批从干净工作区开始，不与伏笔批次混杂

在门禁完成前，不修改知识状态生产代码。

---

### 阶段 9：Knowledge Store 首个纵向切片

状态：`complete`

严格 TDD 顺序：

1. 只写公开存储失败测试：
   ```text
   establish fact@1 → character learns@2
   ```
2. 断言：
   - Truth 和 EstablishedAt 保留；
   - KnownBy 只有目标角色；
   - LearnedAt 为第 2 章。
3. 最小实现 `domain` 数据类型和 `WorldStore` JSON + Markdown 投影。
4. 再分别驱动：
   - 未知 fact 的 learn 拒绝；
   - 同角色重复 learn 幂等；
   - 相同 ID 不同 Truth 拒绝；
   - 同章重放结果不漂移。
5. 不先改 ChapterFacts、Commit、Prompt 或 Context。

建议事实文件：

```text
knowledge_state.json
knowledge_state.md
```

它们是 ChapterRecord 的当前投影，不是新的事件源。

---

### 阶段 10：ChapterFacts 与 Commit Saga 接入

状态：`complete`

切片顺序：

1. 给 `chapterfacts.Properties/Validate` 增加 `knowledge_updates` 严格数组契约。
2. 通过 `CommitChapterTool.Execute` 提交 `establish`，确认公共入口落盘。
3. 提交 `learn` 并确认角色知识状态更新。
4. 提交前按 payload 顺序维护临时事实可见性，使 `establish → learn` 合法。
5. 未知引用和冲突 Truth 必须在创建 PendingCommit 前拒绝。
6. 构造 `CommitStageStarted` 重放，确认 Knowledge 更新幂等且 PendingCommit 正常清除。
7. 不改变 Commit Saga 阶段划分。

兼容要求：

- `KnowledgeUpdates` 为新增零值切片字段；旧 ChapterRecord 可直接解码。
- 如果测试证明现有版本校验无法兼容，先停下复核，不能顺手提升格式版本。

---

### 阶段 11：章节修订与 Projector 重建

状态：`complete`

TDD 切片：

1. 构造 ChapterRecord：
   ```text
   establish@1 → 林墨 learn@3
   ```
2. 通过 `revision.NewProjector(st).Apply(records)` 重建知识投影。
3. 断言 Truth、EstablishedAt、KnownBy、LearnedAt 完整恢复。
4. 增加非法历史：未知 fact learn、冲突 establish，Projector 必须拒绝。
5. 增加含知识更新章节的 Rewrite 集成用例，验证 PendingRevision/Projector 完成且不锁死。
6. `revisionAnalysisSchema` 自动复用 `chapterfacts.Properties`；同步 revision Prompt 的语义说明。

---

### 阶段 12：Import 契约与发布同步

状态：`complete`

1. 给 `ImportedChapterFacts` 增加 `KnowledgeUpdates`。
2. 同步 import 严格 Schema 和 Prompt。
3. 导入连续性账本必须允许批次内 `establish → learn`。
4. `publish.go` 必须把知识更新传入 `commit_chapter`。
5. 加集成测试证明导入章节发布后知识投影存在。
6. 不将人物证据或世界证据自动猜测为知识状态；只有正文明确表明角色获知时才输出 learn。

---

### 阶段 13：Writer Context 有界消费

状态：`complete`

目标不是把全量作者真相常驻上下文，而是复用现有 Context 选择机制。

首个切片：

1. 当前章大纲文本明确涉及角色 A。
2. 账本中角色 A 与角色 B 各自知道不同事实。
3. `ContextTool.Execute` 只注入与当前角色相关的知识条目，并明确 `truth + known_by` 边界。
4. 对长书设置有界数量，加入现有预算裁剪顺序。
5. 若大纲无法识别角色，仅补近期 learn 变更，不注入全量账本。
6. 复用 `matchOutlineCharacters`、Context envelope 和 `RecallItem`；不新增检索服务。

必须验证：

- Writer 能看到角色“不知道”的边界，避免全知泄漏；
- 不相关角色知识不会挤占上下文；
- Context 裁剪后仍有 `_trimmed` 可观测性。

---

### 阶段 14：Writer/Editor 语义纪律

状态：`complete`

只在代码事实闭环完成后更新 Prompt：

- Writer 仅记录正文实际建立或获知的知识变化；
- `establish` 不等于角色知道；
- 角色不得使用其 KnownBy 中不存在的信息；
- Editor 一致性检查增加“角色越权知情”；
- 不让 Prompt 承担未知引用、冲突 Truth 或 Saga 校验。

同步资源 golden/契约测试。

---

### 阶段 15：全量验证与范围审计

状态：`complete`

验证矩阵：

| 接缝 | 必须覆盖 |
|---|---|
| WorldStore | establish、learn、冲突、幂等 |
| ChapterFacts | 严格 Schema 和确定性 Validate |
| Commit | 正常提交、同 payload、Pending 前拒绝、崩溃重放 |
| ChapterRecord | 旧记录零值兼容 |
| Revision | Analyze 契约、Rewrite、Projector 重建 |
| Import | Schema、连续性、publish |
| Context | 角色相关选择、预算裁剪 |
| Prompt | Writer/Editor 明确知识边界 |

最终命令：

```bash
gofmt -w <本批 Go 文件>
go test ./internal/store -run 'Knowledge' -count=1
go test ./internal/tools -run 'Knowledge|ContextTool' -count=1
go test ./internal/revision -run 'Knowledge' -count=1
go test ./internal/host/imp -run 'Knowledge' -count=1
go test ./assets -count=1
go test ./...
go vet ./...
git diff --check
git status --short
git diff --stat
```

范围审计：

- [x] 没有新增 Service、Repository 或通用状态机接口
- [x] 没有实现 belief、reader reveal、forget 或 reveal plan
- [x] 没有新增数据库或 Web 事实源
- [x] 没有复制外部项目代码/Prompt
- [x] 正常提交、重放、重写、导入、Context 结果一致

---

## 里程碑 B 完成定义

以下条件已满足：

1. `establish/learn` 在 Store、ChapterFacts、Commit、PendingCommit、Revision、Projector、Import 和 Context 中语义一致。
2. 未知引用、冲突 Truth 和破坏后续 learn 依赖的重写都在创建 PendingCommit 前拒绝。
3. 同 ID 同 Truth、重复 learn 和崩溃重放保持幂等。
4. Context 仅选择当前角色已知的历史真相，过滤未来信息，最多 8 条并可预算裁剪。
5. 旧 ChapterRecord 缺少 `knowledge_updates` 时仍可读取，无需提升记录版本。
6. Import 逐章分析 Schema 升级为 v3，旧 v2 缓存自然失效。
7. 全量测试、vet、diff check 和范围审计全部通过。

---

## 里程碑 C1：读者揭示状态

### 为什么下一步只做读者揭示

现有 Knowledge 已能区分：

```text
作者认定的客观真相
某个角色是否已经知道
```

但仍无法表达：

```text
读者是否已经知道该真相
```

这会影响悬疑、马甲、误会、权谋和戏剧性反讽：Writer 不知道哪些 Truth 可以作为读者已知背景使用，Editor 也无法判断提前泄底或重复揭秘。

错误信念比读者揭示更复杂，需要描述角色相信的内容、与 Truth 的关系、纠正和撤销语义。为避免一次扩张成完整认知状态机，C1 只增加全局读者揭示；belief 继续延期。

### 最小模型增量

在现有类型上增量扩展：

```go
type KnowledgeEntry struct {
    ID               string
    Truth            string
    EstablishedAt    int
    KnownBy          []KnowledgeHolder
    ReaderRevealedAt int
}

type KnowledgeUpdate struct {
    ID        string
    Action    string // establish / learn / reveal_to_reader
    Truth     string
    Character string
}
```

动作语义：

```text
reveal_to_reader：正文在本章明确让读者知道已有 Truth；不自动让任何角色知道
```

不变量：

- 必须引用已 establish 的知识 ID。
- `establish → reveal_to_reader` 可在同一 payload 中发生。
- 重复 reveal 幂等，保留首次 `ReaderRevealedAt`。
- reveal 不修改 KnownBy。
- `learn` 不自动代表读者知道；角色可在场外或省略场景中获知。
- 第一版只支持完整 Truth 揭示；部分信息继续拆为独立 KnowledgeEntry 或伏笔 `partial_payoff`。
- 不实现 conceal、unreveal、reader_belief、角色错误信念或多读者群体。

### 已确认公共接缝

1. `WorldStore.UpdateKnowledge` / `LoadKnowledgeState`
2. `chapterfacts.Properties` / `Validate`
3. `CommitChapterTool.Execute`
4. `revision.ValidateRecordSet` / `Projector.Apply`
5. Import `analysisContract → buildLedger → publishChapter`
6. `ContextTool.Execute`
7. Writer / Editor / Revision / Import Prompt

---

### 阶段 16：隔离 Knowledge 批次

状态：`complete`

执行前门禁：

- [x] 已创建独立提交：`6a7b7f7 功能：追踪作者真相与角色知情状态`
- [x] 提交前 `go test ./... -timeout=5m` 通过
- [x] `go vet ./...` 通过
- [x] `git diff --check` 通过
- [x] C1 从干净工作区开始

建议提交信息：

```text
feat: track author truths and character knowledge
```

在阶段 16 完成前，不修改 reader reveal 生产代码。

---

### 阶段 17：Store 的 reader reveal 首个切片

状态：`complete`

严格 TDD 顺序：

1. 公开 Store 测试：
   ```text
   establish@1 → reveal_to_reader@3
   ```
2. 断言：
   - Truth / EstablishedAt 不变；
   - `ReaderRevealedAt == 3`；
   - KnownBy 仍为空。
3. 再驱动：
   - 未知 ID reveal 拒绝；
   - 重复 reveal 幂等并保留首次章节；
   - 同 payload `establish → reveal_to_reader`；
   - JSON 与 Markdown 投影显示“读者于第 N 章获知”。
4. 不先修改 ChapterFacts、Commit 或 Prompt。

---

### 阶段 18：ChapterFacts 与 Commit Saga

状态：`complete`

1. 扩展严格动作枚举为：
   ```text
   establish / learn / reveal_to_reader
   ```
2. `chapterfacts.Validate` 校验 reveal 只需要 ID，不接受用 Truth 静默改写已有事实。
3. Commit 正常提交 reader reveal。
4. 提交前临时状态支持 `establish → reveal_to_reader`。
5. 未知引用必须在 PendingCommit 前拒绝。
6. `CommitStageStarted` 重放不改变首次揭示章。
7. 保持现有 Saga 阶段和 Knowledge Store 接口，不新增服务。

---

### 阶段 19：Revision、Projector 与 Rewrite 安全

状态：`complete`

1. Projector 重建：
   ```text
   establish@1 → reveal_to_reader@4
   ```
2. 重复 reveal 保留首次章节。
3. 未知 reveal 历史记录拒绝。
4. Rewrite 删除唯一 establish、但后续仍有 reveal 时，必须由现有候选记录集校验在 PendingCommit 前拒绝。
5. Rewrite 删除 reveal 本身应允许，并重建为读者尚未知；不新增 `RestoreOwnReveal`。
6. Revision Prompt 只根据修改后正文明确揭示情况生成动作。

---

### 阶段 20：Import 契约、缓存与发布

状态：`complete`

1. Import Schema 接受 `reveal_to_reader`。
2. `validateBatch` 支持批次内 `establish → reveal_to_reader`。
3. 跨批次 ledger 显示 Reader 是否已知。
4. `publishChapter` 真实发布 reader reveal。
5. 提升 `analysisSchemaVersion`，使旧 v3 分析缓存失效；不提升 workspace 版本。
6. Import Prompt 只有正文明确向读者揭示 Truth 时才输出 reveal。

---

### 阶段 21：Writer Context 的信息差表达

状态：`complete`

目标：Writer 能区分“读者知道”和“角色知道”，但不获得无关隐藏 Truth。

选择规则：

1. 当前大纲涉及角色已知的 Truth：继续注入。
2. 已 `ReaderRevealedAt < currentChapter` 的 Truth：即使当前角色未知，也可注入，并明确标记 ReaderKnown/角色未知，用于戏剧性反讽。
3. 作者已建立但读者和当前角色都未知的 Truth：不注入，避免泄底。
4. 当前章或未来才 reveal 的 Truth：不注入。
5. 继续沿用最多 8 条与预算裁剪，不新增检索服务。

TDD 场景：

```text
读者已知、林墨未知 → 当前林墨章节可看到信息差标记
读者未知、苏晚独知 → 当前林墨章节不可看到 Truth
```

---

### 阶段 22：Writer / Editor 纪律

状态：`complete`

Writer：

- `reveal_to_reader` 只在正文明确让读者知道完整 Truth 时提交；
- reveal 不等于任何角色知道；
- 使用 ReaderKnown Truth 制造信息差时，角色行为仍受 KnownBy 限制；
- 不为一般暗示、模糊怀疑或伏笔强化提交完整 reveal。

Editor：

- 检查提前泄底；
- 检查重复揭秘；
- 检查读者已知但角色未知时，角色是否越权；
- 不把 `partial_payoff` 自动当成完整 reader reveal。

同步 Revision/Import Prompt 和 Writer golden。

---

### 阶段 23：全量验证与范围审计

状态：`complete`

验证矩阵：

| 接缝 | 必须覆盖 |
|---|---|
| Store | reveal、未知引用、幂等、Markdown |
| ChapterFacts | 新枚举和 Validate |
| Commit | 正常提交、同 payload、Pending 前拒绝、重放 |
| Projector | 正常重建、非法历史、首次 reveal |
| Rewrite | 删除 establish 拒绝；删除 reveal 允许 |
| Import | Schema、ledger、缓存版本、publish |
| Context | ReaderKnown/CharacterKnown 信息差、有界裁剪 |
| Prompt | Writer/Editor/Revision/Import 纪律 |
| Compatibility | 旧 KnowledgeEntry 缺 ReaderRevealedAt 时为 0 |

最终命令：

```bash
gofmt -w <本批 Go 文件>
go test ./internal/store -run 'Knowledge|Reader' -count=1
go test ./internal/tools -run 'Knowledge|Reader|ContextTool' -count=1
go test ./internal/revision -run 'Knowledge|Reader' -count=1
go test ./internal/host/imp -run 'Knowledge|Reader|AnalysisSchemaVersion' -count=1
go test ./assets -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

范围审计：

- [x] 不实现错误信念、撤销揭示或多读者模型
- [x] 不增加 Service、Repository、数据库或格式迁移
- [x] 不把全部作者 Truth 暴露给 Writer
- [x] 不用 Prompt 代替引用、幂等和 Saga 校验
- [x] 与现有 establish/learn 完整兼容

---

## 里程碑 C1 完成定义

以下条件已满足：

1. `reveal_to_reader` 在 Store、ChapterFacts、Commit、Projector、Rewrite、Import 和 Context 中语义一致。
2. 未知 reveal 在 PendingCommit 前拒绝，重复 reveal 保留首次 `ReaderRevealedAt`。
3. reveal 只接受 ID，不携带 Truth 或 Character；三层确定性校验一致。
4. 删除唯一 establish 且后续仍 reveal 时，Rewrite 在 PendingCommit 前拒绝；删除 reveal 本身允许。
5. Context 只注入当前角色已知或当前章之前读者已知的 Truth，不暴露当前章/未来揭示或无关隐藏 Truth。
6. Import 分析 Schema 升级为 v4，旧 v3 缓存失效；旧 KnowledgeEntry 缺字段时兼容为 0。
7. 全量测试、vet、diff check 和范围扫描全部通过。

---

## 里程碑 C2a：最小角色错误信念

### 目标

在现有 Knowledge 主链中表达：

```text
客观 Truth 已建立
→ 某角色形成一个明确但错误的认知
→ 该角色后来 learn 客观 Truth
→ 错误信念被标记为已纠正
```

C2a 只增加 `believe` 动作，不新增 `correct_belief`。现有 `learn` 表示角色确实获知客观 Truth，因此同时纠正该角色对同一 Knowledge ID 的活跃错误信念。

### 最小领域模型

建议在现有类型上增量扩展：

```go
type KnowledgeBelief struct {
    Character   string `json:"character"`
    Content     string `json:"content"`
    FormedAt    int    `json:"formed_at"`
    CorrectedAt int    `json:"corrected_at,omitempty"`
}

type KnowledgeEntry struct {
    ID               string
    Truth            string
    EstablishedAt    int
    KnownBy          []KnowledgeHolder
    BelievedBy       []KnowledgeBelief
    ReaderRevealedAt int
}

type KnowledgeUpdate struct {
    ID        string
    Action    string // establish / believe / learn / reveal_to_reader
    Truth     string
    Character string
    Belief    string
}
```

### 动作与不变量

`believe`：

- 必须引用已建立的 Knowledge ID；
- Character 和 Belief 必填，Truth 为空；
- Belief 必须与客观 Truth 不同；
- 已经在 KnownBy 的角色不能形成该错误信念；
- 同角色、同内容重复 believe 幂等，保留首次 FormedAt；
- 同角色、不同内容的后续 believe 第一版拒绝，不做中途改写；
- 已纠正后不能再次 believe。

`learn`：

- 继续沿用现有获知客观 Truth 语义；
- 首次 learn 添加 KnownBy；
- 同时将该角色的活跃错误信念 `CorrectedAt` 设为当前章；
- 重复 learn 幂等，不漂移 LearnedAt/CorrectedAt。

同 payload 合法序列：

```text
establish → believe
establish → believe → learn
```

### Context 安全边界

不得继续把完整 `KnowledgeEntry` 直接作为 Writer 的 `knowledge_boundaries`。C2a 必须引入仅用于序列化的净化视图，而不是新事实源：

```go
type knowledgeBoundary struct {
    ID          string
    Truth       string // 仅当前角色已知或读者已知时输出
    ReaderKnown bool
    KnownBy     []KnowledgeHolder
    Beliefs     []KnowledgeBelief // 仅当前大纲涉及角色的活跃 belief
}
```

规则：

1. 当前角色只相信错误内容、读者也未知 Truth 时：输出 belief，不输出 Truth；
2. 读者已知 Truth、角色仍错误相信时：同时输出 Truth + belief，支持戏剧性反讽；
3. 角色 learn 后：输出 Truth，不再把已纠正 belief 作为当前认知；
4. 无关角色 belief 不进入当前章 Context；
5. 当前章或未来才形成/纠正的认知不提前注入；
6. 仍使用 8 条上限、预算裁剪和 `_trimmed`。

`knowledgeBoundary` 只是 Context Adapter DTO；Domain/Store 中的 `KnowledgeEntry` 仍是事实投影。

### 本里程碑明确不做

- 不新增 `correct_belief`、`doubt`、`suspect`、`forget`；
- 不允许同一角色对同一 ID 同时持有多个错误信念；
- 不允许错误信念中途改写或纠正后再次形成；
- 不追踪读者错误信念或不可靠叙述；
- 不实现多读者群体；
- 不新增 Service、Repository、数据库或通用认知状态机；
- 不把 belief 塞进通用 `StateChange.Field`；
- 不提升 ChapterRecord/workspace 格式版本，除非兼容测试证明必须。

### 已确认公共接缝

1. `WorldStore.UpdateKnowledge` / `LoadKnowledgeState`
2. `chapterfacts.Properties` / `Validate`
3. `CommitChapterTool.Execute`
4. `revision.ValidateRecordSet` / `Projector.Apply`
5. Import `analysisContract → validateBatch → buildLedger → publishChapter`
6. `ContextTool.Execute`
7. Writer / Editor / Revision / Import Prompt

---

### 阶段 24：基线与 Store 首个 believe 切片

状态：`complete`

- [x] 确认生产代码无改动（允许三份规划文件）且基线为 `03bf271`
- [x] 定向运行现有 Knowledge Store 测试
- [x] 首个失败测试：
  ```text
  establish@1 → 林墨 believe@2 "黑影是杀兄仇人"
  ```
- [x] 断言 Truth 保留、BelievedBy 仅一条、FormedAt=2、CorrectedAt=0、KnownBy 为空
- [x] 只增加 `KnowledgeBelief`、`KnowledgeUpdate.Belief` 与 Store believe 分支
- [x] 不先修改 ChapterFacts、Commit、Projector 或 Prompt

---

### 阶段 25：Store 不变量与 learn 纠正

状态：`complete`

严格 TDD 覆盖：

1. 未知 Knowledge ID 不能 believe；
2. believe 缺 Character/Belief 拒绝；
3. Belief 与 Truth 相同拒绝；
4. 同角色同内容重复 believe 幂等；
5. 同角色不同内容后续 believe 拒绝；
6. 已知 Truth 的角色不能 believe；
7. `believe@2 → learn@4` 后：
   ```text
   LearnedAt = 4
   CorrectedAt = 4
   ```
8. 重复 learn 不漂移；
9. 纠正后再次 believe 拒绝；
10. Markdown 分别显示活跃错误信念与已纠正章节；
11. 旧 KnowledgeEntry 缺 `believed_by` 时解码为空。

---

### 阶段 26：ChapterFacts 与 Commit Saga

状态：`complete`

1. 严格 Schema 动作枚举扩展为：
   ```text
   establish / believe / learn / reveal_to_reader
   ```
2. 字段纪律：
   - establish：Truth 必填，其余空；
   - believe：Character + Belief 必填，Truth 空；
   - learn：Character 必填，Truth/Belief 空；
   - reveal_to_reader：只接受 ID。
3. 提交前临时状态按 payload 顺序模拟 Belief/KnownBy：
   - 支持 `establish → believe → learn`；
   - 拒绝未知引用、真信念、已知后 believe、冲突 belief；
   - 所有非法请求在 PendingCommit 前拒绝。
4. started 重放不复制 Belief，不漂移 FormedAt/CorrectedAt。
5. 不改变现有 Commit Saga 阶段。

---

### 阶段 27：Revision、Projector 与 Rewrite

状态：`complete`

Projector 路径：

```text
establish@1 → believe@2 → learn@5
```

断言：

- Belief FormedAt=2；
- CorrectedAt=5；
- KnownBy LearnedAt=5；
- 重复历史动作幂等；
- 非法 believe 历史记录拒绝。

Rewrite 安全：

- 删除唯一 establish、后续仍 believe 时 Pending 前拒绝；
- 删除 believe 本身允许；
- 删除 learn 后，原 belief 恢复为活跃状态；
- 修改 belief 内容通过候选记录集重新投影，不保留旧内容；
- 不新增 `RestoreOwnBelief`。

---

### 阶段 28：Import 契约、连续性与缓存

状态：`complete`

1. Import Schema/DTO 支持 `believe` 和 `belief` 字段；
2. 首批与同批次 `establish → believe → learn` 校验；
3. 跨批次 ledger 显示：客观 Truth、读者知情、角色已知、活跃错误信念；
4. Import Prompt 只有正文明确呈现角色相信某个具体错误内容时才输出 believe；
5. publish 真实落盘并由 learn 纠正；
6. `analysisSchemaVersion 4 → 5`，旧 v4 缓存失效；
7. 不提升 workspace/ChapterRecord 版本。

---

### 阶段 29：Writer Context 净化视图

状态：`complete`

这是 C2a 的阻塞安全阶段。

TDD 矩阵：

| 场景 | Context 输出 |
|---|---|
| 林墨只持有错误 belief，读者未知 | 输出 belief；不输出 Truth |
| 林墨持有错误 belief，读者已知 | 输出 belief + Truth + ReaderKnown |
| 苏晚独有 belief，当前章只涉及林墨 | 不输出苏晚 belief 或对应隐藏 Truth |
| 林墨已 learn | 输出 Truth；活跃 beliefs 为空 |
| belief/learn 发生在当前章或未来 | 不提前输出 |

实现要求：

- Context 改为净化后的 `knowledgeBoundary` DTO；
- 不在 DTO 中暴露无权看到的 Truth；
- 不改变 Store 事实结构；
- 保持最多 8 条、最近优先、预算裁剪和 `_trimmed`；
- 增加 JSON 级测试，不能只用字符串 contains 证明无泄漏。

---

### 阶段 30：Writer / Editor / Revision / Import 纪律

状态：`complete`

Writer：

- 角色按自己的 active belief 行动，但不得把 belief 当成客观 Truth；
- ReaderKnown Truth 可与角色错误 belief 构成戏剧性反讽；
- 只有正文明确让角色相信具体错误内容时提交 believe；
- 角色确知 Truth 时提交 learn，代码负责纠正 belief。

Editor：

- 检查角色行为是否符合其 active belief；
- 检查已纠正后是否仍按旧 belief 行动；
- 检查叙述是否把角色误解写成作者事实；
- 检查隐藏 Truth 是否提前泄漏。

Revision/Import：

- 仅提取正文明确支持的具体信念；
- 不把怀疑、猜测或一闪而过的念头当成 believe；
- 同步 Writer golden 与资源契约。

---

### 阶段 31：全量验证、范围审计与提交门禁

状态：`complete`

验证矩阵：

| 接缝 | 必须覆盖 |
|---|---|
| Store | believe、输入约束、幂等、learn 纠正、Markdown |
| ChapterFacts | 四动作 Schema、字段纪律 |
| Commit | 正常提交、同 payload、Pending 前拒绝、started 重放 |
| Projector | 重建、纠正、非法历史、Rewrite 恢复 |
| Import | Schema、ledger、publish、v4 缓存失效 |
| Context | Truth 净化、角色过滤、时间边界、有界裁剪 |
| Prompt | Writer/Editor/Revision/Import belief 纪律 |
| Compatibility | 旧 JSON 缺 believed_by/belief 字段 |

最终命令：

```bash
gofmt -w <本批 Go 文件>
go test ./internal/store -run 'Knowledge|Belief' -count=1
go test ./internal/tools -run 'Knowledge|Belief|ContextTool' -count=1
go test ./internal/revision -run 'Knowledge|Belief' -count=1
go test ./internal/host/imp -run 'Knowledge|Belief|AnalysisSchemaVersion' -count=1
go test ./assets -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

范围审计：

- [x] 不新增 correction/doubt/suspect/forget 动作
- [x] 不实现读者错误信念、多 belief 或 belief 再激活
- [x] 不泄漏当前角色和读者均未知的 Truth
- [x] 不新增 Service、Repository、数据库或格式迁移
- [x] 不用 Prompt 代替引用、幂等、纠正和 Saga 校验
- [x] 与 establish/learn/reveal_to_reader 完整兼容

提交门禁：

- 只有阶段 24—31 全部完成且最终验证通过，才允许提交；
- 提交信息必须使用中文，例如：
  ```text
  功能：追踪角色错误信念与纠正状态
  ```

---

## 里程碑 D：Import 发布前全书事实重放门禁

### 目标

修复全项目 Review 发现的 P0 缺口：Import 不能只在单批次内验证事实，也不能把跨批次生命周期合法性寄托在 Prompt ledger 上。

目标链路：

```text
逐章分析批次
→ 当前批次字段/章号校验
→ 已落盘前缀 + 当前候选批次的全书事实重放
→ 通过后才写分析工件
→ synthesis 前再次验证全部分析工件
→ publish 前最后验证
→ 通过后才修改正式 Store
```

非法跨批次事实必须：

```text
定位首个失败章 N
→ 失效 analyses/N..end 与依赖的 synthesis/story-resolution
→ 返回可诊断错误
→ 下次 LoadState 得到 AnalyzedChapters=N-1
→ NextAction 自然回到 ActionAnalyze
```

### Review 复现场景

当前 `validateBatch` 每批创建空 map；`start > 0` 时未知引用会被有意放行，因此以下错误只能在正式发布时由 Commit 发现：

```text
批次 1：establish k_shadow → 林墨 learn k_shadow
批次 2：林墨 believe k_shadow "黑影是仇人"
```

现状失败路径：

```text
分析工件全部落盘
→ synthesis 完成
→ publishFoundation 已修改正式 Store
→ 前几章提交完成
→ 错误章 Commit 拒绝
→ NextAction 仍为 ActionPublish
→ 重跑卡在同一章
```

### 设计决策

1. **复用 `revision.ValidateRecordSet`**：它已经是 Rewrite/Migration 的纯全量事实重放接缝，第二个适配器（Import）复用后可同时覆盖 Knowledge、Foreshadow 和 ChapterFacts 字段纪律。
2. **不在 Import 再造生命周期状态机**：`validateBatch` 保留模型返回的局部形状/章号校验；跨章领域合法性由全书 ChapterRecord 重放裁决。
3. **统一 Import→ChapterFacts 映射**：全书门禁和 `commitArgs` 必须消费同一映射，避免“验证的是一份事实，发布的是另一份事实”。
4. **正常路径先验证后写工件**：`AnalyzeNext` 将 prior facts 与候选 batch 合并重放，失败时当前批次不落盘，因此无需回滚。
5. **截断打捞不可绕过门禁**：salvaged prefix 在落盘前也必须与 prior facts 合并重放。
6. **综合/发布防御性复验**：用于发现旧工件、手工修改或历史 Bug 产物；失败时只删除可重建的 Import 派生工件，不碰源快照、切分确认或正式 Store。
7. **缓存版本 `5 → 6`**：验证语义属于分析契约的一部分；旧 v5 工件未经跨批次重放证明，必须自然失效。
8. **不新增 Import Action/Stage**：仍使用现有 `Facts + NextAction` 推导；失效分析尾部后自然回到 `ActionAnalyze`。

### 已确认接缝

1. `AnalyzeNext(...)`：候选批次与截断打捞的公开分析接缝。
2. `revision.ValidateRecordSet(records)`：全书事实可重放接缝。
3. `LoadState / CollectFacts / NextAction`：失效后恢复动作推导。
4. `runner.synthesize`：综合前防御门禁。
5. `runner.publish`：修改正式 Store 前最后门禁。
6. `publishChapter → CommitChapterTool.Execute`：正式发布协议，保持不变。

---

### 阶段 32：基线、缓存版本与旧工件隔离

状态：`complete`

1. 确认工作区只含本轮规划文件，基线 HEAD 为 `3ee475c`。
2. 运行现有 Import/Revision 定向测试，固定绿灯基线。
3. 首个失败测试手工按 v5 digest 写入分析工件，要求当前版本不再复用。
4. 最小实现：
   ```text
   analysisSchemaVersion 5 → 6
   ```
5. 不提升 `workspaceSchemaVersion` 或 `ChapterRecordVersion`。

验收：旧 v5 逐章分析自然失效，`LoadState` 回到分析阶段。

---

### 阶段 33：统一 ImportedChapterFacts 映射与纯全书门禁

状态：`complete`

先建立一份映射：

```text
ImportedChapterFacts
→ domain.ChapterFacts
→ domain.ChapterRecord（仅用于纯验证）
```

映射必须包含：

- title / summary / characters / key_events；
- timeline / foreshadow / relationship / state / knowledge updates；
- hook_type / dominant_strand；
- key_events 为空时继续复用现有 `core_event` fallback；
- 不伪造 cast_intros、feedback 或 StyleDelta。

TDD 切片：

1. 映射契约测试证明门禁映射与 `commitArgs` JSON round-trip 后的 ChapterFacts 一致。
2. 纯事实序列合法：
   ```text
   establish@1 → believe@2 → learn@3 → reveal@4
   ```
3. 纯事实序列非法：
   ```text
   establish+learn in prior batch → believe in later batch
   ```
4. 再覆盖：
   - 跨批次冲突 Truth；
   - 跨批次未知 Knowledge 引用；
   - 伏笔 `resolve` 后下一批 `advance/reinforce/partial_payoff`；
   - 合法跨批次伏笔生命周期。
5. 实现只调用 `revision.ValidateRecordSet`，不复制 Store/Projector switch。

验收：同一映射同时服务全书门禁与正式发布参数。

---

### 阶段 34：AnalyzeNext 候选批次累计重放

状态：`complete`

首个公开行为测试：

1. 工作区已有新鲜的批次 1 分析：
   ```text
   establish k_shadow → 林墨 learn
   ```
2. mock 模型对批次 2 返回：
   ```text
   林墨 believe k_shadow
   ```
3. 调用 `AnalyzeNext`。
4. 断言：
   - 返回确定性语义错误；
   - 批次 2 的分析工件未写入；
   - `analyzedChapters` 仍停在批次 1；
   - 正式 Store 未发生任何修改。

最小实现：

```text
validateBatch(current)
→ validateImportedFactSequence(prior + current)
→ 全部通过
→ writeArtifact(current)
```

合法跨批次场景必须正常写入当前批次。

不得通过扩大 Prompt、重写 ledger 或延迟到 publish 修复。

---

### 阶段 35：截断前缀打捞也必须经过累计门禁

状态：`complete`

`salvagePrefix` 当前可能在结构化输出截断后直接写入合法 JSON 前缀。新增测试：

1. prior facts 已使林墨知道 Truth；
2. 截断响应的第一份完整候选章让林墨重新 believe；
3. JSON 形状和单章 `validateBatch` 可通过；
4. 累计全书重放必须拒绝；
5. 不写入任何 salvaged analysis 工件；
6. failure metadata 保留原始响应与“累计事实非法”的诊断。

合法 salvage prefix 继续可落盘，不能因门禁取消现有截断恢复能力。

---

### 阶段 36：首个失败章定位与派生工件失效

状态：`complete`

为旧工件、手工修改和历史 Bug 产物增加错误路径修复：

1. 先一次性验证全部 facts；正常路径保持 O(n)。
2. 仅在失败路径逐前缀定位首个非法章节 N；允许 O(n²)，因为只在异常修复路径执行。
3. 复用现有：
   ```text
   discardAnalysesAfter(w, N-1, total)
   ```
4. 同时失效：
   - `synthesis.json`
   - `story-resolution.json`
   - 依赖分析事实的 range digest 缓存（若现有 digest 机制已能自然失效，则用测试证明后可不删除）。
5. 不删除：
   - manifest / intent / source；
   - guidance / segmentation / confirmation；
   - N 之前已经证明合法的分析工件；
   - 正式 Store 文件。
6. 删除失败必须显式返回，不得宣称已回退。

恢复验收：

```text
LoadState.AnalyzedChapters == N-1
NextAction == ActionAnalyze
```

---

### 阶段 37：Synthesis 前全书门禁

状态：`complete`

集成测试构造当前版本但跨批次非法的全部分析工件，然后调用综合流程：

- 不调用 Synthesize 模型；
- 不写新的 `synthesis.json`；
- 定位首个失败章并失效尾部；
- `NextAction` 回到 `ActionAnalyze`；
- 错误包含章节号和原始领域原因。

合法全书继续综合，现有 range digest 与 synthesis InputDigest 语义保持不变。

---

### 阶段 38：Publish 前最后门禁与正式 Store 零污染

状态：`complete`

防御性集成测试：

1. 准备 segmentation、全部分析、synthesis 和 story resolution 工件；
2. 分析事实含跨批次非法 Knowledge 或 Foreshadow 历史；
3. 调用 publish 流程；
4. 断言在以下动作前失败：
   ```text
   publishFoundation
   setCompletionHold
   publishChapter
   ```
5. 正式 Store 保持空：
   - Book 未写入；
   - Premise/Outline 未写入；
   - CompletedChapters 为空；
   - PendingCommit 为空；
   - AdvanceHold 未建立。
6. Import 派生工件按阶段 36 回退，下一次动作是 `ActionAnalyze`。

合法 Import 发布和崩溃恢复测试必须保持通过。

---

### 阶段 39：文档、全量验证与中文提交门禁

状态：`complete`

文档只同步本次稳定事实：

- `docs/import-pipeline.md` 增加“批次局部校验 + 累计全书重放 + 发布前门禁”；
- `docs/architecture.md` 在 Import 描述中明确正式 Store 修改前已证明 ChapterFacts 可重放；
- 不在本批归档历史规划文件，不顺带编写完整 Knowledge glossary。

最终验证：

```bash
gofmt -w <本批 Go 文件>
go test ./internal/host/imp -run 'Analyze|Import|Publish|Fact|Validation' -count=1 -timeout=5m
go test ./internal/revision -run 'ValidateRecordSet|Projector' -count=1
go test ./internal/tools -run 'CommitChapter' -count=1
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/host/imp ./internal/revision ./internal/tools -timeout=10m
git diff --check
```

范围审计：

- [x] 不新增 Import Action/Stage 或第二套恢复状态机；
- [x] 不复制 Knowledge/Foreshadow 生命周期规则到 Import；
- [x] 不修改正式 Commit Saga 阶段；
- [x] 不引入数据库、Service、Repository 或格式迁移；
- [x] 只有全部事实验证通过后才允许正式 Store 首次写入；
- [x] 非法工件能够自然回到 `ActionAnalyze`，不形成永久 `ActionPublish` 循环；
- [x] 与现有截断打捞、InputDigest、发布恢复和已发布终态兼容。

建议中文提交信息：

```text
修复：导入发布前重放并验证全书事实
```

---

## 里程碑 D 完成定义

1. 每个新分析批次在落盘前与既有前缀共同通过 `revision.ValidateRecordSet`。
2. 截断打捞前缀不能绕过累计事实门禁。
3. Synthesis 与 Publish 都在副作用前复验全部分析事实。
4. 非法旧工件会从首个失败章回退，下一动作自然变为 `ActionAnalyze`。
5. 正式 Store 在全书事实验证通过前保持不变。
6. v5 分析缓存自然失效；workspace/ChapterRecord 版本不变。
7. 全量测试、race、vet、diff check 全部通过。
8. 没有引入第二套领域规则或 Import 状态机。

---

## 后续候选里程碑（D 完成后，不进入当前执行范围）

优先级按当前仓库增量价值调整为：

1. **Prose Lint 增量**——优先做重复段落，独立评估短段、对白、回环修辞的误报率。
2. **具体题材/平台资源试点**——在现有 References/覆盖层中先落一个题材或 rubric，不先建 Pack 框架。
3. **cocreate 阶段化访谈**——扩展现有共创对话，不先加第三启动模式。
4. **评审商业维度**——利用现有 ChapterContract/Review Dimensions 增加平台适配或商业节奏 rubric。
5. **扫榜与拆文**——保持独立命令和抽象产物，不侵入主写作 Engine。
6. **更复杂认知状态**——只有出现真实需求后再讨论 doubt/suspect/forget、belief 改写、读者错误信念或多读者模型。

明确暂缓：

- 独立心理状态账本；
- 新 preferences.json；
- PostgreSQL/Web 后端；
- Prompt 社区/工坊；
- 并行写相邻章节；
- 一次性 GenrePack/PlatformPack/StylePack/BenchmarkPack/QualityPack 抽象。

## 里程碑 A 完成定义

以下条件已全部满足：

1. 2 个 S1 和 2 个 S2 全部有回归测试并修复。
2. 2 个 S3 已修复或由用户明确延期。
3. 正常提交、PendingCommit 恢复、章节重写、Projector 重建结果一致。
4. `resolved` 终态在三个接缝中一致。
5. 全量测试、vet、diff check 全部通过。
6. 未引入计划外架构或领域扩张。

## 遇到的错误

| 错误 | 尝试次数 | 处理方案 |
|---|---:|---|
| `rg: command not found` | 1 | 后续使用 `search_files` 或系统自带 `grep`，不再重复调用 `rg` |
| 全量测试未发现 Projector 缺少新动作 | 1 | 增加 Projector 和 Rewrite 集成接缝测试，而不是依赖现有全量绿灯 |
| Saga 夹具批量替换存在非唯一文本 | 1 | 不重复批量替换；改用包含函数上下文的唯一片段逐段编辑 |
| 进度表追加时匹配串不一致 | 1 | 先搜索实际行，再用唯一文本追加；不重复原替换 |
| `check-complete.sh` 未识别项目根规划文件 | 1 | 脚本返回 0 但报告未找到；不重复调用，改用搜索确认无 pending/in_progress/未勾选项 |
| E1 阶段追加锚点已被前次收口改写 | 1 | 头部状态已成功更新；不重复旧锚点替换，读取文件末尾后用真实唯一文本追加 |
| 大纲角色字段搜索正则未转义 `[]` | 1 | 不重复该正则；直接读取 `domain/story.go` 核对结构 |

---

# 里程碑 E1：Knowledge 纯 Apply/Replay 规则收敛

## 目标

把 Knowledge 生命周期从 Store、Commit 临时模拟和 Revision Projector 三份实现，收敛为专用纯函数：

```go
ApplyKnowledgeUpdates(current, chapter, updates) ([]KnowledgeEntry, error)
```

纯函数不做 IO、锁、事务、Prompt、Context、Markdown 或日志；输入不被修改，错误不产生部分结果。

## 已确认公共接缝

1. `domain.ApplyKnowledgeUpdates`：纯领域规则。
2. `WorldStore.UpdateKnowledge/LoadKnowledgeState`：增量持久化。
3. `CommitChapterTool.Execute`：首次冻结前试运行与冻结 payload 恢复。
4. `revision.ValidateRecordSet/Projector.Apply`：全量重建。
5. Import 继续通过 Revision 间接消费，不增加规则。

## 阶段 40：纯 Apply establish

状态：`complete`

- 空投影 establish；输入不变。
- 同 ID 同 Truth 幂等。
- 冲突 Truth 拒绝且输入不变。
- 首轮不接适配器。

## 阶段 41：补齐 believe / learn / reveal

状态：`complete`

- believe 的形成、幂等、冲突、已知后拒绝。
- learn 的首次章节与 active belief 纠正。
- reveal 的首次章节与角色独立性。
- 同章 `believe → learn` 冻结 payload 完整重放幂等。
- 深拷贝和错误原子性。

## 阶段 42：WorldStore 接入

状态：`complete`

`UpdateKnowledge` 只保留锁、读、纯 Apply、JSON/Markdown 写入；删除四动作 switch。

## 阶段 43：Projector 接入

状态：`complete`

`projectKnowledge` 从空投影按章调用纯 Apply；删除 Knowledge switch，并保留章号错误上下文。

## 阶段 44：Commit 前置试运行接入

状态：`complete`

首次 PendingCommit 前加载投影并调用纯 Apply；删除 truth/knownBy/beliefBy 临时模拟。恢复冻结 payload 时仍不按当前投影重新裁决。

## 阶段 45：跨接缝等价与 Saga 回归

状态：`complete`

Store 增量、Projector 全量、Commit 同 payload/重放和 Import 全书门禁结果一致。

## 阶段 46：重复规则与范围审计

状态：`complete`

Store、Commit、Projector 不再维护 Knowledge 生命周期 switch；字段矩阵、Import 局部反馈、Context 净化和 Markdown 保持各自职责。禁止新动作、通用状态机、Service 或 Repository。

## 阶段 47：全量验证与中文提交

状态：`complete`

```text
go test ./internal/domain ./internal/store ./internal/tools ./internal/revision ./internal/host/imp -count=1 -timeout=5m
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/store ./internal/tools ./internal/revision ./internal/host/imp -timeout=10m
git diff --check
```

提交信息：

```text
重构：统一知识状态的应用与重放规则
```
