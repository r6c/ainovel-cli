# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-26
- 基线：`3fddb1e 修复：保护导入与用户正文不被自动返工覆盖`
- 当前里程碑：U——Import 认知事实提取校准
- 当前阶段：阶段 158——当前 Import Prompt 三轮基线（blocked_provider）

阶段 158 首次基线未产出有效结果：人工复核发现 ik09 金标要求 believe 却未建立同 ID Truth，按正式领域规则该样本非法；已终止未完成进程、删除无效结果，修正为“真实动机 establish+reader reveal，许澄持有相反稳定 belief”。不把该试跑计入模型证据。

临时包内 live runner 首次编译因 helper `itoa` 与现有 `synthesize_test.go` 同名失败，尚未调用模型或产生费用；改用 `strconv.Itoa` 内联并删除本地 helper，不重复占用包级名字。

阶段 157 采用 12 条完全自建片段和独立金标，覆盖 establish-only、establish+learn、establish+reveal、三动作、belief 及猜测/谎言/不相信负例。语义约定：reveal 只用于此前隐藏 Truth 的完整揭晓，不机械伴随普通 establish。测试 seam 固定为真实 Import `analysisContract + llmcontract.Execute`，不使用简化 Judge。

真实第 3 章续写完成：导入 Hold 经 Resume 消费，切换 review 后只授权第 3 章；Architect 把 2 章分层大纲扩为 10 章，Writer/Editor 完成第 3 章并停在 3/10。ChapterRecord 3 为 generated、无 PendingCommit、全量重放通过，总费用 `$0.5680684`。

收口审计发现 P1：续写前弧末评审自动把导入第 2 章放入返工队列并覆盖为 `revision=2, origin=generated`，导致导入 Knowledge updates 消失；后续 Projector 只剩第 3 章新 Knowledge。导入分析工件本身正确含 Timeline/Relationship/StateChange 及 `K-red-signal` 的 establish+learn+reveal，问题是自动返工覆盖了 imported 事实源。下一切片在 SaveReview 控制态 seam 上保留评审证据，但禁止 imported/user 章节进入自动返工队列。

正式投影检查：两章正文 SHA 与 ChapterRecord 一致，origin 均为 imported；Knowledge 建立 `K-red-signal`（第2章读者与苏弦已知，顾临未知）及隐藏的 `K-third-relay-record`；5 条活跃伏笔覆盖红灯、铜钥匙、广播“第三枚”、删日志和第三中继器。Progress 位于 writing，NextChapter=3，分层大纲当前仅 2 章，Compass 指向北侧冷阱。真实 P2：Timeline/Relationship/StateChange 投影均为空，尽管源文有明确决定、隐瞒与信任裂痕；本轮不凭空补事实，续写验收观察 Context/Writer 是否仍能依靠摘要、Knowledge、Foreshadow 和 Compass 连贯接力。

阶段 151 自动化基线通过。真实 Import 使用自建两章未完科幻悬疑源文，首次停在 2/2 切分确认，确认前正式 Book/完成章节均为空；显式确认后分析、综合、验证、发布全部完成，费用 `$0.04324`，两条 ChapterRecord 均为 imported，无 PendingCommit，导入 Hold 已建立。首轮投影检查脚本因把第二条 Knowledge 的 `believed_by:null` 当数组遍历而 TypeError；这是验收脚本空值处理错误，不是数据损坏，改为空值安全读取。
- 公共路径：Quick、Cocreate、Headless、Import、Deconstruct、读者成品导出

## Karpathy 约束下的取舍

不新建 E2E 框架，不调用真实 Provider。先复用已有强测试证据，只补真实缺口。盘点显示 Engine、Cocreate、Import、Deconstruct、TXT/EPUB 已有公开路径测试；`internal/entry/headless` 没有测试，是首个最小切片。

## 成功标准

1. fake-model 自动化覆盖 Headless 入口，而不是只覆盖 Host 内部。
2. 五条用户路径都有可执行测试命令和工件检查说明。
3. 人工真实模型验收明确标记为未执行，不消耗用户额度。
4. 发现问题按 P0/P1/P2 分级，不顺带扩展功能。
5. 全量测试、vet、格式与文档链接通过。

## 阶段 112 完成

覆盖盘点确认已有代表性证据：Engine 完整写书、Cocreate 阶段/协议/TUI、Import 端到端与零污染、Deconstruct 增量、TXT/EPUB 隔离。唯一明显空白是 Headless 包无测试。

