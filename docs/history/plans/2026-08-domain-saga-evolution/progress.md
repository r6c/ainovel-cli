# ainovel-cli 分阶段能力演进进度

## 2026-08-24：里程碑 G 启动

- HEAD：`f5e91de 加固：校验章节提交冻结载荷的完整性`；工作区开始时干净。
- 根规划文件合计 138,865 字节，已同时包含早期计划、红灯过程、过期工作区状态和稳定结论。
- README/architecture 尚未出现 KnowledgeEntry、Author Truth、KnownBy、ReaderKnown、KnowledgeBelief 等稳定词汇。
- 本批只做文档归档与导航，不修改 Go 运行时。

## 会话信息

- 任务：伏笔、知识状态、读者揭示与角色错误信念分阶段闭环
- 当前状态：里程碑 A、B、C1、C2a 全部完成
- 执行原则：严格 TDD，红→绿，单切片推进
- 项目根：`ainovel-cli`

## 2026-08-24：里程碑 F 启动

- HEAD：`429bb4a 重构：统一伏笔生命周期的应用与重放规则`；工作区开始时干净。
- session-catchup 未报告未同步上下文。
- 已确认公共接缝：CommitChapterTool 首次冻结/恢复、SignalStore PendingCommit 持久化。
- PendingCommit 当前无摘要字段；恢复会在解码冻结 payload 后直接继续 Saga。
- 普通提交与 Rewrite 各有一个 Pending 构造点，后续必须共享密封逻辑。
- 搜索 WriteJSON 声明时一次不完整正则导致解析失败；未重试同一查询，改用字面搜索。
- 随后搜索 ChapterRecord Load 声明又误用了未闭合括号正则；停止猜测声明形式，直接复用仓库现有 `ChapterRecords.Load` 调用观察副作用。
- 阶段 56 首个红灯：带 v1 摘要但 payload 已被替换的 PendingCommit 被错误恢复并提交，证明当前完全忽略密封字段。
- `SignalStore.WriteJSON` 使用 MarshalIndent，会改变 RawMessage 的排版空白；摘要采用 `json.Compact` 后的冻结 JSON 字节，保护字段顺序/内容而忽略无意义空白。

### 阶段 56 完成

v1 sealed Pending 的 payload 或 DraftContent 被替换时，恢复均在 ChapterRecord/Progress 等副作用前拒绝并保留 Pending。旧无摘要重放测试继续通过。

### 阶段 57 完成

摘要匹配但字段矩阵非法的冻结 payload 曾先写 ChapterRecord。现已拆出 `validateFrozenCommitArgs`，首次与恢复均执行 ChapterFacts/章号纯校验；Knowledge/Foreshadow 当前投影试运行仍仅在首次冻结前执行。

### 阶段 58 完成

新增 `sealPendingCommit`：SealVersion=1，payload 使用 compact JSON SHA-256，draft 使用 UTF-8 SHA-256。普通提交与 Rewrite 两个首次 Pending 构造点共同调用；提交、返工、恢复和 SignalStore 测试全绿。

### 阶段 59 完成

定向测试后 tools 全包发现 `progress_marked` 旧 Pending 合法缺少 DraftContent。现已将冻结输入要求限定为 `started/state_applied`；后段 stage 只收尾结果。密封格式、摘要、元数据和纯字段纪律完整通过，tools 全包恢复全绿。

### 阶段 60 完成

合法 legacy started/state_applied 在 payload 解码、章号和纯字段校验后，先原子回写完整 v1 密封再重放；非法 legacy 不密封，升级后再次篡改会按 v1 拒绝。tools 全包通过。

### 阶段 61 完成

新增 sealed state_applied/signal_saved 合法恢复和四 Stage 篡改拒绝矩阵；started 的 Knowledge/Foreshadow/Reader、progress_marked、Rewrite 冻结正文既有测试继续通过。state_applied 夹具曾缺 Summary，补齐真实阶段工件后通过。

### 阶段 62 完成

