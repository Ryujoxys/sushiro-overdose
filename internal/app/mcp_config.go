package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// MCPConfig 只保存本机助手开关。旧数据库字段在读取时忽略。
type MCPConfig struct {
	Enabled   bool `json:"enabled"`
	AutoStart bool `json:"auto_start"`
}

var (
	mcpConfigMu sync.Mutex
)

// MCPConfigPath 返回 MCP 配置落盘路径。
func MCPConfigPath() string {
	return filepath.Join(AppDirPath(), "mcp_config.json")
}

// DefaultMCPConfig 返回默认（空）MCP 配置。
func DefaultMCPConfig() MCPConfig {
	return MCPConfig{}
}

func NormalizeMCPConfig(cfg MCPConfig) MCPConfig {
	return cfg
}

// LoadMCPConfig 读 MCP 配置，失败/不存在返回默认。
func LoadMCPConfig() MCPConfig {
	mcpConfigMu.Lock()
	defer mcpConfigMu.Unlock()
	data, err := os.ReadFile(MCPConfigPath())
	if err != nil {
		return DefaultMCPConfig()
	}
	var cfg MCPConfig
	if json.Unmarshal(data, &cfg) != nil {
		return DefaultMCPConfig()
	}
	return NormalizeMCPConfig(cfg)
}

// SaveMCPConfig 原子写 MCP 配置（0600）。
func SaveMCPConfig(cfg MCPConfig) error {
	mcpConfigMu.Lock()
	defer mcpConfigMu.Unlock()
	cfg = NormalizeMCPConfig(cfg)
	os.MkdirAll(AppDirPath(), 0o755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWriteFile(MCPConfigPath(), data, 0o600)
}
