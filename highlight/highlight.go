// Package highlight produces safe snippets from analyzed token offsets.
package highlight

import (
	"html"
	"sort"
	"strings"

	"github.com/23jdd/Koris/analysis"
)

type Options struct {
	PreTag       string
	PostTag      string
	FragmentSize int
	MaxFragments int
	EscapeHTML   bool
}

func DefaultOptions() Options {
	return Options{PreTag: "<em>", PostTag: "</em>", FragmentSize: 160, MaxFragments: 1, EscapeHTML: true}
}

type Fragment struct {
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// Extract highlights tokens whose normalized terms are in terms. Fragments
// are byte-safe and overlap is coalesced.
func Extract(analyzer analysis.Analyzer, text string, terms []string, options Options) []Fragment {
	if analyzer == nil || text == "" || len(terms) == 0 {
		return nil
	}
	applyDefaults(&options)
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
