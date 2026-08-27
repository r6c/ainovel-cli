# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-27
- 基线：`40a3613 发布：完成 v0.1.2 发布后稳定性观察`
- 当前里程碑：外部更新审查与兼容性收敛
- 当前阶段：AB1 推理内容与最终内容隔离——complete；下一阶段 AB2 上游差异审查——pending
- AB1 核心红→绿：thinking-only 不再回退为用户回复；内嵌 `<think>/<thinking>` 在结构化解析、共创最终回复和流式预览前清理；会话日志不保存完整 reasoning，仅保留可见内容与 `thinking_len`。
- 稳定版本：`v0.1.2`
- AB1 已修改现有 Host/Store/llmcontract 的输出边界并完成回归；未安装外部 Skill、未调用真实 Provider。

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

### 阶段 AB2：上游 ainovel-cli 差异审查——pending

只做行为级差异测试：

- `LatestCompleted()`；
- 分层完成状态补偿；
- `ChapterRecordStore.Prepare`；
- 返工伏笔恢复。

不直接 merge/cherry-pick 上游。

### 阶段 AB3：字数口径与完成收口复核——pending

对照 `visible_chars_v1` 与当前 `domain.WordCount`、`chapter_target_chars`、120% 上限和来源政策；若口径一致，只保留测试/文档结论。

### 阶段 AB4：作者记忆边界设计——pending

评估跨项目长期偏好与当前 UserRules 的分界；只考虑明确记住、确认、回执、相关性上限、冲突和撤回。

### 阶段 AB5：细纲照搬检测候选——pending

评估同章 outline/prose 是否有稳定可比文本；先 advisory，不直接阻断 Commit。

### 阶段 AB6：设定词典候选——pending

评估术语首现线索和 ReaderKnown 复用；优先诊断 warning，不建平行状态机。

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

每阶段完成后更新本文件；外部网页内容只进入 `findings.md`，完整过程进入日期归档。当前下一步是 AB1 的确定性 reasoning 隔离回归测试。
