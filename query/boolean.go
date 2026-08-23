package query

// BooleanQuery combines arbitrary child queries. Must clauses intersect,
// Should clauses add score (and form the candidate set when Must is empty), and
// MustNot clauses exclude matching documents.
type BooleanQuery struct {
	Must    []Query
	Should  []Query
	MustNot []Query
}

func (q BooleanQuery) Execute(searcher Searcher) ([]Hit, error) {
	var candidates map[uint64]float64
	for clauseIndex, clause := range q.Must {
		hits, err := clause.Execute(searcher)
		if err != nil {
			return nil, err
		}
		clauseScores := hitScoreMap(hits)
		if clauseIndex == 0 {
			candidates = clauseScores
			continue
		}
		for docID := range candidates {
			if score, found := clauseScores[docID]; found {
				candidates[docID] += score
			} else {
				delete(candidates, docID)
			}
		}
	}
	for _, clause := range q.Should {
		hits, err := clause.Execute(searcher)
		if err != nil {
			return nil, err
		}
		if candidates == nil {
			candidates = make(map[uint64]float64)
		}
		for _, hit := range hits {
			if len(q.Must) == 0 {
				candidates[hit.DocID] += hit.Score
			} else if _, found := candidates[hit.DocID]; found {
				candidates[hit.DocID] += hit.Score
			}
		}
	}
	if candidates == nil {
		candidates = make(map[uint64]float64)
		if len(q.MustNot) > 0 {
			ids, err := searcher.AllDocumentIDs()
			if err != nil {
				return nil, err
			}
			for _, docID := range ids {
				candidates[docID] = 0
			}
		}
	}
	for _, clause := range q.MustNot {
		hits, err := clause.Execute(searcher)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			delete(candidates, hit.DocID)
		}
	}
	return hitsFromScores(searcher, candidates)
}

func hitScoreMap(hits []Hit) map[uint64]float64 {
	result := make(map[uint64]float64, len(hits))
	for _, hit := range hits {
		result[hit.DocID] = hit.Score
	}
	return result
}
