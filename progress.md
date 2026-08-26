# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-26
- 基线：`d9b9a1f 评测：完成导入认知动作三轮 A/B 验证`
- 当前里程碑：S1——Import Prompt 缓存一致性修复
- 当前阶段：Import Prompt 缓存一致性修复——complete

本次修复：Import Analyze Prompt 已从 `analyze-v1` 提升为 `analyze-v2`。旧版本分析工件在当前运行时不再复用；通过新增回归测试同时证明旧版本工件在旧版本参数下可复用、在当前版本下失效。

候选 3 的完整三轮 baseline/calibrated A/B 已完成：12 条自建样本、3 轮顺序变化，共 36/36 结果有效、Provider 错误与超时为 0。calibrated 的 `learn` recall 提升，但整体 precision 和动作集合完全匹配下降，因此当前 Prompt 作为折中版本保留，不继续追加规则。详细脱敏统计见 `evals/import-knowledge/ab-summary.json` 与 `report.md`。

真实 Architect 扩弧后的第 3 章 Context 端到端验收仍未完成：代码层 ReaderKnown/CharacterKnown 边界已有回归，但真实 Provider 在弧末评审/扩弧阶段多次阻塞，未生成第 3 章 OutlineEntry。该限制与已完成的 Import 认知 A/B 评测分开记录。

本次采用每次单片段调用和独立超时，结果只保存动作级脱敏数据和 Usage；Provider 长连接/HTTP 502 的无效运行不计入模型质量证据。

阶段 161 真实两章 Import/Context 回归：使用当前 Prompt 与 sss/gpt-5.6-sol 在全新隔离目录完成 ingest→segment→analysis→synthesis→publish。两条 ChapterRecord 均为 imported/revision=1，Knowledge 投影 2 条、Foreshadow 投影 2 条、无 PendingCommit、phase=writing；K001/K002 的 KnownBy 均为苏弦，ReaderRevealedAt 为 1/2，原文保持。第 3 章尚无 OutlineEntry，因此 novel_context(chapter=3) 按安全策略不注入 knowledge_boundaries，但保留北侧冷阱/中继器线索且未泄露全量 Truth；未手工扩展大纲或续写。阶段 161 标记 partial_evidence，阶段 162 暂不收口。

阶段 161 后续扩弧尝试：使用同一隔离 Import 作品调用 Host.Resume，正确消费导入 Hold；切换 review 并请求第 3 章后，现有弧末评审开始读取第 1/2 章并进入卷摘要维护，但 Provider 长时间无响应，约 45 秒内无新事件，未生成第 3 章 OutlineEntry 或正文。临时驱动已安全停止；Progress 仍为 completed=[1,2]、total_chapters=2、无 PendingCommit，原两章 imported/revision=1 未改变。该次不计为产品逻辑失败，也不把第 3 章认知边界验收标记为通过。

阶段 161 有界重试：重新创建两章隔离 Import 并调用真实 Host.Resume→review→AdvanceOneChapter。Import Hold 被正常消费，随后弧末评审开始读取章节并维护卷摘要；Provider 在约 45 秒内没有新事件，进程被安全终止。未生成第 3 章 OutlineEntry/正文，Progress 仍为 completed=[1,2]、total_chapters=2、无 PendingCommit；两章 imported/revision=1 未改变。该次继续记为 Provider 阻塞，不修改生产代码、不手工写大纲。

阶段 163/164 收口：新增 `internal/tools/novel_context_reader_boundary_test.go`，在现有 Context 公共夹具中提供第 3 章 OutlineEntry，验证代码层边界：读者已知但当前角色未知的 Truth 可见并保留 ReaderRevealedAt；苏弦已知的 Truth 带 KnownBy；顾临独知/读者未知的 Truth 不泄露。internal/tools、revision、host/imp 回归通过。该测试不冒充真实 Architect 扩弧；真实扩弧仍因 Provider 阻塞未生成第 3 章大纲，因此阶段 161 的真实链路限制继续保留。

