package tokenizer

import (
	"strings"
	"unicode"

	"github.com/23jdd/Koris/analysis"
)

// StandardTokenizer recognizes words, numbers, e-mail addresses, URLs and
// standalone symbols. It is intentionally deterministic and allocation-light,
// rather than attempting to duplicate every Lucene UAX#29 edge case.
type StandardTokenizer struct{}

func (StandardTokenizer) Tokenize(text string) []analysis.Token {
	runes := []rune(text)
	byteOffsets := make([]int, len(runes)+1)
	bytePos := 0
	for i, r := range runes {
		byteOffsets[i] = bytePos
		bytePos += len(string(r))
	}
	byteOffsets[len(runes)] = len(text)

	var tokens []analysis.Token
	var position uint32
	for i := 0; i < len(runes); {
		if unicode.IsSpace(runes[i]) {
			i++
			continue
		}
		start := i
		kind := "symbol"
		if hasURLPrefix(runes[i:]) {
			kind = "url"
			for i < len(runes) && !unicode.IsSpace(runes[i]) {
				i++
			}
			for i > start && strings.ContainsRune(".,;:!?)]}", runes[i-1]) {
				i--
			}
		} else if isWordRune(runes[i]) {
			kind = "word"
			hasAt := false
			for i < len(runes) && (isWordRune(runes[i]) || strings.ContainsRune("._+-@", runes[i])) {
				hasAt = hasAt || runes[i] == '@'
				i++
			}
			if hasAt {
				kind = "email"
			} else if allNumberLike(runes[start:i]) {
				kind = "number"
			}
		} else {
			i++
		}
		if i > start {
			tokens = appendToken(tokens, text, byteOffsets[start], byteOffsets[i], position, kind)
			position++
		} else {
			i++
		}
	}
	return tokens
}

func hasURLPrefix(runes []rune) bool {
	s := strings.ToLower(string(runes))
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "www.")
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

func allNumberLike(runes []rune) bool {
	if len(runes) == 0 {
		return false
	}
	for _, r := range runes {
		if !unicode.IsDigit(r) && r != '.' && r != ',' && r != '+' && r != '-' {
			return false
		}
	}
	return true
}
