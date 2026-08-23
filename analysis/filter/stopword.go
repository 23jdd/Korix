package filter

import "github.com/23jdd/Koris/analysis"

// EnglishStopWords is a compact default set suitable for general text.
var EnglishStopWords = []string{
	"a", "an", "and", "are", "as", "at", "be", "by", "for", "from", "has", "he", "in", "is", "it", "its", "of", "on", "that", "the", "to", "was", "were", "will", "with",
}

type StopWordFilter struct{ words map[string]struct{} }

func NewStopWordFilter(words []string) StopWordFilter {
	filter := StopWordFilter{words: make(map[string]struct{}, len(words))}
	for _, word := range words {
		filter.words[word] = struct{}{}
	}
	return filter
}

func (f StopWordFilter) Filter(tokens []analysis.Token) []analysis.Token {
	result := make([]analysis.Token, 0, len(tokens))
	for _, token := range tokens {
		if _, found := f.words[token.Term]; !found {
			result = append(result, token)
		}
	}
	return result
}
