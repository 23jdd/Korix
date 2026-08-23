package index

import (
	"sort"

	"github.com/23jdd/Koris/query"
)

// FacetBucket 是一个原始字段值及其在命中文档中的出现次数。
type FacetBucket struct {
	Value string `json:"value"`
	Count uint64 `json:"count"`
}

// Facet 对查询命中文档中的原始字段值做精确计数。
//
// 结果按 Count 降序、Value 升序排列，limit <= 0 返回全部 bucket。该实现无需
// 单独列存，适合 category/status 等低基数字段；高基数、大结果集应使用专用
// doc-values 结构以避免逐文档读取。
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
	// Value 作为第二排序键，确保相同计数下的输出稳定。
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
