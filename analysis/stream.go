package analysis

// TokenStream 提供游标式 token 消费协议。
//
// 调用者先反复调用 Next，在返回 true 后读取 Token；Reset 会丢弃旧输入并把游标
// 放回起点。接口为未来真正的增量 tokenizer 保留了空间，调用端无需关心 token
// 是一次性生成还是按需生成。
type TokenStream interface {
	Next() bool
	Token() Token
	Reset(text string)
}

// StreamAnalyzer 是同时支持批量和游标读取的 Analyzer 扩展接口。
type StreamAnalyzer interface {
	Analyzer
	Stream(text string) TokenStream
}

type sliceTokenStream struct {
	analyzer Analyzer
	tokens   []Token
	current  int
}

// NewTokenStream 使用 Analyzer 创建可复用流。
//
// 当前实现先生成 []Token 再通过游标访问，保证所有 Analyzer 都能立即获得流式
// API；未来可以加入惰性实现而不改变调用代码。
func NewTokenStream(analyzer Analyzer, text string) TokenStream {
	s := &sliceTokenStream{analyzer: analyzer}
	s.Reset(text)
	return s
}

func (s *sliceTokenStream) Next() bool {
	// current 表示“下一次 Token() 应读取的位置之后一位”。初始为 0，成功
	// Next 后递增，因此 Token 使用 current-1。
	if s.current >= len(s.tokens) {
		return false
	}
	s.current++
	return true
}

func (s *sliceTokenStream) Token() Token {
	// 在第一次 Next 之前或遍历结束之后返回零值，避免暴露越界 panic。
	if s.current == 0 || s.current > len(s.tokens) {
		return Token{}
	}
	return s.tokens[s.current-1]
}

func (s *sliceTokenStream) Reset(text string) {
	s.tokens = s.analyzer.Analyze(text)
	s.current = 0
}
