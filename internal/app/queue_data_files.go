package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 排队数据 JSONL 文件的读取缓存与行数裁剪。
//
// queue_observations.jsonl / queue_baseline.jsonl 由采样/基准采集持续追加
// （每店每 5 分钟各一条），长期运行会膨胀到几十 MB；而 advisor / trends /
// dashboard 的一个请求路径会多次调用 loader，每次全量读盘 + 逐行反序列化。
// 这里给两个 loader 加 (size, mtime) 内存缓存——文件没变就直接复用上次解析
// 结果——并照抄 history.go 的 trimHistoryLocked 模式给追加写加行数上限。

const (
	// queueObservationMaxLines / queueBaselineMaxLines 是两个 JSONL 的行数上限。
	// 单店 5 分钟一帧时 100k 行 ≈ 1 年；多店按比例缩短，超限裁掉最旧的。
	queueObservationMaxLines = 100000
	queueBaselineMaxLines    = 100000
	// queueJSONLTrimInterval：每追加多少次检查一次是否超限，避免每次 append
	// 都触发昂贵的全量 rewrite（与 history.go historyTrimInterval 同思路）。
	queueJSONLTrimInterval = 200
)

// jsonlReadCache 缓存一次 JSONL 解析结果，键为 (size, mtime)。
// 返回的切片是缓存本体：调用方只能读，不得排序/改写/回填——需要派生数据
// 时像现有调用方一样 range 拷贝或建新切片。
type jsonlReadCache[T any] struct {
	mu      sync.Mutex
	size    int64
	modNano int64
	rows    []T
}

// load 返回 path 的解析结果；(size, mtime) 与上次一致时直接命中缓存，
// 跳过全量读盘和逐行反序列化。未命中才解析，且解析后用「新 stat」作键：
// 解析期间若有并发追加，新 stat 与解析内容对齐，下次 stat 变化会重读——
// 宁可多解析一次也不能把旧键配新内容。
func (c *jsonlReadCache[T]) load(path string, normalize func(*T)) []T {
	if normalize == nil {
		normalize = func(*T) {}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	info, err := os.Stat(path)
	if err != nil {
		c.size, c.modNano, c.rows = 0, 0, nil
		return nil
	}
	if c.rows != nil && c.size == info.Size() && c.modNano == info.ModTime().UnixNano() {
		return c.rows
	}
	rows, parseErr := parseJSONLFile[T](path, normalize)
	if parseErr != nil && !os.IsNotExist(parseErr) {
		LogMessage(time.Now(), "读取排队数据文件失败（"+path+"）："+parseErr.Error())
	}
	if info2, err2 := os.Stat(path); err2 == nil {
		info = info2
	} else {
		// 解析期间文件被删：按空数据处理。
		c.size, c.modNano, c.rows = 0, 0, nil
		return nil
	}
	c.size = info.Size()
	c.modNano = info.ModTime().UnixNano()
	c.rows = rows
	return c.rows
}

// parseJSONLFile 逐行解析 JSONL。与旧 loader 的差异：scanner 错误不再被静默
// 吞掉（磁盘错误/超长行会让调用方拿到半截数据且毫无感知）。
func parseJSONLFile[T any](path string, normalize func(*T)) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := []T{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row T
		if json.Unmarshal([]byte(line), &row) == nil {
			normalize(&row)
			out = append(out, row)
		}
	}
	return out, scanner.Err()
}

// trimJSONLFileLocked 在文件超过 maxLines 时保留最近 maxLines 行并原子 rewrite。
// 调用方必须已持该文件的追加写锁（与 history.go trimHistoryLocked 同一模式）。
// rewrite 失败只记日志不动原文件。
func trimJSONLFileLocked(path string, maxLines int, now time.Time) {
	lines, err := readAllLines(path)
	if err != nil {
		LogMessage(now, "裁剪排队数据文件时读取失败（"+path+"）："+err.Error())
		return
	}
	if len(lines) <= maxLines {
		return
	}
	keep := lines[len(lines)-maxLines:]
	data := make([]byte, 0, len(keep)*160)
	for _, l := range keep {
		data = append(data, l...)
		data = append(data, '\n')
	}
	if err := AtomicWriteFile(path, data, 0o600); err != nil {
		LogMessage(now, "裁剪排队数据文件时写入失败（"+path+"）："+err.Error())
		return
	}
	LogMessage(now, "排队数据已裁剪（"+path+"）：从 "+strconv.Itoa(len(lines))+" 条保留最近 "+strconv.Itoa(len(keep))+" 条")
}

func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

var (
	queueObservationsReadCache jsonlReadCache[QueueObservation]
	queueBaselineReadCache     jsonlReadCache[QueueBaselineRecord]

	// 追加写计数器：与各自的追加锁同锁保护，达到间隔触发一次裁剪检查。
	queueObservationWriteCounter int
	queueBaselineWriteCounter    int
)
