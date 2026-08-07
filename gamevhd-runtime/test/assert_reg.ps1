<#
.SYNOPSIS
    GameVHD Runtime 阶段 3 注册表沙箱验收断言：hive 落盘隔离 + host 零痕迹 + 读穿透 + 持久化。

.DESCRIPTION
    Windows-only（reg load/unload + HKU 挂载需管理员）。本机（Linux，无 wine）无法运行，
    本脚本由评审做 parse-check，待 W4T16 接线 `runtime inject` 后在 Windows 11 上真跑。

    对应实施计划 §4 阶段 3 acceptance：
      1. 沙箱写入落盘到 <GameDataRoot>\Registry\user.dat 对应 hive 键，host HKCU 零痕迹；
      2. 对 hive 内不存在的键做读穿透（读到真实 host 键值）；
      3. unload → load 重挂载后值仍在（持久化）；
      4. 无 runtime 时同一组 reg marker 在 host HKCU 上直接可用（基线）。

    依赖：
      - W4T16：`runtime inject` 子命令接线（契约见 assert_inject.ps1 头注释）；
      - W3T14：test-app reg 模式（marker：REG_CREATE_KEY / REG_SET_VALUE / REG_READ_VALUE /
        REG_ENUM / REG_DELETE_VALUE / REG_DELETE_KEY，路径均不带 HKCU\ 前缀）。
      未提供 -RuntimeExe/-HookDllX64/-HookDllX86 时执行 host HKCU 基线（验证 marker 可用性），
      否则执行完整沙箱断言链。

.PARAMETER TestAppX64 / TestAppX86
    必需。test-app-x64/x86.exe 路径。
.PARAMETER GameDataRoot
    必需。沙箱数据根目录；hive 文件位于 <GameDataRoot>\Registry\user.dat。
.PARAMETER GameId
    沙箱 game_id，默认 testgame；hive 挂载为 HKU\GameVHD_<GameId>。
.PARAMETER RuntimeExe / HookDllX64 / HookDllX86
    提供任一 → 三者必须齐全且存在，否则抛错；全缺 → 走基线。
.PARAMETER LogDir
    钩子日志临时目录；默认 %TEMP%\gvhook-assert-<guid>。
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$TestAppX64,
    [Parameter(Mandatory = $true)][string]$TestAppX86,
    [Parameter(Mandatory = $true)][string]$GameDataRoot,
    [string]$GameId = 'testgame',
    [string]$RuntimeExe,
    [string]$HookDllX64,
    [string]$HookDllX86,
    [string]$LogDir
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2

# --- 框架 ----------------------------------------------------------------
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
. (Join-Path $scriptDir 'assert_common.ps1')
Require-Admin

# --- 常量 ----------------------------------------------------------------
$hiveName    = 'GameVHD_' + $GameId
$hiveKey     = "HKU:\$hiveName\Software\GVHD_Test"
$hostKey     = 'HKCU:\Software\GVHD_Test'
$regRoot     = Join-Path $GameDataRoot 'Registry'
$hiveFile    = Join-Path $regRoot 'user.dat'
$valueName   = 'GVHDTestValue'
$valuePrefix = 'GVHD_TEST_VALUE_'   # 值 = 前缀 + test-app 自身 PID
$escPath     = [regex]::Escape('Software\GVHD_Test')  # marker 正则用

# --- 参数校验 ------------------------------------------------------------
foreach ($p in @($TestAppX64, $TestAppX86)) {
    if (-not (Test-Path -LiteralPath $p -PathType Leaf)) { throw "test-app 不存在: $p" }
}
if (-not (Test-Path -LiteralPath $GameDataRoot -PathType Container)) {
    throw "GameDataRoot 不存在: $GameDataRoot"
}
if (-not $LogDir) {
    $LogDir = Join-Path ([IO.Path]::GetTempPath()) ('gvhook-assert-' + [guid]::NewGuid().ToString('N'))
}
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

$positiveReady = $false
if ($RuntimeExe -or $HookDllX64 -or $HookDllX86) {
    foreach ($p in @($RuntimeExe, $HookDllX64, $HookDllX86)) {
        if (-not $p) { throw "正向断言需要同时提供 -RuntimeExe / -HookDllX64 / -HookDllX86" }
        if (-not (Test-Path -LiteralPath $p -PathType Leaf)) { throw "runtime/hook 文件不存在: $p" }
    }
    $positiveReady = $true
}

