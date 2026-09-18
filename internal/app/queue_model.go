package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

type LocalQueueModel struct {
	Version       int                 `json:"version"`
	Method        string              `json:"method"`
	GeneratedAt   string              `json:"generated_at"`
	SampleCount   int                 `json:"sample_count"`
	DayCount      int                 `json:"day_count"`
	SourceStamp   string              `json:"source_stamp"`
	CalendarStamp string              `json:"calendar_stamp"`
	Baseline      QueueBaselineExport `json:"baseline"`
}

type LocalQueueModelStatus struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	GeneratedAt string `json:"generated_at,omitempty"`
	SampleCount int    `json:"sample_count"`
	DayCount    int    `json:"day_count"`
	StoreCount  int    `json:"store_count"`
	Stale       bool   `json:"stale"`
}

func queueModelPath() string { return filepath.Join(AppDirPath(), "queue_model.json") }

func queueFileStamp(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
}

func buildLocalQueueModel(records []QueueBaselineRecord, now time.Time) LocalQueueModel {
	if now.IsZero() {
		now = time.Now()
	}
	baseline := buildLocalQueueBaselineExport(records, now)
	model := LocalQueueModel{Version: 1, Method: "local_empirical_quantiles", GeneratedAt: baseline.GeneratedAt, Baseline: baseline}
	for _, rollup := range baseline.Rollups {
		model.SampleCount += rollup.SampleCount
	}
	days := map[string]bool{}
	for _, record := range records {
		normalizeQueueBaselineRecordForRead(&record)
		at, err := time.Parse(time.RFC3339, record.CollectedAt)
		if err == nil && record.StoreID > 0 && !at.After(now) {
			days[at.In(SushiroTimezone).Format("2006-01-02")] = true
		}
	}
	model.DayCount = len(days)
	return model
}

func loadLocalQueueModel() (LocalQueueModel, error) {
	var model LocalQueueModel
	data, err := os.ReadFile(queueModelPath())
	if err != nil {
		return model, err
	}
	if err := json.Unmarshal(data, &model); err != nil {
		return model, err
	}
	if model.Version != 1 || model.Baseline.Version != queueBaselineSchemaVersion || model.Baseline.Source != "local" {
		return model, fmt.Errorf("本机模型版本或数据来源不匹配")
	}
	return model, nil
}

func localQueueModelCurrent(model LocalQueueModel, now time.Time) bool {
	at, err := time.Parse(time.RFC3339, model.GeneratedAt)
	return err == nil && !at.After(now) && now.Sub(at) < 24*time.Hour &&
		model.SourceStamp == queueFileStamp(queueBaselineRecordsPath()) &&
		model.CalendarStamp == queueFileStamp(filepath.Join(AppDirPath(), "holidays.json"))
}

// Called by the public writer while holding the cross-process collection lock.
func rebuildLocalQueueModel(now time.Time) error {
	source := queueFileStamp(queueBaselineRecordsPath())
	calendar := queueFileStamp(filepath.Join(AppDirPath(), "holidays.json"))
	if model, err := loadLocalQueueModel(); err == nil && localQueueModelCurrent(model, now) {
		return nil
	}
	records, err := parseJSONLFile[QueueBaselineRecord](queueBaselineRecordsPath(), normalizeQueueBaselineRecordForRead)
	if err != nil {
		return err
	}
	model := buildLocalQueueModel(records, now)
	if source != queueFileStamp(queueBaselineRecordsPath()) {
		return fmt.Errorf("历史文件正在变化，下轮重新生成模型")
	}
	model.SourceStamp, model.CalendarStamp = source, calendar
	data, err := json.Marshal(model)
	if err != nil {
		return err
	}
	return AtomicWriteFile(queueModelPath(), data, 0o600)
}

func localQueueModelStatus(now time.Time) LocalQueueModelStatus {
	status := LocalQueueModelStatus{Status: "empty", Message: "尚未生成本机模型；选店后开启公开采集即可，不需要认证。已有历史也会参与下一轮统计。"}
	model, err := loadLocalQueueModel()
	if err != nil {
		if !os.IsNotExist(err) {
			status.Status, status.Message = "error", "本机模型读取失败："+err.Error()
		}
		return status
	}
	status.GeneratedAt, status.SampleCount, status.DayCount = model.GeneratedAt, model.SampleCount, model.DayCount
	status.StoreCount = model.Baseline.Stats.StoreCount
	status.Stale = !localQueueModelCurrent(model, now)
	status.Status, status.Message = "collecting", "正在积累本机规律；样本和覆盖日期不足时，不保证预测准确。"
	if model.DayCount >= 7 && model.SampleCount >= 100 {
		status.Status, status.Message = "history_available", "已有跨日历史，可参考等待区间与时段规律；仍以门店叫号为准。"
	}
	if status.Stale {
		status.Message += " 模型待下一轮采集更新。"
	}
	return status
}

func localQueueBaselineForRecords(records []QueueBaselineRecord, now time.Time) QueueBaselineExport {
	if model, err := loadLocalQueueModel(); err == nil && localQueueModelCurrent(model, now) {
		return model.Baseline
	}
	return buildLocalQueueBaselineExport(records, now)
}
