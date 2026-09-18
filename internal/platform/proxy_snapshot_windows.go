//go:build windows

package platform

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Ryujoxys/sushiro-overdose/internal/core"
)

func windowsProxySnapshotPath() string {
	return filepath.Join(core.AppDirPath(), "windows_proxy_backup.json")
}

func captureWindowsProxy() ([]byte, error) {
	out, err := hiddenPowerShellCommand(`
$ErrorActionPreference = 'Stop'
$key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Software\Microsoft\Windows\CurrentVersion\Internet Settings')
if ($null -eq $key) { throw 'Internet Settings unavailable' }
try {
  $names = $key.GetValueNames()
  $rows = @(foreach ($name in @('ProxyServer','ProxyEnable','AutoConfigURL','AutoDetect','ProxyOverride')) {
    if ($names -contains $name) {
      @{Name=$name; Present=$true; Kind=$key.GetValueKind($name).ToString(); Value=$key.GetValue($name, $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)}
    } else { @{Name=$name; Present=$false} }
  })
  $json = ConvertTo-Json -InputObject $rows -Compress -Depth 4
  [Console]::Write([Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($json)))
} finally { $key.Dispose() }
`).Output()
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(string(out))
	if err != nil || !json.Valid(data) {
		return nil, fmt.Errorf("原代理配置格式无效")
	}
	return data, nil
}

func restoreWindowsProxy(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("代理备份损坏，请保留备份并手动检查")
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	_, err := hiddenPowerShellCommand(`
$ErrorActionPreference = 'Stop'
$rows = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($args[0])) | ConvertFrom-Json
$names = @('ProxyServer','ProxyEnable','AutoConfigURL','AutoDetect','ProxyOverride')
if (@($rows).Count -ne $names.Count) { throw 'Invalid proxy snapshot' }
$key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Software\Microsoft\Windows\CurrentVersion\Internet Settings', $true)
try {
  foreach ($row in $rows) {
    if ($names -notcontains $row.Name) { throw 'Invalid proxy value name' }
    if (!$row.Present) { $key.DeleteValue($row.Name, $false); continue }
    $kind = [Enum]::Parse([Microsoft.Win32.RegistryValueKind], [string]$row.Kind)
    $value = $row.Value
    switch ($row.Kind) {
      'DWord' { $value = [int]$value }
      'QWord' { $value = [long]$value }
      'Binary' { $value = [byte[]]$value }
      'MultiString' { $value = [string[]]$value }
      'String' { $value = [string]$value }
      'ExpandString' { $value = [string]$value }
      default { throw 'Unsupported proxy value kind' }
    }
    $key.SetValue($row.Name, $value, $kind)
  }
} finally { if ($key) { $key.Dispose() } }
`, encoded).CombinedOutput()
	refreshProxySettings()
	return err
}

func deleteWindowsProxyValue(name string) error {
	// DeleteValue's false flag treats a missing value as success, not an
	// excuse to swallow access-denied or other registry errors.
	_, err := hiddenPowerShellCommand(`
$ErrorActionPreference = 'Stop'
$key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Software\Microsoft\Windows\CurrentVersion\Internet Settings', $true)
try { $key.DeleteValue($args[0], $false) } finally { $key.Dispose() }
`, name).CombinedOutput()
	return err
}

func setWindowsProxyValues(values [][3]string, remove string) error {
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	steps := []func() error{func() error { return deleteWindowsProxyValue(remove) }}
	for _, value := range values {
		steps = append(steps, func() error {
			return runHiddenWindowsCommand("reg", "add", key, "/v", value[0], "/t", value[1], "/d", value[2], "/f")
		})
	}
	if err := applyProxyTransaction(windowsProxySnapshotPath(), captureWindowsProxy, restoreWindowsProxy, steps...); err != nil {
		return err
	}
	// Do not modify machine-wide WinHTTP settings from an asInvoker GUI.
	// Modern WeChat versions may bypass WinINet; expose that limitation instead.
	refreshProxySettings()
	blockSushiroQUIC()
	return nil
}
