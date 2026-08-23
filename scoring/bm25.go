// Package scoring implements document ranking functions.
package scoring

import "math"

// Scorer is implemented by BM25-compatible ranking strategies.
type Scorer interface {
	Score(tf, df, documentCount uint64, documentLength, averageLength float64) float64
}

// BM25 is Okapi BM25. K1 controls term-frequency saturation and B controls
// document-length normalization.
type BM25 struct {
	K1 float64
	B  float64
}

func DefaultBM25() BM25 { return BM25{K1: 1.2, B: 0.75} }

// Score returns a non-negative Robertson/Sparck Jones BM25 contribution.
func (b BM25) Score(tf, df, documentCount uint64, documentLength, averageLength float64) float64 {
	if tf == 0 || df == 0 || documentCount == 0 {
		return 0
	}
	if b.K1 <= 0 {
		b.K1 = 1.2
	}
	if b.B < 0 || b.B > 1 {
		b.B = 0.75
	}
	if averageLength <= 0 {
		averageLength = 1
	}
	idf := math.Log(1 + (float64(documentCount)-float64(df)+0.5)/(float64(df)+0.5))
	numerator := float64(tf) * (b.K1 + 1)
	denominator := float64(tf) + b.K1*(1-b.B+b.B*documentLength/averageLength)
	return idf * numerator / denominator
}
