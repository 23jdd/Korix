package tokenizer

import (
	"strings"
	"unicode"

	"github.com/23jdd/Koris/analysis"
)

// StandardTokenizer 识别普通字词、数字、Email、URL 与独立符号。
//
// 实现以确定性和清晰度为目标，并不宣称完整复刻 Lucene 的 UAX#29 行为。扫描在
// rune 层完成以正确判断 Unicode 类别，同时维护 rune→byte 映射供 offset 使用。
type StandardTokenizer struct{}

func (StandardTokenizer) Tokenize(text string) []analysis.Token {
	runes := []rune(text)
	// byteOffsets[n] 是第 n 个 rune 在原 UTF-8 字符串中的起点；最后一项是
	// len(text)，从而可统一处理结尾 token。
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
			// URL 允许内部包含常见标点，一直读取到空白；句尾标点会退回扫描游标，
			// 下一轮仍可作为 symbol token 处理。
			kind = "url"
			for i < len(runes) && !unicode.IsSpace(runes[i]) {
				i++
			}
			for i > start && strings.ContainsRune(".,;:!?)]}", runes[i-1]) {
				i--
			}
		} else if isWordRune(runes[i]) {
			// 普通字词允许 Email/小数常见连接字符；扫描完成后再根据 @ 和字符
			// 组成区分 email、number 与 word。
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
