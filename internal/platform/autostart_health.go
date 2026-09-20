package platform

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func checkAutoStartTarget(status AutoStartStatus, target string) AutoStartStatus {
	current, err := os.Executable()
	if err != nil {
		status.Error = "无法读取当前程序位置"
		return status
	}
	return assessAutoStartTarget(status, target, current, runtime.GOOS == "windows", os.Stat)
}

func assessAutoStartTarget(status AutoStartStatus, target, current string, windows bool, stat func(string) (os.FileInfo, error)) AutoStartStatus {
	status.TargetPath, status.CurrentPath = target, current
	if !status.Enabled {
		return status
	}
	if target == "" {
		status.NeedsUpdate, status.Error = true, "无法识别自启动位置，请更新为当前程序"
		return status
	}
	if info, err := stat(target); err != nil || info.IsDir() {
		status.NeedsUpdate, status.Error = true, "自启动程序已移动或无法访问，请更新位置"
		return status
	}
	a, b := filepath.Clean(target), filepath.Clean(current)
	if windows {
		a, b = strings.ToLower(a), strings.ToLower(b)
	}
	if a != b {
		status.NeedsUpdate, status.Error = true, "自启动仍指向其他位置，请更新为当前程序"
	}
	return status
}

func collectorCommandTarget(command string) string {
	command = strings.TrimSpace(command)
	if !strings.HasSuffix(command, " --queue-collector-child") {
		return ""
	}
	target := strings.TrimSpace(strings.TrimSuffix(command, " --queue-collector-child"))
	if strings.HasPrefix(target, `"`) && strings.HasSuffix(target, `"`) {
		return strings.TrimSuffix(strings.TrimPrefix(target, `"`), `"`)
	}
	if strings.ContainsAny(target, "\"\t\r\n ") {
		return ""
	}
	return target
}

func plistCollectorTarget(data []byte) string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		token, err := decoder.Token()
		if err != nil {
			return ""
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "key" {
			continue
		}
		var key string
		if decoder.DecodeElement(&key, &start) != nil || key != "ProgramArguments" {
			continue
		}
		for {
			token, err = decoder.Token()
			if err == io.EOF || err != nil {
				return ""
			}
			if start, ok = token.(xml.StartElement); ok {
				if start.Name.Local != "array" {
					return ""
				}
				var arguments struct {
					Values []string `xml:"string"`
				}
				if decoder.DecodeElement(&arguments, &start) == nil && len(arguments.Values) == 2 && arguments.Values[1] == "--queue-collector-child" {
					return arguments.Values[0]
				}
				return ""
			}
		}
	}
}
