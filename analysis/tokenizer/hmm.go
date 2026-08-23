package tokenizer

import (
	"math"
	"unicode"

	"github.com/23jdd/Koris/analysis"
)

// HMMState 是 BMES 隐马尔可夫模型中的一个词边界状态。
type HMMState uint8

const (
	// BeginState 表示多字词的首字。
	BeginState HMMState = iota
	// MiddleState 表示多字词的中间字。
	MiddleState
	// EndState 表示多字词的末字。
	EndState
	// SingleState 表示单字独立成词。
	SingleState
	hmmStateCount
)

// HMMModel 保存 BMES 模型的对数概率。
//
// 使用 log probability 可以把路径概率连乘变成加法，并避免长文本下浮点下溢。
// Emission 为每个 rune 保存四个状态的发射分数；训练语料未出现的 rune 使用
// UnknownEmission。非法状态转移应设置为 math.Inf(-1)。
type HMMModel struct {
	Start           [hmmStateCount]float64
	Transition      [hmmStateCount][hmmStateCount]float64
	Emission        map[rune][hmmStateCount]float64
	UnknownEmission [hmmStateCount]float64
}

// DefaultHMMModel 返回只有通用边界先验的模型，可保证 Viterbi 路径合法。
// 它适合作为回退或示例；生产中文分词应从语料训练 Emission，并可同时替换起始和
// 转移概率。调用方只需替换模型数据，不需要修改 tokenizer 算法。
func DefaultHMMModel() HMMModel {
	negativeInfinity := math.Inf(-1)
	model := HMMModel{Emission: make(map[rune][hmmStateCount]float64)}
	model.Start = [hmmStateCount]float64{-0.5, negativeInfinity, negativeInfinity, -0.8}
	// 先禁止全部转移，再只开放合法 BMES 边：B→M/E、M→M/E、
	// E→B/S、S→B/S。这样即使发射分数异常也不会生成非法词界。
	for from := HMMState(0); from < hmmStateCount; from++ {
		for to := HMMState(0); to < hmmStateCount; to++ {
			model.Transition[from][to] = negativeInfinity
		}
	}
	model.Transition[BeginState][MiddleState] = -0.7
	model.Transition[BeginState][EndState] = -0.7
	model.Transition[MiddleState][MiddleState] = -0.7
	model.Transition[MiddleState][EndState] = -0.7
	model.Transition[EndState][BeginState] = -0.7
	model.Transition[EndState][SingleState] = -0.7
	model.Transition[SingleState][BeginState] = -0.7
	model.Transition[SingleState][SingleState] = -0.7
	model.UnknownEmission = [hmmStateCount]float64{-0.6, -0.8, -0.6, -0.9}
	return model
}

// Segment 使用 Viterbi 动态规划寻找概率最大的 BMES 状态路径，并按路径切词。
// 时间复杂度 O(n×4²)，空间复杂度 O(n×4)，其中 n 是输入 rune 数。
func (m HMMModel) Segment(text string) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	scores := make([][hmmStateCount]float64, len(runes))
	back := make([][hmmStateCount]HMMState, len(runes))
	// 第 0 列由起始概率与首字发射概率初始化。
	for state := HMMState(0); state < hmmStateCount; state++ {
		scores[0][state] = m.Start[state] + m.emission(runes[0], state)
	}
	// scores[position][state] 保存“在该位置以 state 结束”的最优路径分数；
	// back 保存其前驱，用于结束后反向恢复整条路径。
	for position := 1; position < len(runes); position++ {
		for state := HMMState(0); state < hmmStateCount; state++ {
			bestScore := math.Inf(-1)
			bestPrevious := BeginState
			for previous := HMMState(0); previous < hmmStateCount; previous++ {
				candidate := scores[position-1][previous] + m.Transition[previous][state]
				if candidate > bestScore {
					bestScore = candidate
					bestPrevious = previous
				}
			}
			scores[position][state] = bestScore + m.emission(runes[position], state)
			back[position][state] = bestPrevious
		}
	}
	// 合法句尾只能是 E 或 S，不能以一个尚未结束的 B/M 状态收尾。
	finalState := EndState
	if scores[len(runes)-1][SingleState] > scores[len(runes)-1][EndState] {
		finalState = SingleState
	}
	states := make([]HMMState, len(runes))
	states[len(states)-1] = finalState
	for position := len(states) - 1; position > 0; position-- {
		states[position-1] = back[position][states[position]]
	}
	return wordsFromStates(runes, states)
}

func (m HMMModel) emission(r rune, state HMMState) float64 {
	if scores, found := m.Emission[r]; found {
		return scores[state]
	}
	return m.UnknownEmission[state]
}

func wordsFromStates(runes []rune, states []HMMState) []string {
	// E 和 S 都表示一个词在当前位置闭合；如果自定义模型产生异常未闭合路径，
	// 尾部 fallback 仍会保留所有字符而不是静默丢失。
	words := make([]string, 0)
	start := 0
	for i, state := range states {
		if state == EndState || state == SingleState {
			words = append(words, string(runes[start:i+1]))
			start = i + 1
		}
	}
	if start < len(runes) {
		words = append(words, string(runes[start:]))
	}
	return words
}

// HMMTokenizer 对连续汉字片段应用 BMES 模型，同时把连续拉丁字母/数字保留为
// 一个 word token。它既可以单独作为字段 tokenizer，也可以用于词典分词之外的
// 未登录词处理。
type HMMTokenizer struct{ Model HMMModel }

// NewHMMTokenizer 使用给定模型构造无状态、可并发复用的 tokenizer。
func NewHMMTokenizer(model HMMModel) HMMTokenizer { return HMMTokenizer{Model: model} }

func (t HMMTokenizer) Tokenize(text string) []analysis.Token {
	runes := []rune(text)
	offsets := runeByteOffsets(text, runes)
	result := make([]analysis.Token, 0, len(runes))
	var position uint32
	for i := 0; i < len(runes); {
		if unicode.IsSpace(runes[i]) || unicode.IsPunct(runes[i]) {
			i++
			continue
		}
		start, end, kind := i, i+1, "symbol"
		if unicode.Is(unicode.Han, runes[i]) {
			// HMM 一次处理完整连续汉字片段，避免标点和拉丁文本影响边界概率。
			kind = "han"
			for end < len(runes) && unicode.Is(unicode.Han, runes[end]) {
				end++
			}
			wordStart := start
			for _, word := range t.Model.Segment(string(runes[start:end])) {
				wordEnd := wordStart + len([]rune(word))
				result = appendToken(result, text, offsets[wordStart], offsets[wordEnd], position, kind)
				position++
				wordStart = wordEnd
			}
			i = end
			continue
		}
		if unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) {
			kind = "word"
			for end < len(runes) && (unicode.IsLetter(runes[end]) || unicode.IsDigit(runes[end])) && !unicode.Is(unicode.Han, runes[end]) {
				end++
			}
		}
		result = appendToken(result, text, offsets[start], offsets[end], position, kind)
		position++
		i = end
	}
	return result
}
