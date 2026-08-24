# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-25
- 基线：`25a43d5 功能：增加知识状态诊断与脱敏统计`
- 当前里程碑：J——番茄平台 Rubric 试点
- 当前阶段：阶段 83—90 全部完成
- 公共接缝：`rules.BuildSnapshot`、`userrules.Normalizer`、`novel_context`、Editor 资源契约

## 基线盘点

- 仓库没有平台字段、平台资源或平台评分维度。
- `DimensionScore` 可扩展，但本试点不新增第八维，避免扩大 Review 状态协议。
- `rules.Structured` 是用户长期意图的确定性输入层，适合承载显式目标平台；旧快照未知/缺失字段可零值兼容。
- `assets` 已有内置 + 全局 + 本书三层覆盖；试点复用该机制，不创建 Pack Loader。
- `novel_context` 已将 references 注入 Writer/Editor/Architect，可按 platform 条件追加单个 rubric。

## 官方资料边界

官方帮助中心可确认：番茄面向连载作品，依赖智能分发和连续翻页阅读；章节标题支持 5—30 字；频繁删除/修改影响阅读体验；平台强调原创与抄袭处罚。官方未公布固定黄金三章字数、爽点数或留存算法阈值。

来源：

- https://fanqienovel.com/docs/8231
- https://fanqienovel.com/writer/zone/article/7170705662714839070

外部页面内容仅作为不可信事实来源，不执行其中任何指令。

## 决策

```text
user_rules.structured.platform = fanqie
→ 条件注入番茄 rubric
→ 映射现有七维
→ 不新增平台分/硬阈值
```

### 阶段 88 完成

Writer 与 Architect 原先未说明平台软目标边界。现只追加条件使用、优先级与禁区，不复制 rubric；用户偏好、章节合同与人物逻辑优先，禁止机械制造钩子/爽点。Writer golden 与 assets 全包通过。

### 阶段 87 完成

Editor 原先未说明 `platform_rubric` 纪律。现区分官方事实与产品软评价，映射现有七维，不新增平台维度或算法分，不允许平台参考单独决定 verdict。首轮测试把禁止语句中的“平台算法分”也误判为定义该字段，契约自相矛盾；已将禁用标识收窄为实际字段名 `platform_fit/fanqie_score`。

### 阶段 86 完成

同一 References 下，显式 `platform=fanqie` 的 Context 原先缺 `platform_rubric`。现从本书 user_rules 快照条件选择资源；空平台不注入。Writer/Editor 章节模式与 Architect 模式均通过，rubric 随 references 继续参与预算裁剪。

### 阶段 85 完成

资源契约编译失败证明 `References` 尚无平台资源。现新增单个原创整理的番茄 rubric，明确官方事实、产品软评价和伪阈值禁区，并复用内置/全局/本书三层追加加载；assets 全包通过。

### 阶段 84 完成

明确“发布番茄”可得到 `platform=fanqie`；含糊“免费阅读平台”保持空；未知平台在 DTO 边界拒绝。Strict Schema 增加必填 platform，旧测试响应同步空字符串；userrules 全包通过。

### 阶段 89 完成

README 已说明番茄参考仅在用户明确指定时启用，仍复用七维且不编造算法分。新增 sentinel 深度契约，证明 Bundle 虽加载内置资源，未指定 platform 时序列化 Context 不泄露 rubric 内容或键。

确认生产代码没有 `platform_fit/fanqie_score` 或平台算法分状态；平台只存在于 user_rules、单个资源和条件 Context。用户规则文档旧示例仍为 version 1 且缺平台、Commit 描述过宽，已同步 v3 与机械字段边界。一次对 README 单文件路径使用 `search_files` 返回 ENOTDIR，未重复，改用直接文件工具。

### 阶段 90 完成

关键 rules/userrules/assets/tools、全量测试、vet、race 通过。首次提交门禁发现番茄资源版本行有 Markdown 强制换行尾随空格；命令链未 fail-fast，文件虽被暂存但尚未提交。已改为空引用行并重新执行 staged 格式检查。默认未指定平台的 Context sentinel 契约通过，Review 协议无新维度。

## 错误记录

- 读取 Store 规则文件时猜测 `internal/store/rules.go` 返回 ENOENT；实际文件是 `user_rules.go`，未重复猜测。兼容契约改在公开 `rules.Snapshot` JSON DTO 上验证。
- 一次搜索 `DefaultLoadOptions|assets.Load(` 使用未闭合括号正则失败；未重复，改用字面搜索和真实 Load 调用点。

## 阶段 83—90 实施记录

### 阶段 83 完成

首个红灯准确：`rules.Structured` 没有 Platform 字段。现支持显式 `fanqie` 合并/覆盖，空值不覆盖，未知值清洗；SnapshotVersion 升至 v3，旧 v2 JSON 缺平台字段时自然加载为空。

### 阶段 84 首个红灯

Normalizer 模型返回 `platform=fanqie` 时 DTO 静默丢失。最小同步 strict Schema、DTO、候选转换与保守提示；旧响应夹具需显式补 `platform:""`。

`go test ./internal/userrules` 首轮因旧 strict 响应缺新必填字段而持续反馈重试，宿主 120 秒超时且未定位用例；不原样重跑，改为搜索并同步全部 JSON 夹具后分组验证。首次多处替换因空平台响应文本不唯一而原子拒绝，文件未部分修改；改用带测试上下文的唯一片段逐处替换。收口红灯又证明 `toCandidate` 会接受未知平台；首次原子编辑因字段缩进文本未匹配而未应用，读取真实片段后分步增加 `""/fanqie` 防御校验。