新增 `ErrPendingCommitIntegrity`；payload/draft 摘要不匹配、半密封、未知版本均可用 errors.Is 稳定识别。错误提示检查 `meta/pending_commit.json`，不输出正文或完整 payload；业务字段/状态错误保留原分类。

### 阶段 63 收口审计红灯

真实 Rewrite 恢复测试确认：密封后同时把 `Rewrite=true/mode=rewrite` 改成普通提交的自洽组合，恢复会错误走普通路径并成功。最小修复增加 IntentDigest，覆盖 Chapter、Rewrite、RewriteMode；Stage/Output/Result/时间戳是 Saga 可变字段，不纳入。

IntentDigest 修复后，四阶段篡改测试先因手工 v1 夹具仍是旧“两摘要”形状而失败。这是夹具协议升级，不是产品回归；合法 v1 夹具改为调用 `sealPendingCommit` 后只篡改目标字段，故意半密封/未知版本夹具保持显式构造。同步夹具后一次编译错误来自同一作用域重复 `:= err`，改为赋值，不作为产品红灯。

### 阶段 63 / 里程碑 F 完成

生产审计确认只有普通提交与 Rewrite 两个首次 Pending 构造点，均在首次副作用前密封；后续 Stage 保存沿用同一密封字段。最终验证全部通过：

```text
go test ./internal/tools -run 'CommitChapter|PendingCommit|Replay|Integrity|Tampered|Sealed|Legacy' -count=1 -timeout=5m
go test ./internal/store -run 'PendingCommit|Signal' -count=1
go test ./internal/revision -run 'Projector|Rewrite|ValidateRecordSet' -count=1
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/store ./internal/tools ./internal/revision -timeout=10m
git diff --check
```

没有新增 Saga Stage、数据库、签名/HMAC、自动删除或自动重签损坏工件。

## 2026-08-24：里程碑 E2 启动

- HEAD：`a041b5c 重构：统一知识状态的应用与重放规则`；工作区开始时干净。
- session-catchup 未报告未同步上下文。
- 已确认公共接缝：domain 纯 Apply、WorldStore、CommitChapterTool、Revision Projector；Import 继续经 Revision 间接消费。
- Store/Commit/Projector 仍各自维护伏笔生命周期；E2 不新增伏笔状态或通用状态机。
- 下一步先锁定 Store 增量与 Projector 全量重建等价，再写纯 plant 红灯。

### 阶段 48 完成

新增真实双 Store 等价基线：`plant→reinforce→partial_payoff→advance→resolve` 经 WorldStore 增量与 Projector 全量重建后完整 ForeshadowEntry 完全一致。现有实现直接通过，作为重构保护保留。

### 阶段 49 完成

纯 plant 经红→绿锁定：建立 planted 条目、输入不变、重复 plant 不复制/不重新打开、旧投影空字段兼容补全、空 ID/Description 原子拒绝、nil 投影保真。追加测试时一次非唯一锚点编辑被拒绝，改用完整函数尾部后完成。

### 阶段 50 完成

advance、reinforce、partial_payoff、resolve 逐动作红→绿完成。纯规则统一未知引用、未来 PlantedAt、resolved 终态、Status/LastAdvancedAt/ResolvedAt 与同章 resolve 重放；跨章重复 resolve 保持既有赋值式语义。

### 阶段 51 完成

`WorldStore.UpdateForeshadow` 已只保留写锁、读取、纯 Apply 与 JSON/Markdown 写入；原动作 switch 删除。全部 Foreshadow 定向测试及 Store 全包通过。

### 阶段 52 完成

`revision.projectWorld` 保留 Timeline/关系/状态聚合职责，伏笔部分改为逐章调用纯 Apply 并补充章号错误上下文；原伏笔 switch 删除。Foreshadow/Projector/ValidateRecordSet 定向测试及 revision 全包通过。

### 阶段 53 完成

Commit 删除 plantedAt/status 临时模拟，改用完整账本纯 Apply 试运行。初次接入时两条错误文案契约红灯，通过 domain 共享可见性 helper 恢复 `unknown id` 与“种植于第 N 章”信息；Foreshadow/CommitChapter 定向测试及 tools 全包通过。

