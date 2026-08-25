# 发布验收清单

本清单用于发布前验证 `ainovel-cli` 的完整用户路径。自动化部分只使用 fake/mock 模型，不访问真实 Provider；人工部分只有操作者明确配置 Provider、接受费用后才执行。

## 1. 问题分级

| 级别 | 定义 | 发布处理 |
|---|---|---|
| P0 | 数据丢失、恢复死锁、正式 Store 污染、创作秘密泄漏、无法完成一本书 | 阻止发布 |
| P1 | 核心流程难以完成或修复，如共创无法 ready、Import 无法回退、状态明显漂移 | 阻止正式发布；允许修复版候选 |
| P2 | 文案、布局、诊断或格式等非阻塞摩擦 | 记录后可发布 |

问题记录至少包含：操作步骤、预期/实际结果、输出目录、相关工件、日志路径和是否可稳定复现。不得附带 API Key。

## 2. 自动化发布基线

### 2.1 Quick / Engine 完整写书

```bash
go test ./internal/host \
  -run 'TestEngine_(WritesBookToCompletion|ReviewPermitWritesExactlyOneNewChapter)$' \
  -count=1 -timeout=5m
```

验证：fake 模型通过真实工具完成整本书；review 许可只稳定一个新章节。

重点工件：

```text
chapters/
meta/chapter_records/
meta/checkpoint.json
meta/progress.json
```

### 2.2 Headless 空工作区失败边界

```bash
go test ./internal/entry/headless -count=1 -timeout=5m
```

验证：无 Prompt 且没有可恢复会话时返回可操作错误，不伪报恢复成功，并生成脱敏诊断。

重点工件：

```text
meta/diag-export.md
logs/headless.log
```

### 2.3 Cocreate 阶段化启动

```bash
go test ./internal/entry/startup \
  -run 'TestCoCreateSession(AdvancesInterviewOneStageAtATime|ReadyRequiresCompleteInterviewDraft|HistoryReflectsAcceptedStageAndReadiness)$' \
  -count=1

go test ./internal/host \
  -run 'Test(ParseCoCreateResponseExtractsInterviewStage|ColdStartCoCreatePromptDefinesOrderedInterviewStages)$' \
  -count=1

go test ./internal/entry/tui \
  -run 'Test(ColdStartCoCreateCtrlSDoesNotStartBeforeReady|CoCreateRequestFailureKeepsInterviewState|ExitColdStartCoCreateRestoresInitialInput)$' \
  -count=1
```

验证：阶段只能逐步前进；完整 Draft 和用户确认前不能启动；失败、取消不会写正式作品事实。

### 2.4 Import 分析、回退与发布

```bash
go test ./internal/host/imp \
  -run 'Test(RunEndToEnd|PublishRejectsInvalidFullBookFactsBeforeWritingOfficialStore|InvalidWorkspaceFactsReturnStateToAnalyze|ValidateWorkspaceFactsInvalidatesFromFirstIllegalChapter|PublishChapterHandlesStalePendingCommit|PublishChaptersPersistsKnowledgeState)$' \
  -count=1 -timeout=5m
```

验证：合法工作区可发布；非法全书事实在正式 Store 写入前失败；首错章之后的分析失效并回到 Analyze；stale PendingCommit 可恢复。

重点工件：

```text
meta/import/
meta/chapter_records/
knowledge_state.json
meta/pending_commit.json
```

### 2.5 本地拆文方法画像

```bash
go test ./internal/host/sim \
  -run 'TestRunner(GeneratesProfileThenSkipsUnchanged|IncrementallyAnalyzesNewAndChangedSources|UsesDeconstructionMethodProfileLanguage)$|TestAnalyzeSourceUsesNativeSchema' \
  -count=1 -timeout=5m

go test ./internal/entry/deconstruct -count=1
```

验证：首次生成画像，未变语料第二次零模型调用，新增/修改文件增量处理；Prompt 只提炼抽象方法，不要求模仿具体作者。

重点工件：

```text
meta/simulation_profile.json
meta/simulation_sources/
```

### 2.6 TXT / EPUB 读者成品隔离

```bash
go test ./internal/host/exp \
  -run 'TestRun_(HappyPath_DefaultsToNovelDir|EPUB_FromExtension)$|TestRunReaderExportsNeverIncludeKnowledgeState' \
  -count=1 -timeout=5m
```

验证：TXT/EPUB 可生成，且不包含作者 Truth、角色错误信念或 Knowledge 内部投影。

### 2.7 Linux / Docker 无头入口

由 CI 自动执行：

```text
linux/amd64 与 linux/arm64 静态跨编译
空 HOME、空 DISPLAY 的 --help / deconstruct --help
无网络、无配置挂载的 Docker ENTRYPOINT 冒烟
Linux 缺 notify-send 时日志降级
```

## 3. 人工真实模型验收清单

> 执行状态见第 5 节。以下步骤会调用真实 Provider 并产生费用，必须由操作者明确选择模型、预算和测试目录后手动执行；未在第 5 节记录的项目均视为未执行。

### 3.1 Quick 短篇闭环

前置：单独测试目录；已配置 Provider；预算上限明确。

步骤：

