package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildWriterPrompt_ByteIdenticalToPreSplit 是文风层验收标准 ①:
// 不放任何覆盖文件时,组装产物与拆分前的 writer.md 管线逐字节一致。
// golden 是拆分前 writer.md 的原始快照(testdata/writer-golden.md)。
func TestBuildWriterPrompt_ByteIdenticalToPreSplit(t *testing.T) {
	golden, err := os.ReadFile("testdata/writer-golden.md")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	protocol := mustRead(promptsFS, "prompts/writer.md")
	voice := mustRead(voiceFS, "voice.md")

	// 文件级:占位符回填 == 拆分前原文
	if got := strings.Replace(protocol, voicePlaceholder, strings.TrimSpace(voice), 1); got != string(golden) {
		t.Fatalf("占位符回填与拆分前不一致:\n--- 长度 golden=%d got=%d", len(golden), len(got))
	}

	// 管线级:新组装 == 旧管线(writer.md → simGuidance → style)
	const style = "## 某风格\n\n- 测试"
	old := WithSimulationGuidance(string(golden), "writer") + "\n\n" + style
	got := BuildWriterPrompt(WithSimulationGuidance(protocol, "writer"), voice, style)
	if got != old {
		t.Fatal("组装管线与拆分前不等价")
	}

	// 无风格追加时也等价
	if BuildWriterPrompt(WithSimulationGuidance(protocol, "writer"), voice, "") != WithSimulationGuidance(string(golden), "writer") {
		t.Fatal("无 style 时组装管线与拆分前不等价")
	}
}

// TestLoad_NoOverrides 零覆盖时 Voice/AntiAITone 与内置逐字节一致。
func TestLoad_NoOverrides(t *testing.T) {
	b := Load("default", LoadOptions{})
	if b.Voice != mustRead(voiceFS, "voice.md") {
		t.Fatal("无覆盖时 Voice 应与内置逐字节一致")
	}
	if b.References.AntiAITone != mustRead(referencesFS, "references/anti-ai-tone.md") {
		t.Fatal("无覆盖时 AntiAITone 应与内置逐字节一致")
	}
	if _, ok := b.Styles["default"]; !ok {
		t.Fatal("内置风格集应含 default")
	}
}

func TestWriterPromptDescribesReaderRevealPayloadDiscipline(t *testing.T) {
	writer := mustRead(promptsFS, "prompts/writer.md")
	for _, phrase := range []string{"`reveal_to_reader` 只能携带已有知识 ID", "不得携带 truth/character/belief", "未知知识 ID 不得自行创造"} {
		if !strings.Contains(writer, phrase) {
			t.Fatalf("Writer 提示缺少 reader reveal 载荷纪律 %q", phrase)
		}
	}
}

func TestKnowledgePromptsDescribeTruthAndLearningBoundaries(t *testing.T) {
	writer := mustRead(promptsFS, "prompts/writer.md")
	for _, phrase := range []string{"knowledge_boundaries", "establish", "believe", "learn", "reveal_to_reader", "越权知情", "读者已知", "错误信念", "纠正"} {
		if !strings.Contains(writer, phrase) {
			t.Fatalf("Writer 提示缺少知识边界纪律 %q", phrase)
		}
	}

	editor := mustRead(promptsFS, "prompts/editor.md")
	for _, phrase := range []string{"knowledge_boundaries", "越权知情", "错误信念", "已纠正", "提前泄底", "重复揭秘"} {
		if !strings.Contains(editor, phrase) {
			t.Fatalf("Editor 提示缺少知识一致性检查 %q", phrase)
		}
	}

	for name, prompt := range map[string]string{
		"revision": mustRead(promptsFS, "prompts/revision-analyze.md"),
		"import":   mustRead(promptsFS, "prompts/import-analyze.md"),
	} {
		for _, phrase := range []string{"establish", "believe", "learn", "reveal_to_reader", "正文明确", "怀疑"} {
			if !strings.Contains(prompt, phrase) {
				t.Fatalf("%s 提示缺少知识事实提取纪律 %q", name, phrase)
			}
		}
	}
}

func TestImportPromptExplainsKnowledgeMultiActionExtraction(t *testing.T) {
	prompt := mustRead(promptsFS, "prompts/import-analyze.md")
	for _, phrase := range []string{
		"同一客观 Truth",
		"多个 `knowledge_updates`",
		"正文发生顺序",
		"establish → learn → reveal_to_reader",
		"明确验证并接受",
		"完整答案告诉读者",
		"听见但不相信",
		"猜测",
		"部分兑现",
		"不等于 `learn`",
		"不等于 `reveal_to_reader`",
		"未经确认的说法不等于客观 Truth",
		"未决指控不等于作者事实",
	} {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("导入分析提示缺少认知动作多动作边界 %q", phrase)
		}
	}
}

