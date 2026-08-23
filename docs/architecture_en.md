# Koris Architecture

[中文](architecture.md) | English

## Goals and Scope

Koris is an in-process library rather than a standalone service. The caller owns the index lifecycle, storage path, and analyzer configuration. The search core does not depend on a third-party search engine; Bbolt only implements the generic `Store` interface.

The design priorities are correctness, clear module boundaries, replaceable storage, recoverability, compact data, and query performance—in that order. The current implementation is intended for single-node, small-to-medium indexes, education, and further development. Large-scale sharding and distributed replication are outside the scope of the embedded core.

## Module Diagram

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

Dependencies always point toward small interfaces. Analysis knows nothing about Index; Query only depends on the read-only `Searcher`; Index knows about Store and Query; Store knows nothing about search semantics. As a result, Memory and Bbolt can be exchanged without changing upper-level search logic.

## Document Layer

The external model is:

```go
type Document struct {
    ID     string
    Fields map[string]string
}
```

An ID must be unique within an index. The first write allocates a monotonically increasing internal document ID. Adding the same external ID again performs an atomic update while preserving that internal ID. Fields are analyzed, counted, and queried independently, so the same term in `title` and `content` has separate dictionary entries and postings.

## Analysis Layer

The standard pipeline is:

```text
UTF-8 text → Tokenizer → []Token → Filter₁ → … → Filterₙ
```

Token offsets are UTF-8 byte offsets, allowing the original string to be sliced directly and safely. Positions are used by phrase queries. `StopWordFilter` removes tokens without renumbering the remaining positions, preserving the distance between terms.

Built-in tokenizers include:

- Whitespace: splits on Unicode whitespace and preserves punctuation;
- Simple: groups consecutive Unicode letters and digits;
- Standard: recognizes words, numbers, URLs, email addresses, and symbols;
- Chinese: uses a Trie dictionary with FMM, RMM, and BMM;
- HMM: accepts configurable BMES probabilities and applies Viterbi decoding to unknown words.

`TokenStream` exposes Next, Token, and Reset operations. The default implementation wraps the analyzer result as a stream. The interface allows future tokenizers to produce tokens incrementally and reduce peak memory for very large inputs without changing callers.

## Write Path

Add and Update perform the following steps inside one transaction:

1. Resolve the internal document ID from the external ID;
2. For an update, use the persisted term vector to delete old postings and reverse DF, TF, and document-length statistics;
3. Store the document;
4. Run the analyzer for each field and group positions by term;
5. Write compressed postings;
6. Update `TermInfo`, including document frequency and total term frequency;
7. Write document metadata and the term vector;
8. Update the document count, total field lengths, and next document ID;
9. Commit once.

Persisting term vectors is essential for correct updates. Even if the analyzer configuration changes after a restart, the writer can remove the exact terms produced by the old analyzer rather than trying to reconstruct them with the new rules.

AddBatch executes the same flow for every document in a single transaction. If any document fails, the whole batch is rolled back. Bbolt supplies crash-safe commits, while MemoryStore uses a copy-on-write map to provide equivalent semantics.

## Reading and Querying

A Term Query scans the posting-key prefix for a term. Fixed-width document IDs make those postings naturally ordered. A Match Query analyzes its input before combining terms with AND or OR. A Boolean Query composes arbitrary child queries. A Phrase Query checks ordered positions for documents common to all terms. Prefix and Fuzzy queries first expand the term dictionary and then score the expanded terms.

The query-string parser uses the precedence `NOT > AND > OR`; adjacent clauses imply OR. It supports parentheses, `field:value`, `"phrase"`, and a trailing `*` prefix wildcard.

## BM25

The contribution of one term to one document is:

```text
idf = ln(1 + (N - df + 0.5) / (df + 0.5))

score = idf × tf × (k1 + 1)
              ─────────────────────────────────────────
              tf + k1 × (1 - b + b × dl / avgdl)
```

The defaults are `k1=1.2` and `b=0.75`. N is the number of live documents; dl and avgdl are maintained per field. Multi-term scores are added, phrase matches receive a fixed 1.5 multiplier, and fuzzy matches are discounted by edit distance.

## Concurrency and Lifecycle

Index uses an RWMutex to protect management operations, while each Store implementation is also concurrency-safe. A Store write transaction becomes visible atomically, so concurrent readers never observe half a posting or incomplete metadata. Separate reads within one query may span two fully committed versions; this provides low-overhead read-committed semantics.

An index must not be reused after `Close`. BboltStore copies matching keys and values inside the view transaction, so iterators do not retain database pages or block writers.

## Recovery and Auditing

`Check` compares the global document count with stored document metadata and verifies each term's document frequency against its actual posting count. `Rebuild` treats `document/*` as the only source of truth, removes all derived keys, and atomically recreates them with the current analyzer. If an original document is corrupt, Rebuild fails and rolls the transaction back.

## Advanced Capabilities

- Highlighting reuses analyzer offsets, merges nearby matches, and escapes HTML by default;
- Faceting counts exact stored field values and is intended for low-cardinality fields;
- Fuzzy matching uses Unicode Levenshtein distance with a maximum edit distance of three;
- Posting positions use delta encoding plus unsigned varints;
- `PostingIterator.SkipTo` performs binary search, while `SkipListIterator` provides explicit sqrt(N) skip points.
