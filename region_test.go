package main

import (
	"testing"
)

func TestSearchRegion(t *testing.T) {
	regions := []*regionInfo{
		newRegionInfo("", "a", 10),
		newRegionInfo("a", "c", 20),
		newRegionInfo("c", "", 30),
	}

	check := func(key string, expected int) {
		n := searchRegion(key, regions)
		if n != expected {
			t.Fatalf("expectd %d but got %d for %s", expected, n, key)
		}
	}

	check("a", 1)
	check("b", 1)
	check("c", 2)
	check("d", 2)
}

func TestRangeRegions(t *testing.T) {
	regions := []*regionInfo{
		newRegionInfo("", "b", 10),
		newRegionInfo("b", "c", 20),
		newRegionInfo("c", "e", 30),
		newRegionInfo("e", "", 30),
	}

	check := func(start string, end string, e1 int, e2 int) {
		r1, r2 := rangeRegionIndices(start, end, regions)
		if r1 != e1 || r2 != e2 {
			t.Fatalf("expected [%d, %d) but got [%d, %d) for [%s, %s)", e1, e2, r1, r2, start, end)
		}
	}

	check("a", "a1", 0, 1)
	check("a1", "a2", 0, 1)
	check("a1", "b1", 0, 2)
	check("a", "b", 0, 1)
	check("a", "b1", 0, 2)
	check("d", "d1", 2, 3)
	check("e", "f", 3, 4)
	check("c", "f", 2, 4)
}
