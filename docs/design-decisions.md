# 设计决策

## 为什么使用 Store 接口

索引层只依赖 `Get/Put/Delete/Scan/Transaction`，因此测试可使用零配置 MemoryStore，生产可使用 Bbolt，未来也能实现 Pebble、Badger 或远程事务 KV，而不改变 analyzer/query。Scan 返回快照迭代器，避免把底层事务生命周期泄漏给调用者。

## 为什么 posting 按文档拆 key

Koris 以嵌入式写入与实现清晰度为优先。按文档拆分能让 Update/Delete 只触碰相关 posting，并借助 KV 排序直接迭代。大型只读索引通常会用 immutable segment、块压缩和 merge；那是后续兼容优化，不是当前正确性的前提。

## 为什么 offset 使用字节

Go 字符串以 UTF-8 字节索引。字节 offset 可零转换切片并用于高亮；tokenizer 在 rune 层识别 Unicode 后映射回字节边界。以 rune offset 对用户更直观，但每次高亮都需要再次扫描。

## 为什么 term vector 持久化

仅保存原文会让 Update 必须用“当前” analyzer 重算旧 token。如果停用 stopword、更新中文字典或更换 stemming，重算结果与旧 posting 不一致并造成幽灵 term。存储旧 term→position 是少量空间换取更新可靠性。

## 为什么查询层定义 Searcher

如果 Query 直接导入 Index，而 Index 又提供 Search(Query)，Go 会出现包循环。只读 Searcher 让查询可单测、可包装、也能用于未来 segment reader。Index 通过方法集合隐式实现它。

## Boolean 的语义

- Must 全部命中并累加得分；
- Should 在没有 Must 时形成并集，有 Must 时只给已有候选加分；
- MustNot 从候选中删除；
- 只有 MustNot 时，候选从全部有效文档开始。

## Prefix 与 Fuzzy 的成本

字段/term 使用 Base64 key 后，原始字符串前缀不再等于编码前缀。因此 Prefix 当前扫描字段词典并在解码后过滤。Fuzzy 同样扫描词典并计算带 cutoff 的 Unicode Levenshtein。它们适用于中小词典；大词典可在 `Terms` 后面加入 FST/Trie，而查询 API 不变。

## 中文分词

词典模式使用 Trie：FMM 从左选最长，RMM 从右选最长，BMM 优先 token 数更少、单字更少的结果。HMM 使用可配置 BMES log probability 与 Viterbi；默认模型只有边界先验，真实语料应提供训练后的发射概率。

## 已知边界

- 当前没有跨进程并发 writer 协调之外的分布式锁；BboltStore 自身保证单文件写事务；
- Facet 读取原始文档，不适合百万级高基数字段；
- Highlight 生成片段而非完整复杂 HTML DOM 高亮；
- StandardTokenizer 是实用的 Unicode tokenizer，不声称完全复刻 Lucene UAX#29；
- Stemmer 是保守规则实现，不是完整 Porter2；
- 查询 parser 不支持正则、范围查询或字段 boost。

这些边界均位于稳定接口后面，可逐步替换实现。
