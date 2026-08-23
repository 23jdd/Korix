package tokenizer

import (
	"unicode"

	"github.com/23jdd/Koris/analysis"
)

type trieNode struct {
	children map[rune]*trieNode
	terminal bool
}

// ChineseMode 指定 Trie 词典的最大匹配方向。
type ChineseMode uint8

const (
	// ForwardMaximumMatching 从左到右选择当前位置能匹配到的最长词。
	ForwardMaximumMatching ChineseMode = iota
	// ReverseMaximumMatching 从右到左选择当前位置能匹配到的最长词。
	ReverseMaximumMatching
	// BidirectionalMaximumMatching 同时计算正向和逆向结果，优先选择词数更少，
	// 其次选择单字词更少的结果。
	BidirectionalMaximumMatching
)

// ChineseTokenizer 使用 Trie 词典实现中文最大匹配分词。
//
// 未登录汉字回退为单字，确保任何输入都不会丢失；连续拉丁字母与数字组成一个
// word token。Tokenizer 构造完成后只读，因此可以被多个并发索引/查询复用。
type ChineseTokenizer struct {
	root *trieNode
	mode ChineseMode
}

// NewChineseTokenizer 构建词典 Trie。插入成本与词典总 rune 数成正比，查询时
// 每个输入位置只沿 Trie 向下遍历，不需要逐个比较全部词条。
func NewChineseTokenizer(dictionary []string) *ChineseTokenizer {
	t := &ChineseTokenizer{root: &trieNode{children: make(map[rune]*trieNode)}}
	for _, word := range dictionary {
		// terminal 标记“从根到当前节点”构成完整词；非 terminal 节点只是前缀。
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

// WithMode 修改匹配策略并返回接收者，便于链式配置。该方法会修改 tokenizer，
// 应在并发使用之前完成配置。
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
		// 双向模式分别生成两个完整候选，再用稳定规则选择；相同输入与词典总会
		// 得到相同结果。
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
			// longest 返回开区间结尾；没有词典命中时保留默认的单 rune 长度。
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
	// 从右向左生成的 token 先存入 reversed，最后翻转并统一重建 Position，保证
	// 下游永远看到从左到右的词流。
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
	// 沿 Trie 扫描并记住最后一个 terminal，而不是遇到第一个完整词就停止，
	// 这正是“最大匹配”中的最长词选择。
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
	// 逆向匹配需要找“恰好在 end 结束”的最长词。从最左候选开始尝试，首次
	// 完整命中即拥有最小 start，也就是最长结果。
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
	// BMM 的常用消歧规则：先最少词数，再最少单字词；完全相同则保持正向结果，
	// 使选择稳定且易于测试。
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
	// range 直接产生每个 rune 的 UTF-8 字节起点；额外的哨兵项表示字符串末尾。
	offsets := make([]int, len(runes)+1)
	i := 0
	for offset := range text {
		offsets[i] = offset
		i++
	}
	offsets[len(runes)] = len(text)
	return offsets
}
