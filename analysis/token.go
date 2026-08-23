// Package analysis 定义文本分析框架。
//
// 同一个 Analyzer 会同时用于写入和查询：写入时生成倒排索引中的 term，
// 查询时把用户输入归一化为相同的 term。该包刻意不依赖 index，避免分析规则
// 与存储实现耦合，也防止 Go 包之间形成循环依赖。
package analysis

// Token 表示分析链输出的一个词项，以及它在原始 UTF-8 文本中的位置。
//
// StartOffset 和 EndOffset 是字节偏移而不是 rune 下标，所以可以直接使用
// text[StartOffset:EndOffset] 截取原文，Highlight 也依赖这个约定。Position 是
// 逻辑词位：Filter 可以删除 token，但不应重排剩余 Position，否则短语查询将
// 无法区分 "go fast" 和 "go is fast"。
type Token struct {
	Term        string `json:"term"`
	Position    uint32 `json:"position"`
	StartOffset uint32 `json:"start_offset"`
	EndOffset   uint32 `json:"end_offset"`
	Type        string `json:"type,omitempty"`
}

// Tokenizer 是分析流水线的第一阶段，负责切分文本并填写词位与偏移。
// 实现必须按原文顺序返回 token，且不能返回越过 UTF-8 字符边界的 offset。
type Tokenizer interface {
	Tokenize(text string) []Token
}

// TokenFilter 对 tokenizer 的输出做归一化或筛选。
// Filter 应返回独立结果，避免修改调用者持有的 token slice；删除 token 时应保留
// 原 Position，以维持短语距离语义。
type TokenFilter interface {
	Filter(tokens []Token) []Token
}

// Analyzer 把原始文本转换为最终可索引 token。Analyzer 必须是确定性的，并且
// 在并发查询中可安全复用；有可变状态的实现应自行同步或为每次调用创建状态。
type Analyzer interface {
	Analyze(text string) []Token
}
