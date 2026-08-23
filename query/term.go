package query

// TermQuery performs an exact analyzed-term lookup. Use MatchQuery when raw
// user text still needs analysis.
type TermQuery struct {
	Field string
	Term  string
}

func (q TermQuery) Execute(searcher Searcher) ([]Hit, error) {
	if q.Field == "" || q.Term == "" {
		return nil, nil
	}
	postings, err := searcher.Postings(q.Field, q.Term)
	if err != nil {
		return nil, err
	}
	scores, err := scorePostings(searcher, q.Field, q.Term, postings)
	if err != nil {
		return nil, err
	}
	hits, err := hitsFromScores(searcher, scores)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint64]uint32, len(postings))
	for _, posting := range postings {
		byID[posting.DocID] = posting.Frequency
	}
	for i := range hits {
		hits[i].Explanation = explanation(q.Field, q.Term, byID[hits[i].DocID])
	}
	return hits, nil
}