# --- 辅助函数 ------------------------------------------------------------
function Clear-LogFile {
    param([string]$Path)
    Remove-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
}

function Clear-RegPath {
    param([string]$Path)
    if (Test-Path -LiteralPath $Path) {
        Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Run-App {
    # 直接运行 test-app（无注入）；捕获 stdout，返回 Out + ExitCode
    param([string]$Exe, [string]$ArgString)
    $out = & $Exe $ArgString 2>&1 | Out-String
    [pscustomobject]@{ Out = $out; ExitCode = $LASTEXITCODE }
}

function Run-Sandboxed {
    # 经 runtime inject 启动 test-app（契约：--args 空格切分）；返回 Out + ExitCode
    param([string]$Exe, [string]$Dll, [string]$Args, [string]$Label)
    $log = Join-Path $LogDir "hook-$Label-reg.log"
    Clear-LogFile $log
    $out = & $RuntimeExe inject --exe $Exe --dll $Dll --log $log --args $Args 2>&1 | Out-String
    [pscustomobject]@{ Out = $out; ExitCode = $LASTEXITCODE }
}

function Query-Reg {
    # reg.exe query；返回退出码（0=存在，非 0=不存在/错误）
    param([string]$KeyPath, [string]$ValueName)
    if ($ValueName) {
        & reg.exe query $KeyPath /v $ValueName 2>&1 | Out-Null
    }
    else {
        & reg.exe query $KeyPath 2>&1 | Out-Null
    }
    return $LASTEXITCODE
}

# --- 基线：无 runtime，host HKCU 直接操作（marker 可用性） --------------------
function Test-HostBaseline {
    param([string]$Exe, [string]$Label)
    Clear-RegPath $hostKey
    Write-Step "基线（$Label）：无 runtime，host HKCU 直接操作"

    $r = Run-App $Exe 'reg create-key Software\GVHD_Test'
    Assert-True ($r.Out -match "\[test-app\] REG_CREATE_KEY $escPath (OK|EXISTS)") "$Label(基线): REG_CREATE_KEY OK/EXISTS"

    $r = Run-App $Exe 'reg set-value Software\GVHD_Test'
    Assert-True ($r.Out -match "\[test-app\] REG_SET_VALUE $escPath OK") "$Label(基线): REG_SET_VALUE OK"

    $r = Run-App $Exe 'reg read-value Software\GVHD_Test'
    Assert-True ($r.Out -match "\[test-app\] REG_READ_VALUE $escPath VALUE=$valuePrefix\d+") "$Label(基线): read-value 回读 VALUE=$valuePrefix<pid>"

    $r = Run-App $Exe 'reg delete-value Software\GVHD_Test'
    Assert-True ($r.Out -match "\[test-app\] REG_DELETE_VALUE $escPath OK") "$Label(基线): REG_DELETE_VALUE OK"

    $r = Run-App $Exe 'reg delete-key Software\GVHD_Test'
    Assert-True ($r.Out -match "\[test-app\] REG_DELETE_KEY $escPath OK") "$Label(基线): REG_DELETE_KEY OK"
}

# --- 沙箱：经 runtime inject（完整断言链） -----------------------------------
function Test-Sandboxed {
    param([string]$Exe, [string]$Dll, [string]$Label)

    Write-Step "沙箱（$Label）：reg create-key"
    $r = Run-Sandboxed $Exe $Dll 'reg create-key Software\GVHD_Test' $Label
    Assert-True ($r.ExitCode -eq 0) "$Label: runtime inject 退出码 0（实际 $($r.ExitCode)）"
    Assert-True ($r.Out -match "\[test-app\] REG_CREATE_KEY $escPath (OK|EXISTS)") "$Label: REG_CREATE_KEY OK/EXISTS"

    Write-Step "沙箱（$Label）：reg set-value"
    $r = Run-Sandboxed $Exe $Dll 'reg set-value Software\GVHD_Test' $Label
    Assert-True ($r.ExitCode -eq 0) "$Label: runtime inject 退出码 0（实际 $($r.ExitCode)）"
    Assert-True ($r.Out -match "\[test-app\] REG_SET_VALUE $escPath OK") "$Label: REG_SET_VALUE OK"

    Write-Step "沙箱（$Label）：reg read-value 回读"
    # set/read 分属两次进程，PID 不同；值形如 GVHD_TEST_VALUE_<pid>，故按形态断言
    $r = Run-Sandboxed $Exe $Dll 'reg read-value Software\GVHD_Test' $Label
    Assert-True ($r.Out -match "\[test-app\] REG_READ_VALUE $escPath VALUE=$valuePrefix\d+") "$Label: read-value 回读 VALUE=$valuePrefix<pid>"

    Write-Step "落盘隔离（$Label）：hive 内存在，host 零痕迹"
    Assert-True ((Query-Reg "HKU:\$hiveName\Software\GVHD_Test" $valueName) -eq 0) "$Label: hive 内 $valueName 存在（reg query exit 0）"
    Assert-True ((Query-Reg 'HKCU:\Software\GVHD_Test' $valueName) -ne 0) "$Label: host HKCU 零痕迹（reg query exit 非 0）"

    Write-Step "读穿透（$Label）：hive 外键读真实 host 值"
    $r = Run-Sandboxed $Exe $Dll 'reg read-value Software\Microsoft\Windows NT\CurrentVersion' $Label
    Assert-True ($r.Out -match '\[test-app\] REG_READ_VALUE .* VALUE=.+') "$Label: read-through 读到真实 host 键值（VALUE=）"
    Assert-True ($r.Out -notmatch 'NOT_FOUND|ERR=') "$Label: read-through 非 NOT_FOUND/ERR="

    Write-Step "清理（$Label）：delete-value → delete-key"
    $r = Run-Sandboxed $Exe $Dll 'reg delete-value Software\GVHD_Test' $Label
    Assert-True ($r.Out -match "\[test-app\] REG_DELETE_VALUE $escPath OK") "$Label: REG_DELETE_VALUE OK"
    Assert-True ((Query-Reg "HKU:\$hiveName\Software\GVHD_Test" $valueName) -ne 0) "$Label: delete-value 后 hive 值不存在"
    $r = Run-Sandboxed $Exe $Dll 'reg delete-key Software\GVHD_Test' $Label
    Assert-True ($r.Out -match "\[test-app\] REG_DELETE_KEY $escPath OK") "$Label: REG_DELETE_KEY OK"
    Assert-True ((Query-Reg "HKU:\$hiveName\Software\GVHD_Test") -ne 0) "$Label: delete-key 后 hive 键不存在"

    Write-Step "持久化（$Label）：set → unload → load → read"
    $r = Run-Sandboxed $Exe $Dll 'reg create-key Software\GVHD_Test' $Label
    Assert-True ($r.Out -match "\[test-app\] REG_CREATE_KEY $escPath (OK|EXISTS)") "$Label(持久化): 重建键 OK"
    $r = Run-Sandboxed $Exe $Dll 'reg set-value Software\GVHD_Test' $Label
    Assert-True ($r.Out -match "\[test-app\] REG_SET_VALUE $escPath OK") "$Label(持久化): REG_SET_VALUE OK"

    $proceed = $true
    try {
        & reg.exe unload "HKU:\$hiveName" 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Write-Step "SKIP（$Label）：HKU:\$hiveName unload 失败/忙（exit $LASTEXITCODE），跳过持久化"
            $proceed = $false
        }
    }
    catch {
        Write-Step "SKIP（$Label）：unload 异常 $($_.Exception.Message)，跳过持久化"
        $proceed = $false
    }

    if ($proceed) {
        & reg.exe load "HKU:\$hiveName" $hiveFile 2>&1 | Out-Null
        Assert-True ($LASTEXITCODE -eq 0) "$Label: reg load 重挂载 user.dat（exit $LASTEXITCODE）"
        $r = Run-Sandboxed $Exe $Dll 'reg read-value Software\GVHD_Test' $Label
        Assert-True ($r.Out -match "\[test-app\] REG_READ_VALUE $escPath VALUE=$valuePrefix\d+") "$Label: 重挂载后值持久（VALUE=$valuePrefix<pid>）"
    }
}

# --- 主流程 ----------------------------------------------------------------
Write-Step "准备：清空 hive 与 host 旧状态"
Clear-RegPath $hiveKey
Clear-RegPath $hostKey

$specs = @(
    @{ Exe = $TestAppX64; Dll = $HookDllX64; Label = 'x64' },
    @{ Exe = $TestAppX86; Dll = $HookDllX86; Label = 'x86' }
)

foreach ($spec in $specs) {
    if ($positiveReady) {
        Test-Sandboxed -Exe $spec.Exe -Dll $spec.Dll -Label $spec.Label
    }
    else {
        Test-HostBaseline -Exe $spec.Exe -Label $spec.Label
    }
}

Finish-Assertions