### 阶段 54 收口审查

回归与动作归属扫描全绿，但发现两个未覆盖边界：`append(nil, empty...)` 会让非 nil 空账本漂成 nil；同章完整 payload `plant→reinforce→partial_payoff→advance→resolve` 全部写入后，started 恢复重放会在 resolved 的 reinforce 处失败。两条纯函数测试均准确变红，现已通过 nil/空保真与受限同章 resolved payload 重放特例修复，孤立终态转换仍拒绝。

进一步审查发现 Projector 原实现从非 nil 空 ledger 开始，无伏笔记录时投影 JSON 为 `[]`；接入纯函数时误改成 nil，公开测试准确变红。现已恢复非 nil 空初值；Foreshadow/Projector、Saga、Import、Context、Diagnostics 回归全部通过。

新增真实 CommitStageStarted 多动作伏笔重放契约并直接通过。随后尝试规范化恢复调用 JSON 时精确替换未匹配；读取真实源码后确认参数本来就是有效 `{"chapter":1}`，无需修改，仅是工具参数转义层级不同。

### 阶段 55 / 里程碑 E2 完成

最终验证全部通过：

```text
go test ./internal/domain ./internal/store ./internal/tools ./internal/revision ./internal/host/imp ./internal/diag -count=1 -timeout=5m
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/store ./internal/tools ./internal/revision ./internal/host/imp -timeout=10m
git diff --check
```

范围扫描确认伏笔生命周期动作 switch 只存在于 `internal/domain/foreshadow.go`。没有新增伏笔状态、通用 StateMachine、Service、Repository、数据库或格式迁移。

## 2026-08-24：里程碑 E1 启动

- HEAD：`cafb752 修复：导入发布前重放并验证全书事实`；工作区开始时干净。
- session-catchup 未报告未同步上下文。
- 已确认公共接缝：domain 纯 Apply、WorldStore、CommitChapterTool、Revision Projector；Import 只间接消费。
- 盘点确认 Store/Commit/Projector 三份 Knowledge 生命周期规则，并记录同章 belief→learn 恢复幂等差异。
- 阶段 40—47 已写入计划；下一步只写 domain establish 的首个失败测试。

### 阶段 40 完成

纯 Apply 的 establish 经过三轮红→绿：公开函数不存在、重复 establish 复制条目、冲突 Truth 被静默接受。当前实现深拷贝输入；同 Truth 幂等并保留首次 EstablishedAt；冲突返回 nil+error 且原输入不变。

### 阶段 41 完成

believe、learn、reveal_to_reader 逐动作红→绿完成。纯函数已锁定 belief 形成/幂等/冲突/已知后拒绝，learn 首次章节与 active belief 纠正，reader reveal 首次章节，以及同章 `establish→believe→learn` 完整冻结 payload 的二次重放幂等。

### 阶段 42 完成

Store 接入后字段矩阵红灯已通过迁移既有直接入口约束到纯 Apply 修复。`WorldStore.UpdateKnowledge` 现在只负责写锁、读取当前投影、调用 `domain.ApplyKnowledgeUpdates`、写 JSON/Markdown；全部 Knowledge/Belief/Reader Store 测试通过。

### 阶段 43 完成

`revision.projectKnowledge` 已改为从空投影逐章调用纯 Apply，删除原 Knowledge switch；错误附加当前章上下文。Knowledge/Belief/ValidateRecordSet/Projector 定向测试和 revision 全包均通过。

### 阶段 44 完成

Commit 前置校验删除 Knowledge truth/knownBy/beliefBy 临时 map 模拟。适配器只过滤 `EstablishedAt > 当前章` 的未来 Truth，再调用纯 Apply 试运行并包装为 `ErrToolPrecondition`。全部 Knowledge/Belief/Reader/CommitChapter 定向测试和 tools 全包通过；已冻结 PendingCommit 恢复仍跳过当前投影语义重判。

### 阶段 45 完成

