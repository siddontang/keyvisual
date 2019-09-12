package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/rs/cors"
)

var (
	addr      = flag.String("addr", "0.0.0.0:8000", "Listening address")
	pdAddr    = flag.String("pd", "http://127.0.0.1:2379", "PD address")
	bucketNum = flag.Int("N", 256, "Max Bucket number in the histogram")
	interval  = flag.Duration("I", time.Minute, "Interval to collect metrics")
)

func perr(err error) {
	if err == nil {
		return
	}

	println(err.Error())
	os.Exit(1)
}

// Bucket is the total value in a range [start, end)
type Bucket struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Value uint64 `json:"value"`
}

// Hist contains all the buckets
type Hist struct {
	Buckets []Bucket `json:"buckets"`
}

func newHist(regions []*regionInfo, getValue func(r *regionInfo) uint64) Hist {
	var h Hist
	n := len(regions)
	if n > *bucketNum {
		n = *bucketNum
	}

	h.Buckets = make([]Bucket, n)
	step := (len(regions) + 1) / n
	for i := 0; i < n; i++ {
		index := i * step
		h.Buckets[i].Start = regions[index].StartKey
		for j := 0; j < step && index+j < len(regions); j++ {
			r := regions[index+j]
			h.Buckets[i].End = r.EndKey
			h.Buckets[i].Value += getValue(r)

		}
	}

	return h
}

// Stat saves all regions for one minutes
type Stat struct {
	Time    time.Time `json:"time"`
	Regions []*regionInfo
}

type ringStat struct {
	items   []*Stat
	head    int
	tail    int
	size    int
	maxSize int
}

func newRingStat(maxSize int) *ringStat {
	r := new(ringStat)

	r.size = maxSize
	r.head = 0
	r.tail = 0

	//for a empty item
	r.maxSize = r.size + 1

	r.items = make([]*Stat, r.maxSize)

	return r
}

func (r *ringStat) Len() int {
	if r.head == r.tail {
		return 0
	} else if r.tail > r.head {
		return r.tail - r.head
	} else {
		return r.tail + r.maxSize - r.head
	}
}

func (r *ringStat) Cap() int {
	return r.size - r.Len()
}

func (r *ringStat) Push(item *Stat) error {
	if r.Cap() == 0 {
		r.head = (r.head + 1) % r.maxSize
	}

	tail := r.tail % r.maxSize
	r.items[tail] = item

	r.tail = (r.tail + 1) % r.maxSize

	return nil
}

func (r *ringStat) Get(index int) *Stat {
	return r.items[(r.head+index)%r.maxSize]
}

// RingStat stores stats of every minute in a ring
type RingStat struct {
	sync.RWMutex

	*ringStat
}

func (r *RingStat) append(regions []*regionInfo) {
	s := Stat{
		Time:    time.Now(),
		Regions: regions,
	}

	r.Lock()
	defer r.Unlock()

	r.Push(&s)
}

func (r *RingStat) rangeStats(startTime time.Time, endTime time.Time) []*Stat {
	r.RLock()
	defer r.RUnlock()

	size := r.Len()
	if size == 0 {
		return nil
	}

	start := r.Get(0)

	count := int(endTime.Sub(startTime) / *interval)

	offset := int(startTime.Sub(start.Time) / *interval)
	if offset >= size {
		offset = size - 1
	} else if offset < 0 {
		offset = 0
	}

	left := size - offset
	if count > left {
		count = left
	} else if count == 0 {
		count = 1
	}

	stats := make([]*Stat, 0, count)
	for i := 0; i < count; i++ {
		stats = append(stats, r.Get(offset+i))
	}

	return stats
}

type regionInfo struct {
	ID           uint64 `json:"id"`
	StartKey     string `json:"start_key"`
	EndKey       string `json:"end_key"`
	WrittenBytes uint64 `json:"written_bytes,omitempty"`
	ReadBytes    uint64 `json:"read_bytes,omitempty"`
	WrittenKeys  uint64 `json:"written_keys,omitempty"`
	ReadKeys     uint64 `json:"read_keys,omitempty"`
}

func scanRegions() []*regionInfo {
	const limit = 1024
	var key []byte
	regions := make([]*regionInfo, 0, 1024)
	for {
		uri := fmt.Sprintf("pd/api/v1/regions/key?key=%s&limit=%d", url.QueryEscape(string(key)), limit)
		resp, err := http.Get(fmt.Sprintf("%s/%s", *pdAddr, uri))
		perr(err)
		r, err := ioutil.ReadAll(resp.Body)
		perr(err)
		resp.Body.Close()

		type regionsInfo struct {
			Regions []*regionInfo `json:"regions"`
		}
		var info regionsInfo
		err = json.Unmarshal([]byte(r), &info)
		perr(err)

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

var stat RingStat

func updateStat(ctx context.Context) {
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			regions := scanRegions()
			stat.append(regions)
		}
	}
}

type outStat struct {
	StartTime time.Time `json:"start"`
	EndTime   time.Time `json:"end"`
	Unit      string    `json:"unit"`

	Heatmaps []Heatmap `json:"heatmaps"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// start=-10m&end=-1m&tag=written_bytes
	start := r.FormValue("start")
	end := r.FormValue("end")
	tag := r.FormValue("tag")

	endTime := time.Now()
	startTime := endTime.Add(-*interval)

	if start != "" {
		if d, err := time.ParseDuration(start); err == nil {
			startTime = endTime.Add(d)
		}
	}
	if end != "" {
		if d, err := time.ParseDuration(end); err == nil {
			endTime = endTime.Add(d)
		}
	}

	f := func(r *regionInfo) uint64 {
		switch tag {
		case "read_bytes":
			return r.ReadBytes
		default:
			// written_bytes
			return r.WrittenBytes
		}
	}

	stats := stat.rangeStats(startTime, endTime)
	if len(stats) == 0 {
		return
	}

	regions := make([][]*regionInfo, len(stats))
	for i := 0; i < len(regions); i++ {
		regions[i] = stats[i].Regions
	}

	heatmap := newHeatmap(regions, *bucketNum, f)
	heatmap.Labels = []string{""}

	output := outStat{
		StartTime: stats[0].Time,
		EndTime:   stats[len(stats)-1].Time,
		Unit:      interval.String(),
		Heatmaps: []Heatmap{
			heatmap,
		},
	}

	data, _ := json.Marshal(output)
	w.Write(data)
}

func main() {
	flag.Parse()
	stat.ringStat = newRingStat(1024)

	go updateStat(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	// cors.Default() setup the middleware with default options being
	// all origins accepted with simple methods (GET, POST). See
	// documentation below for more options.
	h := cors.Default().Handler(mux)
	http.ListenAndServe(*addr, h)
}
