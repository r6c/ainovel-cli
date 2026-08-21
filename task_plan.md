# ainovel-cli 分阶段能力演进计划

## 规划总览

- 总体状态：`planned`
- 当前执行门禁：阶段 8——隔离已完成的伏笔生命周期批次
- 下一领域里程碑：最小知识事实闭环（作者真相 + 角色获知）
- 规划原则：先复用既有 ChapterFacts/Commit Saga/WorldStore/Projector/Context，再增加最小领域数据；不从参考项目移植运行架构
- 当前风险：工作区已有 18 个受跟踪文件、812+ / 51- 的未提交伏笔改动，禁止直接叠加知识状态实现

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

状态：`pending`

执行门禁：在开始知识状态代码前，必须任选一种方式建立独立边界：

1. 用户授权创建 Git 提交；或
2. 用户自行提交；或
3. 明确建立可恢复的独立分支/stash（不得丢失当前 18 个文件改动和规划文件）。

门禁验收：

- [ ] 当前伏笔批次有独立提交或等价可恢复边界
- [ ] `go test ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] `git diff --check` 通过
- [ ] 下一批 diff 不与伏笔批次混杂

在门禁完成前，不修改知识状态生产代码。

---

### 阶段 9：Knowledge Store 首个纵向切片

状态：`pending`

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

状态：`pending`

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

状态：`pending`

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

状态：`pending`

1. 给 `ImportedChapterFacts` 增加 `KnowledgeUpdates`。
2. 同步 import 严格 Schema 和 Prompt。
3. 导入连续性账本必须允许批次内 `establish → learn`。
4. `publish.go` 必须把知识更新传入 `commit_chapter`。
5. 加集成测试证明导入章节发布后知识投影存在。
6. 不将人物证据或世界证据自动猜测为知识状态；只有正文明确表明角色获知时才输出 learn。

---

### 阶段 13：Writer Context 有界消费

状态：`pending`

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

状态：`pending`

只在代码事实闭环完成后更新 Prompt：

- Writer 仅记录正文实际建立或获知的知识变化；
- `establish` 不等于角色知道；
- 角色不得使用其 KnownBy 中不存在的信息；
- Editor 一致性检查增加“角色越权知情”；
- 不让 Prompt 承担未知引用、冲突 Truth 或 Saga 校验。

同步资源 golden/契约测试。

---

### 阶段 15：全量验证与范围审计

状态：`pending`

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

- [ ] 没有新增 Service、Repository 或通用状态机接口
- [ ] 没有实现 belief、reader reveal、forget 或 reveal plan
- [ ] 没有新增数据库或 Web 事实源
- [ ] 没有复制外部项目代码/Prompt
- [ ] 正常提交、重放、重写、导入、Context 结果一致

---

## 后续候选里程碑（不进入当前执行范围）

优先级按当前仓库增量价值调整为：

1. **知识阶段 C：错误信念与读者已知**——只在 `establish/learn` 稳定后设计。
2. **Prose Lint 增量**——在 `rules.Lint` 中优先做重复段落、异常标点或章节截断，每条规则独立评估误报率。
3. **具体题材/平台资源试点**——在现有 References/覆盖层中先落一个题材或 rubric，不先建 Pack 框架。
4. **cocreate 阶段化访谈**——扩展现有共创对话，不先加第三启动模式。
5. **评审商业维度**——利用现有 ChapterContract/Review Dimensions 增加平台适配或商业节奏 rubric。
6. **扫榜与拆文**——保持独立命令和抽象产物，不侵入主写作 Engine。

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
| 大纲角色字段搜索正则未转义 `[]` | 1 | 不重复该正则；直接读取 `domain/story.go` 核对结构 |
