# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-25
- 基线：`ade8108 测试：加固 Linux 与无头环境兼容性`
- 当前里程碑：N——端到端创作验收与发布就绪检查
- 当前阶段：阶段 112—118 全部完成
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

人工真实模型文学质量验收仍明确标记为未执行：没有调用用户 Provider，也没有消耗额度。提交信息：`测试：建立端到端创作发布验收基线`。
