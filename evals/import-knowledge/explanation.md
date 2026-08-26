# Import 认知动作 A/B 结果解释

日期：2026-08-26

## 结论摘要

本报告只解释已提交的聚合统计，不重新调用 Provider，也不修改 Import Prompt。

当前 calibrated Prompt 的真实效果不是全面提升，而是一个明确的权衡：

- `learn` 漏报显著减少；
- `reveal_to_reader` 的误报增加；
- `establish` 的召回略降，但精确率上升；
- `believe` 没有改善；
- 三类负例在当前 9 次 arm 观测中均保持空数组；
- 整体精确率和完整动作集合准确率下降。

因此保留当前 calibrated Prompt 作为折中版本，不继续在这批结果上堆 Prompt 规则。

## 评测数据边界

样本类别与金标来自：

```text
samples.json
labels.json
```

聚合结果来自：

```text
ab-summary.json
```

`ab-summary.json` 只保存按动作聚合的 TP/FP/FN、总体 exact-match 和 Usage；没有保存每次运行的逐样本预测集合。因此本报告：

- 可以解释动作级混淆矩阵；
- 可以解释总体指标变化；
- 可以解释预期动作数量和误报总量；
- 不能声称某个具体 `ikXX` 在三轮中的逐次预测，除非已有独立逐样本证据；
- 不把总计数反推成不存在的逐样本事实。

## 金标动作总量

12 条样本、3 轮的预期动作总量为：

| 动作 | 金标样本来源 | 预期次数 |
|---|---|---:|
| `establish` | ik01—ik09 | 27 |
| `learn` | ik03、ik04、ik07、ik08 | 12 |
| `reveal_to_reader` | ik05—ik09 | 15 |
| `believe` | ik09 | 3 |
| 合计 | 9 条正例 × 3 轮 | 57 |

三类负例为 ik10—ik12，各运行 3 次，共 9 次负例场景。

## 动作级混淆矩阵

### `establish`

```text
                 baseline   calibrated   变化
TP                    20          19       -1
FP                     1           0       -1
FN                     7           8       +1
precision         0.9524       1.0000
recall            0.7407       0.7037
```

解释：calibrated 对“未经确认的说法不能自动成为作者 Truth”的边界更谨慎，因而消除了一个误报，但也少识别了一个应建立的 Truth。当前数据支持“精确率提高、召回略降”，不支持断言具体是哪条样本造成变化。

### `learn`

```text
                 baseline   calibrated   变化
TP                     7          12       +5
FP                     0           0        0
FN                     5           0       -5
precision            1.000       1.000
recall            0.5833       1.0000
```

解释：新增的“同一客观 Truth 可以按正文顺序输出多个动作”说明，明显减少了模型只输出 `establish`、遗漏角色明确接受 Truth 的问题。当前样本中没有观察到 `learn` 误报增加。

### `reveal_to_reader`

```text
                 baseline   calibrated   变化
TP                    15          15        0
FP                     0           6       +6
FN                     0           0        0
precision            1.000       0.7143
recall               1.000       1.000
```

解释：baseline 已经能够识别全部金标 `reveal_to_reader`，因此 calibrated 没有召回增益空间；新增的多动作和完整揭示说明让模型在更多情况下主动输出 reveal，但也把部分不应标记为 reveal 的情况纳入，导致精确率下降。这是本轮最主要的回归风险。

### `believe`

```text
                 baseline   calibrated   变化
TP                     3           3        0
FP                     3           3        0
FN                     0           0        0
precision            0.500       0.500
recall               1.000       1.000
```

解释：错误信念样本数量很少，且两版都存在同等 FP。当前结果不能支持通过 Prompt 继续修复 `believe`；需要更多“客观 Truth、角色错误判断、角色仅怀疑、角色故意说谎”分层样本后再决定。

## 总体指标变化

| 指标 | baseline | calibrated | 变化 |
|---|---:|---:|---:|
| 总 TP | 45 | 49 | +4 |
| 总 FP | 4 | 9 | +5 |
| 总 FN | 12 | 8 | -4 |
| 总 precision | 0.9184 | 0.8448 | -0.0736 |
| 总 recall | 0.7895 | 0.8596 | +0.0701 |
| 动作顺序完全匹配 | 16/36 | 17/36 | +1 |
| 动作集合完全匹配 | 21/36 | 19/36 | -2 |

calibrated 增加了 4 个正确动作，同时增加了 5 个错误动作；因此 recall 上升，但 precision 下降。动作顺序完全匹配略有上升，不代表完整集合质量提升，因为集合完全匹配反而下降。

## 负例表现

ik10—ik12 的 9 次 baseline 与 9 次 calibrated 结果均为空数组：

```text
baseline：9/9 空
calibrated：9/9 空
```

这支持以下有限结论：当前新增 Prompt 规则没有让这三类已定义负例产生新的知识动作。

但它不能外推为所有负例都安全，原因是：

- 只有三种负例类别；
- 每类只有一个片段；
- 没有覆盖复杂叙述视角、间接引语和长上下文；
- `believe` 的 FP 不是这些空数组负例能够解释的。

## 三轮一致性与 Usage

现有聚合文件没有保存逐样本、逐轮预测，因此只能确认：

- 两个 arm 均各有 36/36 有效结果；
- Provider 错误/超时均为 0；
- 结果经历了 3 轮顺序变化；
- Input/Output token 已按 arm 汇总。

不能从当前文件计算严格的“逐样本三轮一致率”。后续若需要该指标，Runner 必须保存脱敏的 `sample_id + round + action_signature`，而不是只保存最终聚合数；这属于后续评测资产改进，不在本阶段补写历史结果。

## Go/No-Go

```text
保留当前 calibrated Prompt
不追加新的 Import Prompt 规则
不把 calibrated 宣称为全面优于 baseline
```

理由：

1. `learn` recall 从 0.5833 提升到 1.0000；
2. `reveal_to_reader` recall 已经是 1.0000，没有召回增益；
3. `reveal_to_reader` precision 从 1.0000 降到 0.7143；
4. 总 precision 和动作集合完全匹配下降；
5. `believe` 没有改善；
6. 当前样本不足以支持更细的动作判定规则。

## 下一步样本扩展方向

下一批新增样本应优先补齐：

- Truth 已建立但只被角色猜测；
- 角色明确接受但读者未完整得知；
- 读者获得完整答案但角色只听到片段；
- 角色故意转述错误说法；
- 不可靠叙述者明确声明不确定；
- `partial_payoff` 与完整 `reveal_to_reader` 的边界；
- 同一 Truth 在跨章节 ledger 中的多动作顺序；
- 不同题材、视角和叙述距离。

新增样本仍必须完全自建，金标与匿名输入分离，不能把动作 ID 命名差异当作语义错误。
