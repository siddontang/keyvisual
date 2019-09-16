package main

import (
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/tablecodec"
	"github.com/pingcap/tidb/util/codec"
)

type Key struct {
	Desc        string   `json:"desc"`
	Ts          uint64   `json:"ts,omitempty"`
	TableID     int64    `json:"table_id,omitempty"`
	RowID       int64    `json:"row_id,omitempty"`
	RowValue    int64    `json:"row_value,omitempty"`
	IndexID     int64    `json:"index_id,omitempty"`
	IndexValues []string `json:"index_values,omitempty"`
}

// RangeBuilder builds the Range,
type RangeBuilder struct {
	Start string
	End   string
}

func (s RangeBuilder) Build() Range {
	r := Range{
		StartKey: decodeKey(s.Start),
		EndKey:   decodeKey(s.End),
	}
	return r
}

func decodeKey(key string) Key {
	var ts uint64
	v, err := hex.DecodeString(key)

	if err != nil {
		panic(err)
	}
	tsString, decode, _ := codec.DecodeBytes(v, nil)
	if len(tsString) == 8 {
		_, ts, _ = codec.DecodeUintDesc(tsString)
	}
	if len(decode) > 0 && decode[0] == 'z' {
		decode = decode[1:]
	}
	desc := string(decode)
	tableID, indexID, isRecord, _ := tablecodec.DecodeKeyHead(kv.Key(desc))
	var (
		rowID       int64
		indexValues []string
	)
	if isRecord {
		_, rowID, _ = tablecodec.DecodeRecordKey(kv.Key(desc))
	} else {
		_, _, indexValues, _ = tablecodec.DecodeIndexKey(kv.Key(desc))
	}
	return Key{
		Desc:        key,
		Ts:          ts,
		TableID:     tableID,
		RowID:       rowID,
		IndexID:     indexID,
		IndexValues: indexValues,
	}
}

// Range is the range of the bucket
type Range struct {
	StartKey Key `json:"start"`
	EndKey   Key `json:"end"`
}

func (r Range) String() string {
	return fmt.Sprintf("[%s, %s)", r.StartKey.Desc, r.EndKey.Desc)

}

// Heatmap saves the statistics in a time range
type Heatmap struct {
	Labels []string   `json:"labels"`
	Ranges []Range    `json:"ranges"`
	Values [][]uint64 `json:"values"`
}

func calValues(ranges []RangeBuilder, values [][]uint64, regionsVec [][]*regionInfo, index int, getValue func(r *regionInfo) uint64) {
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

func squashRanges(ranges []RangeBuilder, values [][]uint64, maxBuckets int) ([]RangeBuilder, [][]uint64) {
	n := len(ranges)
	if n > maxBuckets {
		n = maxBuckets
	}

	newRanges := make([]RangeBuilder, n)
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

func buildRanges(regions [][]*regionInfo) []RangeBuilder {
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

	ranges := make([]RangeBuilder, len(keys))
	for i := 0; i < len(keys)-1; i++ {
		ranges[i] = RangeBuilder{
			Start: keys[i],
			End:   keys[i+1],
		}
	}

	ranges[len(keys)-1] = RangeBuilder{
		Start: keys[len(keys)-1],
		End:   regions[0][len(regions[0])-1].EndKey,
	}
	return ranges
}

func newHeatmap(regions [][]*regionInfo, maxBuckets int, getValue func(r *regionInfo) uint64) Heatmap {
	rs := buildRanges(regions)

	values := make([][]uint64, len(rs))
	for i := 0; i < len(values); i++ {
		values[i] = make([]uint64, len(regions))
	}

	for i := 0; i < len(regions); i++ {
		calValues(rs, values, regions, i, getValue)
	}
	builders, values := squashRanges(rs, values, maxBuckets)

	ranges := make([]Range, len(builders))
	for i, b := range builders {
		ranges[i] = b.Build()
	}
	return Heatmap{
		Ranges: ranges,
		Values: values,
	}
}