1. 使用 Quick 创建 3—5 章短篇。
2. 观察 Foundation、Writer、Editor、Commit 和完结流程。
3. 导出 TXT 与 EPUB，人工阅读成品。
4. 检查伏笔、Knowledge、关系和角色状态投影。

预期：无重复章节、无事实漂移、可自然完结；成品不泄露内部状态。

### 3.2 Cocreate 开书

1. 完成核心定位、深度定制、标题简介、规划确认。
2. 故意尝试提前 `Ctrl+S`，确认不会启动。
3. 明确选择标题并确认最终指令后启动。

预期：不重复盘问，不自行替用户确认，不丢失已确认内容。

### 3.3 Headless 强杀恢复

1. 在独立目录用 `--headless --prompt` 启动小型作品。
2. 在章节提交期间终止进程。
3. 再次运行 `--headless` 恢复。
4. 检查 PendingCommit、Checkpoint、Progress 和 ChapterRecord。

预期：不复制章节、不重复推进；密封校验通过；stdout/stderr 可用于日志系统。

### 3.4 外部正文修改与 Revision

1. 完成至少两章。
2. 手工修改已接纳章节正文。
3. 重启并完成 Revision 分析/接纳。
4. 检查后续 Knowledge/Foreshadow/关系投影。

预期：修改被识别；派生事实按新正文重建；失败不会锁死返工 Saga。

### 3.5 Import 后续写

1. 导入一篇用户有权处理的本地小说文本。
2. 完成分章、分析、综合和发布。
3. 继续写 1—2 章。

预期：错误可定位到章节并回退；正式 Store 发布前零污染；续写角色不会越权知情。

### 3.6 本地拆文人工合规检查

1. 使用用户有权处理的 2—3 篇本地文本运行 `deconstruct`。
2. 第二次不修改语料再次运行。
3. 阅读 `simulation_profile.json` 并让小样本写作消费该画像。

预期：第二次增量跳过；画像不包含连续原文、签名短语、专名或具体作者模仿指令；写作不因画像而僵化。

## 4. 发布判定

发布前必须满足：

```text
自动化基线全部通过
无未处置 P0/P1
真实模型验收若执行，结果和费用已记录但不包含密钥
全量 go test / go vet / git diff --check 通过
```

真实模型验收未执行时，发布说明必须如实标记，不得用 fake-model 自动化结果替代文学质量结论。

## 5. 真实 Provider 验收记录

### 2026-08-25 · sss / gpt-5.6-sol · Headless 单章短篇

状态：**通过工程闭环，发现 3 项 P2；其余人工场景未执行。**

范围与费用：

```text
单章完结科幻短篇
目标约 1200 字
首次硬预算 $0.25；恢复后上调为 $0.60
实际累计费用约 $0.410
```

实际过程：

1. 在独立临时目录启动 Headless，完成启动裁定与 Foundation。
2. 外部测试执行器在约 120 秒处终止进程；当时 Progress 为 outline、零章节、无 PendingCommit。
3. 无 Prompt 恢复后进入第 1 章写作；预算在累计约 $0.264 时安全硬停，仍无章节、无 ChapterRecord、无 PendingCommit。
4. 将隔离项目预算上调到 $0.60 后再次无 Prompt 恢复。
5. Writer 两次提交因未知 Foreshadow/Knowledge ID 被前置门禁拒绝，第三次自修复成功。
6. 最终完成 1/1 章、2092 字，进程退出码 0，无 PendingCommit；全量 ChapterRecord 重放验证通过。
7. TXT 与 EPUB 均成功导出，未发现 Knowledge、PendingCommit 或内部协议泄漏。
8. 已完结目录再次无 Prompt Headless 启动不发模型请求、不改工件和费用，但当前返回退出码 1。

通过项：

- 强杀后可从 Foundation/Outline 状态恢复，不重复章节。
- 预算告警和硬停有效；停机点没有留下部分提交。
- 提交前 Knowledge/Foreshadow 引用校验可被真实模型根据错误自修复。
- Progress、ChapterRecord、Timeline、Relationship、StateChange 和 Checkpoint 自洽。
- `revision.ValidateRecordSet` 对真实 ChapterRecord 重放通过。
- TXT/EPUB 成品不泄露作者侧认知状态。
- `diag-export.md` 正文出包为 0，且未发现运行时异常。

P2：

1. **篇幅偏差**：用户目标约 1200 字，实际 2092 字；当前没有机械字数门禁。
2. **Markdown 残留**：正文含 6 个 `**` 标记；`markdown_residue` 已正确记录 warning，但未自动返修。
3. **完结态 Headless 文案/退出码**：完结后无 Prompt 再启动返回“需要 --prompt 或可恢复会话”和退出码 1；无数据或费用副作用，但不够友好。

未执行：

- 真实 Cocreate 五阶段完整会话
- 真实 PendingCommit 中间 Stage 强杀
- 真实外部正文 Revision
- 真实 Import 后续写
- 真实 Deconstruct 文学效果对比

停止扩展原因：单章工程闭环已经覆盖核心 Provider、恢复、预算、提交、投影和导出；继续执行其余场景会产生额外真实费用，应由操作者单独确认预算后进行。
