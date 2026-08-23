package index

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

var (
	globalMetaKey = []byte("meta/global")
	docPrefix     = []byte("document/")
	docMetaPrefix = []byte("docmeta/")
	idPrefix      = []byte("id/")
	postingPrefix = []byte("posting/")
	termPrefix    = []byte("term/")
)

func component(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeComponent(value string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return string(decoded), err
}

func docIDPart(docID uint64) string { return fmt.Sprintf("%016x", docID) }

func parseDocIDPart(part string) (uint64, error) { return strconv.ParseUint(part, 16, 64) }

func documentKey(docID uint64) []byte     { return []byte("document/" + docIDPart(docID)) }
func documentMetaKey(docID uint64) []byte { return []byte("docmeta/" + docIDPart(docID)) }
func externalIDKey(id string) []byte      { return []byte("id/" + component(id)) }

func termKey(field, term string) []byte {
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
	value := string(key)
	if i := strings.LastIndexByte(value, '/'); i >= 0 {
		return value[i+1:]
	}
	return value
}
