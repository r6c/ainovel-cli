# 发布说明

## ainovel-cli 当前发布候选

ainovel-cli 是一个面向单机、Linux 服务器/NAS、macOS 和 Windows 的可恢复 AI 小说创作引擎。核心原则是：模型负责开放语义，代码负责状态、约束、事务、恢复和验证。

## 主要能力

- 串行确定性 Engine 驱动 Architect、Writer、Editor；Arbiter 只处理边界清晰的语义裁定。
- ChapterRecord 作为章节事实重建输入，Knowledge、Foreshadow、Timeline 等是可重建投影。
- PendingCommit Saga 具有 v2 payload/draft/intent 密封和 legacy/v1 兼容恢复。
- Import 先在独立工作区完成分章、分析、综合和全书事实重放，再发布到正式 Store。
- imported/user 正文拥有保护性写权限，不能被自动 generated rewrite 覆盖。
- Writer Context 区分 Author Truth、Character Known、Reader Known 和 Character Belief。
- 用户可通过 `deconstruct` 分析主动提供的本地文本，生成抽象 SimulationProfile，并增量复用未变化来源。

## 安装与无头使用

帮助和版本命令不需要配置、模型、TTY 或桌面环境：

```bash
ainovel-cli --help
ainovel-cli version
ainovel-cli deconstruct --help
```

Headless、Linux amd64/arm64 静态构建和 Docker 无网络帮助冒烟已纳入 CI。Linux 缺少 `notify-send` 时通知仅降级为日志，不阻断创作或恢复。

## 兼容性

- 旧 UserRules 快照缺少新增字段时按兼容零值读取；当前快照版本为 v4。
- 旧 ChapterRecord 与 SimulationProfile v1 保持读取兼容。
- PendingCommit 的 legacy/v1 工件会在安全条件满足时升级；未来未知格式明确拒绝。
- 新 PendingCommit 使用 v2 密封，并把 origin 纳入冻结意图。

## 本地拆文边界

`deconstruct` 只读取用户主动提供的 `.txt`、`.md`、`.markdown` 文件。画像只提炼结构、节奏、冲突、信息释放、钩子和情绪方法，不应包含连续原文、具体作者模仿、专名或签名式语言。扫榜、网页抓取、URL 下载、Chrome/CDP、浏览器登录态和反爬均不支持。

## 验收状态

已通过自动化与部分真实 Provider 验收：Quick/Headless、Cocreate、Revision、Import 后续写、Deconstruct、Linux/无头、恢复和成品隔离均有记录。Import 认知动作完整统计及真实 Architect 扩弧后的 Context 仍受 Provider 可用性限制；已有定向证据和代码层回归，不将未完成统计伪报为完整通过。
