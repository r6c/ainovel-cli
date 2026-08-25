package rules

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Lint 内置产品底线检查：扫描正文中的机制残留，与用户规则无关，commit 时始终执行。
// 与 Check 同契约——只生成事实，不在 Lint 内裁决。正文接纳 adapter 可按 provenance
// 将特定事实作为前置条件：模型生成正文拒绝 markdown_residue，Import 原文只记录事实。
//
// 当前三类（全部来自真实长跑产物的实证缺陷）：
//   - markdown_residue：正文残留 ** 加粗、首行之外的 # 标题行（导出 txt 会裸露符号）
//   - non_cjk_fragments：连续拉丁字母片段（模型语言混杂，如中文正文裸混 "pattern"）
//   - duplicate_paragraph：同章完全重复的长正文段落；仅报事实，不判断是否有意复沓
func Lint(text string) []Violation {
	var vs []Violation
	vs = appendMarkdownResidue(vs, text)
	vs = appendNonCJKFragments(vs, text)
	vs = appendDuplicateParagraphs(vs, text)
	return vs
}

func appendMarkdownResidue(vs []Violation, text string) []Violation {
	if n := strings.Count(text, "**"); n > 0 {
		vs = append(vs, Violation{
			Rule:     "markdown_residue",
			Target:   "**",
			Actual:   n,
			Severity: SeverityWarning,
		})
	}
	headings := 0
	seenContent := false
	for line := range strings.SplitSeq(text, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		// 第一个非空行的 # 标题是章文件的合法格式（不按行号写死，容忍前导空行）
		first := !seenContent
		seenContent = true
		if !first && strings.HasPrefix(t, "#") {
			headings++
		}
	}
	if headings > 0 {
		vs = append(vs, Violation{
			Rule:     "markdown_residue",
			Target:   "#",
			Actual:   headings,
			Severity: SeverityWarning,
		})
	}
	return vs
}

const (
	duplicateParagraphMinRunes    = 24
	duplicateParagraphTargetRunes = 48
)

func duplicateParagraphTarget(paragraph string) string {
	runes := []rune(paragraph)
	if len(runes) <= duplicateParagraphTargetRunes {
		return paragraph
	}
	return string(runes[:duplicateParagraphTargetRunes]) + "…"
}

func appendDuplicateParagraphs(vs []Violation, text string) []Violation {
	counts := make(map[string]int)
	var order []string
	for line := range strings.SplitSeq(text, "\n") {
		paragraph := strings.TrimSpace(line)
		if paragraph == "" || strings.HasPrefix(paragraph, "#") || utf8.RuneCountInString(paragraph) < duplicateParagraphMinRunes {
			continue
		}
		if counts[paragraph] == 0 {
			order = append(order, paragraph)
		}
		counts[paragraph]++
	}
	for _, paragraph := range order {
		if counts[paragraph] < 2 {
			continue
		}
		vs = append(vs, Violation{
			Rule:     "duplicate_paragraph",
			Target:   duplicateParagraphTarget(paragraph),
			Limit:    1,
			Actual:   counts[paragraph],
			Severity: SeverityWarning,
		})
	}
	return vs
}

var latinFragmentRe = regexp.MustCompile(`[A-Za-z]{2,}`)

// appendNonCJKFragments 报告拉丁字母片段的总次数与去重示例。
// 现代题材的合法英文（品牌名/缩写）也会命中——warning 级事实，由评审按题材裁定。
func appendNonCJKFragments(vs []Violation, text string) []Violation {
	matches := latinFragmentRe.FindAllString(text, -1)
	if len(matches) == 0 {
		return vs
	}
	seen := make(map[string]struct{})
	var examples []string
	for _, m := range matches {
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		if len(examples) < 3 {
			examples = append(examples, m)
		}
	}
	return append(vs, Violation{
		Rule:     "non_cjk_fragments",
		Target:   strings.Join(examples, "、"),
		Actual:   len(matches),
		Severity: SeverityWarning,
	})
}