新增 `headless.Run` 公开入口契约：本地 ollama 配置、空 Prompt、空工作区不发模型请求，必须返回带目录的“需要 --prompt 或可恢复会话”错误，stdout 为空、不伪报恢复成功，并导出脱敏诊断。现有实现直接通过，无生产修改。没有为测试引入 Host 模型注入框架。

## 阶段 113 完成

代表性测试全绿：fake 模型 + 真实工具完成整本书；review permit 恰好只稳定一个新章节；TXT 与 EPUB happy path 通过且均不泄露 Knowledge。现有证据已足够，没有新增重复 Engine/Exporter 测试。

## 阶段 114 完成

Session 单步推进、完整 Draft/ready 门禁、规范化 History；Host `<stage>` 解析与冷启动 Prompt；TUI Ctrl+S、请求失败和取消恢复全部通过。三层公共契约已形成完整证据，无需新增整套启动测试。

## 阶段 115 完成

Import fake-model 端到端、全书非法事实发布前零污染、首错章回退 Analyze、stale PendingCommit 恢复和 Knowledge 正式发布全部通过。没有发现 P0/P1 缺口。

## 阶段 116 完成

拆文 Runner 首次生成/第二次零模型调用、增量新增/修改、合规措辞与 strict schema、Host 显式目录和 CLI 参数契约全部通过。没有发现 P0/P1 缺口。

## 阶段 117 完成

新增 `docs/release-acceptance.md`，覆盖自动化命令、重点工件、P0/P1/P2 定义和六组人工真实模型步骤。文档明确人工状态未执行、会产生费用、不得记录 API Key；所有自动化命令已逐条实际运行，README/CONTEXT 链接有效。

## 阶段 118 完成

全量测试、`go vet`、Headless/Startup/Deconstruct/Import/Sim/Export 关键 race、gofmt、diff、文档链接和安全边界全部通过。自动化验收未发现 P0/P1。

人工真实模型文学质量验收在 `efcef9f` 时仍未执行。用户随后明确授权使用已配置的 `sss` Provider 直接验证；已脱敏确认默认模型为 `gpt-5.6-sol`、凭证和 Base URL 存在，未读取/输出密钥。

## 阶段 119—122 完成

已在隔离临时目录用 `sss / gpt-5.6-sol` 完成真实单章 Headless 验收。外部执行器在 Foundation 阶段强杀后可无 Prompt 恢复；首次 `$0.25` 硬预算在累计约 `$0.264` 时安全停机；上调隔离项目预算到 `$0.60` 后再次恢复并完成 1/1 章。实际 2092 字，退出码 0，总费用约 `$0.410`，无 PendingCommit。

Writer 前两次 Commit 分别因未知 Foreshadow/Knowledge ID 被前置门禁拒绝，第三次依据错误自修复成功。真实 ChapterRecord 通过全量重放；Timeline/Relationship/StateChange 章号正确；TXT/EPUB 成功且无内部认知状态泄漏；diag-export 正文出包 0。

P2：目标约 1200 字但实际 2092 字；正文残留 6 个 `**` 且 Lint 已报告；完结态无 Prompt Headless 重启无副作用但返回退出码 1 和不够准确的错误文案。

基于已产生约 `$0.410` 费用，本轮不自动扩展到真实 Cocreate、Import、Revision 或 Deconstruct。临时小说和日志保留在系统临时目录供当前会话检查，不进入仓库。

## 里程碑 O 启动

按 O1→O2→O3 独立切片收敛真实验收 P2。公共接缝已确认：`headless.Run`、`CommitChapterTool.Execute`、`UserRules → Context/Contract → Commit`。首轮只处理完结态 Headless。

