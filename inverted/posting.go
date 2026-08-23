// Package inverted contains storage-neutral inverted-index primitives.
package inverted

import "sort"

// Posting is the occurrence summary of one term in one document.
type Posting struct {
	DocID     uint64   `json:"doc_id"`
	Frequency uint32   `json:"frequency"`
	Positions []uint32 `json:"positions,omitempty"`
}

// NewPosting normalizes positions and derives frequency.
func NewPosting(docID uint64, positions []uint32) Posting {
	copyPositions := append([]uint32(nil), positions...)
	sort.Slice(copyPositions, func(i, j int) bool { return copyPositions[i] < copyPositions[j] })
	return Posting{DocID: docID, Frequency: uint32(len(copyPositions)), Positions: copyPositions}
}