里程碑 V 阶段 165/166：新增 `internal/tools/commit_process_recovery_test.go`，使用 Go 测试自调用子进程和真实磁盘 Store，验证 `progress_marked` 密封 PendingCommit 可跨进程收尾：checkpoint 追加、PendingCommit 清理、Progress 完成章节保持不变。首轮夹具缺少正文、摘要和 ChapterRecord 工件，按真实 Checkpoint 摘要校验逐层补齐；没有把夹具错误当产品红灯。现有 `started/state_applied/progress_marked/signal_saved`、密封、Imported provenance、Knowledge/Foreshadow 组合恢复矩阵与 Race 均已通过。评估后不新增生产故障注入开关；阶段 167 进入终态与全量收口。

里程碑 W 阶段 168—172：离线 scanner/runner/CLI/Context 基线全绿。使用两篇自建本地语料和一个被忽略的 JSON 文件完成真实 `deconstruct`：首次 2 篇，未修改二次运行 `analyze/merge=0`，新增 `c.markdown` 只分析 1 篇，修改 `b.txt` 只分析 1 篇，画像累计 3 篇。真实 Provider 为 `sss/gpt-5.6-sol`，修复后运行总费用约 `$0.199158`，`per_agent.simulation` 记录 input=22084、output=15499、missing_usage=0。首次真实运行暴露 `Host.SimulateDir` 绕过 UsageTracker、usage.json 误为零的 P1；通过 Host fake-model 红灯→绿灯，以 `newUsageTrackedModel(..., "simulation", h.usage.Record)` 最小修复，并用新输出目录真实复跑确认用量落盘。

真实 SimulationProfile 结构版本正确、3 篇来源/报告齐全；画像没有自建语料专名、连续正文行或具体模仿指令。命中“仿写/签名短语”仅位于 `do_not_copy` 与抽象语言特征字段，语义为禁止复制/抽象方法，不是模仿建议。现有 SimulationProfile/Context/Exporter 回归全绿；临时语料、配置、日志和画像目录均未进入 Git。

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

## 里程碑 X：发布候选稳定化与交付检查（2026-08-26）

- 阶段 174：干净 HOME 帮助/版本/`deconstruct --help` 通过；五目标 CGO_DISABLED 交叉编译通过。
- 阶段 175：UserRules v1—v3、ChapterRecord/ProjectFormat 旧版本、PendingCommit legacy/v1/v2、SimulationProfile v1 与未知未来格式拒绝矩阵通过。
- 阶段 176：八类模型入口 UsageTracker 接线审计通过，`simulation` 入口已在前一提交修复。
- 阶段 177：Commit、Rewrite、Knowledge、Foreshadow、Imported provenance、complete、跨进程恢复和 TXT/EPUB 隔离矩阵通过。
- 阶段 178：新增 `CHANGELOG.md`、`docs/release-notes.md`、`docs/upgrade.md`，README/CONTEXT 导航同步。
- 阶段 179：全量测试、vet、关键 race、gofmt、git diff、134 个 Markdown 链接、敏感信息与临时产物扫描通过。当前未创建版本标签；X 仅完成候选验证和交付文档，不代表已发布 GitHub Release。

## 候选 2/3/4 规划（2026-08-26）

本会话只完成路线规划，没有修改生产代码。

阶段 180 已完成：生成 `docs/test-asset-map.md`，盘点 Commit/Context/Import 测试函数的 seam 归属。基线数量为 commit 74、Context 31、Import analyze 23、runner 14、contracts 5；`go test ./internal/tools ./internal/host/imp`、`go test ./...`、`go vet ./...`、`git diff --check` 全部通过。没有移动文件或修改 Go 代码。Context 已有独立 simulation/reader-boundary 测试文件，不做机械合并。

阶段 181 当前开始：先拆 Commit 测试，不触碰生产代码；保持原测试名、数量、公共接口和 `-run` 筛选兼容。