func TestAntiAIToneReferenceUsesCalibratedNovelBoundaries(t *testing.T) {
	ref := mustRead(referencesFS, "references/anti-ai-tone.md")
	for _, phrase := range []string{
		"AI 来源判据", "一般审美问题", "目标风格优先", "信息守恒", "叙事功能优先",
		"段首零回指评论", "提示性冒号", "理想化职业人格喻体",
		"短句或短段本身", "问句本身", "比喻本身", "句内排比本身",
	} {
		if !strings.Contains(ref, phrase) {
			t.Fatalf("anti-ai-tone 缺少校准边界 %q", phrase)
		}
	}
	for _, overclaim := range []string{
		"三段式 / 排比三连", "每段长度、句式高度雷同", "明喻套句", "口语有停顿、省略、答非所问",
	} {
		if strings.Contains(ref, overclaim) {
			t.Fatalf("anti-ai-tone 仍保留未经功能判断的泛化 %q", overclaim)
		}
	}
}

func TestSimulationPromptsDescribeAbstractDeconstructionNotAuthorImitation(t *testing.T) {
	for name, prompt := range map[string]string{
		"source": mustRead(promptsFS, "prompts/simulation-source.md"),
		"merge":  mustRead(promptsFS, "prompts/simulation-merge.md"),
	} {
		for _, phrase := range []string{"拆文方法画像", "不得模仿具体作者", "不得输出连续原文表达"} {
			if !strings.Contains(prompt, phrase) {
				t.Fatalf("%s prompt missing compliance boundary %q", name, phrase)
			}
		}
		if strings.Contains(prompt, "仿写画像") {
			t.Fatalf("%s prompt still uses imitation-facing name", name)
		}
	}
}

func TestLoadIncludesBoundedFanqieRubricAndOverrides(t *testing.T) {
	builtin := Load("default", LoadOptions{}).References.FanqieRubric
	for _, phrase := range []string{
		"官方可核事实", "产品软评价", "5—30字", "连续翻页", "不得机械扣分", "黄金三章", "爽点数量", "留存算法",
		"2026-08-25", "https://fanqienovel.com/docs/8231", "https://fanqienovel.com/writer/zone/article/7170705662714839070",
	} {
		if !strings.Contains(builtin, phrase) {
			t.Fatalf("番茄 rubric 缺少边界声明 %q", phrase)
		}
	}

	home := t.TempDir()
	book := t.TempDir()
	writeFile(t, filepath.Join(home, "platforms", "fanqie.md"), "全局番茄补充")
	writeFile(t, filepath.Join(book, "platforms", "fanqie.md"), "本书番茄补充")
	got := Load("default", LoadOptions{HomeStyleDir: home, BookStyleDir: book}).References.FanqieRubric
	gi, bi := strings.Index(got, "全局番茄补充"), strings.Index(got, "本书番茄补充")
	if gi < 0 || bi < 0 || gi > bi {
		t.Fatalf("番茄 rubric 覆盖顺序错误: global=%d book=%d\n%s", gi, bi, got)
	}
}

func TestEditorPromptExplainsReviewScopeSelection(t *testing.T) {
	editor := mustRead(promptsFS, "prompts/editor.md")
	for _, phrase := range []string{
		"多章返工改用 `scope=global`",
		"单章返工改用 `scope=chapter`",
		"只有任务明确给出弧的起止章节和弧末章节",
	} {
		if !strings.Contains(editor, phrase) {
			t.Fatalf("Editor 提示缺少评审范围规则 %q", phrase)
		}
	}
}

func TestEditorPromptKeepsPlatformRubricInsideExistingDimensions(t *testing.T) {
	editor := mustRead(promptsFS, "prompts/editor.md")
	for _, phrase := range []string{"platform_rubric", "官方可核事实", "产品软评价", "现有七维", "不得新增平台维度", "不得单独决定 verdict"} {
		if !strings.Contains(editor, phrase) {
			t.Fatalf("Editor 提示缺少平台 rubric 边界 %q", phrase)
		}
	}
	for _, forbidden := range []string{"platform_fit", "fanqie_score"} {
		if strings.Contains(editor, forbidden) {
			t.Fatalf("Editor 提示不应定义平台评分状态 %q", forbidden)
		}
	}
}

