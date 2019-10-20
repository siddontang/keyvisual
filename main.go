package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/rs/cors"
)

const addr = "0.0.0.0:8000"

var (
	pdAddr    = flag.String("pd", "http://127.0.0.1:2379", "PD address")
	tidbAddr  = flag.String("tidb", "http://127.0.0.1:10080", "TiDB Address")
	bucketNum = flag.Int("N", 256, "Max Bucket number in the histogram")
	interval  = flag.Duration("I", time.Minute, "Interval to collect metrics")
	ingoreSys = flag.Bool("no-sys", true, "Ignore system database")
)

func perr(err error) {
	if err == nil {
		return
	}

	println(err.Error())
	debug.PrintStack()
	os.Exit(1)
}

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

			updateTables()
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

	tbls := loadTables()
	heatmaps := make([]Heatmap, 0, len(tbls))
	for _, tbl := range tbls {
		if *ingoreSys {
			if tbl.DB == "mysql" {
				continue
			}
		}
		heatmaps = tableHeatmap(heatmaps, tbl, regions, *bucketNum, f)
	}

	output := outStat{
		StartTime: stats[0].Time,
		EndTime:   stats[len(stats)-1].Time,
		Unit:      interval.String(),
		Heatmaps:  heatmaps,
	}

	data, _ := json.Marshal(output)
	w.Write(data)
}

func main() {
	flag.Parse()
	stat.ringStat = newRingStat(1024)

	go updateStat(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/heatmaps", handler)

	// cors.Default() setup the middleware with default options being
	// all origins accepted with simple methods (GET, POST). See
	// documentation below for more options.
	fs := http.FileServer(http.Dir("./frontend"))
	mux.Handle("/", fs)

	h := cors.Default().Handler(mux)
	fmt.Printf("Please access http://%s to enjoy it\n", addr)
	http.ListenAndServe(addr, h)
}