阶段 181 首个切片已完成：新增 `internal/tools/commit_payload_test.go`，移动篇幅、Markdown、Lint、Schema 与嵌套字段测试。原测试名保持不变；只修正测试文件 import 依赖。定向测试、`internal/tools`、全量测试、vet 与 diff check 全部通过。后续 Knowledge/Foreshadow/Rewrite/Integrity/Completion 测试继续小切片移动。

阶段 181 第二个切片已完成：新增 `internal/tools/commit_knowledge_test.go`，移动 20 个 Knowledge/Belief/ReaderKnown/Knowledge Replay 测试；原测试名保持不变，未修改生产代码。定向 Knowledge 测试、`internal/tools`、全量测试、vet 与 diff check 全部通过。下一切片处理 Rewrite/Integrity/Completion 测试。

阶段 181 第三个切片已完成：新增 `internal/tools/commit_foreshadow_test.go`，移动 9 个 Foreshadow 生命周期、Rewrite 重建与 Saga 重放测试；原测试名保持不变，未修改生产代码。Foreshadow 定向测试、`internal/tools`、全量测试、vet 与 diff check 全部通过。

阶段 181 最后切片已完成：
- `commit_rewrite_test.go`：15 个 Rewrite/完成态测试；
- `commit_integrity_test.go`：16 个 PendingCommit 密封/恢复测试；
- `commit_completion_test.go`：2 个基础/兼容测试；
- 原 `commit_chapter_test.go` 保留 2 个基础测试；
- 独立 `commit_process_recovery_test.go` 保留 1 个跨进程测试。

Commit 测试共 75 个，名称唯一且与拆分前一致。Rewrite、Integrity、Completion 定向测试、`internal/tools`、全量测试、vet 与 diff check 全部通过。阶段 181 已完成，候选 4 进入下一阶段。

阶段 181 收口验证：Commit 测试集合与 HEAD 中的 `commit_chapter_test.go`、`commit_process_recovery_test.go` 完全一致，共 75 个唯一测试，无丢失、重复或新增。Commit 测试已按 seam 分布到 payload、knowledge、foreshadow、rewrite、integrity、side_effects 和 process_recovery 文件；生产代码无差异。全量测试、vet、关键 race、diff check 全部通过。

阶段 182 首个切片已完成：新增 `internal/tools/context_knowledge_test.go`，移动 6 个 Knowledge/Belief/ReaderKnown 测试。完整 Context 测试集合与 HEAD 对比为 33 个唯一测试，未丢失、重复或新增；定向测试、`internal/tools`、全量测试、vet 与 diff check 通过。生产代码无差异。

阶段 182 第二个切片已完成：新增 `internal/tools/context_recall_test.go`，移动 8 个 Recall/伏笔/Review 记忆测试。完整 Context 测试集合仍为 33 个唯一测试，未丢失、重复或新增；Recall 定向测试、`internal/tools`、全量测试、vet 与 diff check 全部通过。生产代码无差异。

阶段 182 第三个切片已完成：新增 `internal/tools/context_budget_test.go` 与 `internal/tools/context_references_test.go`，移动 7 个 Budget/Platform Rubric/References 测试。完整 Context 测试集合仍为 33 个唯一测试，未丢失、重复或新增；Budget/References 定向测试、`internal/tools`、全量测试、vet 与 diff check 全部通过。生产代码无差异。

阶段 182 已完成：最后 10 个测试拆分为 `context_errors_test.go`、`context_modes_test.go`、`context_envelope_test.go`；原文件仅保留共享 helper，Simulation 与 Reader Boundary 独立文件保持不动。全部 Context 测试与 HEAD 对比为 33 个唯一测试，未丢失、重复或新增；定向测试、`internal/tools`、全量测试、vet 与 diff check 全部通过。候选 4 下一步进入阶段 183 Import 测试资产拆分。

阶段 182 收口：Context 测试按 Knowledge、Recall、Budget、References、Modes、Envelope、Errors 七个消费 seam 完成整理；`novel_context_simulation_test.go` 和 `novel_context_reader_boundary_test.go` 保持独立。完整测试集合与 HEAD 保持 33 个唯一测试，生产代码无差异。阶段 183 已进入进行中。

