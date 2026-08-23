package Koris

import (
	"github.com/23jdd/Koris/document"
	indexpkg "github.com/23jdd/Koris/index"
	"github.com/23jdd/Koris/query"
)

type (
	Index    = indexpkg.Index
	Option   = indexpkg.Option
	Document = document.Document
	Hit      = query.Hit
	Query    = query.Query
)

var (
	WithAnalyzer      = indexpkg.WithAnalyzer
	WithFieldAnalyzer = indexpkg.WithFieldAnalyzer
	WithBM25          = indexpkg.WithBM25
)
