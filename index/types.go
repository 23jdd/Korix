// Package index owns documents, metadata and inverted-index persistence.
package index

import (
	"errors"

	"github.com/23jdd/Koris/inverted"
)

var (
	ErrInvalidDocument = errors.New("index: document ID and fields are required")
	ErrDocumentMissing = errors.New("index: document not found")
	ErrCorruptIndex    = errors.New("index: corrupt stored value")
)

// DocumentMetadata exposes BM25 length information without stored term
// vectors. The internal persisted representation additionally retains vectors
// so updates remain correct if analyzer configuration changes.
type DocumentMetadata struct {
	DocID      uint64            `json:"doc_id"`
	ExternalID string            `json:"external_id"`
	Lengths    map[string]uint32 `json:"lengths"`
}

type GlobalStats struct {
	DocumentCount    uint64            `json:"document_count"`
	TotalFieldLength map[string]uint64 `json:"total_field_length"`
	NextDocumentID   uint64            `json:"next_document_id"`
}

type storedDocumentMetadata struct {
	DocumentMetadata
	TermVectors map[string]map[string][]uint32 `json:"term_vectors"`
}

type TermStats struct {
	Field string
	Term  string
	Info  inverted.TermInfo
}

// ConsistencyReport summarizes a full read-only index audit.
type ConsistencyReport struct {
	Documents uint64
	Terms     uint64
	Postings  uint64
	Problems  []string
}

func (r ConsistencyReport) Valid() bool { return len(r.Problems) == 0 }
