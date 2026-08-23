# 存储 Schema

所有键和值都存放在一个有序 KV 命名空间中。字段名、term 和外部 ID 使用无 padding 的 URL-safe Base64 编码，避免 `/`、NUL、URL 等内容破坏 key 边界。下表中的 `{field}`、`{term}`、`{id}` 均指编码值，`{docID}` 是固定 16 位十六进制。

| Key | Value | 作用 |
| --- | --- | --- |
| `document/{docID}` | JSON Document | 原始文档，是 Rebuild 的事实来源 |
| `id/{id}` | 十进制 uint64 | 外部字符串 ID 到内部 DocID |
| `docmeta/{docID}` | JSON metadata | 外部 ID、各字段长度和 term vector |
| `term/{field}/{term}` | JSON TermInfo | 文档频率 DF 与总词频 |
| `posting/{field}/{term}/{docID}` | 二进制 Posting | 该文档中的 TF 与 position |
| `meta/global` | JSON GlobalStats | 文档数、各字段总长度、next DocID |

## Posting 编码

Posting 使用以下 uvarint 序列：

```text
docID | frequency | positionCount | firstPosition | delta₂ | delta₃ | ...
```

position 先排序，第一项写绝对值，后续写差值。例如 `[1, 8, 9, 100]` 编为 `[1, 7, 1, 91]`。常见的小 DocID、TF 和 position delta 只占一个字节。

每个文档独立一个 posting key，带来三个好处：更新/删除无需重写整个 term posting list；前缀 scan 天然按 DocID 顺序；事务冲突范围清晰。代价是 key 开销高于段式连续 posting，未来可以在不改变查询接口的情况下加入 segment/merge codec。

## 结构解释

`document/*` 和 `id/*` 提供双向用户访问；`docmeta/*` 为 BM25 提供 dl，同时保留 term vector 以支持精确更新。`term/*` 既是 Prefix/Fuzzy 的词典，也是 BM25 的 DF 来源。`posting/*` 保存 TF 和 Phrase 所需 position。`meta/global` 用总字段长度除以文档数得到 avgDL。

文档数不连续是正常情况：Delete 不复用 DocID，避免旧引用误指向新文档。Rebuild 可以重新编号内部 DocID，但外部 ID 不变。

## 事务不变量

一次成功提交后必须满足：

- `meta/global.document_count == count(document/*)`；
- 每个 `docmeta` 都有对应 document 和 id 映射；
- `term.document_frequency == count(posting/{field}/{term}/*)`；
- posting 的 frequency 等于 position 数；
- 各字段总长度等于有效 docmeta 长度之和。

Writer 在同一事务中维护这些结构；`Check` 审计最关键的文档数与 DF 不变量。
