// Package query defines composable, storage-independent search queries.
package query

import (
	"sort"

	"github.com/23jdd/Koris/analysis"
	"github.com/23jdd/Koris/document"
	"github.com/23jdd/Koris/inverted"
	"github.com/23jdd/Koris/scoring"
)

// Hit is one ranked search result.
type Hit struct {
	DocID       uint64  `json:"doc_id"`
	ID          string  `json:"id"`
	Score       float64 `json:"score"`
	Explanation string  `json:"explanation,omitempty"`
}

// Searcher is the read-only index surface required by queries.
type Searcher interface {
	Analyzer(field string) analysis.Analyzer
	BM25() scoring.BM25
	Postings(field, term string) ([]inverted.Posting, error)
	TermInfo(field, term string) (inverted.TermInfo, error)
	Terms(field, prefix string) ([]string, error)
	QueryMetadata(docID uint64) (externalID string, lengths map[string]uint32, err error)
	SearchStats() (documentCount uint64, totalFieldLength map[string]uint64, err error)
	AllDocumentIDs() ([]uint64, error)
	Document(docID uint64) (document.Document, error)
}

type Query interface {
	Execute(searcher Searcher) ([]Hit, error)
}

func sortHits(hits []Hit) {
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].DocID < hits[j].DocID
		}
		return hits[i].Score > hits[j].Score
	})
}

func hitsFromScores(searcher Searcher, scores map[uint64]float64) ([]Hit, error) {
	hits := make([]Hit, 0, len(scores))
	for docID, score := range scores {
		externalID, _, err := searcher.QueryMetadata(docID)
		if err != nil {
			return nil, err
		}
		hits = append(hits, Hit{DocID: docID, ID: externalID, Score: score})
	}
	sortHits(hits)
	return hits, nil
}
