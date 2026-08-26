# Import Knowledge 校准集

这是一套 ainovel-cli 自建的中文小说逐章认知事实 worked examples，用于校准 Import 对以下动作的提取：

```text
establish
believe
learn
reveal_to_reader
```

- `samples.json`：匿名片段；真实模型只读取此文件。
- `labels.json`：独立产品金标；不进入模型 Prompt。
- `report.md`：三轮 A/B 脱敏统计、Go/No-Go 结论与已知限制；Provider 阻塞仅保留在历史过程记录。
- 后续有效基线产物只保存动作级脱敏结果与 Usage，不保存完整模型响应。

语义约定：`establish` 让 Truth 成为作者层正式事实；`reveal_to_reader` 只用于此前隐藏的完整 Truth 在本章明确揭晓，不等同于每条普通世界事实。猜测、故意说谎、听到但明确不相信均不能自动生成 `learn/believe/reveal_to_reader`。

所有片段均为本项目自建，不含第三方小说文本。

## 可断点评测 Runner

阶段 189 的 Runner 位于 `internal/eval/import_knowledge_runner.go`，只负责离线评测状态与脱敏结果工件，不属于运行时 Import 管线。它按样本写入：

```text
results/<sample_id>.json
errors/<sample_id>.json
```

成功结果包含样本 ID、分类、Prompt 身份、动作级结果和 Usage；失败结果只包含错误摘要，不保存原始模型响应。只有样本、分类和 Prompt 名称/摘要都匹配时才会断点跳过；损坏结果会 fail-loud，不能被模型调用覆盖。
