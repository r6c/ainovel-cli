# 发布后反馈

## 版本

- 当前稳定版本：`v0.1.2`
- 上一稳定版本：`v0.1.1`
- 本次验收日期：2026-08-27
- 验收方式：实际 GitHub Release 资产

## 反馈记录

| 编号 | 环境 | 路径 | 结果 | 级别 | 状态 |
|---|---|---|---|---|---|
| AA-001 | macOS arm64 | 指定 `v0.1.1` 安装脚本 | 原脚本访问旧仓库 `voocel/ainovel-cli` 并返回 404；实际 Release 位于 `r6c/ainovel-cli` | P1 | 已修复 |
| AA-002 | macOS arm64 | 本地修复脚本安装 `v0.1.1` | checksum、安装、`--version`、`--help`、`deconstruct --help` 均通过 | - | complete |
| AA-003 | macOS arm64 | 远端 `v0.1.2` 安装脚本与正式资产回归 | 仓库地址、checksum、版本、帮助和 `deconstruct --help` 均通过 | - | complete |

## 验收边界

本记录不包含 Provider 凭证、API Key、完整模型响应或用户作品内容。

v0.1.2 远端回归已确认：

- macOS arm64 安装脚本与正式资产；
- Linux amd64/arm64 资产已由 Release/Checksum 验收；
- Windows amd64/arm64 zip 已由 Release 资产验收；
- Docker `linux/amd64` 与 `linux/arm64` 工作流成功；
- checksum 清单存在且选定资产校验通过；
- 稳定版 `--version`、`--help` 和 `deconstruct --help` 通过。

## 严重级别

- **P0**：数据丢失、正文覆盖、恢复死锁、内部状态泄漏。
- **P1**：核心流程无法完成、安装失败、升级失败。
- **P2**：文案、布局、非关键操作摩擦。


## v0.1.2 收口

- `v0.1.2` 已发布，CI、Release、Docker 均成功，正式资产 7 项。
- 远端安装脚本从 `r6c/ainovel-cli` 拉取并安装 Darwin arm64 资产，checksum、版本、`--help` 和 `deconstruct --help` 均通过。
- AA-001 安装链 P1 已关闭。
