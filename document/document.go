// Package document 定义 Koris 存储和索引的用户文档模型。
package document

// Document 是面向调用者的文档模型。
// ID 在一个索引内必须唯一；同 ID 再次 Add 表示更新。Fields 的每个值独立分析，
// 因而 title:go 与 content:go 使用不同词典和 posting。字段值可以为空。
type Document struct {
	ID     string            `json:"id"`
	Fields map[string]string `json:"fields"`
}

// Clone 返回深拷贝，主要用于跨 API 边界返回文档，避免调用者修改 Fields map 后
// 无意改变内存中的源值。
func (d Document) Clone() Document {
	clone := Document{ID: d.ID, Fields: make(map[string]string, len(d.Fields))}
	for name, value := range d.Fields {
		clone.Fields[name] = value
	}
	return clone
}
