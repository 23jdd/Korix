# Design Decisions

[中文](design-decisions.md) | English

## Why Use a Store Interface?

The index layer only depends on `Get`, `Put`, `Delete`, `Scan`, and `Transaction`. Tests can therefore use a zero-configuration MemoryStore, production can use Bbolt, and future implementations can target Pebble, Badger, or a remote transactional key-value store without changing analyzers or queries. Scan returns a snapshot iterator so the lifetime of an underlying transaction never leaks to the caller.

## Why Use One Posting Key per Document?

Koris prioritizes embedded writes and implementation clarity. Per-document posting keys allow Update and Delete to touch only related postings and use key-value ordering for direct iteration. Large read-only indexes often use immutable segments, block compression, and merging. Those are compatible future optimizations, not prerequisites for the current correctness model.

## Why Are Offsets Measured in Bytes?

Go indexes strings by UTF-8 byte offsets. Byte offsets allow zero-conversion slicing and highlighting; tokenizers classify Unicode text at the rune level and map boundaries back to bytes. Rune offsets would be more intuitive to some users, but every highlight operation would need to scan the text again.

## Why Persist Term Vectors?

If only the original text were retained, Update would have to reanalyze the old document with the current analyzer. Disabling a stop word, changing a Chinese dictionary, or replacing the stemmer could produce different terms and leave ghost postings behind. Persisting old term-to-position mappings trades a modest amount of space for reliable updates.

## Why Does the Query Layer Define Searcher?

If Query imported Index directly while Index also exposed Search(Query), Go would have a package cycle. A read-only Searcher makes queries independently testable and wrappable, and it can support future segment readers. Index implements the interface implicitly through its method set.

## Boolean Semantics

- Every Must clause must match, and its score is added;
- Should clauses form a union when there is no Must clause and only add score to existing candidates when Must is present;
- MustNot removes matching candidates;
- A query containing only MustNot starts from all live documents.

## Prefix and Fuzzy Costs

Because field and term components are Base64-encoded, a raw string prefix is not also an encoded-key prefix. Prefix currently scans a field dictionary and filters decoded terms. Fuzzy also scans the dictionary and computes cutoff-aware Unicode Levenshtein distance. These approaches suit small-to-medium dictionaries. A large dictionary can place an FST or Trie behind `Terms` without changing the query API.

## Chinese Segmentation

Dictionary mode uses a Trie. FMM selects the longest match from the left, RMM selects the longest match from the right, and BMM favors fewer tokens and then fewer single-character tokens. HMM uses configurable BMES log probabilities and Viterbi decoding. The default model only contains boundary priors; real corpora should supply trained emission probabilities.

## Known Boundaries

- There is no distributed lock beyond cross-process writer coordination provided by the storage backend; BboltStore itself guarantees single-file write transactions;
- Faceting reads original documents and is not suitable for million-scale, high-cardinality fields;
- Highlighting produces fragments rather than performing full, complex HTML DOM highlighting;
- StandardTokenizer is a practical Unicode tokenizer and does not claim full Lucene UAX #29 compatibility;
- Stemmer is a conservative rule-based implementation, not complete Porter2;
- The query parser does not support regular expressions, range queries, or field boosts.

All of these boundaries sit behind stable interfaces and can be improved incrementally.
