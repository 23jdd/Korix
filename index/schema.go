package index

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

var (
	// 所有 schema 前缀集中在此处，Writer、Reader 与恢复逻辑不得各自拼写常量，
	// 否则一次命名变化容易产生无法被清理的孤儿键。
	globalMetaKey = []byte("meta/global")
	docPrefix     = []byte("document/")
	docMetaPrefix = []byte("docmeta/")
	idPrefix      = []byte("id/")
	postingPrefix = []byte("posting/")
	termPrefix    = []byte("term/")
)

func component(value string) string {
	// RawURLEncoding 不包含 '/' 且没有 '=' padding，适合安全嵌入路径式 key。
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeComponent(value string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return string(decoded), err
}

// 固定宽度十六进制使 key 的字典序等于 DocID 数值序，前缀扫描得到的 posting
// 因而天然有序，无需 Reader 再排序。
func docIDPart(docID uint64) string { return fmt.Sprintf("%016x", docID) }

func parseDocIDPart(part string) (uint64, error) { return strconv.ParseUint(part, 16, 64) }

func documentKey(docID uint64) []byte     { return []byte("document/" + docIDPart(docID)) }
func documentMetaKey(docID uint64) []byte { return []byte("docmeta/" + docIDPart(docID)) }
func externalIDKey(id string) []byte      { return []byte("id/" + component(id)) }

func termKey(field, term string) []byte {
	// field 与 term 分别编码，避免原值中的斜杠、NUL 或 URL 破坏 key 边界。
	return []byte("term/" + component(field) + "/" + component(term))
}

func fieldTermPrefix(field string) []byte { return []byte("term/" + component(field) + "/") }

func postingKey(field, term string, docID uint64) []byte {
	return []byte("posting/" + component(field) + "/" + component(term) + "/" + docIDPart(docID))
}

func termPostingPrefix(field, term string) []byte {
	return []byte("posting/" + component(field) + "/" + component(term) + "/")
}

func encodeUint64(value uint64) []byte {
	return []byte(strconv.FormatUint(value, 10))
}

func decodeUint64(value []byte) (uint64, error) { return strconv.ParseUint(string(value), 10, 64) }

func lastKeyComponent(key []byte) string {
	// 只用于内部已验证 schema 的 key；调用方随后仍需解析并校验结果。
	value := string(key)
	if i := strings.LastIndexByte(value, '/'); i >= 0 {
		return value[i+1:]
	}
	return value
}
