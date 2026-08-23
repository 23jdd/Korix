package index

import (
	"sync"

	"github.com/23jdd/Koris/analysis"
	"github.com/23jdd/Koris/analysis/standard"
	"github.com/23jdd/Koris/scoring"
	"github.com/23jdd/Koris/storage"
)

// Index coordinates analyzers and a persistence backend. Store transactions
// provide durability; the mutex makes compound operations safe across stores.
type Index struct {
	store           storage.Store
	defaultAnalyzer analysis.Analyzer
	fieldAnalyzers  map[string]analysis.Analyzer
	scorer          scoring.BM25
	mu              sync.RWMutex
}

type Option func(*Index)

func WithAnalyzer(analyzer analysis.Analyzer) Option {
	return func(index *Index) {
		if analyzer != nil {
			index.defaultAnalyzer = analyzer
		}
	}
}

func WithFieldAnalyzer(field string, analyzer analysis.Analyzer) Option {
	return func(index *Index) {
		if field != "" && analyzer != nil {
			index.fieldAnalyzers[field] = analyzer
		}
	}
}

func WithBM25(bm25 scoring.BM25) Option { return func(index *Index) { index.scorer = bm25 } }

func New(store storage.Store, options ...Option) (*Index, error) {
	if store == nil {
		store = storage.NewMemoryStore()
	}
	idx := &Index{
		store: store, defaultAnalyzer: standard.New(),
		fieldAnalyzers: make(map[string]analysis.Analyzer), scorer: scoring.DefaultBM25(),
	}
	for _, option := range options {
		option(idx)
	}
	if err := idx.initialize(); err != nil {
		return nil, err
	}
	return idx, nil
}

func (i *Index) initialize() error {
	return i.store.Transaction(func(tx storage.Tx) error {
		if _, err := tx.Get(globalMetaKey); err == nil {
			return nil
		} else if err != storage.ErrNotFound {
			return err
		}
		return putJSON(tx, globalMetaKey, GlobalStats{TotalFieldLength: make(map[string]uint64), NextDocumentID: 1})
	})
}

func (i *Index) Analyzer(field string) analysis.Analyzer {
	if analyzer := i.fieldAnalyzers[field]; analyzer != nil {
		return analyzer
	}
	return i.defaultAnalyzer
}

func (i *Index) BM25() scoring.BM25 { return i.scorer }

func (i *Index) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store.Close()
}
