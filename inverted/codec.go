package inverted

import (
	"encoding/binary"
	"errors"
)

var ErrCorruptPosting = errors.New("inverted: corrupt posting")

// EncodePosting uses unsigned varints and delta-encoded positions.
func EncodePosting(posting Posting) []byte {
	capacity := 20 + len(posting.Positions)*3
	buffer := make([]byte, 0, capacity)
	buffer = appendUvarint(buffer, posting.DocID)
	buffer = appendUvarint(buffer, uint64(posting.Frequency))
	buffer = appendUvarint(buffer, uint64(len(posting.Positions)))
	var previous uint32
	for i, position := range posting.Positions {
		delta := position
		if i > 0 {
			delta = position - previous
		}
		buffer = appendUvarint(buffer, uint64(delta))
		previous = position
	}
	return buffer
}

func DecodePosting(data []byte) (Posting, error) {
	var posting Posting
	var ok bool
	posting.DocID, data, ok = consumeUvarint(data)
	if !ok {
		return Posting{}, ErrCorruptPosting
	}
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