新增真实双 Store 等价测试：同一 `establish→believe→reveal→learn` 序列经 WorldStore 逐章增量应用与 Projector ChapterRecord 全量重建，完整 KnowledgeEntry 投影完全一致。domain/store/tools/revision/import 五包、Commit Saga 和 Import 全书门禁全部通过。

### 阶段 46 完成

nil 投影形状红灯已通过 clone 的 nil 保真修复。依赖方向为 domain ← Store/Tools/Revision ← Import，无循环。扫描确认 Store、Commit、Projector 已无 Knowledge 生命周期 switch；剩余动作枚举仅服务 ChapterFacts 字段纪律、Import ledger/局部反馈和 domain 正式纯规则。架构文档已同步。

### 阶段 47 最终审查发现

Commit 适配器过滤 `EstablishedAt > 当前章` 的完整条目后再试运行，会丢失未来同 ID 的冲突证据：早期重写 `establish` 不同 Truth 可能被当作新 ID 放行，Store 才在 Pending 创建后拒绝。修复方向是把引用动作的章节可见性收进纯 Apply，并让 Commit 对完整投影试运行；这样 establish 仍能检测全局 ID 冲突，believe/learn/reveal 则统一拒绝未来 Truth。

纯函数未来引用测试取得真实红灯：`believe` 被错误应用到 `EstablishedAt=5` 的 Truth（当前章 3）。Commit 测试第一次编译失败 `undefined: newCommitTestStore`，第二次因猜测 `Drafts.Save` 失败（真实 DraftStore 无此方法），两者都不作为功能红灯。按三次失败协议停止猜 API，搜索并复用同文件既有草稿写法。

修正夹具后取得真实 Commit 红灯：冲突未来 Truth 的第 2 章请求留下 `PendingCommit{stage:started}`。纯 Apply 增加未来引用检查，Commit 改用完整投影试运行后，两条红灯及全部 Knowledge Commit 测试转绿。

### 阶段 47 / 里程碑 E1 完成

最终两轮验证均通过：

```text
go test ./internal/domain ./internal/store ./internal/tools ./internal/revision ./internal/host/imp -count=1 -timeout=5m
go test ./... -timeout=5m
go vet ./...
go test -race ./internal/store ./internal/tools ./internal/revision ./internal/host/imp -timeout=10m
git diff --check
```

范围扫描确认 Knowledge 生命周期动作 switch 只存在于 `internal/domain/knowledge.go`；没有新增认知动作、通用 StateMachine、Service、Repository、数据库或格式迁移。

## 2026-08-24：全项目 Review 后里程碑 D 规划

### 基线

- 工作区开始时干净；
- HEAD：`3ee475c 功能：追踪角色错误信念与纠正状态`；
- `session-catchup.py` 未报告未同步上下文；
- 全项目 Review 已通过全量测试、vet 和关键包 race；
- 本次只更新 `task_plan.md`、`findings.md`、`progress.md`，不修改生产代码。

### 规划结论

下一里程碑确定为 **D：Import 发布前全书事实重放门禁**，优先修复跨批次非法事实只能在正式发布中途被发现、且重跑永久停留在 ActionPublish 的风险。

阶段 32—39 已写入计划：v5→v6 缓存隔离、统一事实映射、候选批次累计重放、salvage 门禁、首错章定位与派生工件失效、synthesis 门禁、publish 零污染门禁、文档与最终验证。

架构边界：复用 `revision.ValidateRecordSet` 和现有 `Facts + NextAction`；不新增 Import Action/Stage，不复制 Knowledge/Foreshadow switch，不修改 Commit Saga，不顺带做领域模块重构。

### 下一步

用户确认执行后加载 TDD，从阶段 32 的旧 v5 分析缓存失败测试开始；首个生产改动只能是 `analysisSchemaVersion 5 → 6`。

### 阶段 32 红灯

将既有缓存失效契约从旧 v4 更新为旧 v5 后：

```text
go test ./internal/host/imp -run '^TestAnalyzedChaptersInvalidatesPreviousAnalysisSchemaVersion$' -count=1
```

失败为 `got 1 reusable chapters`，证明当前 `analysisSchemaVersion=5` 会继续复用未经全书重放证明的 v5 工件。

