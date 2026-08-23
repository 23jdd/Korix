# Koris

Koris 是一个使用 Go 从零实现的轻量级嵌入式全文搜索引擎
## 功能

- 文档与字段级检索，用户字符串 ID + 稳定内部 `uint64` DocID
- Unicode Standard / Whitespace / Simple tokenizer
- Trie 中文词典、正向/逆向/双向最大匹配、BMES-HMM Viterbi 分词
- Lowercase、StopWord、英文词干过滤器和可复用 TokenStream
- 词典、压缩 posting、词频、位置、文档长度和全局统计
- Term、Match、Boolean、Phrase/Slop、Prefix 和 Fuzzy 查询
- Okapi BM25 排序、结果解释、查询字符串解析
- 高亮、低基数字段分面、posting 跳表迭代
- MemoryStore 和 BboltStore，原子批量写入、回滚、更新、删除与重建
- 并发安全读写、完整性检查、单元测试和 Benchmark

## 快速开始

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

持久化索引只需将 `koris.New()` 换成：

```go
idx, err := koris.Open("./data/koris.db")
```

查询字符串 API 支持字段、短语、前缀和布尔运算：

```go
hits, err := idx.SearchString(
    `content:"distributed system" OR title:go* AND NOT content:legacy`,
    "content",
    20,
)
```

## 常用查询

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

TermQuery 接收已经分析过的精确 term；原始用户文本应使用 MatchQuery 或 `SearchString`。

## 分析器配置

索引与查询共用同一字段分析器：

```go
chinese := analysis.PipelineAnalyzer{
    Tokenizer: tokenizer.NewChineseTokenizer(dictionary).
        WithMode(tokenizer.BidirectionalMaximumMatching),
    Filters: []analysis.TokenFilter{filter.LowercaseFilter{}},
}

idx, err := koris.New(koris.WithFieldAnalyzer("content", chinese))
```

需要统计模型时可改用 `tokenizer.NewHMMTokenizer(model)`；`HMMModel` 接收训练后的 BMES 起始、转移和发射对数概率。

## 一致性与恢复

每个 Add、Update、Delete 和 AddBatch 都在一个 Store 事务内更新文档、posting、词典与统计。`Check` 进行只读全量审计；`Rebuild` 从原始文档原子重建全部派生数据：

```go
report, err := idx.Check()
if err == nil && !report.Valid() {
    err = idx.Rebuild()
}
```

## 测试与 Benchmark

```bash
go test ./...
go test -race ./...
go test -run '^$' -bench . -benchmem ./...
```

详细设计参见 [架构设计](docs/architecture.md)、[存储 Schema](docs/storage-schema.md) 和 [设计决策](docs/design-decisions.md)。可运行示例位于 `examples/basic`。

## 许可证

[MIT](LICENSE)
