# Storage Schema

[中文](storage-schema.md) | English

All keys and values live in one ordered key-value namespace. Field names, terms, and external IDs use unpadded URL-safe Base64 encoding so that `/`, NUL, URLs, and other input cannot break key boundaries. In the table below, `{field}`, `{term}`, and `{id}` refer to encoded components; `{docID}` is a fixed-width, 16-digit hexadecimal number.

| Key | Value | Purpose |
| --- | --- | --- |
| `document/{docID}` | JSON Document | Original document and source of truth for Rebuild |
| `id/{id}` | Decimal uint64 | Maps an external string ID to an internal document ID |
| `docmeta/{docID}` | JSON metadata | External ID, per-field lengths, and term vectors |
| `term/{field}/{term}` | JSON TermInfo | Document frequency and total term frequency |
| `posting/{field}/{term}/{docID}` | Binary Posting | TF and positions for this term in this document |
| `meta/global` | JSON GlobalStats | Document count, total field lengths, and next document ID |

## Posting Encoding

A posting is encoded as the following unsigned-varint sequence:

```text
docID | frequency | positionCount | firstPosition | delta₂ | delta₃ | ...
```

Positions are sorted first. The initial position is absolute; every following position is stored as a delta from its predecessor. For example, `[1, 8, 9, 100]` becomes `[1, 7, 1, 91]`. Common small document IDs, term frequencies, and position deltas use only one byte each.

Each document has a separate posting key. This has three benefits: Update and Delete do not need to rewrite an entire term posting list; prefix scans naturally produce document-ID order; and transaction conflicts remain localized. The tradeoff is more key overhead than contiguous segment postings. A segment-and-merge codec can be added later without changing the query interface.

## Structure Rationale

`document/*` and `id/*` provide user-facing access in both directions. `docmeta/*` supplies BM25 document lengths and retains term vectors for exact updates. `term/*` serves as both the Prefix/Fuzzy dictionary and the source of BM25 document frequency. `posting/*` stores TF and Phrase positions. `meta/global` yields avgDL by dividing each total field length by the document count.

Gaps between document IDs are normal. Delete never reuses an ID, preventing an old reference from pointing at a new document. Rebuild may renumber internal IDs, but external IDs remain unchanged.

## Transaction Invariants

After a successful commit, the following conditions must hold:

- `meta/global.document_count == count(document/*)`;
- every `docmeta` entry has a corresponding document and external-ID mapping;
- `term.document_frequency == count(posting/{field}/{term}/*)`;
- a posting's frequency equals its number of positions;
- every total field length equals the sum of that field's lengths in live document metadata.

The Writer maintains these structures in the same transaction. `Check` audits the most important document-count and document-frequency invariants.
