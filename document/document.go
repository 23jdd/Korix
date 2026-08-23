// Package document defines values stored and indexed by Koris.
package document

// Document is the user-facing document model. ID must be unique within an
// index. Each field is analyzed independently, allowing field-scoped queries.
type Document struct {
	ID     string            `json:"id"`
	Fields map[string]string `json:"fields"`
}

// Clone returns a deep copy suitable for returning across API boundaries.
func (d Document) Clone() Document {
	clone := Document{ID: d.ID, Fields: make(map[string]string, len(d.Fields))}
	for name, value := range d.Fields {
		clone.Fields[name] = value
	}
	return clone
}
