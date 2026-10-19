package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const latestReleaseAPI = "https://api.github.com/repos/Ryujoxys/sushiro-overdose/releases/latest"

type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	URL             string `json:"url,omitempty"`
	CheckedAt       string `json:"checked_at"`
	Error           string `json:"error,omitempty"`
}

var updateCache = struct {
	sync.Mutex
	info      UpdateInfo
	checkedAt time.Time
}{}

func CheckLatestRelease(ctx context.Context) UpdateInfo {
	updateCache.Lock()
	// A failed check is retried soon, so a later click is not stuck on an old network error.
	ttl := 30 * time.Minute
	if updateCache.info.Error != "" {
		ttl = time.Minute
	}
	if !updateCache.checkedAt.IsZero() && time.Since(updateCache.checkedAt) < ttl {
		info := updateCache.info
		updateCache.Unlock()
		return info
	}
	updateCache.Unlock()

	info := UpdateInfo{
		CurrentVersion: Version,
		CheckedAt:      time.Now().Format(time.RFC3339),
	}
	if strings.TrimSpace(Version) == "" || Version == "dev" {
		return cacheUpdateInfo(info)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseAPI, nil)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sushiro-overdose/"+Version)
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		info.Error = err.Error()
		return cacheUpdateInfo(info)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		info.Error = resp.Status
		return cacheUpdateInfo(info)
	}
	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		info.Error = err.Error()
		return cacheUpdateInfo(info)
	}
	info.LatestVersion = payload.TagName
	info.URL = payload.HTMLURL
	info.UpdateAvailable = compareVersions(payload.TagName, Version) > 0
	return cacheUpdateInfo(info)
}

func cacheUpdateInfo(info UpdateInfo) UpdateInfo {
	updateCache.Lock()
	updateCache.info = info
	updateCache.checkedAt = time.Now()
	updateCache.Unlock()
	return info
}

func compareVersions(a, b string) int {
	aa := versionParts(a)
	bb := versionParts(b)
	for i := 0; i < 3; i++ {
		if aa[i] > bb[i] {
			return 1
		}
		if aa[i] < bb[i] {
			return -1
		}
	}
	return 0
}

func versionParts(v string) [3]int {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	chunks := strings.Split(v, ".")
	var out [3]int
	for i := 0; i < len(chunks) && i < 3; i++ {
		n, _ := strconv.Atoi(strings.TrimFunc(chunks[i], func(r rune) bool { return r < '0' || r > '9' }))
		out[i] = n
	}
	return out
}