O3 Normalizer 首个字段切片通过后，`userrules` 全包暴露旧 strict JSON 夹具缺少新必填 `chapter_target_chars`；scripted model 重复最后一条无效响应，产生大量反馈日志。未原样重跑，改为先定位并同步旧夹具为空值 0，再分组验证。一次对 `novel_context_test.go` 单文件路径使用 `search_files` 返回 `ENOTDIR`，未重复该方式，改用直接读取。阶段 158 首次基线未产出有效结果：人工复核发现 ik09 金标要求 believe 却未建立同 ID Truth，按正式领域规则该样本非法；已终止未完成进程、删除无效结果，修正为“真实动机 establish+reader reveal，许澄持有相反稳定 belief”。不把该试跑计入模型证据。一次批量编辑把测试路径误写到不存在的项目目录，未修改目标文件；随后使用真实路径修正。O3 Prompt 契约首次编辑漏掉循环闭合大括号，测试语法失败未当作行为红灯；先修正测试结构再重跑。真实 O 回归完成后，检查脚本先猜错 ChapterRecord 文件名/JSON 外形（实际 `000001.json`，violations 外层对象），未据此判失败；后续按真实工件复查。一次 ChapterRecord API 搜索正则括号未闭合，改用字面搜索。初次读取错误猜测文件名为 `headless.go/headless_test.go`，实际为 `run.go/run_test.go`；未修改文件，已改用真实路径。首次 O1 测试又猜测了不存在的 `DraftStore.SaveChapterText`，编译失败未当作产品红灯；已转为读取真实 DraftStore API。修正后测试目录又因缺 `meta/format.json` 进入合法迁移门禁，仍未视为产品红灯；夹具已补当前格式版本。O1 首轮绿灯后真实目录字段比较发现 Progress/费用/Token 不变，但 `Host.Close()` 会刷新 `usage.updated_at`；因此终态判断前移到 Host 构造前，并由非空 Usage 工件测试锁定。

## 阶段 123 完成

已完结当前格式项目在无 Prompt Headless 启动时，直接只读 Progress/Book，输出标题、章数和字数摘要并返回成功；不构造 Host，不刷新 Usage、不写 Progress、不创建 PendingCommit，也不调用模型。空工作区和旧格式项目仍走既有错误/迁移路径。真实《月背回声》目录回归：退出码 0，Progress 与 Usage 原始字节均不变。全量测试、vet、Headless/Host race 和 diff check 通过。

## 阶段 124 完成

`CommitChapterTool.Execute` 在首次冻结普通/Rewrite PendingCommit 前，复用 `rules.Lint` 并仅将 `markdown_residue` 升格为最终正文格式门禁。错误包含具体标记与次数，真实模型可在同一回合修正；旧终稿和返工队列保持不变。`duplicate_paragraph` 等审美 warning 继续成功提交并持久化。全量测试、vet、Tools/Revision/Rules race 和 diff check 通过。

## 阶段 125 完成

UserRules 快照升级 v4，新增 `structured.chapter_target_chars`。Normalizer 只提升明确的单章/每章单一目标值；区间、全书目标和含糊篇幅保持 preferences/uncertain。Architect/Writer/Editor 通过现有 Context 消费；普通提交与 Rewrite 复用 `domain.WordCount`，只在超过目标 120% 时于 PendingCommit 前拒绝，不设机械下限。旧 v1-v3 快照兼容。Rules/UserRules/Tools/Assets/Host 关键包、全量测试、vet、race、文档链接和 diff check 通过。

## 阶段 126 完成

在全新隔离目录用 `sss / gpt-5.6-sol` 复用同类单章科幻需求，项目硬预算 `$0.60`。快照 v4 正确记录目标 1200，最终《静海回声》1/1 章、1311 字（+9.25%，低于 1440 上限），Markdown/其它规则违规均为 0，无 PendingCommit，退出码 0，费用约 `$0.161`。完结态再次无 Prompt 启动返回 0 且 Progress/Usage 原始字节不变；真实 ChapterRecord 全量重放与 TXT/EPUB 隔离通过。未继续调用 Provider。

## 2026-08-25 全项目复审

只读审查覆盖近期热区与主链：Headless、Host/Engine/Flow、Commit Saga、Revision Projector/Service、Import analyze/publish、UserRules v4、Context、Prompt、Diagnostics、Exporter、Linux/无头入口。

全量 `go test ./...`、`go vet` 和 Store/Tools/Revision/Host/Headless/Import race 通过，工作区基线干净。宏观架构未跑偏。

发现两个 S1：

1. 最后一章在 PendingCommit 收尾前已 `phase=complete`；Headless 快路径、Resume label、Route/precheck 都可能跳过该恢复。
2. Import 用户原文复用生成正文 Markdown 门禁；发布前只验证事实、不验证正文，可能先写 Foundation/Hold 后卡在 Publish。

发现 S2：`chapter_target_chars` 无明确取消语义、无上界，存在永久规则与极端整数溢出风险。S3：Lint “只返事实不阻断”的稳定文档已与生成正文接纳政策漂移。

