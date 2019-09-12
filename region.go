package main

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
)

type regionInfo struct {
	ID           uint64 `json:"id"`
	StartKey     string `json:"start_key"`
	EndKey       string `json:"end_key"`
	WrittenBytes uint64 `json:"written_bytes,omitempty"`
	ReadBytes    uint64 `json:"read_bytes,omitempty"`
	WrittenKeys  uint64 `json:"written_keys,omitempty"`
	ReadKeys     uint64 `json:"read_keys,omitempty"`
}

func (r *regionInfo) String() string {
	return fmt.Sprintf("[%s, %s)", r.StartKey, r.EndKey)
}

func scanRegions() []*regionInfo {
	const limit = 1024
	var key []byte
	var err error
	regions := make([]*regionInfo, 0, 1024)
	for {
		uri := fmt.Sprintf("pd/api/v1/regions/key?key=%s&limit=%d", url.QueryEscape(string(key)), limit)

		type regionsInfo struct {
			Regions []*regionInfo `json:"regions"`
		}
		var info regionsInfo
		readBody(*pdAddr, uri, &info)

		if len(info.Regions) == 0 {
			break
		}

		regions = append(regions, info.Regions...)

		lastEndKey := info.Regions[len(info.Regions)-1].EndKey
		if lastEndKey == "" {
			break
		}

		key, err = hex.DecodeString(lastEndKey)
		perr(err)
	}

	return regions
}

func searchRegion(key string, regions []*regionInfo) int {
	i := sort.Search(len(regions), func(i int) bool {
		return regions[i].StartKey >= key
	})
	if i < len(regions) {
		if regions[i].StartKey == key {
			return i
		}

		if i > 0 && regions[i-1].EndKey > key {
			return i - 1
		}
	}

	if regions[len(regions)-1].EndKey == "" {
		return len(regions) - 1
	}

	return -1
}

func rangeRegionIndices(start string, end string, regions []*regionInfo) (int, int) {
	startIndex := searchRegion(start, regions)
	endIndex := searchRegion(end, regions)

	if startIndex == -1 || endIndex == -1 {
		return 0, 0
	}

	if regions[endIndex].EndKey == "" || (regions[endIndex].EndKey > end && regions[endIndex].StartKey != end) {
		endIndex = endIndex + 1
	}

	return startIndex, endIndex
}

func rangeRegions(start string, end string, regions [][]*regionInfo) [][]*regionInfo {
	newRegions := make([][]*regionInfo, len(regions))
	for i := 0; i < len(regions); i++ {
		startIndex, endIndex := rangeRegionIndices(start, end, regions[i])
		newRegions[i] = regions[i][startIndex:endIndex]
	}

	return newRegions
}

func tableHeatmap(heats []Heatmap, t *Table, regions [][]*regionInfo, maxNumber int, getValue func(r *regionInfo) uint64) []Heatmap {
	// for record
	startRecord := GenTableRecordPrefix(t.ID)
	endRecord := GenTableRecordPrefix(t.ID + 1)

	rr := rangeRegions(startRecord, endRecord, regions)
	h := newHeatmap(rr, maxNumber, getValue)
	h.Labels = []string{t.DB, t.Name, ""}

	heats = append(heats, h)
	for idx, name := range t.Indices {
		startIndex := GenTableIndexPrefix(t.ID, idx)
		endIndex := GenTableIndexPrefix(t.ID, idx+1)

		rr = rangeRegions(startIndex, endIndex, regions)
		h = newHeatmap(rr, maxNumber, getValue)
		h.Labels = []string{t.DB, t.Name, name}
		heats = append(heats, h)
	}

	return heats
}
