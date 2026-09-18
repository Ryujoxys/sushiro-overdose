package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// 路径约定：所有运行期数据统一放在用户主目录下的 ~/.sushiro（点开头表示隐藏目录）。
// 这样无论从哪个 CWD 启动都能找到同一份数据（区别于早期把文件散落在当前目录的写法）。
// 之所以不用 XDG/平台标准目录，是为了单二进制、零依赖、跨平台一致。
const (
	appDir      = ".sushiro"
	pidFile     = "sushiro.pid"
	logFileName = "sushiro.log"
)

// AtomicWriteFile 原子地写入文件：先写同目录的随机临时文件，再用 os.Rename 原子替换目标。
// os.Rename 在同一目录内是原子的；临时文件必须与目标同目录，否则跨文件系统会退化为非原子 copy。
// 相比裸 os.WriteFile（O_TRUNC+write）的截断窗口：并发读在截断期间会读到半截 JSON 解析失败
// （例如采样循环读到 Enabled=false 后静默停止，或 preferences 被静默重置为默认值）。
// 用 os.CreateTemp 生成唯一临时名，避免并发写同一目标时多个写者争抢固定的 path+".tmp"。
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := func() { _ = os.Remove(tmp) }
	if _, err := f.Write(data); err != nil {
		f.Close()
		cleanup()
		return err
	}
	if err := f.Chmod(perm); err != nil {
		f.Close()
		cleanup()
		return err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// HasCustomDataHome reports an explicitly isolated application data root.
func HasCustomDataHome() bool { return os.Getenv("SUSHIRO_DATA_HOME") != "" }

// ValidateDataHome rejects ambiguous paths before any startup reads or writes.
func ValidateDataHome() error {
	if home := os.Getenv("SUSHIRO_DATA_HOME"); home != "" && !filepath.IsAbs(home) {
		return fmt.Errorf("SUSHIRO_DATA_HOME 必须是绝对路径；请勿通过覆盖 HOME 隔离应用数据")
	}
	return nil
}

// DataHomeDir isolates app files without changing the OS user's keychain or profile.
func DataHomeDir() string {
	if home := os.Getenv("SUSHIRO_DATA_HOME"); home != "" {
		return filepath.Clean(home)
	}
	home, _ := os.UserHomeDir()
	return home
}

// AppDirPath 返回本应用的数据目录，默认 ~/.sushiro。
func AppDirPath() string {
	return filepath.Join(DataHomeDir(), appDir)
}

// PidFilePath 返回守护进程 PID 文件路径（~/.sushiro/sushiro.pid），用于单实例与进程管理。
func PidFilePath() string { return filepath.Join(AppDirPath(), pidFile) }

// StateFilePath 返回运行态文件路径（~/.sushiro/.sushiro_state.json）。
// 注意：这是「抓包/取号」链路的默认 state 路径；config.go 里 LoadSettings 的 state 文件
// 默认在 config.json 同目录，二者可能不同（手动配置 vs 默认）。
func StateFilePath() string { return filepath.Join(AppDirPath(), ".sushiro_state.json") }

// LogPath 返回运行日志文件路径（~/.sushiro/sushiro.log）。
func LogPath() string { return filepath.Join(AppDirPath(), logFileName) }