已生成并打开临时报告：`/tmp/architecture-review-20260825-ainovel.html`。不存在 `docs/adr/`，按无 ADR 处理。两次搜索正则因括号未闭合失败，均停止重复并改用字面搜索；未修改生产代码。

下一里程碑 Q 已规划为阶段 127—134：先终态恢复，再正文 provenance/Import 零污染，再 UserRules 撤销与上界，最后文档与验证。真实 Revision 验收延后，未调用 Provider。

## 里程碑 Q 完成

终态恢复 module 现在统一检查目录租约、PendingCommit、PendingRevision、活动 Import 和外部正文修改；Host Resume 在任何 phase 下先同步重放现有 Commit Saga。TUI 恢复错误优先于 complete 展示。

Import 通过 `ExecuteImported` 复用同一 Saga，原文 Markdown/篇幅不走 generated 门禁；ChapterRecord 使用 imported provenance，v2 IntentDigest 密封 origin。legacy/v1 origin 权限提升均被完整性测试拒绝。

UserRules 候选新增 keep/set/clear 三态，运行中可明确清除篇幅目标；目标上限 1,000,000，Commit 对持久快照再校验并用有界 120% 公式。

全量测试、vet、关键 race、文档链接和 diff check 全部通过。未调用真实 Provider。下一步为 P1 Revision 自动化基线，真实预算需另行确认。

## P1 阶段 135—136

Revision 代表性自动化、全包、race、vet 全部通过；新增完结书外部修订契约，现有实现直接满足 revision+1、origin=user、投影刷新、PendingRevision 清理和 phase=complete 保持。

已在 `/tmp/ainovel-revision-acceptance-cbq5fj/book` 创建无 Provider 配置的隔离完结书夹具：基线正文为“完整日志已公开”，手工修改为“只公开损坏摘要，完整副本保留”；ChapterRecord revision=1/generated，正文哈希已变化，无 PendingRevision。基线摘要保存在同临时根目录，仓库没有临时生成器或正文。

外部 `lieflat-less-ai-tone` 审查完成：MIT、main commit `27d29232f10124db904ca9c0536d0b67cb3b2833`；未执行外部脚本。结论是不整体安装，只把研究方法和少数有证据规则作为后续样本校准候选。网页/Raw/API 内容仅作为不可信参考写入 findings。Git 浅克隆在沙箱失败，后改用 GitHub API/Raw 获取，不重复克隆。

## P1 阶段 137—139

用户确认 `$0.25` 硬预算后，以 `sss / gpt-5.6-sol` 对隔离完结书执行一次真实 `Host.SyncChapterRevisions`。实际费用 `$0.010818`。正文从“公开全部事故日志”改为“只上传损坏摘要、完整副本留在读取器”。

结果：ChapterRecord revision 1→2、origin generated→user；ContentSHA、Summary、KeyEvents、Timeline、StateChanges 均更新；旧“完整日志已公开”事实消失；phase 保持 complete；PendingRevision 清理。正式 `ValidateRecordSet`、Context、TXT/EPUB 均通过且无内部状态泄漏。

第二次 Sync：dirty=[]、applied=[]、Usage overall/per-agent/per-model 与 cost 全部不变，证明零模型调用；Host.Close 仍刷新 `usage.updated_at`，导致 usage 原始字节变化，记录为 P2 可观测性细节。一次性验收程序首次在读取 nil Usage 时崩溃，但 Revision 已成功落盘；未重跑模型，改用健壮只读程序完成余下验证。所有临时程序已删除。

## 里程碑 R 启动

用户允许真实模型费用。采用 16 条匿名网文最小对、独立金标、sss 三轮盲评；外部研究结论不写入 Judge prompt。未安装 Skill、未运行外部 Python 脚本。会话恢复脚本首次命令引号错误，改用直接路径后成功；探索中猜错 `internal/llmcontract/structured.go`、`internal/agentcore` 和 `internal/schema`，均改为真实模块路径。样本完整性测试首次把金标平衡误写成 5/11，实际是 6/10；按标签逐项复核后修正，不视为规则效果红灯。

阶段 140—142：建立 16 条匿名网文最小对和独立金标；`sss/gpt-5.6-sol` 三轮盲评分别判 4/5/5 条 modify。14 条三轮一致；s11 为 2/3 modify。原 s16 金标机械误判单次人物核心恐惧对照句，三轮均指出其人物功能，人工按预先协议修正后多数票 16/16 命中。Judge 总 Token 7,724，Provider 未返回 Cost，记 unavailable。

