package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// MCP 助手的设计：
// Claude Desktop 自己通过 claude_desktop_config.json 拉起 MCP server（command/args/env）。
// 桌面端不常驻 MCP 子进程（避免和 Claude Desktop 起的重复、避免孤儿进程）。
// 桌面端职责：① 准备 venv（首次启用，让 Claude Desktop 能拉起）
//             ② 写/删 claude_desktop_config.json 的 sushiro 条目（开关）
//             ③ 开机自启时确保 venv 就绪（无需常驻进程）
// 真正的 MCP 进程生命周期由 Claude Desktop 管理。

// MCPStatusJSON 是 /api/mcp/status 的返回。
type MCPStatusJSON struct {
	Enabled             bool   `json:"enabled"`
	AutoStart           bool   `json:"auto_start"`
	PythonReady         bool   `json:"python_ready"` // venv + 依赖是否就绪
	VenvPath            string `json:"venv_path,omitempty"`
	MCPDir              string `json:"mcp_dir,omitempty"`
	ClaudeConfigWritten bool   `json:"claude_config_written"` // claude_desktop_config.json 是否含 sushiro 条目
	ClaudeConfigPath    string `json:"claude_config_path,omitempty"`
	Message             string `json:"message,omitempty"`
	InstallHint         string `json:"install_hint,omitempty"`
}

// mcpVenvPython 返回 venv 内 python 可执行路径。
func mcpVenvPython(mcpDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(mcpDir, "venv", "Scripts", "python.exe")
	}
	return filepath.Join(mcpDir, "venv", "bin", "python")
}

// mcpVenvReady 报告 venv + 关键依赖是否就绪。
func mcpVenvReady(mcpDir string) bool {
	py := mcpVenvPython(mcpDir)
	if _, err := os.Stat(py); err != nil {
		return false
	}
	cmd := exec.Command(py, "-c", "import mcp, httpx, mcp_server.server")
	cmd.Env = append(os.Environ(), "PYTHONPATH="+mcpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		LogMessagef("MCP venv 依赖探测失败: %s: %s", err, string(out))
		return false
	}
	return true
}

// EnsureMCPVenv 确保 mcp/venv 存在且装好依赖。首次启用时调用（可能耗时几十秒）。
func EnsureMCPVenv(mcpDir string) error {
	if mcpVenvReady(mcpDir) {
		return nil
	}
	uv, _ := exec.LookPath("uv")
	python3, _ := exec.LookPath("python3")
	if python3 == "" {
		python3, _ = exec.LookPath("python")
	}
	if uv == "" && python3 == "" {
		return fmt.Errorf("本机未找到 python3 或 uv，请先安装 Python 3.9+")
	}
	if uv != "" {
		LogMessagef("MCP: 用 uv 准备 venv（首次可能需几十秒）...")
		if err := mcpRunVisible(uv, mcpDir, "venv", "venv"); err != nil {
			return fmt.Errorf("uv venv 失败: %w", err)
		}
		if err := mcpRunVisible(uv, mcpDir, "pip", "install", "--python", mcpVenvPython(mcpDir), "-e", "."); err != nil {
			return fmt.Errorf("uv pip install 失败: %w（可能网络问题，稍后重试）", err)
		}
	} else {
		LogMessagef("MCP: 用 python3 准备 venv（首次可能需几十秒）...")
		if err := mcpRunVisible(python3, mcpDir, "-m", "venv", "venv"); err != nil {
			return fmt.Errorf("python venv 失败: %w", err)
		}
		if err := mcpRunVisible(mcpVenvPython(mcpDir), mcpDir, "-m", "pip", "install", "-e", "."); err != nil {
			return fmt.Errorf("pip install 失败: %w（可能网络问题，稍后重试）", err)
		}
	}
	if !mcpVenvReady(mcpDir) {
		return fmt.Errorf("依赖安装后仍探测失败")
	}
	return nil
}

// mcpRunVisible 跑命令（cwd=mcpDir），输出进日志。
func mcpRunVisible(name, dir string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PYTHONPATH="+dir)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		LogMessagef("MCP venv [%s]: %s", name, string(out))
	}
	return err
}

// MCPEnable 开启：准备 venv + 写 claude_desktop_config.json。返回错误（UI 显示）。
func MCPEnable(cfg MCPConfig) error {
	mcpDir, err := materializeMCPAssets()
	if err != nil {
		return fmt.Errorf("释放内嵌 MCP 资源失败: %w", err)
	}
	if err := EnsureMCPVenv(mcpDir); err != nil {
		return err
	}
	if err := registerClaudeDesktop(mcpDir, cfg); err != nil {
		return fmt.Errorf("写入 Claude Desktop 配置失败: %w", err)
	}
	LogMessagef("MCP 已启用：venv 就绪，Claude Desktop 配置已写入")
	return nil
}

