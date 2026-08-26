# Import Knowledge 校准阶段报告

日期：2026-08-26
状态：阶段 158 `blocked_provider`

## 已完成

- 建立 12 条完全自建中文小说片段。
- 建立独立动作金标，覆盖 establish-only、establish+learn、establish+reveal、三动作、belief，以及猜测/谎言/明确不相信负例。
- 通过确定性数据契约测试。
- 确认真实测试 seam 为当前 Import `analysisContract + llmcontract.Execute`，没有新增 Judge Schema，也没有修改生产 Prompt。

## Provider 通道验证

使用当前配置的 `sss / gpt-5.6-sol`：

- 单批 2 条普通事实验证成功，约 31 秒。
- 两条样本均提取为 establish。
- 结果含 provider/model 与 Usage，未保存完整模型响应。

## 未完成的基线

完整三轮基线没有取得可统计结果：

1. 一次 9 批运行长时间停在本地代理 TCP 连接，结果文件为空，已终止。
2. 改为每批 2 条后，第 1 轮第 3 批中模型返回了不一致的 Knowledge ID；正式 Import 校验进入未知引用自愈，随后遇到 HTTP 502，并在 5 分钟 context timeout 结束。

这两次都不计入 precision/recall、一致性或 Prompt 质量证据。没有因无效试跑修改 `import-analyze.md`。

## 当前决策

- 阶段 157：`complete`。
- 阶段 158：`blocked_provider`。
- 阶段 159/160 不启动。
- Provider 通道稳定后，从阶段 158 重新执行三轮基线。
- 在取得有效基线前，不根据单批结果推断 learn/reveal 漏报，也不加入 worked examples 到生产 Prompt。
- 临时 live runner、结果文件、日志和一次性进程均已删除。

## 许可证与来源

片段均为本项目自建，不含第三方文本。没有安装外部 Skill、没有运行外部脚本、没有复制外部代码或 Prompt。
