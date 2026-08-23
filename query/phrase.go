package query

import "github.com/23jdd/Koris/inverted"

type PhraseQuery struct {
	Field string
	Text  string
	Slop  uint32
}

func (q PhraseQuery) Execute(searcher Searcher) ([]Hit, error) {
	tokens := searcher.Analyzer(q.Field).Analyze(q.Text)
	if len(tokens) == 0 {
		return nil, nil
	}
	postingSets := make([]map[uint64]inverted.Posting, len(tokens))
	termScores := make([]map[uint64]float64, len(tokens))
	for i, token := range tokens {
		postings, err := searcher.Postings(q.Field, token.Term)
		if err != nil {
			return nil, err
		}
		postingSets[i] = make(map[uint64]inverted.Posting, len(postings))
		for _, posting := range postings {
			postingSets[i][posting.DocID] = posting
		}
		termScores[i], err = scorePostings(searcher, q.Field, token.Term, postings)
		if err != nil {
			return nil, err
		}
	}
	scores := make(map[uint64]float64)
	for docID, first := range postingSets[0] {
		positions := make([][]uint32, len(tokens))
		positions[0] = first.Positions
		matched := true
		for i := 1; i < len(tokens); i++ {
			posting, found := postingSets[i][docID]
			if !found {
				matched = false
				break
			}
			positions[i] = posting.Positions
		}
		relative := make([]uint32, len(tokens))
		for i := range tokens {
			relative[i] = tokens[i].Position - tokens[0].Position
		}
		if matched && phrasePositionsMatch(positions, relative, q.Slop) {
			for i := range tokens {
				scores[docID] += termScores[i][docID]
			}
			// Phrase matches are more specific than the sum of their terms.
			scores[docID] *= 1.5
		}
	}
	return hitsFromScores(searcher, scores)
}

func phrasePositionsMatch(positions [][]uint32, relative []uint32, slop uint32) bool {
	if len(positions) == 0 {
		return false
	}
	for _, start := range positions[0] {
		previous := start
		usedSlop := uint32(0)
		matched := true
		for i := 1; i < len(positions); i++ {
			expectedGap := uint32(1)
			if i < len(relative) && relative[i] > relative[i-1] {
				expectedGap = relative[i] - relative[i-1]
			}
			next, found := nextPosition(positions[i], previous+expectedGap)
			if !found || next < previous+expectedGap {
				matched = false
				break
			}
			usedSlop += next - (previous + expectedGap)
			if usedSlop > slop {
				matched = false
				break
			}
			previous = next
		}
		if matched {
			return true
		}
	}
	return false
}

func nextPosition(positions []uint32, minimum uint32) (uint32, bool) {
	for _, position := range positions {
		if position >= minimum {
			return position, true
		}
	}
	return 0, false
}
