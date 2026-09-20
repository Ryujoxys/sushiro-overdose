package app

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRecordViewKeepsBundledAreaForLocalStoreSearch(t *testing.T) {
	t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
	pack, err := loadBundledHistory()
	if err != nil {
		t.Fatal(err)
	}
	var store historyStore
	for _, candidate := range pack.Stores {
		if candidate.City == "" && strings.Contains(candidate.Area, "广州") && len(pack.byStore[candidate.ID]) > 0 {
			store = candidate
			break
		}
	}
	if store.ID == 0 {
		t.Fatal("expected a real bundled store with its city recorded only in area")
	}
	row := QueueBaselineRecord{StoreID: store.ID, Name: store.Name, CollectedAt: "2026-09-18T11:00:00+08:00", StoreStatus: "OPEN", WaitMinutes: 12, DisplayCalledNo: 345}
	rows := []QueueBaselineRecord{row}
	for _, includeHistory := range []bool{true, false} {
		got := buildRecordView(rows, pack, recordViewSettings{IncludeHistory: includeHistory}, localRecordsQuery{store: store.ID}, historyTestNow())
		var found *historyStore
		for i := range got.AvailableStores {
			if got.AvailableStores[i].ID == store.ID {
				found = &got.AvailableStores[i]
			}
		}
		if found == nil || found.Area != store.Area || found.City != "" {
			t.Fatalf("area-only search metadata lost (history=%t): %+v", includeHistory, found)
		}
		if !includeHistory {
			if len(got.AvailableStores) != 1 || got.History.Included || got.Source != "local" || got.HistorySamples != 0 || got.CalledHistorySamples != 0 {
				t.Fatalf("metadata fallback included historical stores or samples: %+v", got)
			}
			if got.LocalSamples != 1 || got.CalledLocalSamples != 1 || len(got.Points) != 1 || got.Points[0].Median != 12 || len(got.CalledPoints) != 1 || got.CalledPoints[0].Median != 345 {
				t.Fatalf("personal curves changed by search metadata: %+v", got)
			}
		}
	}
	if !reflect.DeepEqual(rows, []QueueBaselineRecord{row}) {
		t.Fatal("search metadata modified personal source records")
	}
	for _, path := range []string{queueBaselineRecordsPath(), queueModelPath()} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("search metadata created personal data at %s: %v", path, err)
		}
	}
}

func TestRecordViewKeepsLocalAreaSearchMetadata(t *testing.T) {
	t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
	pack := fixtureHistory(t)
	pack.Stores = []historyStore{{ID: 1, Name: "历史店名", City: "广州", Area: "广州天河区"}}
	for _, tc := range []struct {
		name    string
		storeID int
		city    string
	}{
		{name: "new area-only store outside bundle", storeID: 100001},
		{name: "local metadata wins over bundle", storeID: 1, city: "佛山"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := []QueueBaselineRecord{
				{StoreID: tc.storeID, Name: "本地店名", City: tc.city, Area: "佛山南海区", CollectedAt: "2026-09-18T11:00:00+08:00", StoreStatus: "OPEN", WaitMinutes: 12},
				{StoreID: tc.storeID, CollectedAt: "2026-09-18T11:05:00+08:00", StoreStatus: "OPEN", WaitMinutes: 15},
			}
			for _, includeHistory := range []bool{true, false} {
				got := buildRecordView(rows, pack, recordViewSettings{IncludeHistory: includeHistory}, localRecordsQuery{store: tc.storeID}, historyTestNow())
				if len(got.Stores) != 1 || got.Stores[0].Name != "本地店名" || got.Stores[0].City != tc.city || got.Stores[0].Area != "佛山南海区" {
					t.Fatalf("local metadata lost (history=%t): %+v", includeHistory, got.Stores)
				}
				var found *historyStore
				for i := range got.AvailableStores {
					if got.AvailableStores[i].ID == tc.storeID {
						found = &got.AvailableStores[i]
					}
				}
				if found == nil || found.Name != "本地店名" || found.City != tc.city || found.Area != "佛山南海区" {
					t.Fatalf("local search metadata lost (history=%t): %+v", includeHistory, found)
				}
				if !includeHistory && (got.HistorySamples != 0 || got.LocalSamples != 1 || len(got.Points) != 1 || got.Points[0].Median != 15) {
					t.Fatalf("metadata changed local samples: %+v", got)
				}
			}
		})
	}
}
