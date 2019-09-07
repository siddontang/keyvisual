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

// Stat collects the statistics for one interval
type Stat struct {
	Time         time.Time `json:"time"`
	WrittenBytes Hist      `json:"written_bytes"`
	// WrittenKeys  Hist      `json:"written_keys"`
	ReadBytes Hist `json:"read_bytes"`
	// ReadKeys     Hist      `json:"read_keys"`
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
	// index must < Len
	return r.items[(r.head+index)%r.maxSize]
}

// RingStat stores stats of every minute in a ring
type RingStat struct {
	sync.RWMutex

	*ringStat
}

func (r *RingStat) append(regions []*regionInfo) {
	s := Stat{
		Time:         time.Now(),
		WrittenBytes: newHist(regions, func(r *regionInfo) uint64 { return r.WrittenBytes }),
		// WrittenKeys:  newHist(regions, func(r *regionInfo) uint64 { return r.WrittenKeys }),
		ReadBytes: newHist(regions, func(r *regionInfo) uint64 { return r.ReadBytes }),
		// ReadKeys:     newHist(regions, func(r *regionInfo) uint64 { return r.ReadKeys }),
	}

	r.Lock()
	defer r.Unlock()

	r.Push(&s)
}

func (r *RingStat) at(t time.Time) *Stat {
	r.RLock()
	defer r.RUnlock()

	size := r.Len()
	if size == 0 {
		return nil
	}

	start := r.Get(0)

	n := int(t.Sub(start.Time) / *interval)
	if n >= size {
		n = size
	} else if n < 0 {
		n = 0
	}

	return r.Get(n)
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

func handler(w http.ResponseWriter, r *http.Request) {
	s := stat.at(time.Now())

	data, _ := json.Marshal(s)
	w.Write(data)
}

func main() {
	flag.Parse()
	stat.ringStat = newRingStat(1024)

	go updateStat(context.Background())

	http.HandleFunc("/", handler)
	err := http.ListenAndServe(*addr, nil)
	perr(err)
}
