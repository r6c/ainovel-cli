# ainovel-cli 当前进度

## 当前会话

- 日期：2026-08-24
- 基线：`2f6768b 文档：归档演进计划并补充领域词汇表`
- 当前里程碑：H——确定性重复段落 Prose Lint
- 当前阶段：阶段 69—75 全部完成
- 公共测试接缝：`rules.Lint(text) []Violation`

## 基线盘点

- `rules.Lint` 当前只检测 `markdown_residue` 与 `non_cjk_fragments`。
- `CommitChapterTool.checkRules` 会把 `rules.Lint` 结果写入提交输出及 `meta/rule_violations.jsonl`。
- `revision.Projector` 会对接纳正文重新执行 `rules.Lint`，无需新增第二条管线。
- `Violation` 只陈述事实，不阻断提交；Editor 经现有 `rule_violations` 语义评审。

## 重复段落第一版边界

```text
段落 = TrimSpace 后的非空、非 # 标题单行
完全相同
长度 >= 24 Unicode 字符
出现次数 >= 2
severity = warning
limit = 1
actual = 出现次数
```

短对话、拟声词、标题、近似改写不报；Target 必须截断，不能复制整段正文。

## 错误记录

- 搜索 Markdown 中双换行时，`search_files` 底层 rg 不接受字面换行；未重复该方式，改用正文测试夹具与写作参考确认项目按行分段惯例。

## 实施记录

### 阶段 69 完成

首个红灯准确：重复长段落原先返回空列表。现已按首次出现顺序统计 TrimSpace 后完全相同、长度至少 24 个 Unicode 字符的非标题正文行，输出 warning、limit=1、actual=次数；既有 Lint 测试通过。

### 阶段 70 完成

短对话、重复标题和只差一个标点的近似段落均不报；TrimSpace 归一化首尾空白，空行和 CRLF 不影响识别，三个副本准确报告 actual=3。所有契约现有实现直接满足。

### 阶段 71 完成

160 字重复段落曾完整复制到 Target；现仅输出前 48 个 Unicode 字符 + `…`，完整段落仍用于去重。多个重复组按首次出现顺序稳定输出，Violation 注释已同步 `duplicate_paragraph` warning。

### 阶段 72 集成进度

- 首轮 Commit 测试停在 `phase=init`，补齐合法 `PhaseWriting` 后通过；重复段落 warning 出现在返回 JSON 和持久化规则事实中，提交不阻断。
- Revision Projector 测试集中在 `revision_test.go`；读取不存在的 `projector_test.go` 返回 ENOENT，已停止猜测并改用真实文件。
- Commit 与 Projector 公共集成均通过：提交返回并持久化重复段落 warning；Projector 会刷新旧违规事实而非追加。
- Editor 资源契约红灯准确：原提示未识别 `duplicate_paragraph`。最小同步到现有 aesthetic 映射，要求区分有意复沓和复制退化，不新增评审维度。

### 阶段 74—75 完成

- 23 个 Unicode 字符不报，24 个字符开始报告；只 TrimSpace，不归一化内部空白。
- 范围扫描确认没有 similarity/Levenshtein/Jaccard/fuzzy 或跨章状态。
- 关键 rules/tools/revision/assets 测试通过；`go test ./... -timeout=5m`、`go vet ./...`、`git diff --check` 通过。
- 一次对 README 单文件路径调用 `search_files` 返回 ENOTDIR；未重复，最终审阅改用 Git diff 与直接文件工具。

## 最终验证

```text
go test ./internal/rules -count=1
go test ./internal/tools -run 'CommitChapterPersistsDuplicateParagraph|RuleViolation' -count=1 -timeout=5m
go test ./internal/revision -run 'ProjectorRefreshesDuplicateParagraph|RuleViolation' -count=1
go test ./assets -count=1
go test ./... -timeout=5m
go vet ./...
git diff --check
```

全部通过。里程碑 H 完成；下一候选为 Knowledge 最小诊断与导出投影。
