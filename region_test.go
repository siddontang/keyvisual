package main

import (
	"testing"
)

func TestSearchRegion(t *testing.T) {
	regions := []*regionInfo{
		newRegionInfo("", encodeTablePrefix(1), 10),
		newRegionInfo(encodeTablePrefix(1), encodeTablePrefix(3), 20),
		newRegionInfo(encodeTablePrefix(3), "", 30),
	}

	check := func(key string, expected int) {
		n := searchRegion(key, regions)
		if n != expected {
			t.Fatalf("expectd %d but got %d for %s", expected, n, key)
		}
	}

	check("7480000000000000ff0100000000000000f8", 1)
	check("7480000000000000ff0200000000000000f8", 1)
	check("7480000000000000ff0300000000000000f8", 2)
	check("7480000000000000ff0400000000000000f8", 2)
}

func TestRangeRegions(t *testing.T) {
	regions := []*regionInfo{
		newRegionInfo("", encodeTablePrefix(2), 10),
		newRegionInfo(encodeTablePrefix(2), encodeTablePrefix(3), 20),
		newRegionInfo(encodeTablePrefix(3), encodeTablePrefix(5), 20),
		newRegionInfo(encodeTablePrefix(5), "", 30),
	}

	check := func(start string, end string, e1 int, e2 int) {
		r1, r2 := rangeRegionIndices(start, end, regions)
		if r1 != e1 || r2 != e2 {
			t.Fatalf("expected [%d, %d) but got [%d, %d) for [%s, %s)", e1, e2, r1, r2, start, end)
		}
	}

	check(encodeTablePrefix(1), encodeTableIndexPrefix(1, 1), 0, 1)
	check(encodeTableIndexPrefix(1, 1), encodeTableIndexPrefix(1, 2), 0, 1)
	check(encodeTableIndexPrefix(1, 1), encodeTableIndexPrefix(2, 1), 0, 2)
	check(encodeTablePrefix(1), encodeTablePrefix(2), 0, 1)
	check(encodeTablePrefix(1), encodeTableIndexPrefix(2, 1), 0, 2)
	check(encodeTablePrefix(4), encodeTableIndexPrefix(4, 1), 2, 3)
	check(encodeTablePrefix(5), encodeTablePrefix(6), 3, 4)
	check(encodeTablePrefix(3), encodeTablePrefix(6), 2, 4)
}
