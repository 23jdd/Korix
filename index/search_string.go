package index

import "github.com/23jdd/Koris/query"

// SearchString 解析字段化查询表达式后执行搜索。defaultField 用于没有显式
// field: 前缀的 clause；语法错误原样返回 query.ErrInvalidQuery 包装错误。
func (i *Index) SearchString(expression, defaultField string, limit int) ([]query.Hit, error) {
	parsed, err := query.Parse(expression, defaultField)
	if err != nil {
		return nil, err
	}
	return i.Search(parsed, limit)
}
