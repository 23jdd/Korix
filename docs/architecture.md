# Koris 整体设计

## 目标与边界

Koris 是进程内库而不是独立服务。调用者拥有索引生命周期、存储路径和 analyzer 配置。搜索核心不依赖第三方搜索引擎；Bbolt 仅实现通用 `Store`。

设计优先级依次是正确性、清晰的模块边界、可替换存储、可恢复性，再是紧凑数据与查询性能。当前实现适合单机、中小索引和教学/二次开发；超大规模分片、分布式复制不属于嵌入式核心范围。

## 模块图

```text
API (Koris / index.Index)
        │
        ├── Document ───────────────► Store ──► Memory / Bbolt
        │                                ▲
        ├── Analysis                     │
        │   Tokenizer → TokenStream → Filter
        │       │                        │
        ▼       ▼                        │
Index Writer → term vector → posting/term/meta
        │
        ▼
Index Reader → Query → Boolean/Phrase iterator → BM25 → Hit
                                      │
                                      ├── Highlight
                                      └── Facet
```

依赖方向始终朝向小接口：Analysis 不知道 Index；Query 只依赖只读 `Searcher`；Index 知道 Store 和 Query；Store 不知道搜索语义，因此 Memory/Bbolt 可以无差别替换。

## 文档层

外部模型为：

```go
type Document struct {
    ID     string
    Fields map[string]string
}
```

ID 在索引内唯一，并在文档、metadata、posting、Hit 和所有 API 中直接使用同一个字符串，不存在内部数字 DocID。同 ID 再次 Add 是原子 Update。字段分别分析、统计与检索，因此同一个 term 在 title 和 content 中拥有不同词典项和 posting。

## 分析层

标准链路为：

```text
UTF-8 text → Tokenizer → []Token → Filter₁ → … → Filterₙ
```

Token 的 offset 是 UTF-8 字节偏移，可直接、安全地切片原字符串；Position 用于 Phrase Query。StopWordFilter 删除 token 但不重排 Position，因此保留词间距离。

内置 tokenizer：

- Whitespace：按 Unicode 空白切分，保留标点；
- Simple：连续 Unicode 字母/数字；
- Standard：识别字词、数字、URL、Email 和符号；
- Chinese：Trie 词典与 FMM/RMM/BMM；
- HMM：可配置 BMES 概率并使用 Viterbi 解码未登录词。

`TokenStream` 提供 Next/Token/Reset 消费方式。当前默认实现将 analyzer 结果包装成流；接口允许未来 tokenizer 直接增量产出，以降低超大文本峰值内存。

## 写入路径

Add/Update 的事务内步骤：

1. 用 `Document.ID` 定位文档 metadata；
2. 若为更新，根据持久化 term vector 删除旧 posting 并回退 DF/TF/文档长度；
3. 保存文档；
4. 对每个字段运行 analyzer，按 term 聚合 position；
5. 写入带字符串 ID 的压缩 posting；
6. 更新 TermInfo（DF、总 TF）；
7. 写入文档 metadata 与 term vector；
8. 更新文档数和各字段总长度；
9. 一次提交。

持久化 term vector 是更新正确性的关键：即使进程重启后 analyzer 配置改变，也能精确删除旧索引数据，而不必用新 analyzer 猜测旧 token。

AddBatch 在一个事务里执行上述流程，任一文档失败则整个批次回滚。Bbolt 提供崩溃安全提交；MemoryStore 使用 copy-on-write map 模拟相同语义。

## 读取与查询

Term Query 前缀扫描 term 的 posting key，并按字符串 ID 排序 posting。Match Query 先分析文本，再按 AND/OR 合并 term 得分。Boolean Query 组合任意子查询；Phrase Query 对相同文档 ID 检查有序 position；Prefix/Fuzzy Query 先扩展词典，再执行 term scoring。

查询字符串 parser 的优先级为 `NOT > AND > OR`，相邻 clause 隐式 OR。支持括号、`field:value`、`"phrase"` 与尾部 `*`。

## BM25

每个 term/document 的贡献为：

```text
idf = ln(1 + (N - df + 0.5) / (df + 0.5))

score = idf × tf × (k1 + 1)
              ─────────────────────────────────────────
              tf + k1 × (1 - b + b × dl / avgdl)
```

默认 `k1=1.2`、`b=0.75`。N 是有效文档数；dl/avgdl 按字段维护。多 term 得分相加，Phrase 命中额外乘 1.5，Fuzzy 按编辑距离衰减。

## 并发与生命周期

Index 用 RWMutex 保护管理操作，Store 自身也支持并发。一次 Store 写事务对读者原子可见，因此并发查询不会看到半条 posting 或半份 metadata。查询中的多个独立读取可能跨越两个已完整提交的版本；这是低开销的 read-committed 语义。

调用 `Close` 后不可再使用索引。BboltStore 的迭代器在 view 事务内复制匹配键值，所以不会持有数据库页面或阻塞写事务。

## 恢复与审计

`Check` 比较全局文档数、term DF 和实际 posting 数。`Rebuild` 仅以 `document/*` 原始文档为事实来源，删除所有派生键后用当前 analyzer 原子重建，并始终保留原字符串 ID。它也会清理旧版 `id/*` 映射，可用于把早期数字 ID schema 迁移为字符串 ID schema。若原始文档自身损坏，Rebuild 会失败并回滚。

## 高级能力

- 高亮复用 analyzer offset，合并临近命中并默认 HTML escape；
- Facet 对查询命中的原始字段值计数，适合低基数字段；
- Fuzzy 使用 Unicode Levenshtein，最大编辑距离限制为 3；
- posting 内 position 使用 delta + uvarint；
- `PostingIterator.SkipTo` 二分跳转，`SkipListIterator` 提供 sqrt(N) 显式跳点。
