package index

import (
	"sync"

	"github.com/23jdd/Koris/analysis"
	"github.com/23jdd/Koris/analysis/standard"
	"github.com/23jdd/Koris/scoring"
	"github.com/23jdd/Koris/storage"
)

// Index 协调 Analyzer、Store、Writer、Reader 与 BM25 配置。
//
// Store 事务保证一次写入原子提交；mu 让复合管理操作在不同 Store 实现上都具备
// 一致的并发语义。fieldAnalyzers 在构造完成后视为只读，可被并发查询安全复用。
type Index struct {
	store           storage.Store
	defaultAnalyzer analysis.Analyzer
	fieldAnalyzers  map[string]analysis.Analyzer
	scorer          scoring.BM25
	mu              sync.RWMutex
}

type Option func(*Index)

// WithAnalyzer 设置没有字段专用配置时使用的默认 Analyzer。nil 会被忽略。
// Analyzer 同时用于写入和查询，打开已有索引时应保持与原配置一致。
func WithAnalyzer(analyzer analysis.Analyzer) Option {
	return func(index *Index) {
		if analyzer != nil {
			index.defaultAnalyzer = analyzer
		}
	}
}

// WithFieldAnalyzer 为单个字段覆盖默认 Analyzer。例如 title 可使用轻量分析链，
// content 使用中文词典分词。空字段名或 nil Analyzer 会被忽略。
func WithFieldAnalyzer(field string, analyzer analysis.Analyzer) Option {
	return func(index *Index) {
		if field != "" && analyzer != nil {
			index.fieldAnalyzers[field] = analyzer
		}
	}
}

// WithBM25 覆盖默认 k1/b 参数。Score 会对越界参数执行安全回退。
func WithBM25(bm25 scoring.BM25) Option { return func(index *Index) { index.scorer = bm25 } }

// New 使用给定 Store 打开或创建索引。store 为 nil 时创建 MemoryStore；已有
// meta/global 会原样保留，新索引则在事务中写入初始统计。
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
	// 初始化也使用事务，避免两个并发 opener 观察到半初始化 metadata。
	return i.store.Transaction(func(tx storage.Tx) error {
		if _, err := tx.Get(globalMetaKey); err == nil {
			return nil
		} else if err != storage.ErrNotFound {
			return err
		}
		return putJSON(tx, globalMetaKey, GlobalStats{TotalFieldLength: make(map[string]uint64)})
	})
}

// Analyzer 返回字段专用 Analyzer；没有覆盖时返回默认 Analyzer。
func (i *Index) Analyzer(field string) analysis.Analyzer {
	if analyzer := i.fieldAnalyzers[field]; analyzer != nil {
		return analyzer
	}
	return i.defaultAnalyzer
}

// BM25 返回当前评分配置，使 Index 满足 query.Searcher。
func (i *Index) BM25() scoring.BM25 { return i.scorer }

// Close 等待正在进行的管理操作结束后关闭底层 Store。关闭后的 Index 不可复用。
func (i *Index) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store.Close()
}
