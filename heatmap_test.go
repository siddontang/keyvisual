package main

import (
	"encoding/hex"

	"github.com/pingcap/tidb/tablecodec"
	"github.com/pingcap/tidb/util/codec"
	"reflect"
	"testing"
)

func newRegionInfo(start string, end string, value uint64) *regionInfo {

	return &regionInfo{
		StartKey:     start,
		EndKey:       end,
		WrittenBytes: value,
	}
}

func getWrittenBtes(r *regionInfo) uint64 {
	return r.WrittenBytes
}

func encodeTablePrefix(tableID int64) string {
	key := tablecodec.EncodeTablePrefix(tableID)
	raw := codec.EncodeBytes([]byte(nil), key)
	return hex.EncodeToString(raw)
}

func encodeTableIndexPrefix(tableID int64, indexID int64) string {
	key := tablecodec.EncodeTableIndexPrefix(tableID, indexID)
	raw := codec.EncodeBytes([]byte(nil), key)
	return hex.EncodeToString(raw)
}

func TestBuildRange(t *testing.T) {
	regions := [][]*regionInfo{
		{
			newRegionInfo(encodeTablePrefix(1), encodeTablePrefix(2), 10),
			newRegionInfo(encodeTablePrefix(2), encodeTablePrefix(3), 20),
			newRegionInfo(encodeTablePrefix(3), encodeTablePrefix(4), 20),
		},
		{
			newRegionInfo(encodeTablePrefix(1), encodeTablePrefix(2), 10),
			newRegionInfo(encodeTablePrefix(2), encodeTablePrefix(3), 20),
			newRegionInfo(encodeTablePrefix(3), encodeTablePrefix(4), 20),
		},
		{
			newRegionInfo(encodeTablePrefix(1), encodeTablePrefix(2), 10),
			newRegionInfo(encodeTablePrefix(2), encodeTablePrefix(3), 20),
			newRegionInfo(encodeTablePrefix(3), encodeTableIndexPrefix(3, 1), 20),
			newRegionInfo(encodeTableIndexPrefix(3, 1), encodeTablePrefix(4), 20),
		},
	}

	builders := buildRanges(regions)
	ranges := make([]Range, len(builders))
	for i, b := range builders {
		ranges[i] = b.Build()
	}

	expectd := []Range{
		{Key{Desc: "7480000000000000ff0100000000000000f8", TableID: 1}, Key{Desc: "7480000000000000ff0200000000000000f8", TableID: 2}},
		{Key{Desc: "7480000000000000ff0200000000000000f8", TableID: 2}, Key{Desc: "7480000000000000ff0300000000000000f8", TableID: 3}},
		{Key{Desc: "7480000000000000ff0300000000000000f8", TableID: 3}, Key{Desc: "7480000000000000ff035f698000000000ff0000010000000000fa", TableID: 3, IndexID: 1}},
		{Key{Desc: "7480000000000000ff035f698000000000ff0000010000000000fa", TableID: 3, IndexID: 1}, Key{Desc: "7480000000000000ff0400000000000000f8", TableID: 4}},
	}

	if !reflect.DeepEqual(ranges, expectd) {
		t.Fatalf("expect %v, but got %v", expectd, ranges)
	}
}

func TestCalcValues(t *testing.T) {
	ranges := []RangeBuilder{
		{"", "7480000000000000ff0100000000000000f8"},
		{"7480000000000000ff0100000000000000f8", "7480000000000000ff0200000000000000f8"},
		{"7480000000000000ff0200000000000000f8", "7480000000000000ff0300000000000000f8"},
		{"7480000000000000ff0300000000000000f8", "7480000000000000ff0400000000000000f8"},
		{"7480000000000000ff0400000000000000f8", ""},
	}

	regions := [][]*regionInfo{
		{
			newRegionInfo("", encodeTablePrefix(1), 10),
			newRegionInfo(encodeTablePrefix(1), encodeTablePrefix(3), 20),
			newRegionInfo(encodeTablePrefix(3), "", 20),
		},
		{
			newRegionInfo(encodeTablePrefix(1), encodeTablePrefix(2), 10),
			newRegionInfo(encodeTablePrefix(2), encodeTablePrefix(3), 20),
			newRegionInfo(encodeTablePrefix(3), "", 20),
		},
	}

	values := [][]uint64{
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
	}

	calValues(ranges, values, regions, 0, getWrittenBtes)
	calValues(ranges, values, regions, 1, getWrittenBtes)

	expected := [][]uint64{
		{10, 0},
		{10, 10},
		{10, 20},
		{10, 10},
		{10, 10},
	}
	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("want %v, but got %v", expected, values)
	}
}

func TestSquashRanges(t *testing.T) {
	ranges := []RangeBuilder{
		{"", "a"},
		{"a", "b"},
		{"b", "c"},
		{"c", "d"},
		{"d", ""},
	}

	values := [][]uint64{
		{1, 1, 1},
		{2, 2, 2},
		{1, 1, 1},
		{2, 2, 2},
		{3, 3, 3},
	}

	newRanges, newValues := squashRanges(ranges, values, 2)

	expectedRanges := []RangeBuilder{
		{"", "c"},
		{"c", ""},
	}

	if !reflect.DeepEqual(newRanges, expectedRanges) {
		t.Fatalf("want %v, but got %v", expectedRanges, newRanges)
	}

	expectedValues := [][]uint64{
		{4, 4, 4},
		{5, 5, 5},
	}
	if !reflect.DeepEqual(newValues, expectedValues) {
		t.Fatalf("want %v, but got %v", expectedValues, newValues)
	}
}

func TestHeatmap(t *testing.T) {
	regions := [][]*regionInfo{
		{
			newRegionInfo(encodeTablePrefix(1), encodeTablePrefix(2), 10),
			newRegionInfo(encodeTablePrefix(2), encodeTablePrefix(3), 20),
			newRegionInfo(encodeTablePrefix(3), encodeTablePrefix(4), 20),
		},
		{
			newRegionInfo(encodeTablePrefix(1), encodeTablePrefix(2), 10),
			newRegionInfo(encodeTablePrefix(2), encodeTablePrefix(4), 20),
		},
	}

	h := newHeatmap(regions, 2, getWrittenBtes)

	expectedRanges := []Range{
		{Key{Desc: "7480000000000000ff0100000000000000f8", TableID: 1}, Key{Desc: "7480000000000000ff0300000000000000f8", TableID: 3}},
		{Key{Desc: "7480000000000000ff0300000000000000f8", TableID: 3}, Key{Desc: "7480000000000000ff0400000000000000f8", TableID: 4}},
	}

	if !reflect.DeepEqual(expectedRanges, h.Ranges) {
		t.Fatalf("want %v, but got %v", expectedRanges, h.Ranges)
	}
}