func TestArchitectPromptsRecoverStaleFoundationFingerprint(t *testing.T) {
	for name, prompt := range map[string]string{
		"architect_short": mustRead(promptsFS, "prompts/architect-short.md"),
		"architect_long":  mustRead(promptsFS, "prompts/architect-long.md"),
	} {
		for _, phrase := range []string{"stale_foundation_fingerprint", "current_fingerprint", "不要重复提交旧 fingerprint"} {
			if !strings.Contains(prompt, phrase) {
				t.Fatalf("%s 提示缺少 fingerprint 恢复纪律 %q", name, phrase)
			}
		}
	}
}

func TestWriterAndArchitectPromptsTreatPlatformRubricAsConditionalSoftReference(t *testing.T) {
	for name, prompt := range map[string]string{
		"writer":          mustRead(promptsFS, "prompts/writer.md"),
		"architect_short": mustRead(promptsFS, "prompts/architect-short.md"),
		"architect_long":  mustRead(promptsFS, "prompts/architect-long.md"),
	} {
		for _, phrase := range []string{"platform_rubric", "存在时", "软参考", "用户偏好", "章节合同", "人物逻辑", "不得机械"} {
			if !strings.Contains(prompt, phrase) {
				t.Fatalf("%s 提示缺少平台软目标边界 %q", name, phrase)
			}
		}
	}
}

func TestPromptsConsumeStructuredChapterTargetWithoutEncouragingPadding(t *testing.T) {
	writer := mustRead(promptsFS, "prompts/writer.md")
	for _, phrase := range []string{"structured.chapter_target_chars", "120%", "不得为达到下限注水", "提交前"} {
		if !strings.Contains(writer, phrase) {
			t.Fatalf("Writer 提示缺少篇幅目标纪律 %q", phrase)
		}
	}

	for name, prompt := range map[string]string{
		"architect_short": mustRead(promptsFS, "prompts/architect-short.md"),
		"architect_long":  mustRead(promptsFS, "prompts/architect-long.md"),
	} {
		for _, phrase := range []string{"structured.chapter_target_chars", "单章承载量", "规划规模相容", "不得宣称最终正文"} {
			if !strings.Contains(prompt, phrase) {
				t.Fatalf("%s 提示缺少篇幅规划纪律 %q", name, phrase)
			}
		}
	}

	editor := mustRead(promptsFS, "prompts/editor.md")
	for _, phrase := range []string{"structured.chapter_target_chars", "120%", "偏短", "pacing", "不得要求注水"} {
		if !strings.Contains(editor, phrase) {
			t.Fatalf("Editor 提示缺少篇幅审阅边界 %q", phrase)
		}
	}
}

func TestEditorPromptConsumesDuplicateParagraphFacts(t *testing.T) {
	editor := mustRead(promptsFS, "prompts/editor.md")
	for _, phrase := range []string{"duplicate_paragraph", "rule_violations", "有意复沓", "不新增评审维度"} {
		if !strings.Contains(editor, phrase) {
			t.Fatalf("Editor 提示缺少重复段落机械事实纪律 %q", phrase)
		}
	}
}

func TestForeshadowPromptsDescribeLifecycleActions(t *testing.T) {
	writer := mustRead(promptsFS, "prompts/writer.md")
	for _, phrase := range []string{"reinforce", "partial_payoff", "部分兑现", "resolve"} {
		if !strings.Contains(writer, phrase) {
			t.Fatalf("Writer 提示缺少伏笔生命周期指引 %q", phrase)
		}
	}

	importAnalyze := mustRead(promptsFS, "prompts/import-analyze.md")
	for _, action := range []string{"plant", "advance", "reinforce", "partial_payoff", "resolve"} {
		if !strings.Contains(importAnalyze, action) {
			t.Fatalf("导入分析提示缺少伏笔动作 %q", action)
		}
	}
}

func TestInterventionPromptsKeepScopeContract(t *testing.T) {
	prompts := loadPrompts()
	for _, phrase := range []string{"上下文不等于修改授权", "最小充分范围", "分析范围不等于修改范围"} {
		if !strings.Contains(prompts.ArbiterIntervention, phrase) {
			t.Fatalf("Arbiter 干预提示缺少范围契约 %q", phrase)
		}
	}
	for _, phrase := range []string{"用户原始干预", "分析范围不等于修改范围", "最小充分章节集合"} {
		if !strings.Contains(prompts.Editor, phrase) {
			t.Fatalf("Editor 提示缺少范围契约 %q", phrase)
		}
	}
}

