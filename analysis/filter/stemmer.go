package filter

import (
	"strings"

	"github.com/23jdd/Koris/analysis"
)

// StemmerFilter 执行保守的英文后缀词干化，例如 running → run。
//
// 这里不是完整 Porter2 实现：规则刻意偏保守，目标是处理常见屈折变化，同时
// 避免引入语言库或把差异很大的词过度合并。需要严格语言学行为时应替换为自定义
// TokenFilter，并确保写入与查询端配置一致。
type StemmerFilter struct{}

func (StemmerFilter) Filter(tokens []analysis.Token) []analysis.Token {
	result := make([]analysis.Token, len(tokens))
	copy(result, tokens)
	for i := range result {
		result[i].Term = stem(result[i].Term)
	}
	return result
}

func stem(word string) string {
	switch {
	case len(word) > 5 && strings.HasSuffix(word, "ing"):
		word = strings.TrimSuffix(word, "ing")
		// running -> run, but sing -> sing is guarded by length above.
		if len(word) >= 2 && word[len(word)-1] == word[len(word)-2] {
			word = word[:len(word)-1]
		}
	case len(word) > 4 && strings.HasSuffix(word, "ied"):
		word = strings.TrimSuffix(word, "ied") + "y"
	case len(word) > 4 && strings.HasSuffix(word, "ed"):
		word = strings.TrimSuffix(word, "ed")
	case len(word) > 4 && strings.HasSuffix(word, "es"):
		word = strings.TrimSuffix(word, "es")
	case len(word) > 3 && strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss"):
		word = strings.TrimSuffix(word, "s")
	}
	return word
}
