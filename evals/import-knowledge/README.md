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
- `report.md`：当前基线状态、Provider 阻塞证据与限制。
- 后续有效基线产物只保存动作级脱敏结果与 Usage，不保存完整模型响应。

语义约定：`establish` 让 Truth 成为作者层正式事实；`reveal_to_reader` 只用于此前隐藏的完整 Truth 在本章明确揭晓，不等同于每条普通世界事实。猜测、故意说谎、听到但明确不相信均不能自动生成 `learn/believe/reveal_to_reader`。

所有片段均为本项目自建，不含第三方小说文本。
