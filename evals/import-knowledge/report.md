# Import Knowledge 校准阶段报告

日期：2026-08-26
状态：阶段 160 `partial_evidence`

## 已完成

- 建立 12 条完全自建中文小说片段。
- 建立独立动作金标，覆盖 establish-only、establish+learn、establish+reveal、三动作、belief，以及猜测/谎言/明确不相信负例。
- 通过确定性数据契约测试。
- 确认真实测试 seam 为当前 Import `analysisContract + llmcontract.Execute`，没有新增 Judge Schema；阶段 159 仅对 `import-analyze.md` 做了最小语义修订，未修改 Go 生产代码。

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

## 追加通道复核（2026-08-26）

将完整三轮改为每次单片段、每次独立 5 分钟 context timeout，以排除跨样本 Knowledge ID 污染。该运行仍在本地代理 TCP 连接上长时间无输出，未生成结果文件，已终止；不计入模型质量证据。当前不修改 Prompt，继续等待 Provider 通道稳定。

## 定向 Prompt 修订证据

旧 Prompt 下的定向探针显示：

- `ik03`：正文明确角色验证并接受 Truth，但只输出 `establish`；
- `ik04`：正文明确角色复核数据并接受结论，但只输出 `establish`；
- `ik05`：正文完整向读者揭示此前隐藏 Truth，但输出空数组；
- `ik07`：输出 `establish + learn + reveal_to_reader`，但额外生成了 `believe`，需要继续复核误报边界。

据此只做了最小 Prompt 修订：说明同一 Truth 可在同章按正文顺序输出多个动作，并补充 `establish → learn → reveal_to_reader` 示例，以及“听见但不相信、猜测、部分兑现不等于对应动作”的负例边界。

修订后定向证据：

```text
ik03：establish → learn(苏弦) → reveal_to_reader
ik04：establish → learn(林澈) → reveal_to_reader
ik05：establish → reveal_to_reader
ik07：establish → learn(苏弦) → reveal_to_reader，并额外输出顾临 believe
```

`ik07` 的 baseline 也输出同一个额外 `believe(顾临)`，因此该误报不是本次 Prompt 修订引入的回归。负例复验结果：`ik10`、修正后的 `ik11`、`ik12` 均输出空数组。修订方向改善了 learn/reveal 召回，也暴露了既有 believe 误报风险；完整三轮基线和 Prompt A/B 仍未完成。

## 当前决策

- 阶段 157：`complete`。
- 阶段 158：`blocked_provider`。
- 阶段 159：`partial_prompt_revision`，已完成有证据的最小 Prompt 修订，但不宣称统计改善。
- 阶段 160：`partial_evidence`，已完成有限 A/B 与负例探针，尚未完成完整三轮统计。
- Provider 通道稳定后，从阶段 158/160 继续完整 baseline/A-B；在此之前不进入阶段 161。
- 临时 live runner、结果文件、日志和一次性进程均已删除。

## 许可证与来源

片段均为本项目自建，不含第三方文本。没有安装外部 Skill、没有运行外部脚本、没有复制外部代码或 Prompt。临时 runner、结果文件和日志已在每次探针完成后删除；版本库只保留脱敏样本、标签、报告和阶段记录。