### 阶段 32 绿灯

只将 `analysisSchemaVersion` 从 5 提升到 6；旧 v5 缓存失效测试和既有上游摘要失效测试均通过。未提升 workspace/ChapterRecord 版本，也未提前实现事实重放。

### 阶段 33 映射红灯

新增 Import 事实映射与真实 `commitArgs` JSON 解码结果一致性测试；编译失败 `undefined: importedChapterFacts`，证明验证与发布尚无共享映射。

### 阶段 33 映射绿灯与门禁红灯

新增唯一 `importedChapterFacts` 映射，并让 `commitArgs` 从该映射取值；包含全部派生字段与 core_event fallback 的一致性测试通过。随后新增跨章 `establish+learn → believe` 测试，编译失败 `undefined: validateImportedFactSequence`，准确证明 Import 尚无全书重放门禁。

### 阶段 33 绿灯

`validateImportedFactSequence` 将 Import facts 转为临时 ChapterRecord 并复用 `revision.ValidateRecordSet`。合法 Knowledge/Foreshadow 生命周期通过；跨章 belief-after-learn、冲突 Truth、未知 Knowledge 引用和 resolved 后 advance 均被拒绝。

### 阶段 34 红灯

预写第 1 章 `establish+learn` 工件，再让 `AnalyzeNext` 的第二批返回同角色 believe。当前调用返回成功并写入第二章，因此测试失败于 `expected cumulative fact replay to reject second batch`，证明正常候选批次尚未接入全书门禁。

### 阶段 34 首次绿灯尝试超时

把累计门禁接入 `callStructured` 验证回调后，原测试 mock 永远返回同一非法事实，触发模型语义重试与退避，宿主命令 120 秒超时。该行为说明门禁位于正确的“模型可修正语义错误”接缝，但测试设计不应假设首次非法响应立即返回。不会原样重跑；改为首轮非法、次轮合法的顺序响应，观察非法候选未落盘且修正响应成功。

### 阶段 34 绿灯

顺序 mock 首轮返回非法 belief、第二轮修正为空更新；`AnalyzeNext` 恰好重问一次，只写修正后的第 2 章工件，累计前缀变为 2。正常上游失效测试保持通过。

### 阶段 35 测试编写错误

插入严格 Import JSON helper 时误删原 `factsJSON` 函数声明，首次运行是 Go 语法错误而非功能红灯。记录后恢复声明，再获取真实打捞行为红灯。

### 阶段 35 真实红灯

第 1 章已 establish+learn；长度截断响应中可解析的第 2 章让同角色 believe。现有 `salvagePrefix` 局部校验通过，`AnalyzeNext` 记录 `salvaged=1` 并返回成功，证明截断分支绕过累计全书门禁。

### 阶段 35 绿灯

打捞前缀在落盘前与 prior facts 一起重放。非法累计前缀返回错误、不写分析工件，并在 failures 中保留原响应及“累计事实非法”诊断；原有合法截断打捞继续通过。

### 阶段 36 红灯

构造第 2 章首次非法、后有第 3 章以及 synthesis/story-resolution 的工作区；编译失败 `undefined: validateWorkspaceFacts`，证明尚无首错章定位和派生工件回退行为。

### 阶段 36 绿灯

`validateWorkspaceFacts` 先做一次 O(n) 全书重放，失败时逐前缀定位首个非法章，复用 `discardAnalysesAfter` 删除尾部，并失效 synthesis/story-resolution。真实 source/segmentation/confirmation 测试证明回退后 `LoadState.AnalyzedChapters=1` 且 `NextAction=ActionAnalyze`。

### 阶段 37 红灯

给 runner.synthesize 准备两章当前版本非法 facts 和合法 synthesis 模型响应。现有流程实际调用模型、产出综合并返回成功，测试失败于 `expected invalid full-book facts to block synthesis`，证明综合前没有事实门禁。

### 阶段 37 绿灯

`runner.synthesize` 在 emit、range digest 和模型调用之前执行 `validateWorkspaceFacts`。非法事实不调用模型、不写 synthesis，并回退分析尾部；合法 `TestRunEndToEnd` 保持通过。

