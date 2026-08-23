package inverted

// TermInfo is the persistent term dictionary value.
type TermInfo struct {
	DocumentFrequency uint64 `json:"document_frequency"`
	TotalFrequency    uint64 `json:"total_frequency"`
}
