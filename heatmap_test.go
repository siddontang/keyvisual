package main

import (
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

func TestBuildRange(t *testing.T) {
	regions := [][]*regionInfo{
		{
			newRegionInfo("", "a", 10),
			newRegionInfo("a", "c", 20),
			newRegionInfo("c", "", 20),
		},
		{
			newRegionInfo("", "a", 10),
			newRegionInfo("a", "b", 20),
			newRegionInfo("b", "", 20),
		},
		{
			newRegionInfo("", "a", 10),
			newRegionInfo("a", "c", 20),
			newRegionInfo("c", "d", 20),
			newRegionInfo("d", "", 20),
		},
	}

	ranges := buildRanges(regions)

	expectd := []Range{
		{"", "a"},
		{"a", "b"},
		{"b", "c"},
		{"c", "d"},
		{"d", ""},
	}

	if !reflect.DeepEqual(ranges, expectd) {
		t.Fatalf("expect %v, but got %v", expectd, ranges)
	}
}

func TestCalcValues(t *testing.T) {
	ranges := []Range{
		{"", "a"},
		{"a", "b"},
		{"b", "c"},
		{"c", "d"},
		{"d", ""},
	}

	regions := [][]*regionInfo{
		{
			newRegionInfo("", "a", 10),
			newRegionInfo("a", "c", 20),
			newRegionInfo("c", "", 20),
		},
		{
			newRegionInfo("a", "b", 10),
			newRegionInfo("b", "c", 20),
			newRegionInfo("c", "", 20),
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
	ranges := []Range{
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

	expectedRanges := []Range{
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
			newRegionInfo("", "a", 10),
			newRegionInfo("a", "c", 20),
			newRegionInfo("c", "", 20),
		},
		{
			newRegionInfo("", "b", 20),
			newRegionInfo("b", "", 20),
		},
	}

	h := newHeatmap(regions, 2, getWrittenBtes)

	expectedRanges := []Range{
		{"", "b"},
		{"b", ""},
	}

	if !reflect.DeepEqual(expectedRanges, h.Ranges) {
		t.Fatalf("want %v, but got %v", h.Ranges, expectedRanges)
	}
}
