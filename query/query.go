// Package query 定义可组合、与具体 Store 无关的查询执行器。
// Query 只依赖只读 Searcher 接口，因此既不会与 index 形成包循环，也便于使用
// mock/segment reader 单独测试查询算法。
package query

import (
	"sort"

	"github.com/23jdd/Koris/analysis"
	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/inverted"
	"github.com/23jdd/Koris/scoring"
)

// Hit 是一条按相关性排序的搜索结果。
// ID 就是 Document.ID；Koris 不分配第二套内部数字 ID。Score 越大越相关。
// Explanation 只包含轻量解释信息，不承诺作为稳定的机器可解析格式。
type Hit struct {
	ID          string  `json:"id"`
	Score       float64 `json:"score"`
	Explanation string  `json:"explanation,omitempty"`
}

// Searcher 是 Query 所需的最小只读索引视图。
//
// Analyzer 保证查询与写入生成相同 term；Postings/TermInfo/Metadata/Stats 提供
// BM25 与位置查询所需数据；Terms 用于多 term 扩展；Document 为高层能力保留。
type Searcher interface {
	Analyzer(field string) analysis.Analyzer
	BM25() scoring.BM25
	Postings(field, term string) ([]inverted.Posting, error)
	TermInfo(field, term string) (inverted.TermInfo, error)
	Terms(field, prefix string) ([]string, error)
	QueryMetadata(id string) (lengths map[string]uint32, err error)
	SearchStats() (documentCount uint64, totalFieldLength map[string]uint64, err error)
	AllDocumentIDs() ([]string, error)
	Document(id string) (document.Document, error)
}

// Query 是所有查询类型的统一协议。Execute 必须返回按 Score 降序排列的 Hit；
// 相同分数按文档 ID 升序，保证重复执行结果稳定。
type Query interface {
	Execute(searcher Searcher) ([]Hit, error)
}

func sortHits(hits []Hit) {
	// 文档 ID 作为稳定 tie-breaker，避免 Go map 遍历顺序影响最终结果。
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].Score > hits[j].Score
	})
}

func hitsFromScores(scores map[string]float64) ([]Hit, error) {
	// 文档 ID 已经是 scores 的 key，无需再做内部 ID 到外部 ID 的映射。
	hits := make([]Hit, 0, len(scores))
	for id, score := range scores {
		hits = append(hits, Hit{ID: id, Score: score})
	}
	sortHits(hits)
	return hits, nil
}
