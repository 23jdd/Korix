package filter

import "github.com/23jdd/Koris/analysis"

// EnglishStopWords 是默认英文停用词集合。它有意保持精简，避免删除可能承载
// 业务含义的词；调用者可通过 NewStopWordFilter 提供领域专用集合。
var EnglishStopWords = []string{
	"a", "an", "and", "are", "as", "at", "be", "by", "for", "from", "has", "he", "in", "is", "it", "its", "of", "on", "that", "the", "to", "was", "were", "will", "with",
}

// StopWordFilter 删除完全匹配的 term。匹配本身区分大小写，因此通常应把
// LowercaseFilter 放在它之前。
type StopWordFilter struct{ words map[string]struct{} }

// NewStopWordFilter 把输入词表构造成 O(1) 查询的只读集合。构造完成后不会保留
// words slice，因此调用者可安全复用或修改原切片。
func NewStopWordFilter(words []string) StopWordFilter {
	filter := StopWordFilter{words: make(map[string]struct{}, len(words))}
	for _, word := range words {
		filter.words[word] = struct{}{}
	}
	return filter
}

func (f StopWordFilter) Filter(tokens []analysis.Token) []analysis.Token {
	// 只删除元素而不重新编号 Position，确保停用词留下的词位间隔仍可被
	// PhraseQuery 感知。
	result := make([]analysis.Token, 0, len(tokens))
	for _, token := range tokens {
		if _, found := f.words[token.Term]; !found {
			result = append(result, token)
		}
	}
	return result
}
