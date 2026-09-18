# sushiro-overdose 一键安装脚本 (Windows)
# 使用方式：
#   irm https://raw.githubusercontent.com/Ryujoxys/sushiro-overdose/master/install/install.ps1 | iex
# 或下载仓库根目录的 install.bat 双击运行

$ErrorActionPreference = "Stop"

$Repo = "Ryujoxys/sushiro-overdose"
$Archive = "sushiro-overdose"   # release 压缩包前缀（沿用仓库名）
$Binary = "sushiro"             # 压缩包内的 exe + 落地命令名

Write-Host ""
Write-Host "=== sushiro 一键安装 ===" -ForegroundColor Green
Write-Host ""

# 架构检测
$Arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") {
        "arm64"
    } else {
        "amd64"
    }
} else {
    Write-Host "不支持 32 位 Windows" -ForegroundColor Red
    exit 1
}

Write-Host "检测系统: windows/$Arch"

# 获取最新版本
Write-Host "查询最新版本..."
try {
    $Latest = (Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing).tag_name
} catch {
    Write-Host "无法获取最新版本，请检查网络连接" -ForegroundColor Red
    exit 1
}
if (-not $Latest) {
    Write-Host "无法解析最新版本号" -ForegroundColor Red
    exit 1
}
Write-Host "最新版本: $Latest"

# 构造下载链接
$Version = $Latest -replace '^v'
$Filename = "${Archive}_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/$Latest/$Filename"
$AppFilename = "Sushiro-Overdose-${Version}-windows-${Arch}.exe"
$AppUrl = "https://github.com/$Repo/releases/download/$Latest/$AppFilename"

$TempDir = [System.IO.Path]::GetTempPath()
$ZipPath = Join-Path $TempDir $Filename
$AppPath = Join-Path $TempDir $AppFilename
$ExtractDir = Join-Path $TempDir "sushiro-install"

Write-Host "下载 $Url ..."
Invoke-WebRequest -Uri $Url -OutFile $ZipPath -UseBasicParsing

# 解压
Write-Host "解压..."
if (Test-Path $ExtractDir) { Remove-Item $ExtractDir -Recurse -Force }
Expand-Archive -Path $ZipPath -DestinationPath $ExtractDir -Force

# 安装到 %LOCALAPPDATA%\sushiro
$InstallDir = Join-Path $env:LOCALAPPDATA "sushiro"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$ExeSource = Get-ChildItem -Path $ExtractDir -Filter "*.exe" -Recurse | Select-Object -First 1
if (-not $ExeSource) {
    Write-Host "解压结果中未找到 exe 文件" -ForegroundColor Red
    exit 1
}

$ExeTarget = Join-Path $InstallDir "$Binary.exe"
$AppTarget = Join-Path $InstallDir "Sushiro Overdose.exe"

# 如果已在运行，先尝试结束（含旧命令名 sushiro-overdose）
foreach ($ProcessName in @($Binary, "sushiro-overdose", "Sushiro Overdose")) {
    Get-Process -Name $ProcessName -ErrorAction SilentlyContinue | ForEach-Object {
        Write-Host "检测到正在运行的旧版本，正在停止..." -ForegroundColor Yellow
        $_ | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 1
    }
}

Copy-Item $ExeSource.FullName $ExeTarget -Force

# 清理旧版本残留的命令行 exe（命令名已从 sushiro-overdose 改为 sushiro）
$LegacyExe = Join-Path $InstallDir "sushiro-overdose.exe"
if (Test-Path $LegacyExe) {
    try {
        Remove-Item $LegacyExe -Force -ErrorAction Stop
        Write-Host "已清理旧版本 sushiro-overdose.exe" -ForegroundColor Yellow
    } catch {
        Write-Host "旧版本 sushiro-overdose.exe 清理失败（可忽略）: $_" -ForegroundColor DarkYellow
    }
}

# 下载桌面直下版：GUI 子系统，双击不显示终端黑框。
$AppInstalled = $false
try {
    Write-Host "下载桌面应用 $AppUrl ..."
    Invoke-WebRequest -Uri $AppUrl -OutFile $AppPath -UseBasicParsing
    Copy-Item $AppPath $AppTarget -Force
    $AppInstalled = $true
} catch {
    Write-Host "桌面应用直下版下载失败，将使用命令行版作为快捷方式目标: $_" -ForegroundColor DarkYellow
}

# 加入用户 PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "已将 $InstallDir 添加到用户 PATH" -ForegroundColor Yellow
}

# 创建桌面快捷方式
try {
    $Desktop = [Environment]::GetFolderPath("Desktop")
    $LnkPath = Join-Path $Desktop "Sushiro Overdose.lnk"
    $Shell = New-Object -ComObject WScript.Shell
    $Shortcut = $Shell.CreateShortcut($LnkPath)
    $ShortcutTarget = if ($AppInstalled) { $AppTarget } else { $ExeTarget }
    $Shortcut.TargetPath = $ShortcutTarget
    $Shortcut.WorkingDirectory = $InstallDir
    $Shortcut.IconLocation = "$ShortcutTarget,0"
    $Shortcut.Description = "寿司郎 Overdose - 本地排队记录与手动取号"
    $Shortcut.Save()
    Write-Host "已创建桌面快捷方式: Sushiro Overdose.lnk" -ForegroundColor Yellow
} catch {
    Write-Host "桌面快捷方式创建失败（可忽略）: $_" -ForegroundColor DarkYellow
}

# 清理临时文件
Remove-Item $ZipPath -Force -ErrorAction SilentlyContinue
Remove-Item $AppPath -Force -ErrorAction SilentlyContinue
Remove-Item $ExtractDir -Recurse -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "安装完成！sushiro $Latest" -ForegroundColor Green
Write-Host "  命令行工具: $ExeTarget" -ForegroundColor Cyan
if ($AppInstalled) {
    Write-Host "  桌面应用: $AppTarget" -ForegroundColor Cyan
}
Write-Host "  使用方式：双击桌面图标，或新开终端执行 sushiro" -ForegroundColor Cyan
Write-Host ""

# 自动启动询问
$envAuto = $env:SUSHIRO_AUTO_LAUNCH
if ($envAuto -eq "1" -or $envAuto -eq "true") {
    $launch = "y"
} else {
    $launch = Read-Host "现在启动 sushiro? [Y/n]"
}
if ($launch -eq "" -or $launch -match '^[Yy]') {
    Write-Host "启动中..." -ForegroundColor Green
    $LaunchTarget = if ($AppInstalled) { $AppTarget } else { $ExeTarget }
    Start-Process -FilePath $LaunchTarget -WorkingDirectory $InstallDir
}
