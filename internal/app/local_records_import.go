package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// Restoring a personal backup only appends validated, new snapshots to
// queue_baseline.jsonl. Existing rows are never rewritten or deleted, and a file
// that cannot be read completely restores nothing.

const recordsImportMaxBytes = 256 << 20

var (
	errRecordsImportEmpty = errors.New("文件里没有可恢复的记录，请选择本应用导出的 .jsonl 备份。")
	errRecordsImportLocal = errors.New("本机记录读写失败，未恢复任何数据，请检查磁盘空间后重试。")
)

var recordsImportMu sync.Mutex

type localRecordsImportResult struct {
	Imported   int    `json:"imported"`
	Duplicates int    `json:"duplicates"`
	Invalid    int    `json:"invalid"`
	Stores     int    `json:"stores"`
	FirstAt    string `json:"first_at,omitempty"`
	LastAt     string `json:"last_at,omitempty"`
}

// localRecordKey identifies one snapshot the same way the record views deduplicate.
func localRecordKey(row QueueBaselineRecord) (string, time.Time, bool) {
	at, err := time.Parse(time.RFC3339, row.CollectedAt)
	if err != nil || row.StoreID <= 0 {
		return "", time.Time{}, false
	}
	return strconv.Itoa(row.StoreID) + "/" + at.UTC().Format(time.RFC3339Nano), at, true
}

func importLocalRecords(src io.Reader, now time.Time) (localRecordsImportResult, error) {
	recordsImportMu.Lock()
	defer recordsImportMu.Unlock()
	var result localRecordsImportResult
	existing, err := parseJSONLFile[QueueBaselineRecord](queueBaselineRecordsPath(), normalizeQueueBaselineRecordForRead)
	if err != nil && !os.IsNotExist(err) {
		return result, errRecordsImportLocal
	}
	seen := make(map[string]bool, len(existing))
	for _, row := range existing {
		if key, _, ok := localRecordKey(row); ok {
			seen[key] = true
		}
	}

	type datedRow struct {
		row QueueBaselineRecord
		at  time.Time
	}
	var rows []datedRow
	stores := map[int]bool{}
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var row QueueBaselineRecord
		if json.Unmarshal(line, &row) != nil {
			result.Invalid++
			continue
		}
		normalizeQueueBaselineRecordForRead(&row)
		key, at, ok := localRecordKey(row)
		if !ok || at.After(now.Add(time.Hour)) {
			result.Invalid++
			continue
		}
		if seen[key] {
			result.Duplicates++
			continue
		}
		seen[key] = true
		stores[row.StoreID] = true
		rows = append(rows, datedRow{row: row, at: at})
	}
	if err := scanner.Err(); err != nil {
		return localRecordsImportResult{}, errors.New("备份文件读取失败或过大，未恢复任何数据。")
	}
	if len(rows) == 0 && result.Duplicates == 0 {
		return result, errRecordsImportEmpty
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].at.Before(rows[j].at) })
	records := make([]QueueBaselineRecord, len(rows))
	for i, item := range rows {
		records[i] = item.row
	}
	if err := appendQueueBaselineRecords(records); err != nil {
		return localRecordsImportResult{}, errRecordsImportLocal
	}
	if len(records) > 0 {
		if err := rebuildLocalQueueModel(now); err != nil {
			// The model is a cache; views fall back to the raw rows that were just saved.
			LogMessage(now, "恢复记录后重建本机模型失败："+err.Error())
		}
		result.FirstAt = records[0].CollectedAt
		result.LastAt = records[len(records)-1].CollectedAt
	}
	result.Imported, result.Stores = len(records), len(stores)
	return result, nil
}

func handleLocalRecordsImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	result, err := importLocalRecords(http.MaxBytesReader(w, r.Body, recordsImportMaxBytes), time.Now())
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errRecordsImportLocal) {
			status = http.StatusInternalServerError
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, result)
}
