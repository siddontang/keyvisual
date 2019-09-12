package main

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
)

var (
	tablePrefix     = []byte{'t'}
	recordPrefixSep = []byte("_r")
	indexPrefixSep  = []byte("_i")
)

const (
	signMask uint64 = 0x8000000000000000

	encGroupSize = 8
	encMarker    = byte(0xFF)
	encPad       = byte(0x0)
)

var pads = make([]byte, encGroupSize)

// EncodeBytes guarantees the encoded value is in ascending order for comparison,
// encoding with the following rule:
//  [group1][marker1]...[groupN][markerN]
//  group is 8 bytes slice which is padding with 0.
//  marker is `0xFF - padding 0 count`
// For example:
//   [] -> [0, 0, 0, 0, 0, 0, 0, 0, 247]
//   [1, 2, 3] -> [1, 2, 3, 0, 0, 0, 0, 0, 250]
//   [1, 2, 3, 0] -> [1, 2, 3, 0, 0, 0, 0, 0, 251]
//   [1, 2, 3, 4, 5, 6, 7, 8] -> [1, 2, 3, 4, 5, 6, 7, 8, 255, 0, 0, 0, 0, 0, 0, 0, 0, 247]
// Refer: https://github.com/facebook/mysql-5.6/wiki/MyRocks-record-format#memcomparable-format
func EncodeBytes(data []byte) []byte {
	// Allocate more space to avoid unnecessary slice growing.
	// Assume that the byte slice size is about `(len(data) / encGroupSize + 1) * (encGroupSize + 1)` bytes,
	// that is `(len(data) / 8 + 1) * 9` in our implement.
	dLen := len(data)
	result := make([]byte, 0, (dLen/encGroupSize+1)*(encGroupSize+1))
	for idx := 0; idx <= dLen; idx += encGroupSize {
		remain := dLen - idx
		padCount := 0
		if remain >= encGroupSize {
			result = append(result, data[idx:idx+encGroupSize]...)
		} else {
			padCount = encGroupSize - remain
			result = append(result, data[idx:]...)
			result = append(result, pads[:padCount]...)
		}

		marker := encMarker - byte(padCount)
		result = append(result, marker)
	}
	return result
}

// EncodeIntToCmpUint make int v to comparable uint type
func EncodeIntToCmpUint(v int64) uint64 {
	return uint64(v) ^ signMask
}

// EncodeInt appends the encoded value to slice b and returns the appended slice.
// EncodeInt guarantees that the encoded value is in ascending order for comparison.
func EncodeInt(b []byte, v int64) []byte {
	var data [8]byte
	u := EncodeIntToCmpUint(v)
	binary.BigEndian.PutUint64(data[:], u)
	return append(b, data[:]...)
}

// GenTableRecordPrefix composes record prefix with tableID: "t[tableID]_r".
func GenTableRecordPrefix(tableID int64) string {
	buf := make([]byte, 0, len(tablePrefix)+8+len(recordPrefixSep))
	buf = appendTableRecordPrefix(buf, tableID)
	return strings.ToUpper(hex.EncodeToString(EncodeBytes(buf)))
}

// GenTableIndexPrefix composes index prefix with tableID: "t[tableID]_i".
func GenTableIndexPrefix(tableID int64, idx int64) string {
	buf := make([]byte, 0, len(tablePrefix)+8+len(indexPrefixSep)+8)
	buf = appendTableIndexPrefix(buf, tableID)
	buf = EncodeInt(buf, idx)
	return strings.ToUpper(hex.EncodeToString(EncodeBytes(buf)))
}

// appendTableRecordPrefix appends table record prefix  "t[tableID]_r".
func appendTableRecordPrefix(buf []byte, tableID int64) []byte {
	buf = append(buf, tablePrefix...)
	buf = EncodeInt(buf, tableID)
	buf = append(buf, recordPrefixSep...)
	return buf
}

// appendTableIndexPrefix appends table index prefix  "t[tableID]_i".
func appendTableIndexPrefix(buf []byte, tableID int64) []byte {
	buf = append(buf, tablePrefix...)
	buf = EncodeInt(buf, tableID)
	buf = append(buf, indexPrefixSep...)
	return buf
}
