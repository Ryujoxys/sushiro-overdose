package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/platform"

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func handleSampling(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, map[string]any{
			"config":      LoadSamplingConfig(),
			"state":       sampler.GetState(),
			"autostart":   SamplingAutoStartStatus(),
			"queue_state": buildQueueSamplingStatus(time.Now(), QueueTrendSummary{}),
		})
	case http.MethodPost, http.MethodPut:
		var cfg SamplingConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
			return
		}
		cfg = NormalizeSamplingConfig(cfg)
		if err := SaveSamplingConfig(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, "保存失败: "+err.Error())
			return
		}
		if !cfg.Enabled {
			sampler.Stop()
		} else if sampler.IsRunning() {
			if err := sampler.Restart(context.Background(), cfg); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
		}
		writeJSON(w, map[string]any{"ok": true, "config": cfg, "state": sampler.GetState()})
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST")
	}
}

func handleSamplingAutoStart(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, SamplingAutoStartStatus())
	case http.MethodPost, http.MethodPut:
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "无效的请求格式: "+err.Error())
			return
		}
		var err error
		if body.Enabled {
			err = setPublicQueueAutoStart(true)
		} else {
			err = setPublicQueueAutoStart(false)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true, "autostart": SamplingAutoStartStatus()})
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST")
	}
}

func handleSamplingStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if err := sampler.Start(context.Background()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "state": sampler.GetState()})
}

func handleSamplingStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	sampler.Stop()
	writeJSON(w, map[string]any{"ok": true, "state": sampler.GetState()})
}

func handleSamplingOnce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	cfg := LoadSamplingConfig()
	cfg.Enabled = true
	result := sampler.RunOnceNow(r.Context(), cfg)
	sampler.applyRunResult(result)
	writeJSON(w, map[string]any{"ok": !result.Skipped, "result": result, "state": sampler.GetState()})
}
