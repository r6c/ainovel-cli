# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-25
- 基线：`ade8108 测试：加固 Linux 与无头环境兼容性`
- 当前里程碑：P1——真实外部正文 Revision 验收
- 当前阶段：阶段 135——Revision 自动化验收基线（pending）
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

O3 Normalizer 首个字段切片通过后，`userrules` 全包暴露旧 strict JSON 夹具缺少新必填 `chapter_target_chars`；scripted model 重复最后一条无效响应，产生大量反馈日志。未原样重跑，改为先定位并同步旧夹具为空值 0，再分组验证。一次对 `novel_context_test.go` 单文件路径使用 `search_files` 返回 `ENOTDIR`，未重复该方式，改用直接读取。O3 Prompt 契约首次编辑漏掉循环闭合大括号，测试语法失败未当作行为红灯；先修正测试结构再重跑。真实 O 回归完成后，检查脚本先猜错 ChapterRecord 文件名/JSON 外形（实际 `000001.json`，violations 外层对象），未据此判失败；后续按真实工件复查。一次 ChapterRecord API 搜索正则括号未闭合，改用字面搜索。初次读取错误猜测文件名为 `headless.go/headless_test.go`，实际为 `run.go/run_test.go`；未修改文件，已改用真实路径。首次 O1 测试又猜测了不存在的 `DraftStore.SaveChapterText`，编译失败未当作产品红灯；已转为读取真实 DraftStore API。修正后测试目录又因缺 `meta/format.json` 进入合法迁移门禁，仍未视为产品红灯；夹具已补当前格式版本。O1 首轮绿灯后真实目录字段比较发现 Progress/费用/Token 不变，但 `Host.Close()` 会刷新 `usage.updated_at`；因此终态判断前移到 Host 构造前，并由非空 Usage 工件测试锁定。

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

阶段 127 启动时一次读取猜错 `internal/store/checkpoint.go`，实际文件为 `checkpoints.go`；未重复错误路径。首个 Headless 恢复测试夹具又因 Book 缺必填 Synopsis 在行为 seam 前失败；不作为产品红灯，补合法作品元数据后重跑。最小恢复实现后 PendingCommit 已清理，但测试用旧 Store 的 CheckpointStore 内存镜像观察不到另一个 Host 新增的 checkpoint；改为重开 Store 读取磁盘事实。随后猜测了不存在的 `internal/host/resume_test.go`；Host 无独立 Resume 测试文件，Headless 公共用例已通过真实 Host.Resume 覆盖，不再猜文件名。阶段 129 批量切换 Headless 到 Host 终态探测时，import/函数片段与预期不完全匹配，结构化编辑未应用；改为读取真实文件后分步替换。阶段 130 首个 Import 测试误把 `LoadRuleViolations` 当作双返回接口，编译失败未当作产品红灯；按真实单返回接口修正。增强篇幅目标夹具时又把 `UserRules.Save` 的指针参数传成值，仍未触达产品行为，已按真实接口修正。阶段 133 首次受限夹具迁移使用 `node`，但用户环境无该命令，迁移未执行；新增断言同时缺 `rules` import。改用已确认存在的 `python3` 并补 import，不重复 Node 方案。
