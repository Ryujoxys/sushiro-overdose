package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"
import "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

// Cache parsed snapshots without deleting personal source data. Storage limits
// belong to derived caches, not the append-only observation and baseline files.

// jsonlReadCache 缓存一次 JSONL 解析结果，键为 (size, mtime)。
// 返回的切片是缓存本体：调用方只能读，不得排序/改写/回填——需要派生数据
// 时像现有调用方一样 range 拷贝或建新切片。
type jsonlReadCache[T any] struct {
	mu      sync.Mutex
	path    string
	size    int64
	modNano int64
	rows    []T
}

// Cache only a stable (path, size, mtime) snapshot. If a writer changes the file
// while parsing, return the available rows without caching a partial snapshot.
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
	if c.path == path && c.rows != nil && c.size == info.Size() && c.modNano == info.ModTime().UnixNano() {
		return c.rows
	}
	rows, parseErr := parseJSONLFile[T](path, normalize)
	if parseErr != nil && !os.IsNotExist(parseErr) {
		LogMessage(time.Now(), "读取排队数据文件失败（"+path+"）："+parseErr.Error())
	}
	if info2, err2 := os.Stat(path); err2 == nil {
		if info2.Size() != info.Size() || info2.ModTime() != info.ModTime() {
			c.rows = nil
			return rows // Never cache a partially read file under its newer fingerprint.
		}
	} else {
		// 解析期间文件被删：按空数据处理。
		c.size, c.modNano, c.rows = 0, 0, nil
		return nil
	}
	c.size = info.Size()
	c.path = path
	c.modNano = info.ModTime().UnixNano()
	c.rows = rows
	return c.rows
}

func lockQueueDataFile(path string) (*platform.FileLock, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return platform.LockFile(ctx, path+".lock")
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

var (
	queueObservationsReadCache jsonlReadCache[QueueObservation]
	queueBaselineReadCache     jsonlReadCache[QueueBaselineRecord]
)
