package main

import (
	"fmt"
	"sort"
)

// Range is the range of the bucket
type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func (r Range) String() string {
	return fmt.Sprintf("[%s, %s)", r.Start, r.End)
}

// Heatmap saves the statistics in a time range
type Heatmap struct {
	Labels []string   `json:"labels"`
	Ranges []Range    `json:"ranges"`
	Values [][]uint64 `json:"values"`
}

func calValues(ranges []Range, values [][]uint64, regionsVec [][]*regionInfo, index int, getValue func(r *regionInfo) uint64) {
	startIndex := 0
	regions := regionsVec[index]
	for i := 0; i < len(regions); i++ {
		region := regions[i]
		startKey := region.StartKey
		endKey := region.EndKey

		for ranges[startIndex].Start != startKey {
			startIndex++
		}

		nextIndex := startIndex
		for ; nextIndex < len(ranges); nextIndex++ {
			if ranges[nextIndex].End == endKey {
				nextIndex++
				break
			}
		}

		count := nextIndex - startIndex
		value := getValue(region) / uint64(count)

		for j := startIndex; j < nextIndex; j++ {
			values[j][index] += value
		}

		startIndex = nextIndex
	}
}

func squashRanges(ranges []Range, values [][]uint64, maxBuckets int) ([]Range, [][]uint64) {
	n := len(ranges)
	if n > maxBuckets {
		n = maxBuckets
	}

	newRanges := make([]Range, n)
	newValues := make([][]uint64, n)

	step := (len(ranges) + 1) / n
	for i := 0; i < n; i++ {
		index := i * step
		newRanges[i].Start = ranges[index].Start
		newRanges[i].End = ranges[index].End
		newValues[i] = values[index]
		for j := 1; j < step && index+j < len(ranges); j++ {
			newRanges[i].End = ranges[index+j].End
			for k := 0; k < len(values[index+j]); k++ {
				newValues[i][k] += values[index+j][k]
			}
		}
	}

	return newRanges, newValues
}

func buildRanges(regions [][]*regionInfo) []Range {
	keySet := make(map[string]struct{}, len(regions[0]))
	// use all the regions' start key to split the whole range
	for i := 0; i < len(regions); i++ {
		for j := 0; j < len(regions[i]); j++ {
			region := regions[i][j]
			keySet[region.StartKey] = struct{}{}
		}
	}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	ranges := make([]Range, len(keys))
	for i := 0; i < len(keys)-1; i++ {
		ranges[i] = Range{
			Start: keys[i],
			End:   keys[i+1],
		}
	}

	ranges[len(keys)-1] = Range{
		Start: keys[len(keys)-1],
		End:   regions[0][len(regions[0])-1].EndKey,
	}
	return ranges
}

func newHeatmap(regions [][]*regionInfo, maxBuckets int, getValue func(r *regionInfo) uint64) Heatmap {
	ranges := buildRanges(regions)

	values := make([][]uint64, len(ranges))
	for i := 0; i < len(values); i++ {
		values[i] = make([]uint64, len(regions))
	}

	for i := 0; i < len(regions); i++ {
		calValues(ranges, values, regions, i, getValue)
	}

	ranges, values = squashRanges(ranges, values, maxBuckets)

	return Heatmap{
		Ranges: ranges,
		Values: values,
	}
}
