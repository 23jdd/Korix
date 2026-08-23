package Koris

import (
	"github.com/23jdd/Koris/document"
	indexpkg "github.com/23jdd/Koris/index"
	"github.com/23jdd/Koris/query"
)

type (
	// Index 是 index.Index 的根包别名。
	Index = indexpkg.Index
	// Option 配置 Analyzer 或 BM25。
	Option = indexpkg.Option
	// Document 是 document.Document 的根包别名。
	Document = document.Document
	// Hit 是 query.Hit 的根包别名。
	Hit = query.Hit
	// Query 是所有可执行查询的接口别名。
	Query = query.Query
)

var (
	// WithAnalyzer 设置默认字段 Analyzer。
	WithAnalyzer = indexpkg.WithAnalyzer
	// WithFieldAnalyzer 为指定字段设置 Analyzer。
	WithFieldAnalyzer = indexpkg.WithFieldAnalyzer
	// WithBM25 设置相关性评分参数。
	WithBM25 = indexpkg.WithBM25
)
