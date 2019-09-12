package main

import (
	"sync"
	"time"
)

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

var stat RingStat
