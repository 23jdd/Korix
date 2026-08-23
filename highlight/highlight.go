// Package highlight 根据 Analyzer 生成的原文 offset 构造安全搜索摘要。
package highlight

import (
	"html"
	"sort"
	"strings"

	"github.com/23jdd/Koris/analysis"
)

// Options 控制高亮标签、片段长度和 HTML 安全策略。
// FragmentSize 使用 UTF-8 字节数，边界会自动调整到合法字符起点。
type Options struct {
	// PreTag 与 PostTag 包裹命中的原文 token；它们不会被 HTML escape。
	PreTag  string
	PostTag string
	// FragmentSize 是每个摘要目标字节数，实际值可能为保证 UTF-8 完整略微扩展。
	FragmentSize int
	// MaxFragments 限制返回片段数；非正数使用默认值。
	MaxFragments int
	// EscapeHTML 控制是否转义片段原文。输出到 HTML 时应保持 true。
	EscapeHTML bool
}

// DefaultOptions 返回 HTML 安全的 <em> 高亮配置，每篇文档最多一个 160 字节片段。
func DefaultOptions() Options {
	return Options{PreTag: "<em>", PostTag: "</em>", FragmentSize: 160, MaxFragments: 1, EscapeHTML: true}
}

// Fragment 包含渲染后的文本，以及该片段在原文中的半开字节区间 [Start, End)。
type Fragment struct {
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// Extract 高亮经过 Analyzer 归一化后存在于 terms 的 token。
//
// 相近命中会合并到同一片段，UTF-8 边界会校正，EscapeHTML 开启时只转义原文，
// PreTag/PostTag 被视为调用方可信模板。没有命中或参数为空时返回 nil。
func Extract(analyzer analysis.Analyzer, text string, terms []string, options Options) []Fragment {
	if analyzer == nil || text == "" || len(terms) == 0 {
		return nil
	}
	applyDefaults(&options)
	// 查询词也走同一个 Analyzer，确保 GO 能匹配索引/原文中的 go。
	wanted := make(map[string]struct{}, len(terms))
	for _, term := range terms {
		for _, token := range analyzer.Analyze(term) {
			wanted[token.Term] = struct{}{}
		}
	}
	matches := make([]span, 0)
	for _, token := range analyzer.Analyze(text) {
		if _, found := wanted[token.Term]; found && int(token.EndOffset) <= len(text) {
			matches = append(matches, span{start: int(token.StartOffset), end: int(token.EndOffset)})
		}
	}
	if len(matches) == 0 {
		return nil
	}
	// 先按距离把命中聚成片段候选，再应用 MaxFragments，避免同一区域重复输出。
	groups := groupMatches(matches, options.FragmentSize)
	if len(groups) > options.MaxFragments {
		groups = groups[:options.MaxFragments]
	}
	fragments := make([]Fragment, 0, len(groups))
	for _, group := range groups {
		start, end := fragmentBounds(text, group.start, group.end, options.FragmentSize)
		fragments = append(fragments, Fragment{
			Text: render(text, matchesInRange(matches, start, end), start, end, options), Start: start, End: end,
		})
	}
	return fragments
}

type span struct{ start, end int }

func applyDefaults(options *Options) {
	defaults := DefaultOptions()
	if options.PreTag == "" {
		options.PreTag = defaults.PreTag
	}
	if options.PostTag == "" {
		options.PostTag = defaults.PostTag
	}
	if options.FragmentSize <= 0 {
		options.FragmentSize = defaults.FragmentSize
	}
	if options.MaxFragments <= 0 {
		options.MaxFragments = defaults.MaxFragments
	}
}

func groupMatches(matches []span, size int) []span {
	sort.Slice(matches, func(i, j int) bool { return matches[i].start < matches[j].start })
	groups := []span{matches[0]}
	for _, match := range matches[1:] {
		last := &groups[len(groups)-1]
		if match.start-last.end <= size {
			last.end = match.end
		} else {
			groups = append(groups, match)
		}
	}
	return groups
}

func fragmentBounds(text string, matchStart, matchEnd, size int) (int, int) {
	// 尽量让命中位于片段中间；靠近开头/结尾时把剩余额度移到另一侧。
	half := (size - (matchEnd - matchStart)) / 2
	if half < 0 {
		half = 0
	}
	start := matchStart - half
	if start < 0 {
		start = 0
	}
	end := start + size
	if end < matchEnd {
		end = matchEnd
	}
	if end > len(text) {
		end = len(text)
		start = end - size
		if start < 0 {
			start = 0
		}
	}
	// 字节窗口可能切到多字节 rune 中间，向外移动到合法 UTF-8 边界。
	for start > 0 && !isUTF8Start(text[start]) {
		start--
	}
	for end < len(text) && !isUTF8Start(text[end]) {
		end++
	}
	return start, end
}

func matchesInRange(matches []span, start, end int) []span {
	result := make([]span, 0)
	for _, match := range matches {
		if match.start >= start && match.end <= end {
			result = append(result, match)
		}
	}
	return result
}

func render(text string, matches []span, start, end int, options Options) string {
	var builder strings.Builder
	cursor := start
	write := func(value string) {
		// 只 escape 原始文本，不 escape 标签；标签来自 Options，必须由调用方控制。
		if options.EscapeHTML {
			builder.WriteString(html.EscapeString(value))
		} else {
			builder.WriteString(value)
		}
	}
	for _, match := range matches {
		if match.start < cursor {
			continue
		}
		write(text[cursor:match.start])
		builder.WriteString(options.PreTag)
		write(text[match.start:match.end])
		builder.WriteString(options.PostTag)
		cursor = match.end
	}
	write(text[cursor:end])
	return builder.String()
}

func isUTF8Start(value byte) bool { return value&0xC0 != 0x80 }
