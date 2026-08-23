package index

import "github.com/23jdd/Koris/query"

// Search executes a query and optionally limits the ranked result set. A
// non-positive limit returns all hits.
func (i *Index) Search(q query.Query, limit int) ([]query.Hit, error) {
	if q == nil {
		return nil, nil
	}
	hits, err := q.Execute(i)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}
