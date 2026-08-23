package tokenizer

import (
	"math"
	"unicode"

	"github.com/23jdd/Koris/analysis"
)

// HMMState is one state in a BMES word-boundary model.
type HMMState uint8

const (
	BeginState HMMState = iota
	MiddleState
	EndState
	SingleState
	hmmStateCount
)

// HMMModel stores log probabilities. Emission maps a rune to the four BMES
// scores; UnknownEmission handles runes absent from training data.
type HMMModel struct {
	Start           [hmmStateCount]float64
	Transition      [hmmStateCount][hmmStateCount]float64
	Emission        map[rune][hmmStateCount]float64
	UnknownEmission [hmmStateCount]float64
}

// DefaultHMMModel is a usable boundary-only model. Production users can load
// trained emission probabilities into Emission without changing the tokenizer.
func DefaultHMMModel() HMMModel {
	negativeInfinity := math.Inf(-1)
	model := HMMModel{Emission: make(map[rune][hmmStateCount]float64)}
	model.Start = [hmmStateCount]float64{-0.5, negativeInfinity, negativeInfinity, -0.8}
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

// Segment runs Viterbi and converts the most likely BMES path into words.
func (m HMMModel) Segment(text string) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	scores := make([][hmmStateCount]float64, len(runes))
	back := make([][hmmStateCount]HMMState, len(runes))
	for state := HMMState(0); state < hmmStateCount; state++ {
		scores[0][state] = m.Start[state] + m.emission(runes[0], state)
	}
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

// HMMTokenizer applies a BMES model to Han runs and preserves grouped Latin
// words/numbers. It is useful alone or as an unknown-word analyzer beside the
// dictionary ChineseTokenizer.
type HMMTokenizer struct{ Model HMMModel }

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
