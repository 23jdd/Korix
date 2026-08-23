package tokenizer

import (
	"unicode"

	"github.com/23jdd/Koris/analysis"
)

// SimpleTokenizer emits contiguous Unicode letters and digits.
type SimpleTokenizer struct{}

func (SimpleTokenizer) Tokenize(text string) []analysis.Token {
	var tokens []analysis.Token
	start := -1
	position := uint32(0)
	for offset, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = offset
			}
			continue
		}
		if start >= 0 {
			tokens = appendToken(tokens, text, start, offset, position, "word")
			position++
			start = -1
		}
	}
	if start >= 0 {
		tokens = appendToken(tokens, text, start, len(text), position, "word")
	}
	return tokens
}
