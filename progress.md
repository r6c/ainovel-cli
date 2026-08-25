# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-25
- 基线：`6b4050f 功能：试点番茄平台创作评审参考`
- 当前里程碑：K——现有 cocreate 阶段化访谈
- 当前阶段：阶段 91—98 全部完成
- 公共接缝：`startup.CoCreateSession`、Host 共创 XML 协议、TUI cocreate 行为

## 基线盘点

- 现有启动模式只有 quick / cocreate；不需要第三种 interview 模式。
- 冷启动和运行中阶段共创共用四段 XML 协议：reply / draft / ready / suggestions。
- `CoCreateSession.CanStart()` 当前只检查 Draft 非空，忽略模型 Ready；第一轮有 Draft 就可 Ctrl+S。
- 协议没有阶段字段，模型可直接宣告 ready，代码无法约束核心定位、深度定制、标题与确认顺序。
- `startup.CoCreateSession` 是非 UI 状态的最佳接缝；TUI 只应展示阶段和调用现有启动/恢复入口。
- 运行中阶段共创已有 story state 摘要、Pause/Resume/Cancel 和独占性测试，本批保持不动。

## 阶段定义

```text
core           题材/主角/核心冲突/规模倾向
customization  世界观、视角、基调、受众、感情线等关键定制
title          书名与无剧透简介候选，用户选择或明确授权
confirmation   汇总完整创作指令并要求用户确认
ready          已确认，可 Ctrl+S 走现有 StartPrepared
```

## 兼容决策

- 阶段状态只存在于冷启动会话内，不落正式 Store。
- 模型回报阶段只能保持或推进一格，不能跳级/回退。
- 回复缺 stage 或非法 stage 时保持当前阶段。
- 阶段共创使用无冷启动阶段门禁的会话构造器，保留现有 Ctrl+S 行为。
- 最终输出仍是一段 Draft Prompt，不直接写 Book/Foundation。

### 阶段 98 完成

全量门禁初次通过后仍发现占位符、标题子串匹配和共享 ready 文案三个协议精度问题；均经新增失败测试修复。最终 startup/host/TUI、全量测试、vet、race、diff 与范围扫描全部通过。

### 阶段 97 范围审计

确认启动模式仍只有 quick/cocreate，Flow/Domain/Store 无 Interview 状态。补真实 Ctrl+S 未 ready 不启动契约，并修正 Host 过期“四段式”注释。

### 阶段 96 完成

新增 TUI 请求失败保留阶段/Draft、取消恢复初始输入的契约；共创日志记录模型回报的 parsed_stage，仅用于诊断，不作为恢复事实。半成品仍不写 Book/Foundation。

### 阶段 96 兼容审计

冷启动回复缺 Draft 时，真实红灯显示 Session 投影保留旧 draftPrompt，但规范化 History 写空 draft。现先应用非空 Prompt 覆盖，再以最终保留 Draft 计算 Ready 并构造 History；阶段共创继续保留原始 Raw。

## 错误记录

- TUI 探索猜测不存在的 `model_test.go`，并使用未转义 `{` 的正则导致搜索失败；不重复。阶段可见性直接测试同包渲染函数，Ctrl+S 复用 Session CanStart 门禁。

- 读取猜测的 `internal/host/cocreate_test.go` 返回 ENOENT；Host 共创测试分散在其它文件，后续通过字面搜索真实接缝，不重复猜文件名。

## 下一步

### 阶段 92 完成

完整 XML 中的 `<stage>` 原先被 Host 丢弃。现支持五阶段值域解析；缺失/非法值返回空，由 Session 保守保持。流式预览与 TUI 历史不显示 stage；阶段共创旧四段响应继续兼容，Host 全包通过。

### 阶段 93 首个红灯

冷启动 Prompt 完全缺少阶段顺序和最低覆盖。首次编辑因“每一轮回复”在冷启动/阶段共创中各出现一次而被原子拒绝；第二次又因 `</draft>` 重复而原子拒绝，两次均未部分修改。停止批量编辑，改用三个独立唯一片段分别修改冷启动前言、冷启动 draft 尾部和共享输出规范。首次编译发现新增文字在 Go raw string 内使用反引号，提前结束字符串；改用普通文本后，Prompt 契约又发现共享尾部仍向阶段共创暴露 `<stage>`。现将共享尾部收窄为“使用本模式前文标签”，由冷启动/阶段共创各自示例定义五/四标签。

### 阶段 95 首个红灯

ready 阶段只要 Draft 非空就会放行。现增加四个规范二级标题的形状检查；阶段共创不应用，BuildPrompt 仍原样返回已确认 Draft。接入后阶段顺序测试的“最终创作指令”旧夹具不再合法，已同步为完整四节指令。

### 阶段 96 审计

代码拒绝跳级后，历史仍保存模型原始跳级 stage/ready，真实红灯确认下一轮上下文会分叉。现先裁决阶段/完整性，再把接受后的五段协议规范化写入冷启动 History；阶段共创仍保存原始 Raw。Prompt 契约随后红灯：模型仍被要求写“主题/关键要素/待澄清”旧章节，可能永远过不了四标题门禁。现同步 `<draft>` 为固定四节；早期未完成节可标待确认，confirmation/ready 禁止占位。

### 阶段 95 完成

最终 Draft 使用四个规范章节：`## 核心定位`、`## 深度定制`、`## 书名与简介`、`## 规划确认`。Session 只校验章节存在，不理解正文语义；语义仍由模型与用户确认负责。

### 阶段 94 首个红灯

冷启动面板只有“继续对话中”，缺少阶段信息。最小增加 TUI 显示映射，阶段来源仍是 Session；运行中阶段共创 Stage 为空，不显示冷启动进度。

### 阶段 94 完成

TUI 只增加冷启动阶段可见性：核心定位 1/5、深度定制 2/5、标题简介 3/5、规划确认 4/5、已确认 5/5。运行中阶段共创不显示该进度；Ctrl+S 继续调用 Session CanStart。

### 阶段 93 完成

冷启动 `<stage>` 定义为“下一轮当前阶段”：本轮信息不足则保持，满足最低覆盖后最多前进一格。这样与 Session 单步裁决一致，也避免“本轮完成阶段/当前阶段”歧义。阶段共创不继承该标签。

### 阶段 91 完成

首个红灯准确：`CoCreateReply` 无 Stage，Session 无阶段访问器/门禁，也没有阶段共创专用构造器。现由 Session 限制冷启动每次最多推进一格；缺失/非法/跳级保持当前阶段，只有 ready + draft + 模型 ready 才可启动。阶段共创使用专用构造器，保留有 Draft 即可应用。