阶段 143：`anti-ai-tone.md` 改为目标风格、叙事功能、信息守恒和最小改动优先；短段/问句/比喻/排比/冒号/对照句式本身不作来源证据；新增段首零回指、提示性冒号、职业人格喻体等候选。资源契约全绿。

阶段 144：Writer smoke baseline/calibrated 各 3 次，六臂全 PASS，总费用 `$2.343592`；校准版平均成本 `$0.352269` vs `$0.428928`，平均工具 15.67 vs 19.33，但不据此判质量。匿名方向互换 Judge 九次为 calibrated 8 胜、baseline 1 胜；九次均判双方剧情功能完整。原始章节留在 gitignored workspace，版本库只保留脱敏指标。

阶段 144 一次性 A/B 运行器首次编译因 `eval.Outcome` 未显式转 string 失败，未调用模型；修正后以普通后台子进程启动，但子进程仍属于宿主进程组，在 120 秒工具超时后被终止，未完成任何臂、无 runs 结果。后续改用 Python `start_new_session=True` 真正脱离，不重复普通后台方案。

修复后一次真实单轮 Usage 检查程序首次编译因误猜持久 DTO 为 `AgentUsageTotals.Role/MissingAssistantUsage` 失败，尚未调用模型；将读取真实 `domain.UsageState` 字段后只修验收程序。

阶段 146 自动化基线全绿。真实 Cocreate 第二次有效验收成功：一次 HTTP 502 由用户重试语义恢复；6 个有效回合严格按 core→customization→title→confirmation→ready，模型第 5 回合自报 ready 但确定性 Session 因 Draft 未完全确认继续拒绝，第 6 回合才放行。最终 Prompt 1231 字符，进入现有启动主链，Foundation 完整落盘后在 writing 立即 Abort；0 章节、无 PendingCommit。持久 Usage `$0.1018428`。审计发现 CoCreate 直接流式模型未经过 UsageTracker，该费用不含多轮共创，属于真实 P1 预算/观测缺口，需先修。

阶段 146 自动化基线全绿。阶段 147 一次性真实 Cocreate 程序首次编译因猜错 `Store.Premise` 访问器失败，尚未调用模型或产生费用；改读真实 Store 结构后只修验收程序，不改生产代码。第一次真实对话 1→4 轮停在 core/customization：模型持续要求用户决定或授权选择求救包真相，固定脚本没有回答，证明模型未擅自补关键谜底；第 5 轮又遇上游 HTTP 502。已终止并清理隔离目录，保留脱敏阶段轨迹；该次 Usage 随目录清理无法复算，费用记 unavailable，不伪造。第二次已明确授权模型选择真相，但第 1 轮即遇同类 HTTP 502，未形成有效对话。第三次按现有 TUI“失败保留 Session、用户重试”语义，在同一轮最多重试 3 次并记录次数；不修改生产流式重试策略。

阶段 127 启动时一次读取猜错 `internal/store/checkpoint.go`，实际文件为 `checkpoints.go`；未重复错误路径。首个 Headless 恢复测试夹具又因 Book 缺必填 Synopsis 在行为 seam 前失败；不作为产品红灯，补合法作品元数据后重跑。最小恢复实现后 PendingCommit 已清理，但测试用旧 Store 的 CheckpointStore 内存镜像观察不到另一个 Host 新增的 checkpoint；改为重开 Store 读取磁盘事实。随后猜测了不存在的 `internal/host/resume_test.go`；Host 无独立 Resume 测试文件，Headless 公共用例已通过真实 Host.Resume 覆盖，不再猜文件名。阶段 129 批量切换 Headless 到 Host 终态探测时，import/函数片段与预期不完全匹配，结构化编辑未应用；改为读取真实文件后分步替换。阶段 130 首个 Import 测试误把 `LoadRuleViolations` 当作双返回接口，编译失败未当作产品红灯；按真实单返回接口修正。增强篇幅目标夹具时又把 `UserRules.Save` 的指针参数传成值，仍未触达产品行为，已按真实接口修正。阶段 133 首次受限夹具迁移使用 `node`，但用户环境无该命令，迁移未执行；新增断言同时缺 `rules` import。改用已确认存在的 `python3` 并补 import，不重复 Node 方案。
