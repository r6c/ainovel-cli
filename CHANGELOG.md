# 变更记录

## 当前发布候选

本发布候选包含最近一轮领域、恢复、真实 Provider 和跨平台验收结果。版本号由 GoReleaser 在打标签发布时注入；源码构建默认显示 `dev`。

### 可靠性与恢复

- Knowledge 与 Foreshadow 生命周期统一由专用纯 Apply/Replay 规则裁决。
- Import 在分析、综合和发布前执行全书 ChapterFacts 重放；非法尾部会回退到 Analyze。
- PendingCommit 使用 v2 密封，保护 payload、正文、chapter/rewrite/rewrite_mode/origin 意图。
- 兼容 legacy、v1、v2 PendingCommit；损坏或摘要不匹配时保留工件并零副作用失败。
- 完结态会先收尾 PendingCommit、PendingRevision、活动 Import 和外部正文修改，再显示完成。
- 跨进程磁盘恢复、Knowledge/Foreshadow 重放和 Imported provenance 已通过自动化验收。

### 创作与协同

- 支持作者 Truth、角色 KnownBy、ReaderKnown 和 CharacterBelief 的独立追踪。
- Import 原文保持逐字内容和 `origin=imported`；自动返工不会覆盖 imported/user 正文。
- UserRules v4 支持明确的单章篇幅目标、清除动作和安全上界；只对 generated 正文执行篇幅上限。
- Cocreate 支持 core → customization → title → confirmation → ready 阶段化访谈。
- Cocreate 与 Deconstruct 的模型调用均进入 UsageTracker 和预算系统。
- `ainovel-cli deconstruct <本地语料目录>` 支持本地拆文画像和指纹增量更新。

### 平台与可移植性

- 番茄仅作为用户显式指定时加载的七维软参考，不新增平台评分或算法分。
- 支持 Linux/macOS/Windows 构建；Linux amd64/arm64 与 Docker 无头入口已验收。
- 不支持扫榜、网页抓取、Chrome/CDP、浏览器登录态或反爬适配。

### 已知限制

- Import 认知动作已完成定向 Prompt 校准，但完整三轮 precision/recall 统计受 Provider 长连接和 HTTP 502 阻塞，未作为完整统计发布。
- 真实 Architect 扩弧后的第 3 章 Context 端到端验收曾受 Provider 阻塞；代码层 ReaderKnown/CharacterKnown 净化边界已有确定性回归。
- `usage.updated_at` 可能在零模型调用的 Host 关闭时刷新；Cost/Token/PerAgent 数值不变。
