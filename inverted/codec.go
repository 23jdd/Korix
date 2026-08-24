package inverted

import (
	"encoding/binary"
	"errors"
)

// ErrCorruptPosting 表示二进制 posting 截断、溢出或内部计数不一致。
var ErrCorruptPosting = errors.New("inverted: corrupt posting")

// EncodePosting 使用 uvarint 编码整数，并对有序 position 做差分编码。
//
// 布局为 docIDByteLength | docIDBytes | frequency | positionCount |
// firstPosition | delta...。ID 长度及数值字段使用 uvarint；文本中的相邻词位
// 通常产生很小的 delta。函数返回新切片，调用方可以直接交给 Store。
func EncodePosting(posting Posting) []byte {
	capacity := 20 + len(posting.DocID) + len(posting.Positions)*3
	buffer := make([]byte, 0, capacity)
	buffer = appendUvarint(buffer, uint64(len(posting.DocID)))
	buffer = append(buffer, posting.DocID...)
	buffer = appendUvarint(buffer, uint64(posting.Frequency))
	buffer = appendUvarint(buffer, uint64(len(posting.Positions)))
	var previous uint32
	for i, position := range posting.Positions {
		// 第一项保存绝对位置，后续保存与前一项的非负差值。
		delta := position
		if i > 0 {
			delta = position - previous
		}
		buffer = appendUvarint(buffer, uint64(delta))
		previous = position
	}
	return buffer
}

// DecodePosting 解码并严格校验 posting。任何 varint 截断、uint32 溢出、尾随
// 数据或 frequency/positionCount 不一致都会返回 ErrCorruptPosting。
func DecodePosting(data []byte) (Posting, error) {
	var posting Posting
	var ok bool
	idLength, rest, ok := consumeUvarint(data)
	if !ok || idLength > uint64(len(rest)) {
		return Posting{}, ErrCorruptPosting
	}
	posting.DocID = string(rest[:idLength])
	data = rest[idLength:]
	frequency, rest, ok := consumeUvarint(data)
	if !ok || frequency > uint64(^uint32(0)) {
		return Posting{}, ErrCorruptPosting
	}
	posting.Frequency = uint32(frequency)
	count, rest, ok := consumeUvarint(rest)
	if !ok || count > uint64(^uint32(0)) {
		return Posting{}, ErrCorruptPosting
	}
	posting.Positions = make([]uint32, 0, count)
	var previous uint64
	for i := uint64(0); i < count; i++ {
		// 使用 uint64 累加后再检查 uint32 上限，避免恶意 delta 在加法时溢出。
		delta, next, valid := consumeUvarint(rest)
		if !valid || previous+delta > uint64(^uint32(0)) {
			return Posting{}, ErrCorruptPosting
		}
		previous += delta
		posting.Positions = append(posting.Positions, uint32(previous))
		rest = next
	}
	if len(rest) != 0 {
		return Posting{}, ErrCorruptPosting
	}
	if uint64(posting.Frequency) != count {
		return Posting{}, ErrCorruptPosting
	}
	return posting, nil
}

func appendUvarint(dst []byte, value uint64) []byte {
	// 栈上 scratch 足以容纳任意 uint64 varint，append 再复制实际使用的字节。
	var scratch [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(scratch[:], value)
	return append(dst, scratch[:n]...)
}

func consumeUvarint(data []byte) (uint64, []byte, bool) {
	value, n := binary.Uvarint(data)
	if n <= 0 {
		return 0, data, false
	}
	return value, data[n:], true
}
