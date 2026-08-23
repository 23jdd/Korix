package query

// Operator controls how MatchQuery combines analyzed terms.
type Operator uint8

const (
	OR Operator = iota
	AND
)

type MatchQuery struct {
	Field    string
	Text     string
	Operator Operator
}

func (q MatchQuery) Execute(searcher Searcher) ([]Hit, error) {
	tokens := searcher.Analyzer(q.Field).Analyze(q.Text)
	if len(tokens) == 0 {
		return nil, nil
	}
	terms := make([]string, 0, len(tokens))
	seen := make(map[string]struct{})
	for _, token := range tokens {
		if _, found := seen[token.Term]; !found {
			seen[token.Term] = struct{}{}
			terms = append(terms, token.Term)
		}
	}
	scores := make(map[uint64]float64)
	matched := make(map[uint64]int)
	for _, term := range terms {
		postings, err := searcher.Postings(q.Field, term)
		if err != nil {
			return nil, err
		}
		termScores, err := scorePostings(searcher, q.Field, term, postings)
		if err != nil {
			return nil, err
		}
		for docID, score := range termScores {
			scores[docID] += score
			matched[docID]++
		}
	}
	if q.Operator == AND {
		for docID := range scores {
			if matched[docID] != len(terms) {
				delete(scores, docID)
			}
		}
	}
	return hitsFromScores(searcher, scores)
}
