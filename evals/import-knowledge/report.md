# Import Knowledge 校准阶段报告

日期：2026-08-26
状态：阶段 192 `complete`

## 评测范围

- 12 条完全自建中文小说片段。
- baseline 与 calibrated 两份 Import Prompt。
- 每个 arm 运行 3 轮，共 36 条有效结果。
- 每轮改变样本顺序，避免固定位置偏差。
- 只比较动作、角色和动作顺序；不要求模型复用金标 Knowledge ID。
- 不保存完整模型响应、原始正文或 Provider 凭证。

## 统计结果

| 指标 | baseline | calibrated |
|---|---:|---:|
| 有效结果 | 36/36 | 36/36 |
| Provider 错误/超时 | 0 | 0 |
| 按动作顺序完全匹配 | 16/36 | 17/36 |
| 按动作集合完全匹配 | 21/36 | 19/36 |
| 总 precision | 0.9184 | 0.8448 |
| 总 recall | 0.7895 | 0.8596 |
| Input tokens | 60,738 | 69,018 |
| Output tokens | 29,261 | 28,663 |

### 动作级统计

| 动作 | 指标 | baseline | calibrated |
|---|---|---:|---:|
| establish | precision | 0.9524 | 1.0000 |
| establish | recall | 0.7407 | 0.7037 |
| learn | precision | 1.0000 | 1.0000 |
| learn | recall | 0.5833 | 1.0000 |
| reveal_to_reader | precision | 1.0000 | 0.7143 |
| reveal_to_reader | recall | 1.0000 | 1.0000 |
| believe | precision | 0.5000 | 0.5000 |
| believe | recall | 1.0000 | 1.0000 |

负例 `ik10/ik11/ik12` 两个 arm 的三轮均为空数组：

```text
baseline：9/9
calibrated：9/9
```

## Go/No-Go

决定：**保留当前 calibrated Prompt，不继续追加 Prompt 规则。**

理由：

- calibrated 将 `learn` recall 从 0.5833 提升到 1.0000；
- `reveal_to_reader` recall 保持 1.0000；
- 三个负例没有新增认知动作；
- 但 `reveal_to_reader` precision 从 1.0000 降到 0.7143；
- 总 precision 从 0.9184 降到 0.8448；
- 按动作集合完全匹配从 21/36 降到 19/36；
- `establish` recall 从 0.7407 降到 0.7037。

因此本次结果不支持继续增加“同章多动作”规则，也不支持把 calibrated Prompt 宣称为全面优于 baseline。当前版本作为一个明确的折中：优先保证明确角色获知的 `learn` 不漏报，同时接受仍需后续人工/Editor 复核的 `reveal_to_reader` 精度风险。

## 局限

- 金标由本项目自建片段和人工规则制定，样本量为 12 条。
- 真实模型输出存在随机性，同一样本不同轮次可能出现不同动作集合。
- 结果只衡量动作级结构提取，不外推 Truth 文本质量。
- 不同动作 ID 的命名差异不计为错误。
- 评测未覆盖长批次 ledger、真实多章上下文和所有题材。
- 本报告不构成平台算法或模型能力的普遍结论。

## 来源与许可

所有片段均为本项目自建，不含第三方小说文本。评测方法参考了项目外部的 `lieflat-less-ai-tone` 研究思路，但没有安装外部 Skill、运行其脚本、复制其代码或复制长段 Prompt。
