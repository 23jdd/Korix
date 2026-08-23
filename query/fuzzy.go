package query

// FuzzyQuery expands a term to dictionary entries within MaxEdits Unicode
// Levenshtein distance. MaxEdits is capped at 3 to avoid accidental full scans.
type FuzzyQuery struct {
	Field    string
	Term     string
	MaxEdits int
}

func (q FuzzyQuery) Execute(searcher Searcher) ([]Hit, error) {
	tokens := searcher.Analyzer(q.Field).Analyze(q.Term)
	if len(tokens) == 0 {
		return nil, nil
	}
	term := tokens[0].Term
	maxEdits := q.MaxEdits
	if maxEdits <= 0 {
		maxEdits = 1
	}
	if maxEdits > 3 {
		maxEdits = 3
	}
	allTerms, err := searcher.Terms(q.Field, "")
	if err != nil {
		return nil, err
	}
	scores := make(map[uint64]float64)
	for _, candidate := range allTerms {
		distance := levenshtein(term, candidate, maxEdits)
		if distance > maxEdits {
			continue
		}
		postings, err := searcher.Postings(q.Field, candidate)
		if err != nil {
			return nil, err
		}
		termScores, err := scorePostings(searcher, q.Field, candidate, postings)
		if err != nil {
			return nil, err
		}
		boost := 1 / float64(distance+1)
		for docID, score := range termScores {
			scores[docID] += score * boost
		}
	}
	return hitsFromScores(searcher, scores)
}

func levenshtein(left, right string, cutoff int) int {
	a, b := []rune(left), []rune(right)
	if len(a)-len(b) > cutoff || len(b)-len(a) > cutoff {
		return cutoff + 1
	}
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, ar := range a {
		current[0] = i + 1
		rowMin := current[0]
		for j, br := range b {
			cost := 0
			if ar != br {
				cost = 1
			}
			current[j+1] = min3(current[j]+1, previous[j+1]+1, previous[j]+cost)
			if current[j+1] < rowMin {
				rowMin = current[j+1]
			}
		}
		if rowMin > cutoff {
			return cutoff + 1
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
