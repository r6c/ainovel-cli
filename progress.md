# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-25
- 基线：`a3d24a1 文档：从产品路线移除扫榜功能`
- 当前里程碑：M——Linux 与无头环境兼容性验收
- 当前阶段：阶段 106—111 全部完成
- 公共接缝：顶层 CLI、deconstruct 子命令、通知 Adapter、Linux CI、Docker ENTRYPOINT

## 基线盘点

1. `.github/workflows/ci.yml` 已在 Ubuntu/Windows 跑 format、vet、全量测试；Ubuntu 另跑关键 race。
2. `.goreleaser.yml` 已覆盖 linux/darwin/windows × amd64/arm64，且 `CGO_ENABLED=0`。
3. Dockerfile 已使用 TARGETOS/TARGETARCH 和 Alpine 运行时；docker workflow 发布 linux/amd64、linux/arm64。
4. Linux 通知缺少 `notify-send` 时已降级为日志；通知本身异步且不介入 Host 控制流。
5. 顶层 `--help` 当前没有专门分支，会进入常规配置/首次引导路径；这是首个真实产品缺口。
6. CI 尚无显式 Linux 双架构构建门禁、原生帮助冒烟或 Docker 帮助冒烟。

## 成功标准

- 顶层帮助无需配置、TTY、DISPLAY 或模型。
- Linux amd64/arm64 都能静态跨编译。
- Ubuntu 原生二进制可执行帮助命令。
- Docker 镜像可在无挂载、无 TTY 下执行帮助命令。
- 通知不可用不影响主流程。
- 不新增 GUI、浏览器或网络抓取依赖。

## 阶段 106 完成

公开测试先证明 `--help/-h/help` 均未被顶层分发处理。最小实现只扩展 `runSubcommand`，三种形式均在配置、首次引导、TTY 和模型之前写 stdout 并返回 0；普通 flags 与 deconstruct 分发保持不变。实际构建后二进制已在空 HOME、空 DISPLAY 下执行帮助成功。

## 阶段 107 完成

现有 CI 已增加 `CGO_ENABLED=0` 的 Linux amd64/arm64 双架构构建门禁，产物写入 RUNNER_TEMP。本机真实跨编译后，`file` 分别确认静态链接 x86-64 与 aarch64 ELF；未新增 workflow 或构建框架。

## 阶段 108 完成

Ubuntu CI 会以空 HOME、空 DISPLAY 原生执行构建出的 amd64 `--help` 与 `deconstruct --help`。新增 Linux-only 通知 Adapter 测试：PATH 中没有 `notify-send` 时返回 nil，只降级日志；当前 macOS 合理 skip，Ubuntu CI 真实执行。相关 Go 包全绿。

## 阶段 109

复用现有 Dockerfile，在 Ubuntu CI 构建本地镜像，并以 `--network none`、无挂载、无 TTY 执行两条帮助命令。发布 Docker workflow 保持原状。

本机 Docker daemon 可用，但首次真实构建在拉取 `golang:1.25` 基础镜像时超过宿主工具时限，未形成代码失败证据。没有原样重复长拉取；随后用 Alpine 容器真实运行 arm64 静态二进制的两条帮助命令，并执行 Linux-only 通知测试，均在 `--network none` 下通过。完整 Dockerfile 构建与 ENTRYPOINT 冒烟由 Ubuntu CI 作为正式门禁。

## 阶段 110 完成

README/CONTEXT 已记录无配置帮助、Linux 双架构、Docker 健康检查和通知降级边界。扫描确认生产代码没有 Chrome/CDP、浏览器登录态、GUI 动态库或绝对临时目录；macOS `osascript` 和 Linux `notify-send` 仅存在于通知 Adapter。

## 阶段 111 完成

以下门禁全部通过：关键包、全量测试、`go vet`、关键 race、gofmt/diff、Markdown 链接、CI YAML 解析、CI 关键步骤审计，以及 Linux amd64/arm64 静态跨编译。范围扫描仅在测试文件发现绝对 `/tmp`，生产代码无 Chrome/CDP、浏览器或绝对临时目录依赖。

本地 Alpine 容器在 `--network none` 下真实执行 arm64 顶层帮助、deconstruct 帮助和 Linux 通知降级测试。完整 Dockerfile 首次基础镜像拉取受宿主工具时限影响，未伪报成功；Ubuntu CI 已配置完整构建和 ENTRYPOINT 冒烟。

提交信息：`测试：加固 Linux 与无头环境兼容性`。
