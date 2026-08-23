package query

import (
	"fmt"

	"github.com/23jdd/Koris/inverted"
)

func scorePostings(searcher Searcher, field, term string, postings []inverted.Posting) (map[uint64]float64, error) {
	// 同一个 term 的 N、DF、avgDL 对全部 posting 相同，循环外读取一次即可。
	documentCount, totalFieldLength, err := searcher.SearchStats()
	if err != nil {
		return nil, err
	}
	info, err := searcher.TermInfo(field, term)
	if err != nil {
		return nil, err
	}
	averageLength := float64(0)
	if documentCount > 0 {
		averageLength = float64(totalFieldLength[field]) / float64(documentCount)
	}
	scores := make(map[uint64]float64, len(postings))
	for _, posting := range postings {
		// 每篇文档只读取自己的字段长度 DL；缺少字段时 map 零值为 0。
		_, lengths, err := searcher.QueryMetadata(posting.DocID)
		if err != nil {
			return nil, err
		}
		scores[posting.DocID] = searcher.BM25().Score(
			uint64(posting.Frequency), info.DocumentFrequency, documentCount,
			float64(lengths[field]), averageLength,
		)
	}
	return scores, nil
}

func explanation(field, term string, frequency uint32) string {
	return fmt.Sprintf("BM25(%s:%s, tf=%d)", field, term, frequency)
}
