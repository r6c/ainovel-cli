package rules

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestLint_CleanText(t *testing.T) {
	if vs := Lint("# 第一章 风起\n他迈步向前。\n夜色渐深。"); len(vs) != 0 {
		t.Errorf("clean text should pass: %+v", vs)
	}
}

func TestLint_MarkdownResidue(t *testing.T) {
	text := "# 第一章\n这是**重点**内容。\n## 小标题\n正文。"
	vs := Lint(text)
	bold := findViolation(vs, "markdown_residue", "**")
	if bold == nil || bold.Actual != 2 {
		t.Errorf("expected ** residue x2: %+v", vs)
	}
	heading := findViolation(vs, "markdown_residue", "#")
	if heading == nil || heading.Actual != 1 {
		t.Errorf("expected 1 heading beyond first line: %+v", vs)
	}
}

func TestLint_DuplicateLongParagraph(t *testing.T) {
	paragraph := "雨水沿着破旧窗棂缓慢滑落，林墨站在黑暗里听见远处钟声再次响起。"
	text := "# 第一章\n" + paragraph + "\n他推开房门，走进没有灯火的长廊。\n" + paragraph

	vs := Lint(text)
	v := findViolation(vs, "duplicate_paragraph", paragraph)
	if v == nil {
		t.Fatalf("expected duplicate_paragraph violation: %+v", vs)
	}
	if v.Limit != 1 || v.Actual != 2 {
		t.Fatalf("duplicate paragraph counts: limit=%v actual=%v", v.Limit, v.Actual)
	}
	if v.Severity != SeverityWarning {
		t.Fatalf("duplicate paragraph severity=%v, want warning", v.Severity)
	}
}

func TestLint_DuplicateShortDialogueIsIgnored(t *testing.T) {
	text := "# 第一章\n“别动。”\n他抬起头。\n“别动。”"
	vs := Lint(text)
	for _, v := range vs {
		if v.Rule == "duplicate_paragraph" {
			t.Fatalf("short repeated dialogue must be ignored: %+v", vs)
		}
	}
}

func TestLint_DuplicateHeadingsAndNearMatchesAreIgnored(t *testing.T) {
	text := strings.Join([]string{
		"# 第一章",
		"## 这是一个很长但合法重复出现的场景标题，用来说明标题不属于正文段落",
		"雨水沿着破旧窗棂缓慢滑落，林墨站在黑暗里听见远处钟声再次响起。",
		"## 这是一个很长但合法重复出现的场景标题，用来说明标题不属于正文段落",
		"雨水沿着破旧窗棂缓慢滑落，林墨站在黑暗里听见远处钟声再次响起！",
	}, "\n")
	vs := Lint(text)
	for _, v := range vs {
		if v.Rule == "duplicate_paragraph" {
			t.Fatalf("headings and near matches must be ignored: %+v", vs)
		}
	}
}

func TestLint_DuplicateParagraphNormalizesOuterWhitespaceAndCountsAllCopies(t *testing.T) {
	paragraph := "雾气从河面一层层漫上石阶，守夜人握着灯笼，却始终没有回头看那扇门。"
	text := "# 第一章\n  " + paragraph + "  \n\n" + paragraph + "\r\n\t" + paragraph
	v := findViolation(Lint(text), "duplicate_paragraph", paragraph)
	if v == nil {
		t.Fatal("expected duplicate paragraph after trimming outer whitespace")
	}
	if v.Actual != 3 {
		t.Fatalf("actual=%v, want 3", v.Actual)
	}
}

func TestLint_DuplicateParagraphTargetIsBoundedExcerpt(t *testing.T) {
	paragraph := strings.Repeat("这是一段不会完整复制进诊断记录的超长正文", 8)
	vs := Lint("# 第一章\n" + paragraph + "\n" + paragraph)
	var got *Violation
	for i := range vs {
		if vs[i].Rule == "duplicate_paragraph" {
			got = &vs[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("expected duplicate paragraph: %+v", vs)
	}
	if utf8.RuneCountInString(got.Target) > 49 {
		t.Fatalf("target leaked too much prose: runes=%d target=%q", utf8.RuneCountInString(got.Target), got.Target)
	}
	if !strings.HasSuffix(got.Target, "…") {
		t.Fatalf("truncated target must end with ellipsis: %q", got.Target)
	}
	if got.Target == paragraph {
		t.Fatal("target must not copy the full paragraph")
	}
}

func TestLint_DuplicateParagraphGroupsFollowFirstAppearanceOrder(t *testing.T) {
	first := "第一组重复段落足够长，先在正文里出现，因此违规事实也必须稳定排在前面。"
	second := "第二组重复段落同样足够长，但首次出现更晚，不能受映射遍历顺序影响。"
	vs := Lint(strings.Join([]string{"# 第一章", first, second, first, second}, "\n"))
	var targets []string
	for _, v := range vs {
		if v.Rule == "duplicate_paragraph" {
			targets = append(targets, v.Target)
		}
	}
	want := []string{first, second}
	if !slices.Equal(targets, want) {
		t.Fatalf("duplicate paragraph order=%q want=%q", targets, want)
	}
}

func TestLint_DuplicateParagraphUsesUnicodeLengthBoundary(t *testing.T) {
	short := strings.Repeat("雨", 23)
	boundary := strings.Repeat("风", 24)
	vs := Lint(strings.Join([]string{"# 第一章", short, short, boundary, boundary}, "\n"))
	if findViolation(vs, "duplicate_paragraph", short) != nil {
		t.Fatalf("23-rune paragraph must be ignored: %+v", vs)
	}
	if findViolation(vs, "duplicate_paragraph", boundary) == nil {
		t.Fatalf("24-rune paragraph must be detected: %+v", vs)
	}
}

func TestLint_DuplicateParagraphDoesNotNormalizeInternalWhitespace(t *testing.T) {
	left := "雨水沿着破旧窗棂缓慢滑落，林墨站在黑暗里，听见钟声再次响起。"
	right := "雨水沿着破旧窗棂缓慢滑落，林墨站在黑暗里， 听见钟声再次响起。"
	vs := Lint(strings.Join([]string{"# 第一章", left, right}, "\n"))
	for _, v := range vs {
		if v.Rule == "duplicate_paragraph" {
			t.Fatalf("internal whitespace differences are not exact duplicates: %+v", vs)
		}
	}
}

func TestLint_NonCJKFragments(t *testing.T) {
	text := "# 第一章\n他发现了一个pattern，这个pattern像DNA一样规律。"
	vs := Lint(text)
	var v *Violation
	for i := range vs {
		if vs[i].Rule == "non_cjk_fragments" {
			v = &vs[i]
			break
		}
	}
	if v == nil {
		t.Fatalf("expected non_cjk violation: %+v", vs)
	}
	if v.Actual != 3 {
		t.Errorf("total count: got %v want 3", v.Actual)
	}
	if !strings.Contains(v.Target, "pattern") || !strings.Contains(v.Target, "DNA") {
		t.Errorf("examples should be distinct: %q", v.Target)
	}
	if v.Severity != SeverityWarning {
		t.Errorf("severity: %v", v.Severity)
	}
}
