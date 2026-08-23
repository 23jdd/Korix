package tokenizer

import (
	"unicode"

	"github.com/23jdd/Koris/analysis"
)

type trieNode struct {
	children map[rune]*trieNode
	terminal bool
}

// ChineseTokenizer uses forward maximum matching for configured dictionary
// words. Unknown Han text falls back to single characters so no text is lost.
// Latin letters and numbers are grouped like SimpleTokenizer.
type ChineseMode uint8

const (
	ForwardMaximumMatching ChineseMode = iota
	ReverseMaximumMatching
	BidirectionalMaximumMatching
)

type ChineseTokenizer struct {
	root *trieNode
	mode ChineseMode
}

func NewChineseTokenizer(dictionary []string) *ChineseTokenizer {
	t := &ChineseTokenizer{root: &trieNode{children: make(map[rune]*trieNode)}}
	for _, word := range dictionary {
		node := t.root
		for _, r := range word {
			if node.children[r] == nil {
				node.children[r] = &trieNode{children: make(map[rune]*trieNode)}
			}
			node = node.children[r]
		}
		node.terminal = true
	}
	return t
}

// WithMode changes dictionary matching while retaining the same trie.
func (t *ChineseTokenizer) WithMode(mode ChineseMode) *ChineseTokenizer {
	t.mode = mode
	return t
}

func (t *ChineseTokenizer) Tokenize(text string) []analysis.Token {
	if t == nil || t.root == nil {
		t = NewChineseTokenizer(nil)
	}
	runes := []rune(text)
	if t.mode == ReverseMaximumMatching || t.mode == BidirectionalMaximumMatching {
		forward := t.tokenizeRunes(text, runes, false)
		reverse := t.tokenizeRunes(text, runes, true)
		if t.mode == ReverseMaximumMatching || preferReverse(forward, reverse) {
			return reverse
		}
		return forward
	}
	return t.tokenizeRunes(text, runes, false)
}

func (t *ChineseTokenizer) tokenizeRunes(text string, runes []rune, reverse bool) []analysis.Token {
	if reverse {
		return t.tokenizeReverse(text, runes)
	}
	offsets := runeByteOffsets(text, runes)
	tokens := make([]analysis.Token, 0, len(runes))
	var position uint32
	for i := 0; i < len(runes); {
		if unicode.IsSpace(runes[i]) || unicode.IsPunct(runes[i]) {
			i++
			continue
		}
		start, end, kind := i, i+1, "han"
		if unicode.Is(unicode.Han, runes[i]) {
			if matched := t.longest(runes, i); matched > end {
				end = matched
			}
		} else if unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) {
			kind = "word"
			for end < len(runes) && (unicode.IsLetter(runes[end]) || unicode.IsDigit(runes[end])) && !unicode.Is(unicode.Han, runes[end]) {
				end++
			}
		} else {
			kind = "symbol"
		}
		tokens = appendToken(tokens, text, offsets[start], offsets[end], position, kind)
		position++
		i = end
	}
	return tokens
}

func (t *ChineseTokenizer) tokenizeReverse(text string, runes []rune) []analysis.Token {
	offsets := runeByteOffsets(text, runes)
	reversed := make([]analysis.Token, 0, len(runes))
	for end := len(runes); end > 0; {
		if unicode.IsSpace(runes[end-1]) || unicode.IsPunct(runes[end-1]) {
			end--
			continue
		}
		start, kind := end-1, "han"
		if unicode.Is(unicode.Han, runes[end-1]) {
			start = t.longestEndingAt(runes, end)
		} else if unicode.IsLetter(runes[end-1]) || unicode.IsDigit(runes[end-1]) {
			kind = "word"
			for start > 0 && (unicode.IsLetter(runes[start-1]) || unicode.IsDigit(runes[start-1])) && !unicode.Is(unicode.Han, runes[start-1]) {
				start--
			}
		} else {
			kind = "symbol"
		}
		reversed = append(reversed, analysis.Token{Term: text[offsets[start]:offsets[end]], StartOffset: uint32(offsets[start]), EndOffset: uint32(offsets[end]), Type: kind})
		end = start
	}
	tokens := make([]analysis.Token, len(reversed))
	for i := range reversed {
		tokens[i] = reversed[len(reversed)-1-i]
		tokens[i].Position = uint32(i)
	}
	return tokens
}

func (t *ChineseTokenizer) longest(runes []rune, start int) int {
	node := t.root
	longest := start + 1
	for i := start; i < len(runes); i++ {
		node = node.children[runes[i]]
		if node == nil {
			break
		}
		if node.terminal {
			longest = i + 1
		}
	}
	return longest
}

func (t *ChineseTokenizer) longestEndingAt(runes []rune, end int) int {
	best := end - 1
	for start := 0; start < end; start++ {
		node := t.root
		matched := true
		for i := start; i < end; i++ {
			node = node.children[runes[i]]
			if node == nil {
				matched = false
				break
			}
		}
		if matched && node.terminal {
			best = start
			break
		}
	}
	return best
}

func preferReverse(forward, reverse []analysis.Token) bool {
	if len(reverse) != len(forward) {
		return len(reverse) < len(forward)
	}
	forwardSingles, reverseSingles := 0, 0
	for _, token := range forward {
		if len([]rune(token.Term)) == 1 {
			forwardSingles++
		}
	}
	for _, token := range reverse {
		if len([]rune(token.Term)) == 1 {
			reverseSingles++
		}
	}
	return reverseSingles < forwardSingles
}

func runeByteOffsets(text string, runes []rune) []int {
	offsets := make([]int, len(runes)+1)
	i := 0
	for offset := range text {
		offsets[i] = offset
		i++
	}
	offsets[len(runes)] = len(text)
	return offsets
}