// MCPDisable 关闭：从 claude_desktop_config.json 删除 sushiro 条目。
func MCPDisable() {
	unregisterClaudeDesktop()
	LogMessagef("MCP 已禁用：Claude Desktop 配置已移除 sushiro 条目")
}

// MCPStatus 返回当前状态。
func MCPStatus() MCPStatusJSON {
	cfg := LoadMCPConfig()
	mcpDir := findMCPDir()
	st := MCPStatusJSON{
		Enabled:   cfg.Enabled,
		AutoStart: cfg.AutoStart,
		MCPDir:    mcpDir,
	}
	if mcpDir != "" {
		st.VenvPath = filepath.Join(mcpDir, "venv")
		st.PythonReady = mcpVenvReady(mcpDir)
		if !st.PythonReady {
			st.Message = "MCP 源码已内嵌；首次启用需本机 Python 或 uv，并联网安装依赖"
		}
	} else {
		st.Message = "未找到 mcp/ 目录"
		st.InstallHint = "请确认 mcp/ 目录随 sushiro 一起部署（与可执行文件同级）"
	}
	if ccp, err := claudeDesktopConfigPath(); err == nil {
		st.ClaudeConfigPath = ccp
		st.ClaudeConfigWritten = claudeConfigHasSushiro(ccp)
		if ccp == "" {
			if st.Message == "" {
				st.Message = "未检测到 Claude Desktop（装了 Claude Desktop 后会自动注册）"
			}
		}
	}
	if st.Message == "" {
		st.Message = "仅通过本机桌面端读取数据，无需数据库密钥；使用时请保持桌面端运行。"
	}
	return st
}

// registerClaudeDesktop 把 MCP server 注册到 Claude Desktop 配置（合并写，不覆盖其他 MCP）。
func registerClaudeDesktop(mcpDir string, cfg MCPConfig) error {
	path, err := claudeDesktopConfigPath()
	if err != nil {
		return err
	}
	if path == "" {
		return nil // Claude Desktop 未安装，静默跳过（venv 仍准备好，等用户装了 Claude 再启用）
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	existing := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	servers, _ := existing["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers["sushiro"] = map[string]any{
		"command": mcpVenvPython(mcpDir),
		"args":    []string{"-m", "mcp_server"},
		"env": map[string]string{
			"SUSHIRO_MCP_DESKTOP_PORT": fmt.Sprint(GetActiveWebPort()),
			"PYTHONPATH":               mcpDir,
		},
	}
	existing["mcpServers"] = servers
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWriteFile(path, data, 0o600)
}

// unregisterClaudeDesktop 从 Claude Desktop 配置移除 sushiro 条目。
func unregisterClaudeDesktop() {
	path, err := claudeDesktopConfigPath()
	if err != nil || path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	existing := map[string]any{}
	if json.Unmarshal(data, &existing) != nil {
		return
	}
	servers, _ := existing["mcpServers"].(map[string]any)
	if servers == nil {
		return
	}
	delete(servers, "sushiro")
	existing["mcpServers"] = servers
	out, _ := json.MarshalIndent(existing, "", "  ")
	_ = AtomicWriteFile(path, out, 0o600)
}

// claudeConfigHasSushiro 报告 Claude Desktop 配置是否含 sushiro 条目。
func claudeConfigHasSushiro(path string) bool {
	if path == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var existing map[string]any
	if json.Unmarshal(data, &existing) != nil {
		return false
	}
	servers, _ := existing["mcpServers"].(map[string]any)
	_, ok := servers["sushiro"]
	return ok
}

// claudeDesktopConfigPath 返回 Claude Desktop 配置路径（仅当 Claude 已安装时）。
func claudeDesktopConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var candidates []string
	if runtime.GOOS == "darwin" {
		candidates = []string{
			filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		}
	} else if runtime.GOOS == "windows" {
		candidates = []string{
			filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json"),
		}
	}
	for _, p := range candidates {
		if fi, err := os.Stat(filepath.Dir(p)); err == nil && fi.IsDir() {
			return p, nil
		}
	}
	return "", nil
}

// LogMessagef 格式化日志桥（dot import core，调 core.LogMessage）。
func LogMessagef(format string, args ...any) {
	LogMessage(time.Now(), fmt.Sprintf(format, args...))
}
