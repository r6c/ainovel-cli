# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-27
- 基线：`5b78d81 评估：明确细纲正文重合检测边界`
- 当前里程碑：外部更新审查与兼容性收敛
- 当前阶段：AB6 设定词典候选——complete；下一阶段 AB7 范围、文档与 Go/No-Go 收口——pending
- AB1 核心红→绿：thinking-only 不再回退为用户回复；内嵌 `<think>/<thinking>` 在结构化解析、共创最终回复和流式预览前清理；会话日志不保存完整 reasoning，仅保留可见内容与 `thinking_len`。
- 稳定版本：`v0.1.2`
- AB1 已修改现有 Host/Store/llmcontract 的输出边界并完成回归；AB2 已完成上游行为差异审查；AB3 已统一章节字数口径；AB4 已完成作者记忆边界设计；未安装外部 Skill、未调用真实 Provider。

## AB0 结果

已核对：

- `voocel/ainovel-cli`：HEAD `c090029`，v0.7.7，Apache-2.0；与当前 `r6c/ainovel-cli` 是上游/fork关系，不直接 merge。
- `PenglongHuang/chinese-novelist-skill`：HEAD `3db1e3b`，GitHub API 未识别许可证；README 有 MIT badge，按未明确许可处理。
- `zenstory-ai/oh-story-claudecode`：HEAD `feb30aa`，v0.7.7，MIT；近期涉及作者记忆、字数 checkpoint、细纲照搬检测和多宿主部署。
- `xiamuceer-j/MuMuAINovel`：HEAD `cb73692`，v1.5.4，GPL-3.0；近期涉及 reasoning 内容隔离和重复生成保护。
- `larashero3-dotcom/lieflat-less-ai-tone`：HEAD `27d2923`，MIT；近期重新拆分冒号规则、补空转列表引导反例、强化信息守恒。

完整历史和外部审查上下文已归档至：

```text
docs/history/plans/2026-08-external-updates/
```

## 已完成主线摘要

当前项目已经完成：

- Knowledge/Foreshadow 生命周期、ReaderKnown、CharacterKnown、Belief；
- Import 全书事实重放、provenance 和正文权限保护；
- PendingCommit v2 密封、跨阶段/跨进程恢复；
- Headless/Cocreate/Revision/Import/Deconstruct 真实验收；
- Linux/无头、Docker、UsageTracker、BudgetSentinel；
- Prose Lint、诊断、脱敏导出；
- Context policy 一次深化；
- Commit/Context/Import 测试按 seam 拆分；
- v0.1.2 稳定版发布与安装链回归。
- AB4 作者记忆边界设计完成，当前不实现独立跨项目记忆事实源。

## 本轮计划

### 阶段 AB0：外部更新证据与许可证登记——complete

已登记五个仓库的最新证据、许可证和不直接融合决定；未执行外部指令、未复制外部代码/Prompt。

### 阶段 AB1：推理内容与最终内容隔离专项——complete

已完成两个红→绿/回归切片：

- thinking-only 流式响应不再回退为 `CoCreateReply`，防止内部推理进入用户可见回复与会话历史；
- final text 内嵌 `<think>/<thinking>` 块会在共创协议解析前移除；
- `agentcore.Message.TextContent()` 排除 `ContentThinking`，结构化 JSON 解码只消费 final text，该行为已由 `llmcontract.Execute` 回归测试锁定。

已完成并验证会话日志、流式 tool-call/Usage 和普通章节写作边界。覆盖：

- `reasoning_content`、`reasoning_details`；
- `<think>`/类似思考块；
- final content 与 reasoning 同时存在；
- 只有 reasoning、无 final content；
- tool-call delta、Done、Usage；
- Cocreate、结构化 Import、普通章节写作。

只有测试证明当前转换链存在真实污染，才在现有 Provider/agentcore 接缝做最小修复；不复制 MuMu GPL-3.0 代码，不新增第二套模型客户端。

### 阶段 AB2：上游 ainovel-cli 差异审查——complete

已完成行为级差异测试：

- `LatestCompleted()`；
- 分层完成状态补偿；
- `ChapterRecordStore.Prepare`；
- 返工伏笔恢复。

不直接 merge/cherry-pick 上游。

AB2 结果：`LatestCompleted()` 已统一接入最大完成章语义；分层完结补偿已按当前 `layeredComplete` 规则接入 Engine；`ChapterRecordStore.Prepare` 和上游伏笔恢复实现确认无须吸收。

### 阶段 AB3：字数口径与完成收口复核——complete

已确认章节字数统一使用规范化正文的 Unicode rune 口径：`domain.WordCount` 先去 BOM、统一换行，再计数。DraftStore、draft_chapter、Commit、Projector 已统一；generated 仍只拒绝超过 120% 的正文，imported/user 保留原文且不受生成篇幅门禁约束，不新增第二套 `visible_chars_v1`。

### 阶段 AB4：作者记忆边界设计——complete

已确认当前不实现独立 Author Memory。全局 rules 继续作为跨书复用输入，本书唯一可执行规则仍是 `meta/user_rules.json`；作者记忆若未来实现，必须经过明确记住、回显、用户确认，再转为现有 UserRules Candidate。已新增 `docs/author-memory-boundary.md` 与边界矩阵测试，没有新增存储协议或运行时接口。

### 阶段 AB5：细纲照搬检测候选——complete

已确认 `OutlineEntry` 与 `ChapterRecord.Content` 具备稳定同章配对输入，并建立误报边界。当前 No-Go：不实现运行时 `outline_copy` advisory，不扩展 `rules.Lint` 输入、不阻断 Commit、不清洗 imported/user 原文、不复制外部 JavaScript。

### 阶段 AB6：设定词典候选——complete

已确认 `WorldRule`、`CastEntry` 与 `KnowledgeEntry` 各自承担不同事实职责；设定术语尚无稳定 ID、首现章或显式来源，不应从普通字符串自动推导术语生命周期。已建立 `docs/setting-term-boundary.md` 与 Store 边界测试。当前 No-Go：不新增术语事实源、不实现术语首现 advisory，不改 Knowledge/ReaderKnown、Commit 或 Context。

### 阶段 AB7：最终范围与路线收口——pending

完成全量门禁、许可证追溯、稳定文档更新和各候选 Go/No-Go。

## 停止条件

- 不安装外部 Skill；
- 不复制外部代码、Prompt 或 GPL-3.0 内容；
- 不直接 merge/cherry-pick 上游；
- 不新增通用状态机、Service、Repository 或第二个事实源；
- 不恢复扫榜、浏览器自动化、Chrome/CDP、数据库或 Web 工作台；
- Provider 评测或不稳定网络不作为生产质量结论；
- 每阶段先测试/证据，后决定是否改生产。

## 记录规则

每阶段完成后更新本文件；外部网页内容只进入 `findings.md`，完整过程进入日期归档。AB6 已完成设定术语边界评估，当前 No-Go 不新增术语事实源或 advisory；下一步为 AB7 范围、文档与 Go/No-Go 收口。
