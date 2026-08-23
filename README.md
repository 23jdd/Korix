# Koris

[中文](README_CN.md) | English

Koris is a lightweight embedded full-text search engine implemented from scratch in Go. It provides its own analysis pipeline, inverted index, term dictionary, postings, query execution, and BM25 ranking. It does not call third-party search engines such as Bleve or Lucene; its only runtime dependency, `bbolt`, is used solely as an optional persistent key-value store.

## Features

- Document and field-level search with user-defined string IDs and stable internal `uint64` document IDs
- Unicode-aware Standard, Whitespace, and Simple tokenizers
- Trie-based Chinese dictionaries, forward/reverse/bidirectional maximum matching, and BMES-HMM Viterbi segmentation
- Lowercase, stop-word, and English stemming filters, plus reusable token streams
- Term dictionary, compressed postings, term frequency, positions, document lengths, and global statistics
- Term, Match, Boolean, Phrase/Slop, Prefix, and Fuzzy queries
- Okapi BM25 ranking, lightweight explanations, and query-string parsing
- Highlighting, low-cardinality field facets, and posting skip iterators
- MemoryStore and BboltStore with atomic batches, rollback, update, delete, and rebuild support
- Concurrent-safe reads and writes, consistency checks, unit tests, race tests, and benchmarks

## Quick Start

```go
package main

import (
    "fmt"

    koris "github.com/23jdd/Koris"
    "github.com/23jdd/Koris/document"
    "github.com/23jdd/Koris/query"
)

func main() {
    idx, err := koris.New()
    if err != nil {
        panic(err)
    }
    defer idx.Close()

    _, err = idx.AddBatch([]document.Document{
        {ID: "1", Fields: map[string]string{
            "title": "Learning Go",
            "content": "Go is a fast programming language",
        }},
        {ID: "2", Fields: map[string]string{
            "title": "Distributed Systems",
            "content": "Build a distributed system with golang",
        }},
    })
    if err != nil {
        panic(err)
    }

    hits, err := idx.Search(query.MatchQuery{
        Field: "content", Text: "distributed golang", Operator: query.AND,
    }, 10)
    if err != nil {
        panic(err)
    }
    fmt.Println(hits[0].ID, hits[0].Score)
}
```

For a persistent index, replace `koris.New()` with:

```go
idx, err := koris.Open("./data/koris.db")
```

The query-string API supports fields, phrases, prefixes, and Boolean operators:

```go
hits, err := idx.SearchString(
    `content:"distributed system" OR title:go* AND NOT content:legacy`,
    "content",
    20,
)
```

## Common Queries

```go
query.TermQuery{Field: "content", Term: "golang"}
query.MatchQuery{Field: "content", Text: "golang database", Operator: query.OR}
query.PhraseQuery{Field: "content", Text: "distributed system", Slop: 0}
query.PrefixQuery{Field: "content", Prefix: "gol"}
query.FuzzyQuery{Field: "content", Term: "gloang", MaxEdits: 2}
query.BooleanQuery{
    Must: []query.Query{query.TermQuery{Field: "content", Term: "golang"}},
    MustNot: []query.Query{query.TermQuery{Field: "content", Term: "legacy"}},
}
```

`TermQuery` accepts an exact term that has already been analyzed. Use `MatchQuery` or `SearchString` for raw user input.

## Analyzer Configuration

Indexing and querying share the same field analyzer:

```go
chinese := analysis.PipelineAnalyzer{
    Tokenizer: tokenizer.NewChineseTokenizer(dictionary).
        WithMode(tokenizer.BidirectionalMaximumMatching),
    Filters: []analysis.TokenFilter{filter.LowercaseFilter{}},
}

idx, err := koris.New(koris.WithFieldAnalyzer("content", chinese))
```

For statistical segmentation, use `tokenizer.NewHMMTokenizer(model)`. `HMMModel` accepts trained BMES start, transition, and emission log probabilities.

## Consistency and Recovery

Every Add, Update, Delete, and AddBatch operation updates the document, postings, dictionary, and statistics within one Store transaction. `Check` performs a full read-only audit, while `Rebuild` atomically recreates all derived data from the original documents:

```go
report, err := idx.Check()
if err == nil && !report.Valid() {
    err = idx.Rebuild()
}
```

## Tests and Benchmarks

```bash
go test ./...
go test -race ./...
go test -run '^$' -bench . -benchmem ./...
```

For implementation details, see the [architecture](docs/architecture_en.md), [storage schema](docs/storage-schema_en.md), and [design decisions](docs/design-decisions_en.md). A runnable example is available under `examples/basic`.

## License

[MIT](LICENSE)
