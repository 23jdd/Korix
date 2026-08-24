# 存储 Schema

所有键和值都存放在一个有序 KV 命名空间中。字段名、term 和 `Document.ID` 使用无 padding 的 URL-safe Base64 编码，避免 `/`、NUL、URL 等内容破坏 key 边界。下表中的 `{field}`、`{term}` 和 `{id}` 均指编码值。

| Key | Value | 作用 |
| --- | --- | --- |
| `document/{id}` | JSON Document | 原始文档，是 Rebuild 的事实来源 |
| `docmeta/{id}` | JSON metadata | 字符串 ID、各字段长度和 term vector |
| `term/{field}/{term}` | JSON TermInfo | 文档频率 DF 与总词频 |
| `posting/{field}/{term}/{id}` | 二进制 Posting | 字符串 ID、该文档中的 TF 与 position |
| `meta/global` | JSON GlobalStats | 文档数与各字段总长度 |

## Posting 编码

Posting 使用以下 uvarint 序列：

```text
idByteLength | idBytes | frequency | positionCount | firstPosition | delta₂ | delta₃ | ...
```

ID 先写 UTF-8 字节长度与原始字节，因此 posting 解码后得到的就是 `Document.ID`。position 先排序，第一项写绝对值，后续写差值。例如 `[1, 8, 9, 100]` 编为 `[1, 7, 1, 91]`。常见的 TF 和 position delta 只占一个字节。

每个文档独立一个 posting key，带来三个好处：更新/删除无需重写整个 term posting list；同一字符串 ID 可直接定位 posting；事务冲突范围清晰。代价是 key 开销高于段式连续 posting，未来可以在不改变查询接口的情况下加入 segment/merge codec。

## 结构解释

`document/*` 直接通过字符串 ID 访问原文；`docmeta/*` 为 BM25 提供 dl，同时保留 term vector 以支持精确更新。`term/*` 既是 Prefix/Fuzzy 的词典，也是 BM25 的 DF 来源。`posting/*` 保存字符串 ID、TF 和 Phrase 所需 position。`meta/global` 用总字段长度除以文档数得到 avgDL。

Koris 不维护 ID 映射表或数字 ID 分配器。Rebuild 直接从每个 Document 恢复相同的字符串 ID。

## 事务不变量

一次成功提交后必须满足：

- `meta/global.document_count == count(document/*)`；
- 每个 `docmeta/{id}` 都有对应的 `document/{id}`；
- `term.document_frequency == count(posting/{field}/{term}/*)`；
- posting 的 frequency 等于 position 数；
- 各字段总长度等于有效 docmeta 长度之和。

Writer 在同一事务中维护这些结构；`Check` 审计最关键的文档数与 DF 不变量。
