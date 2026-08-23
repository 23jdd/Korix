package filter

import (
	"strings"

	"github.com/23jdd/Koris/analysis"
)

// StemmerFilter performs conservative English suffix stemming. It handles the
// common inflections without turning the analyzer into a language dependency.
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