阶段 183 首个切片已完成：严格 Contracts 已确认无需移动，原本集中在 `analyze_test.go` 的 8 个 Knowledge/认知连续性测试已迁移至 `analyze_knowledge_test.go`。Import 全部测试集合与 HEAD 对比为 103 个唯一测试，无丢失、重复或新增；Knowledge 定向、Import 全包、全量测试与 vet 通过。生产代码无差异。

阶段 183 第二个切片已完成：`publish_test.go` 中 4 个映射、Knowledge 发布与元数据规范化测试已迁移至 `publish_provenance_test.go`；`TestPublishChapterHandlesStalePendingCommit` 保留在原文件作为恢复测试。Import 全部测试集合与 HEAD 对比仍为 103 个唯一测试，无丢失、重复或新增；发布定向、Import 全包、全量测试与 vet 通过。生产代码无差异。

阶段 183 第三个切片已完成：`runner_test.go` 中 6 个 Import 主流程、全书事实发布前门禁、Synthesis 门禁、原文保真与 Completion Hold 测试已迁移至 `runner_recovery_test.go`；运行时配置、错误反馈和重分段测试保留在原文件。Import 全部测试集合与 HEAD 对比仍为 103 个唯一测试，无丢失、重复或新增；Runner 定向、Import 全包、全量测试与 vet 通过。生产代码无差异。

阶段 183 第四个切片已完成：`analyze_test.go` 中 6 个全书事实校验与工作区失效测试已迁移至 `analyze_facts_test.go`；批次分析、Salvage、预算和 Schema 失效测试保留在原文件。Import 全部测试集合与 HEAD 对比仍为 103 个唯一测试，无丢失、重复或新增；Analyze 定向、Import 全包、全量测试与 vet 通过。生产代码无差异。

### 候选 4：测试按 seam 拆分

从阶段 180 开始，先建立 Commit/Context/Import 测试函数归属表，再按 seam 移动文件。保持测试名称、数量、公共接口和 `-run` 筛选稳定；每次移动后先局部测试再全量。该线不新增测试框架、不修改生产代码。

### 候选 2：Context selection policy

候选 4 之后进入阶段 185—188。先盘点 Context 输入/输出与测试面，再做 deletion test；只有删除选择、Knowledge 净化或预算逻辑会集中复杂度时才深化模块。否则以测试整理和文档结论收口，不创建 Context Service/Repository，不改变 JSON envelope、ReaderKnown/CharacterKnown、未来信息过滤、8 条上限或预算裁剪。

### 候选 3：Import 认知 A/B

阶段 189—192 独立并行准备。先用 fake model 验证可断点 runner，再在 Provider 稳定时执行有限探针和完整三轮 A/B。每条结果立即落盘；缺结果失败，不允许 skip；连续 Provider 阻塞即停止。完整统计不阻塞 Release Candidate，也不因不完整证据继续修改 Prompt。

### 共同门禁

阶段 193—194 负责候选 2/3/4 的全量测试、vet、race、格式、文档和路线收口。当前下一步是阶段 180：测试资产归属清单。

阶段 183 第五个切片与阶段 184 收口已完成：`segment_test.go`、`source_test.go`、`state_test.go`、`synthesize_test.go`、`workspace_test.go`、`call_test.go`、`contracts_test.go` 原本已按职责独立，无需继续移动。Import 全部测试集合与拆分前 HEAD 保持 103 个唯一测试，未丢失、重复或新增；全量测试、vet、关键回归和 diff check 通过。候选 4 的测试资产整理完成，生产代码无差异。

阶段 185 已完成：建立 `docs/context-policy-map.md`，盘点 ContextTool 的输入、选择、净化、预算、序列化和测试 seam。确认 `ContextTool.Execute` 适合作为适配器，若深化只考虑集中选择/净化纯逻辑，不引入 Context Service/Repository。

阶段 186 开始：进行 deletion test，验证删除 Knowledge 净化、选择、预算裁剪或 envelope builder 是否会造成泄漏、调用方复杂度转移或行为回归；不以文件行数单独决定生产重构。

