// Package scoring 实现与查询类型无关的相关性评分算法。
package scoring

import "math"

// Scorer 是 BM25 统计输入形式的评分接口。自定义实现可替换饱和函数或长度归一化，
// 只要保持相同的 TF、DF、N、DL、avgDL 语义。
type Scorer interface {
	Score(tf, df, documentCount uint64, documentLength, averageLength float64) float64
}

// BM25 实现 Okapi BM25。
// K1 控制词频饱和速度：值越大，高 TF 的额外收益越明显；B 控制文档长度归一化，
// 0 表示完全忽略长度，1 表示完整归一化。
type BM25 struct {
	K1 float64
	B  float64
}

// DefaultBM25 返回文献和常见搜索实现采用的 k1=1.2、b=0.75。
func DefaultBM25() BM25 { return BM25{K1: 1.2, B: 0.75} }

// Score 计算一个 term 对一篇文档的非负 BM25 贡献。
//
// tf 是文档内词频，df 是包含该 term 的文档数，documentCount 是索引文档总数；
// documentLength 与 averageLength 必须来自同一字段。无有效统计时返回 0，非法
// K1/B 则回退默认值，避免配置错误产生 NaN 或负分。
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
	// +1 的 Robertson/Sparck Jones 变体保证极常见 term 的 IDF 仍为非负。
	idf := math.Log(1 + (float64(documentCount)-float64(df)+0.5)/(float64(df)+0.5))
	numerator := float64(tf) * (b.K1 + 1)
	denominator := float64(tf) + b.K1*(1-b.B+b.B*documentLength/averageLength)
	return idf * numerator / denominator
}