### 阶段 38 红灯

直接为 publish 准备非法两章 facts 和有效 synthesis。当前流程先写入 Foundation、completion Hold 和第 1 章，直到第 2 章 Commit 才失败；零污染断言发现正式 Book 已存在，完整复现 Review 中的部分发布风险。

### 阶段 38 绿灯

`runner.publish` 在 resolveStory、AssembleFoundation、publishFoundation、Hold 和逐章 Commit 之前执行同一全书工作区门禁。非法事实保持 Book/Premise/Completed/PendingCommit/Hold 全空并回退分析尾部；合法端到端与 Hold 测试通过。

### 阶段 39 协议审计红灯

全书门禁接受传入 facts `[chapter=1, chapter=3]`，因为 Revision 只按记录章号排序，不知道 Import 工件位置必须连续。该约束属于 Import 坐标，不是领域生命周期；将在 Import→ChapterRecord 适配处要求 `facts[i].Chapter == i+1`。Range digest 测试确认它只消费 compact narrative evidence，Knowledge/Foreshadow 跟踪字段变化无需主动删除 range cache。

### 阶段 39 文档定位错误

两次 `search_files` 将单个 Markdown 文件路径作为目录，返回 ENOTDIR。未重复该方式；改用 `read_file` 已定位 Import §9.4—9.6、§11 和 architecture 目录说明。

### 阶段 39 / 里程碑 D 完成

稳定文档已同步批次局部校验、累计全书重放、首错章回退与发布前零污染。最终验证：

```text
go test ./... -timeout=5m                          通过
go vet ./...                                       通过
go test -race ./internal/host/imp ./internal/revision ./internal/tools -timeout=10m  通过
git diff --check                                   通过
```

范围扫描确认：`analysisSchemaVersion=6`，`workspaceSchemaVersion=1`、`ChapterRecordVersion=1`；没有新增 ActionValidate/StageFactReplay、Service、Repository、数据库或第二套生命周期规则。最终批次在提交前为 12 个文件、1017+ / 33-（其中大部分为测试与规划记录）。

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

### 阶段 24 开始

基线确认：HEAD=`03bf271`，生产代码无改动，仅三份规划文件有变更；现有 `TestKnowledge_*` 全绿。首个接缝为 `WorldStore.UpdateKnowledge/LoadKnowledgeState`。

### 阶段 24 首个红灯

`establish@1 → 林墨 believe@2` 编译红灯：缺少 `KnowledgeUpdate.Belief` 与 `KnowledgeEntry.BelievedBy`，符合预期。首个绿灯仅增加领域字段和 Store believe 分支。

### 阶段 24 完成

Store tracer bullet 已绿：Truth 保留、林墨错误信念记录于第 2 章、KnownBy 不受影响；全部 `TestKnowledge_*` 通过。

### 阶段 31 完成

最终审计补齐两项：

1. 直接 Store 调用严格执行四动作字段矩阵，不再静默忽略多余 Truth/Character/Belief；领域动作注释同步四动作。
2. Context 的 active belief 使用更窄私有 DTO，不输出 `CorrectedAt`，避免本章或未来纠正时间提前泄漏。

最终验证：

```text
go test ./... -timeout=5m 通过
go vet ./... 通过
git diff --check 通过
```

范围扫描确认没有 `correct_belief/doubt/suspect/forget/reader_belief`，没有新增 Service、Repository、数据库或格式迁移。C2a 待按中文提交信息收尾。

### 阶段 30 完成

Writer/Editor/Revision/Import Prompt 已同步 belief 纪律：稳定且影响行动的错误认知才用 believe；角色按 active belief 行动；learn 后不得继续按已纠正信念行动；暂时怀疑、猜测、反问、一闪而过的念头和故意说谎不等于 believe。Writer golden 与资源契约全绿。

### 阶段 29 完成

