# Storage Schema

[中文](storage-schema.md) | English

All keys and values live in one ordered key-value namespace. Field names, terms, and `Document.ID` use unpadded URL-safe Base64 encoding so that `/`, NUL, URLs, and other input cannot break key boundaries. In the table below, `{field}`, `{term}`, and `{id}` refer to encoded components.

| Key | Value | Purpose |
| --- | --- | --- |
| `document/{id}` | JSON Document | Original document and source of truth for Rebuild |
| `docmeta/{id}` | JSON metadata | String ID, per-field lengths, and term vectors |
| `term/{field}/{term}` | JSON TermInfo | Document frequency and total term frequency |
| `posting/{field}/{term}/{id}` | Binary Posting | String ID, TF, and positions for this term in this document |
| `meta/global` | JSON GlobalStats | Document count and total field lengths |

## Posting Encoding

A posting is encoded as the following unsigned-varint sequence:

```text
idByteLength | idBytes | frequency | positionCount | firstPosition | delta₂ | delta₃ | ...
```

The UTF-8 ID byte length and the original ID bytes are written first, so a decoded posting directly contains `Document.ID`. Positions are sorted; the initial position is absolute, and every following position is stored as a delta from its predecessor. For example, `[1, 8, 9, 100]` becomes `[1, 7, 1, 91]`. Common term frequencies and position deltas use only one byte each.

Each document has a separate posting key. This has three benefits: Update and Delete do not need to rewrite an entire term posting list; the same string ID directly locates a posting; and transaction conflicts remain localized. The tradeoff is more key overhead than contiguous segment postings. A segment-and-merge codec can be added later without changing the query interface.

## Structure Rationale

`document/*` provides direct access through the string ID. `docmeta/*` supplies BM25 document lengths and retains term vectors for exact updates. `term/*` serves as both the Prefix/Fuzzy dictionary and the source of BM25 document frequency. `posting/*` stores the string ID, TF, and Phrase positions. `meta/global` yields avgDL by dividing each total field length by the document count.

Koris has no ID mapping table or numeric ID allocator. Rebuild restores the same string ID directly from each Document.

## Transaction Invariants

After a successful commit, the following conditions must hold:

- `meta/global.document_count == count(document/*)`;
- every `docmeta/{id}` entry has a corresponding `document/{id}`;
- `term.document_frequency == count(posting/{field}/{term}/*)`;
- a posting's frequency equals its number of positions;
- every total field length equals the sum of that field's lengths in live document metadata.

The Writer maintains these structures in the same transaction. `Check` audits the most important document-count and document-frequency invariants.
