# 升级说明

## 升级前建议

1. 备份整个作品目录，尤其是 `meta/`、`chapters/` 和 `meta/chapter_records/`。
2. 不要手工删除 `meta/pending_commit.json`、`meta/pending_revision.json` 或 Import 工作区；先让新版本执行恢复和诊断。
3. 若使用自定义 Provider，确认配置中的 provider、model、base_url 和 API key 仍有效；密钥不会写入本说明或诊断导出。

## PendingCommit

新建 PendingCommit 使用密封版本 v2，摘要保护：

- 规范化 payload；
- 冻结正文；
- chapter、rewrite、rewrite_mode、origin 等不可从 payload 推导的提交意图。

legacy 和 v1 PendingCommit 仍可在安全条件下恢复。未知 seal version、半密封工件、摘要不匹配或 legacy 工件带有非空 origin 时会停止恢复并保留工件，避免自动猜测或重签。

## UserRules

当前 UserRules 快照版本为 v4。旧 v1—v3 快照缺少新增字段时按兼容零值读取，不要求用户手工迁移。

单章篇幅目标支持三态：

```text
keep   未声明，保留已有值
set    设置正数目标
clear  明确清除已有目标
```

目标上限为 1,000,000 字符。只有 generated 正文执行目标 120% 上限；Import 原文和用户正文保持原文接纳策略。

## Import 与正文来源

Import 原文记录为 `origin=imported`，用户外部修改记录为 `origin=user`。两类正文不会被自动 generated rewrite 覆盖；用户需要修改它们时，应编辑正文后使用 `/sync`。

导入工作区会在发布前执行全书事实重放。发现非法分析工件时会失效首个错误章节及后续派生工件，并返回 Analyze，而不是部分写入正式 Store 后反复卡在 Publish。

## SimulationProfile

旧 `simulation_profile.v1` 继续读取。`deconstruct` 只接受本地 `.txt`、`.md`、`.markdown` 语料，不支持 URL、网页、排行榜或浏览器登录态。来源指纹用于增量分析；未修改来源不会再次调用模型。

## 配置与平台

番茄参考只有在用户明确指定 `structured.platform=fanqie` 时加载；它是现有七维评审的软参考，不是平台算法分或硬门禁。没有配置的平台字段按空值兼容。

## Linux 与无头环境

Linux amd64/arm64 构建不依赖 CGO。帮助命令、Headless 和 Docker 入口不要求 TTY、DISPLAY 或桌面通知工具。Linux 缺少 `notify-send` 时仅降级为日志。

## 不支持的能力

为保持 Linux、服务器、NAS 和无头环境可移植性，项目不支持：

- 自动扫榜；
- 番茄、起点、晋江网页抓取；
- Chrome/CDP 或浏览器登录态；
- 反爬和平台页面适配。
