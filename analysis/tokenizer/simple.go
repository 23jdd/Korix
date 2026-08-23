package tokenizer

import (
	"unicode"

	"github.com/23jdd/Koris/analysis"
)

// SimpleTokenizer 把连续 Unicode 字母或数字组成一个 term，其余字符均作为分隔符。
// 例如 "Hello, Go123!" 输出 Hello 与 Go123；它不会尝试识别 URL 或 Email。
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