阶段 186 已完成：在临时副本中做了可编译 deletion test。删除 Knowledge 选择会使 Knowledge/Reader 边界测试失败；删除预算裁剪会使预算契约失败；删除 Envelope apply 会使分区和净化结果缺失。结论：三者均承担真实职责，但不值得创建通用 Context Service。阶段 187 限定为同包内的纯 Knowledge 选择/净化策略深化，IO、预算和 JSON envelope 保持在现有适配器。

阶段 187—188 已完成：将 Knowledge 选择、时间边界、ReaderKnown/CharacterKnown 过滤、active belief 净化与 8 条上限提取到 `internal/tools/context_knowledge_policy.go` 的同包纯策略函数。`ContextTool` 继续负责 Store IO、角色匹配、错误降级、预算和 JSON envelope。Knowledge/Reader Boundary、Context 全包、全量测试、vet、关键 race 与 diff check 全部通过；未创建 Context Service/Repository，未改变公共 JSON 行为。

阶段 189 已完成：新增 `internal/eval/import_knowledge_runner.go`，仅用于离线 Import 认知校准。Runner 按样本原子写入脱敏结果，记录 Prompt 名称/摘要，Prompt 身份变化不会错误复用旧结果；单样本错误写入错误摘要并可续跑，损坏结果 fail-loud，未保存原始模型响应。fake executor 测试覆盖首次运行、断点恢复、Prompt 隔离、错误续跑和损坏工件。

阶段 190 有限探针已开始：可断点 Runner 的离线行为已完成。真实单样本探针 `ik03` 的 baseline/calibrated 均成功，但动作输出存在随机差异；`ik04` 本次 baseline 与 calibrated 均输出 `establish→learn(林澈)→reveal_to_reader`，与此前单次结果不同，因此不把单次结果当作 Prompt 因果证据。当前仅继续收集独立样本，结果写入临时目录，完成后清理。


阶段 191 已开始：有限探针 7/7 双侧有效后，启动完整 12 条样本 × baseline/calibrated × 3 轮 A/B。六个 arm 按轮转、反序和原序分别运行；每条结果独立落盘，错误单独记录，不保存原始模型响应。

阶段 191 评测器启动记录：首次启动在调用 Provider 前失败，原因是 shell 计算的 DIGEST_BASE/DIGEST_CAL 未 export 给 Python 协调器；未产生模型调用。将改为协调器内部计算摘要，并修正临时 Runner 的每样本原子写入、单样本 Usage 增量和错误继续策略后再启动。

阶段 190 已完成：7/7 有限探针均取得 baseline/calibrated 有效双侧结果，满足完整 A/B 门槛。阶段 191 已完成：12 条自建样本、两种 Prompt、三轮顺序变化共 36 条结果全部有效，0 Provider 错误、0 超时；结果逐样本落盘，未保存原始响应。阶段 192 已完成：baseline 总 precision/recall=0.9184/0.7895，calibrated=0.8448/0.8596；calibrated 的 learn recall 由 0.5833 提升至 1.0000，但 reveal_to_reader precision 由 1.0000 降至 0.7143，动作集合完全匹配由 21/36 降至 19/36。因此 Go/No-Go 为保留当前 calibrated Prompt，不继续追加规则。脱敏汇总见 `evals/import-knowledge/ab-summary.json` 与 `report.md`。

候选 3 阶段 190—192 已完成：有限探针 7/7 双侧有效；完整 12 条样本 × baseline/calibrated × 3 轮共 36/36 有效结果，0 Provider 错误、0 超时。最终统计：baseline precision/recall=0.9184/0.7895，calibrated=0.8448/0.8596；calibrated 的 learn recall 0.5833→1.0000，但总 precision、reveal_to_reader precision 与动作集合完全匹配下降。负例两版均 9/9 为空。Go/No-Go：保留当前 calibrated Prompt，不继续追加规则。原始模型响应、临时 Runner 和结果已清理，版本库仅保留脱敏汇总。
