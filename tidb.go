package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"sync"
)

// Table saves the info of a table
type Table struct {
	Name string
	DB   string
	ID   int64

	Indices map[int64]string
}

func (t *Table) String() string {
	return fmt.Sprintf("%s.%s", t.DB, t.Name)
}

// TableSlice is the slice of tables
type TableSlice []*Table

func (s TableSlice) Len() int      { return len(s) }
func (s TableSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TableSlice) Less(i, j int) bool {
	if s[i].DB < s[j].DB {
		return true
	}

	return s[i].Name < s[j].Name
}

// id -> map
var tables = sync.Map{}

func loadTables() []*Table {
	tbls := make([]*Table, 0, 1024)

	tables.Range(func(_key, value interface{}) bool {
		tbl := value.(*Table)
		tbls = append(tbls, tbl)
		return true
	})

	sort.Sort(TableSlice(tbls))
	return tbls
}

func readBody(addr string, uri string, v interface{}) {
	resp, err := http.Get(fmt.Sprintf("%s/%s", addr, uri))
	perr(err)
	r, err := ioutil.ReadAll(resp.Body)
	perr(err)
	resp.Body.Close()

	err = json.Unmarshal([]byte(r), v)
	perr(err)
}

func updateTables() {
	type dbStruct struct {
		Name struct {
			O string `json:"O"`
			L string `json:"L"`
		} `json:"db_name"`
		State int `json:"state"`
	}

	dbInfo := make([]dbStruct, 0)
	readBody(*tidbAddr, "schema", &dbInfo)

	// TODO: check schema version to avoid duplicated loading

	type tblStruct struct {
		ID   int64 `json:"id"`
		Name struct {
			O string `json:"O"`
			L string `json:"L"`
		} `json:"name"`
		Indices []struct {
			ID   int64 `json:"id"`
			Name struct {
				O string `json:"O"`
				L string `json:"L"`
			} `json:"idx_name"`
		} `json:"index_info"`
	}
	tblInfos := make([]tblStruct, 0)
	for _, info := range dbInfo {
		if info.State == 0 {
			continue
		}

		readBody(*tidbAddr, fmt.Sprintf("schema/%s", info.Name.O), &tblInfos)

		for _, tbl := range tblInfos {
			indices := make(map[int64]string, len(tbl.Indices))
			for _, idx := range tbl.Indices {
				indices[idx.ID] = idx.Name.O
			}
			table := &Table{
				ID:      tbl.ID,
				DB:      info.Name.O,
				Name:    tbl.Name.O,
				Indices: indices,
			}

			tables.Store(tbl.ID, table)
		}
	}
}
