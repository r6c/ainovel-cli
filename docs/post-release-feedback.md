# 发布后反馈

## 版本

- 当前稳定版本：`v0.1.2`（待远端工作流完成）
- 上一稳定版本：`v0.1.1`
- 本次验收日期：2026-08-27
- 验收方式：实际 GitHub Release 资产

## 反馈记录

| 编号 | 环境 | 路径 | 结果 | 级别 | 状态 |
|---|---|---|---|---|---|
| AA-001 | macOS arm64 | 指定 `v0.1.1` 安装脚本 | 原脚本访问旧仓库 `voocel/ainovel-cli` 并返回 404；实际 Release 位于 `r6c/ainovel-cli` | P1 | 已修复，待 `v0.1.2` 远端资产回归 |
| AA-002 | macOS arm64 | 本地修复脚本安装 `v0.1.1` | checksum、安装、`--version`、`--help`、`deconstruct --help` 均通过 | - | complete |
| AA-003 | 当前工作区 | 修复后安装脚本静态契约与 `v0.1.1` 资产回归 | 仓库地址、checksum、版本和帮助命令均通过；等待补丁版远端资产回归 | - | in_progress |

## 验收边界

本记录不包含 Provider 凭证、API Key、完整模型响应或用户作品内容。

正式补丁发布后需要再次确认：

- Linux amd64/arm64 安装资产；
- macOS 当前架构安装资产；
- Windows amd64/arm64 zip 可下载；
- Docker `linux/amd64` 与 `linux/arm64`；
- checksum 清单；
- 稳定版 `--version`、`--help` 和 `deconstruct --help`。

## 严重级别

- **P0**：数据丢失、正文覆盖、恢复死锁、内部状态泄漏。
- **P1**：核心流程无法完成、安装失败、升级失败。
- **P2**：文案、布局、非关键操作摩擦。
