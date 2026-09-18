package app

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

//go:embed data/queue-history-v1.json.gz
var historyBundleGzip []byte

type historyStore struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	City string `json:"city"`
	Area string `json:"area"`
}

type historyRecord struct {
	StoreID         int    `json:"store_id"`
	CollectedAt     string `json:"collected_at"`
	DateType        string `json:"date_type"`
	StoreStatus     string `json:"store_status"`
	WaitMinutes     *int   `json:"wait_minutes"`
	DisplayCalledNo *int   `json:"display_called_no,omitempty"`
}

type historyMetadata struct {
	Version       int    `json:"version"`
	ID            string `json:"id"`
	Timezone      string `json:"timezone"`
	BucketMinutes int    `json:"bucket_minutes"`
	Source        string `json:"source"`
	ExportedAt    string `json:"exported_at"`
	FirstAt       string `json:"first_at"`
	CutoffAt      string `json:"cutoff_at"`
	QualityNote   string `json:"quality_note"`
}

type historyBundle struct {
	historyMetadata
	Stores  []historyStore  `json:"stores"`
	Records []historyRecord `json:"records"`
	byStore map[int][]recordViewSample
}

type recordViewSample struct {
	storeID  int
	at       time.Time
	dateType string
	wait     *int
	called   *int
	open     bool
	history  bool
}

func (s recordViewSample) key() string {
	return strconv.Itoa(s.storeID) + "/" + s.at.Format("2006-01-02") + "/" + halfHourBucket(s.at)
}

var bundledHistoryOnce sync.Once
var bundledHistory *historyBundle
var bundledHistoryErr error

func loadBundledHistory() (*historyBundle, error) {
	bundledHistoryOnce.Do(func() { bundledHistory, bundledHistoryErr = decodeHistoryBundle(historyBundleGzip) })
	return bundledHistory, bundledHistoryErr
}

func decodeHistoryBundle(data []byte) (*historyBundle, error) {
	z, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer z.Close()
	const maxSize = 128 << 20
	raw, err := io.ReadAll(io.LimitReader(z, maxSize+1))
	if err != nil || len(raw) > maxSize {
		return nil, fmt.Errorf("invalid history archive size or checksum")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var pack historyBundle
	if err := d.Decode(&pack); err != nil {
		return nil, err
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("unexpected trailing history data")
	}
	first, e1 := time.Parse(time.RFC3339, pack.FirstAt)
	cutoff, e2 := time.Parse(time.RFC3339, pack.CutoffAt)
	exported, e3 := time.Parse(time.RFC3339, pack.ExportedAt)
	if pack.Version != 1 || pack.ID == "" || pack.Timezone != "Asia/Shanghai" || pack.BucketMinutes != 30 || pack.Source != "maintainer_snapshot" || e1 != nil || e2 != nil || e3 != nil || first.After(cutoff) || cutoff.After(exported) || len(pack.Records) == 0 {
		return nil, fmt.Errorf("invalid history metadata")
	}
	storeIDs := map[int]bool{}
	for _, s := range pack.Stores {
		if s.ID <= 0 || s.Name == "" || storeIDs[s.ID] {
			return nil, fmt.Errorf("invalid history store")
		}
		storeIDs[s.ID] = true
	}
	pack.byStore = map[int][]recordViewSample{}
	keys := map[string]bool{}
	for _, row := range pack.Records {
		at, err := time.Parse(time.RFC3339, row.CollectedAt)
		if err != nil || !storeIDs[row.StoreID] || at.Before(first) || at.After(cutoff) || (row.WaitMinutes != nil && (*row.WaitMinutes < 0 || *row.WaitMinutes > 1440)) {
			return nil, fmt.Errorf("invalid history sample")
		}
		if row.DisplayCalledNo != nil && (*row.DisplayCalledNo <= 0 || *row.DisplayCalledNo > 2147483647) {
			return nil, fmt.Errorf("invalid historical called number")
		}
		switch row.DateType {
		case "weekday", "workday", "weekend", "holiday":
		default:
			return nil, fmt.Errorf("invalid history date type")
		}
		sample := recordViewSample{storeID: row.StoreID, at: at.In(SushiroTimezone), dateType: row.DateType, wait: row.WaitMinutes, called: row.DisplayCalledNo, open: row.StoreStatus == "OPEN", history: true}
		key := sample.key()
		if keys[key] {
			return nil, fmt.Errorf("duplicate history bucket")
		}
		keys[key] = true
		pack.byStore[row.StoreID] = append(pack.byStore[row.StoreID], sample)
	}
	sort.Slice(pack.Stores, func(i, j int) bool { return pack.Stores[i].ID < pack.Stores[j].ID })
	// Keep only the validated, parsed representation in memory.
	pack.Records = nil
	return &pack, nil
}
