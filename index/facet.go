package index

import (
	"sort"

	"github.com/23jdd/Koris/query"
)

type FacetBucket struct {
	Value string `json:"value"`
	Count uint64 `json:"count"`
}

// Facet counts exact stored field values across query hits. It is suited to
// low-cardinality fields and requires no separate column store.
func (i *Index) Facet(q query.Query, field string, limit int) ([]FacetBucket, error) {
	hits, err := i.Search(q, 0)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]uint64)
	for _, hit := range hits {
		doc, err := i.Document(hit.DocID)
		if err != nil {
			return nil, err
		}
		if value, found := doc.Fields[field]; found {
			counts[value]++
		}
	}
	buckets := make([]FacetBucket, 0, len(counts))
	for value, count := range counts {
		buckets = append(buckets, FacetBucket{Value: value, Count: count})
	}
	sort.Slice(buckets, func(a, b int) bool {
		if buckets[a].Count == buckets[b].Count {
			return buckets[a].Value < buckets[b].Value
		}
		return buckets[a].Count > buckets[b].Count
	})
	if limit > 0 && len(buckets) > limit {
		buckets = buckets[:limit]
	}
	return buckets, nil
}
