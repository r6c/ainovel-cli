# AI 味判据校准集

本目录用于校准 ainovel-cli 的中文网文 AI 味语义判据，不做文本来源鉴定，也不输出 AI 概率。

## 文件

- `samples.json`：16 条 ainovel-cli 自建匿名网文最小对，只包含 ID 和正文。
- `labels.json`：独立项目金标；Judge 不读取此文件。
- `judge-runs.json`：`sss / gpt-5.6-sol` 三轮固定顺序盲评结果和 Token Usage。
- `writer-ab.json`：现有 Writer smoke case 的 baseline/calibrated 三重复指标及匿名偏好结论；原始章节不入 Git。
- `report.md`：协议、复核、限制和采用决策。

## 标签原则

只判断“是否应仅因明显模板化/AI 味做最小修改”。有明确人物、信息、节奏、悬念、任务或体裁功能时，即使包含短段、问句、比喻、排比、冒号或对照句式，也应保留。

金标是项目编辑政策，不是真人/模型来源标签。若模型意见暴露金标本身违反预先原则，应人工复核并记录原因，而不是机械追随多数票。

## 外部研究来源

候选假设参考：

```text
https://github.com/larashero3-dotcom/lieflat-less-ai-tone
commit: 27d29232f10124db904ca9c0536d0b67cb3b2833
license: MIT
reviewed: 2026-08-25
```

本仓库没有安装该 Skill、没有执行其 Python 脚本、没有复制其代码或长段规则文本，也没有采用其不可公开复核的统计数字。正式判据是根据本目录自建网文样本 clean-room 整理的项目规则。

## 限制

- 16 条样本用于证伪泛化规则，不代表统计语料库。
- 同一个 Judge 三轮只能检查顺序敏感性和判断稳定性。
- Writer A/B 有模型随机差异；偏好只能说明无明显回归和趋势，不能证明因果。
- 语义结果不能成为 Commit 硬门禁；确定性门禁仍只认现有规则事实。
