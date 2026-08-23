package index

import "github.com/23jdd/Koris/highlight"

// Highlight 读取命中文档指定字段，并使用该字段的 Analyzer 生成片段。
// terms 应是用户想强调的原始词，highlight.Extract 会再次分析以匹配索引 term。
func (i *Index) Highlight(docID uint64, field string, terms []string, options highlight.Options) ([]highlight.Fragment, error) {
	doc, err := i.Document(docID)
	if err != nil {
		return nil, err
	}
	return highlight.Extract(i.Analyzer(field), doc.Fields[field], terms, options), nil
}