Context 已改用私有净化 `knowledgeBoundary` DTO，而非直接序列化完整 KnowledgeEntry。JSON 级测试证明：角色只有 active belief 且读者未知时输出 belief、不含 truth 键；ReaderKnown + 错误信念同时输出以支持戏剧性反讽；无关角色、已纠正 belief、当前章/未来 belief 均不泄漏。旧 CharacterKnown/ReaderKnown、8 条上限、预算裁剪和 `_trimmed` 全部保持通过。

### 阶段 28 完成

Import 严格 Schema/DTO 已支持 belief 字段和四动作；validateBatch 支持首批/同批次 `establish→believe→learn`，拒绝未知 belief、真信念、非法字段和同批次认知冲突。跨批次 ledger 显示 active belief，并在 learn 后移除该角色当前误信。真实三章发布保留 FormedAt/CorrectedAt；分析缓存版本从 4 提升到 5，整个 Import 包通过。

### 阶段 27 完成

Projector 已重建 `establish→believe→learn`：FormedAt、CorrectedAt、LearnedAt 均按历史章节恢复；已知 Truth 后 believe、真信念、不同内容改写被拒绝，相同 active belief 重放保留首次形成章。删除 learn 后全量投影自然恢复 active belief。Rewrite 候选记录验证已扩展后续 belief 引用：删除唯一 establish 时在 Pending 前拒绝。整个 revision 包通过。

测试首次使用 `slices.Clone` 时漏掉 import，先修正测试编译错误后取得同内容复制/不同内容静默追加的真实红灯。

### 阶段 26 完成

ChapterFacts 严格 Schema/Validate 已扩展为四动作字段矩阵；Commit 正常 believe、未知引用、真信念、已知后 believe、同 payload learn→believe、冲突 belief 均有公开回归测试，非法请求在 PendingCommit 前拒绝。合法 `establish→believe→learn` 同 payload 通过。

started 重放测试发现两层 Saga 问题：

1. 恢复冻结 payload 时，入口会用已经部分应用后的当前投影重复做前置校验，导致历史 believe 被误判；现改为语义前置校验只在首次冻结 PendingCommit 前执行。
2. Store 对“同章形成并同章纠正”的同内容 belief 不具备完整 payload 重放幂等；现仅对该同章形态开放幂等，跨章纠正后再次 believe 继续拒绝。

`chapterfacts`、`tools`、`store` 三包全部通过。

### C2a 会话恢复与阶段 25 完成

开始本会话时发现生产工作区已有未同步的 C2a Store 改动，而不是规划时的纯净状态：

```text
internal/domain/tracking.go
internal/store/world.go
internal/store/world_test.go
```

阶段 24 与阶段 25 的 JSON 行为测试及实现已经存在；定向 Store 测试的唯一红灯是 Markdown 尚未投影 active/corrected belief。没有覆盖或回退这些改动，而是从该真实红灯继续。

最小修复只扩展 `renderKnowledge`：显示错误信念形成章、尚未纠正/纠正章。随后 `TestKnowledge_*` 和整个 Store 包通过。旧 Knowledge JSON 兼容断言同步覆盖缺少 `believed_by` 时为空切片。

### C2a 规划会话

恢复结果：

- `session-catchup.py` 未报告未同步上下文；
- 工作区开始时干净；
- 最近提交为 `03bf271 功能：追踪读者揭示与信息差状态`；
- task_plan 头部残留的“Knowledge 批次未提交”描述已修正。

下一里程碑选择 **C2a 最小角色错误信念**，只新增 `believe`。不新增 `correct_belief`：角色执行现有 `learn` 后，代码将其对同一 Knowledge ID 的活跃错误信念标记为已纠正。

规划阶段为 24—31：Store 首切片、Store 不变量/learn 纠正、ChapterFacts/Commit、Revision/Rewrite、Import、Context 净化视图、Prompt、全量验证与中文提交门禁。

关键安全决策：Writer Context 不得继续直接序列化完整 KnowledgeEntry；必须使用净化 DTO，确保“角色只持有错误 belief、读者未知”时只输出 belief、不输出隐藏 Truth。

本次只修改 `task_plan.md`、`findings.md`、`progress.md`，没有开始 belief 生产代码。

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