func TestStructuredArbiterPromptsContainOnlySemantics(t *testing.T) {
	prompts := loadPrompts()
	for name, prompt := range map[string]string{
		"plan_start": prompts.ArbiterPlanStart,
		"failure":    prompts.ArbiterFailure,
	} {
		for _, duplicate := range []string{"```json", "不要 Markdown", "输出一个 JSON 对象"} {
			if strings.Contains(prompt, duplicate) {
				t.Fatalf("%s 提示词仍重复维护输出格式 %q", name, duplicate)
			}
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestLoad_ThreeTierAppendAndReplace 覆盖三层优先级与逐资产语义(验收标准 ②)。
func TestLoad_ThreeTierAppendAndReplace(t *testing.T) {
	home := t.TempDir()
	book := t.TempDir()
	opts := LoadOptions{HomeStyleDir: home, BookStyleDir: book}

	// voice / anti-ai-tone:追加语义,全局在前、本书在后,带边界标记
	writeFile(t, filepath.Join(home, "voice.md"), "全局:少用成语")
	writeFile(t, filepath.Join(book, "voice.md"), "本书:多写对话")
	writeFile(t, filepath.Join(book, "anti-ai-tone.md"), "本书判据:禁排比")

	// styles:同名整文件替换 + 新名新增;非法名忽略
	writeFile(t, filepath.Join(home, "styles", "fantasy.md"), "全局改写的奇幻")
	writeFile(t, filepath.Join(book, "styles", "xianxia.md"), "自定义仙侠")
	writeFile(t, filepath.Join(book, "styles", "Bad Name!.md"), "非法")

	// 题材参考:同名整文件替换,本书 > 全局
	writeFile(t, filepath.Join(home, "genres", "fantasy", "style-references.md"), "全局参考")
	writeFile(t, filepath.Join(book, "genres", "fantasy", "style-references.md"), "本书参考")

	b := Load("fantasy", opts)

	builtinVoice := mustRead(voiceFS, "voice.md")
	if !strings.HasPrefix(b.Voice, builtinVoice) {
		t.Fatal("追加语义必须保留内置原文为前缀")
	}
	giIdx := strings.Index(b.Voice, "## 用户全局文风覆盖")
	bkIdx := strings.Index(b.Voice, "## 本书文风覆盖")
	if giIdx < 0 || bkIdx < 0 || giIdx > bkIdx {
		t.Fatalf("追加段顺序错误:global=%d book=%d", giIdx, bkIdx)
	}
	if !strings.Contains(b.Voice, "全局:少用成语") || !strings.Contains(b.Voice, "本书:多写对话") {
		t.Fatal("覆盖内容缺失")
	}
	if !strings.Contains(b.References.AntiAITone, "本书判据:禁排比") {
		t.Fatal("anti-ai-tone 本书追加缺失")
	}

	if b.Styles["fantasy"] != "全局改写的奇幻" {
		t.Fatal("styles 同名应整文件替换")
	}
	if b.Styles["xianxia"] != "自定义仙侠" {
		t.Fatal("新增自定义风格应即放即用")
	}
	if _, ok := b.Styles["Bad Name!"]; ok {
		t.Fatal("非法风格名必须被忽略")
	}

	if b.References.StyleReference != "本书参考" {
		t.Fatalf("题材参考应为本书覆盖优先,got %q", b.References.StyleReference)
	}
}

// TestLoad_BookOverridesHomeOnStyles 本书 styles 覆盖全局同名。
func TestLoad_BookOverridesHomeOnStyles(t *testing.T) {
	home := t.TempDir()
	book := t.TempDir()
	writeFile(t, filepath.Join(home, "styles", "romance.md"), "全局版")
	writeFile(t, filepath.Join(book, "styles", "romance.md"), "本书版")
	b := Load("default", LoadOptions{HomeStyleDir: home, BookStyleDir: book})
	if b.Styles["romance"] != "本书版" {
		t.Fatalf("本书应覆盖全局,got %q", b.Styles["romance"])
	}
}

// TestOverrideVoice_SharesAssemblyPath eval 的 voice A/B 与生产同组装路径(验收标准 ④)。
func TestOverrideVoice_SharesAssemblyPath(t *testing.T) {
	b := Load("default", LoadOptions{})
	b.OverrideVoice("## 实验文风\n\n- 一句话")
	got := BuildWriterPrompt(b.Prompts.Writer, b.Voice, "")
	if !strings.Contains(got, "## 实验文风") {
		t.Fatal("OverrideVoice 未生效")
	}
	if strings.Contains(got, voicePlaceholder) {
		t.Fatal("占位符必须被消耗")
	}
	// 协议部分不受 voice 覆盖影响
	if !strings.Contains(got, "## 执行协议") {
		t.Fatal("协议模板不得被 voice 覆盖破坏")
	}
}
