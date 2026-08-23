package query

type PrefixQuery struct {
	Field  string
	Prefix string
}

func (q PrefixQuery) Execute(searcher Searcher) ([]Hit, error) {
	tokens := searcher.Analyzer(q.Field).Analyze(q.Prefix)
	prefix := q.Prefix
	if len(tokens) > 0 {
		prefix = tokens[0].Term
	}
	terms, err := searcher.Terms(q.Field, prefix)
	if err != nil {
		return nil, err
	}
	scores := make(map[uint64]float64)
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
		}
	}
	return hitsFromScores(searcher, scores)
}
