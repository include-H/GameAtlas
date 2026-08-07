<#
.SYNOPSIS
    GameVHD Runtime 阶段 2 文件系统重定向验收断言（W3T11）：test-app file 模式经
    gvhook sandbox 后，%USERPROFILE%\Documents 下的写入被重定向到
    <GameDataRoot>\Users\<username>\Documents\...（VHD 挂载目录），host 零残留。

.DESCRIPTION
    Windows-only（真机）。本脚本即「实施计划 §4 阶段 2：文件系统重定向」的验收
    acceptance：
      - 阶段 1（注入链路）→ assert_inject.ps1
      - 阶段 2（文件系统重定向）→ 本脚本（W3T11）
    依赖波次：
      - W3T10：test-app file 模式（已实现，marker 契约见 .CONTRACT）
      - W3T9 ：gvhook 文件重定向（写入落 sandbox、读穿透、目录列表合并）
      - W2T6 ：gvhook-x64.dll / gvhook-x86.dll
      - W4T16：runtime inject 子命令。须提供 -RuntimeExe/-HookDllX64/-HookDllX86，
               否则 sandbox 断言块整体 SKIP（附明确 Write-Step），仅跑直跑基线
               验证 test-app 自身（W3T10 marker 契约）。本脚本即 sandbox 断言的验收契约。

    管理员：文件读写本身不需要管理员；但 runtime inject（CreateRemoteThread）依赖
    管理员权限，且为与 assert_inject.ps1 保持统一验收行为，故保留 Require-Admin。

    目录映射：
      host 测试目录 : $env:USERPROFILE\Documents\GVHD_FS_TEST
      sandbox 镜像 : <GameDataRoot>\Users\<当前用户名>\Documents\GVHD_FS_TEST
      （%USERPROFILE% → <GameDataRoot>\Users\<username>，Documents 下相对路径保持）

.CONTRACT（test-app file 模式，W3T10；位置参数形式，非 --target）
      test-app.exe file <action> <path> [<path2>]      （rename 需要两个路径）
      等价：file --action <action> <path> ...
      action: new | overwrite | append | read | list | delete | rename | exists

      结果 marker（每行经 tlogf，stdout 形如 "[ts] [test-app] ..."）：
        [test-app] FILE_MODE_START
        [test-app] FILE_NEW <path> OK|EXISTS|ERR=<err>
        [test-app] FILE_OVERWRITE <path> OK|ERR=<err>
        [test-app] FILE_APPEND <path> OK|ERR=<err>
        [test-app] FILE_READ <path> CONTENT=<line>|NOT_FOUND|ERR=<err>
        [test-app] FILE_LIST <path> ENTRY=<name>        （每项一行，已排序）
        [test-app] FILE_DELETE <path> OK|NOT_FOUND|ERR=<err>
        [test-app] FILE_RENAME <old> -> <new> OK|ERR=<err>
        [test-app] FILE_EXISTS <path> YES|NO
      退出码：0 = 全部 OK/YES；1 = 任一 EXISTS/NOT_FOUND/NO/ERR=；2 = 用法错误。
      写入内容固定为 "GVHD_TEST_<TAG>_<pid>\n"（TAG=NEW/OVERWRITE/APPEND），
      append 追加在 EOF（fixture 为 APPEND-SEED-1\n，最终 A+B 两行）。

.CONTRACT（runtime inject，W4T16；本脚本按此调用）
      gamevhd-runtime.exe inject --exe <app> --dll <gvhook.dll> --log <hook-log> --args "<argv...>"
      --args 内路径含空格时以双引号包裹（runtime 双引号感知切分）。注意：Windows
      PowerShell 5.1 会把内嵌双引号转坏，路径含空格时建议 pwsh 7+ 运行或保证用户名
      无空格（默认测试目录在常见用户名下无空格）。

.USAGE
      powershell -ExecutionPolicy Bypass -File assert_fs.ps1 `
          -TestAppX64 "$PWD\test-app\test-app-x64.exe" `
          -TestAppX86 "$PWD\test-app\test-app-x86.exe" `
          -GameDataRoot "E:\GameData" `
          -RuntimeExe "$PWD\runtime\target\release\gamevhd-runtime.exe" `
          -HookDllX64 "$PWD\hook\gvhook-x64.dll" `
          -HookDllX86 "$PWD\hook\gvhook-x86.dll"
      未接线 W4T16 时省略 runtime/hook 参数 → 直跑基线（验证 W3T10 marker 契约）。

.ASSERTION-MATRIX（每个位数一套）
      | 用例               | 直跑基线                           | sandbox（注入）                             |
      | new                | marker OK + host 内容 ✓            | marker OK + 镜像内容 ✓ + host 零残留        |
      | overwrite          | 同上                               | 同上（CREATE_ALWAYS 截断重建）              |
      | append             | marker OK + host A+B               | marker OK + 镜像 A+B + host 零残留          |
      | read               | marker CONTENT= 往返               | marker CONTENT= 往返                        |
      | list               | 两项 ENTRY 均出现                  | host 文件 + 镜像文件 两项 ENTRY 均出现       |
      | delete             | marker OK + host 已删              | marker OK + 镜像已删 + host 从未有          |
      | rename             | marker OK + host 新名在/旧名无     | marker OK + 镜像新名在/旧名无 + host 全无    |
      | exists             | YES/NO + 退出码 0/1                | 同上                                        |
      | 深层目录自动创建     | marker OK + host a\b\c 存在        | marker OK + 镜像 a\b\c 存在 + host 无目录树  |
      | 读穿透（host 既有） | —                                 | marker CONTENT= 命中 host 文件              |

.PARAMETER TestAppX64
    必需。test-app-x64.exe 路径。
.PARAMETER TestAppX86
    必需。test-app-x86.exe 路径。
.PARAMETER GameDataRoot
    必需。sandbox 根（已挂载 VHD 目录），如 E:\GameData。
.PARAMETER RuntimeExe
    gamevhd-runtime.exe 路径。未提供 → sandbox 断言块 SKIP，仅直跑基线。
.PARAMETER HookDllX64
    gvhook-x64.dll 路径。
.PARAMETER HookDllX86
    gvhook-x86.dll 路径。
.PARAMETER LogDir
    日志临时目录；默认 %TEMP%\gvhook-fs-assert-<guid>。
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$TestAppX64,

    [Parameter(Mandatory = $true)]
    [string]$TestAppX86,

    [Parameter(Mandatory = $true)]
    [string]$GameDataRoot,

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

# --- 参数校验 --------------------------------------------------------------
foreach ($p in @($TestAppX64, $TestAppX86)) {
    if (-not (Test-Path -LiteralPath $p -PathType Leaf)) {
        throw "test-app 不存在: $p"
    }
}
if (-not (Test-Path -LiteralPath $GameDataRoot -PathType Directory)) {
    throw "GameDataRoot 不存在（VHD 未挂载或盘符不对）: $GameDataRoot"
}

$sandboxReady = $false
if ($RuntimeExe -or $HookDllX64 -or $HookDllX86) {
    foreach ($p in @($RuntimeExe, $HookDllX64, $HookDllX86)) {
        if (-not $p) {
            throw "sandbox 断言需要同时提供 -RuntimeExe / -HookDllX64 / -HookDllX86"
        }
        if (-not (Test-Path -LiteralPath $p -PathType Leaf)) {
            throw "runtime/hook 文件不存在: $p"
        }
    }
    $sandboxReady = $true
}

if (-not $LogDir) {
    $LogDir = Join-Path ([IO.Path]::GetTempPath()) ('gvhook-fs-assert-' + [guid]::NewGuid().ToString('N'))
}
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

# --- 测试路径与运行态 --------------------------------------------------------
$hostDir = "$env:USERPROFILE\Documents\GVHD_FS_TEST"
$mirrorDir = Join-Path $GameDataRoot ("Users\" + $env:USERNAME + "\Documents\GVHD_FS_TEST")

$script:Mode = 'direct'   # 'direct'（直跑基线）| 'sandbox'（经 runtime inject）
$script:Arch = ''
$script:Exe = ''
$script:Dll = ''
$script:RuntimeExe = $RuntimeExe
$script:LastOutput = ''
$script:LastExit = -1

# --- 工具函数 ----------------------------------------------------------------

function Reset-TestDirs {
    Remove-Item -LiteralPath $hostDir -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $mirrorDir -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force -Path $hostDir | Out-Null
    if ($script:Mode -eq 'sandbox') {
        New-Item -ItemType Directory -Force -Path $mirrorDir | Out-Null
    }
}

function Get-MirrorPath {
    param([string]$HostPath)
    $rel = $HostPath.Substring($hostDir.Length).TrimStart('\')
    return (Join-Path $mirrorDir $rel)
}

# 存储端 = sandbox 模式下为镜像路径，直跑模式下就是 host 路径
function Get-StorePath {
    param([string]$HostPath)
    if ($script:Mode -eq 'sandbox') { return (Get-MirrorPath $HostPath) }
    return $HostPath
}

function New-FileFixture {
    param([string]$Path, [string]$Content)
    $parent = Split-Path -Parent $Path
    if (-not (Test-Path -LiteralPath $parent)) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
    [IO.File]::WriteAllText($Path, $Content)
}

# 运行任意 exe 并捕获 stdout（含 stderr）；退出码留在 $LASTEXITCODE。
# 注：$ErrorActionPreference='Stop' 会把 2>&1 重定向的原生 stderr 变成终止错误，
# 故在函数内临时降级为 Continue，保证 test-app / runtime 的 stderr 输出不致中断脚本。
function Run-App {
    param([string]$Exe, [string[]]$ArgList)
    $prevEAP = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        return (& $Exe @ArgList 2>&1 | Out-String)
    }
    finally {
        $ErrorActionPreference = $prevEAP
    }
}

# --args 内路径含空格时用双引号包裹（runtime 双引号感知切分）
function Format-Arg {
    param([string]$Value)
    if ($Value -match '\s') { return '"' + $Value + '"' }
    return $Value
}

# 拼接 inject --args 字符串：'file <action> <target...>'
function Get-FileArgsString {
    param([string]$Action, [string[]]$Targets)
    $parts = @($Action)
    foreach ($t in $Targets) { $parts += (Format-Arg $t) }
    return ($parts -join ' ')
}

# 直跑用 argv 数组：@('file', '<action>', '<target>', ...)
function Get-FileTokens {
    param([string]$Action, [string[]]$Targets)
    $t = @('file', $Action)
    foreach ($x in $Targets) { $t += $x }
    return ,$t
}

function Invoke-FileAction {
    param([string]$Action, [string[]]$Targets, [string]$LogName)
    if ($script:Mode -eq 'sandbox') {
        $hookLog = Join-Path $LogDir "hook-$($script:Arch)-$LogName.log"
        Remove-Item -LiteralPath $hookLog -Force -ErrorAction SilentlyContinue
        $argsStr = 'file ' + (Get-FileArgsString $Action $Targets)
        $script:LastOutput = Run-App $script:RuntimeExe @(
            'inject', '--exe', $script:Exe, '--dll', $script:Dll,
            '--log', $hookLog, '--args', $argsStr)
        $script:LastExit = $LASTEXITCODE
    }
    else {
        $script:LastOutput = Run-App $script:Exe (Get-FileTokens $Action $Targets)
        $script:LastExit = $LASTEXITCODE
    }
}

# --- 断言辅助 ----------------------------------------------------------------

function Assert-Output {
    param([string]$Regex, [string]$Label)
    Assert-True ($script:LastOutput -match $Regex) "$Label：stdout 匹配 /$Regex/。实际输出：$($script:LastOutput)"
}

function Assert-Exit {
    param([int]$Expected, [string]$Label)
    Assert-True ($script:LastExit -eq $Expected) "$Label：退出码 $Expected（实际 $($script:LastExit)）"
}

function Assert-HostPath {
    param([string]$Path, [bool]$ShouldExist, [string]$Label)
    $exists = Test-Path -LiteralPath $Path -PathType Leaf
    Assert-True ($exists -eq $ShouldExist) "$Label：$Path 存在=$ShouldExist（实际 $exists）"
}

function Assert-StoreContent {
    param([string]$HostPath, [string]$ContentRegex, [string]$Label)
    $sp = Get-StorePath $HostPath
    Assert-HostPath $sp $true "$Label：存储端文件存在（$sp）"
    if (Test-Path -LiteralPath $sp -PathType Leaf) {
        $text = [IO.File]::ReadAllText($sp)
        Assert-True ($text -match $ContentRegex) "$Label：存储端内容匹配 /$ContentRegex/（实际 '$text'）"
    }
}

# --- 各动作用例（直跑与 sandbox 共用；sandbox 额外断言 host 零残留） ------------

function Test-ActionNew {
    param([string]$FileName)
    Reset-TestDirs
    $hp = Join-Path $hostDir $FileName
    Invoke-FileAction -Action 'new' -Targets @($hp) -LogName 'new'
    $lab = "$($script:Arch)-$($script:Mode) new"
    Assert-Output ('\[test-app\] FILE_MODE_START') $lab
    Assert-Output ('\[test-app\] FILE_NEW ' + [regex]::Escape($hp) + ' OK') $lab
    Assert-Exit 0 $lab
    Assert-StoreContent $hp '^GVHD_TEST_NEW_\d+\r?\n?$' $lab
    if ($script:Mode -eq 'sandbox') { Assert-HostPath $hp $false "$lab：host 零残留" }
}

function Test-ActionOverwrite {
    param([string]$FileName)
    Reset-TestDirs
    $hp = Join-Path $hostDir $FileName
    New-FileFixture (Get-StorePath $hp) "OVERWRITE-SEED-1`n"
    Invoke-FileAction -Action 'overwrite' -Targets @($hp) -LogName 'overwrite'
    $lab = "$($script:Arch)-$($script:Mode) overwrite"
    Assert-Output ('\[test-app\] FILE_OVERWRITE ' + [regex]::Escape($hp) + ' OK') $lab
    Assert-Exit 0 $lab
    Assert-StoreContent $hp '^GVHD_TEST_OVERWRITE_\d+\r?\n?$' $lab
    if ($script:Mode -eq 'sandbox') { Assert-HostPath $hp $false "$lab：host 零残留" }
}

function Test-ActionAppend {
    param([string]$FileName)
    Reset-TestDirs
    $hp = Join-Path $hostDir $FileName
    New-FileFixture (Get-StorePath $hp) "APPEND-SEED-1`n"
    Invoke-FileAction -Action 'append' -Targets @($hp) -LogName 'append'
    $lab = "$($script:Arch)-$($script:Mode) append"
    Assert-Output ('\[test-app\] FILE_APPEND ' + [regex]::Escape($hp) + ' OK') $lab
    Assert-Exit 0 $lab
    Assert-StoreContent $hp '^APPEND-SEED-1\r?\nGVHD_TEST_APPEND_\d+\r?\n?$' $lab
    if ($script:Mode -eq 'sandbox') { Assert-HostPath $hp $false "$lab：host 零残留" }
}

function Test-ActionRead {
    param([string]$FileName)
    Reset-TestDirs
    $hp = Join-Path $hostDir $FileName
    New-FileFixture (Get-StorePath $hp) "READ-SEED-CONTENT`n"
    Invoke-FileAction -Action 'read' -Targets @($hp) -LogName 'read'
    $lab = "$($script:Arch)-$($script:Mode) read"
    Assert-Output ('\[test-app\] FILE_READ ' + [regex]::Escape($hp) + ' CONTENT=READ-SEED-CONTENT') $lab
    Assert-Exit 0 $lab
}

function Test-ActionList {
    Reset-TestDirs
    if ($script:Mode -eq 'sandbox') {
        New-FileFixture (Join-Path $hostDir 'hostonly.txt') "HOST-LIST-SEED`n"
        New-FileFixture (Join-Path $mirrorDir 'sandboxonly.txt') "SANDBOX-LIST-SEED`n"
    }
    else {
        New-FileFixture (Join-Path $hostDir 'hostonly.txt') "HOST-LIST-SEED`n"
        New-FileFixture (Join-Path $hostDir 'sandboxonly.txt') "SANDBOX-LIST-SEED`n"
    }
    Invoke-FileAction -Action 'list' -Targets @($hostDir) -LogName 'list'
    $lab = "$($script:Arch)-$($script:Mode) list"
    Assert-Output ('\[test-app\] FILE_LIST ' + [regex]::Escape($hostDir) + ' ENTRY=hostonly\.txt') $lab
    Assert-Output ('\[test-app\] FILE_LIST ' + [regex]::Escape($hostDir) + ' ENTRY=sandboxonly\.txt') $lab
    Assert-Exit 0 $lab
}

function Test-ActionDelete {
    param([string]$FileName)
    Reset-TestDirs
    $hp = Join-Path $hostDir $FileName
    New-FileFixture (Get-StorePath $hp) "DELETE-SEED-1`n"
    Invoke-FileAction -Action 'delete' -Targets @($hp) -LogName 'delete'
    $lab = "$($script:Arch)-$($script:Mode) delete"
    Assert-Output ('\[test-app\] FILE_DELETE ' + [regex]::Escape($hp) + ' OK') $lab
    Assert-Exit 0 $lab
    Assert-HostPath (Get-StorePath $hp) $false "$lab：存储端文件已删除"
    if ($script:Mode -eq 'sandbox') { Assert-HostPath $hp $false "$lab：host 从未有文件" }
}

function Test-ActionRename {
    param([string]$OldName, [string]$NewName)
    Reset-TestDirs
    $hop = Join-Path $hostDir $OldName
    $hnp = Join-Path $hostDir $NewName
    New-FileFixture (Get-StorePath $hop) "RENAME-SEED-1`n"
    Invoke-FileAction -Action 'rename' -Targets @($hop, $hnp) -LogName 'rename'
    $lab = "$($script:Arch)-$($script:Mode) rename"
    Assert-Output ('\[test-app\] FILE_RENAME ' + [regex]::Escape($hop) + ' -> ' + [regex]::Escape($hnp) + ' OK') $lab
    Assert-Exit 0 $lab
    Assert-HostPath (Get-StorePath $hnp) $true "$lab：存储端新名存在"
    Assert-HostPath (Get-StorePath $hop) $false "$lab：存储端旧名已消失"
    if ($script:Mode -eq 'sandbox') {
        Assert-HostPath $hop $false "$lab：host 旧名零残留"
        Assert-HostPath $hnp $false "$lab：host 新名零残留"
    }
}

function Test-ActionExists {
    param([string]$FileName)
    Reset-TestDirs
    $hp = Join-Path $hostDir $FileName
    New-FileFixture (Get-StorePath $hp) "EXISTS-SEED-1`n"
    Invoke-FileAction -Action 'exists' -Targets @($hp) -LogName 'exists-yes'
    $lab = "$($script:Arch)-$($script:Mode) exists"
    Assert-Output ('\[test-app\] FILE_EXISTS ' + [regex]::Escape($hp) + ' YES') $lab
    Assert-Exit 0 $lab
    $missing = Join-Path $hostDir 'missing.txt'
    Invoke-FileAction -Action 'exists' -Targets @($missing) -LogName 'exists-no'
    Assert-Output ('\[test-app\] FILE_EXISTS ' + [regex]::Escape($missing) + ' NO') $lab
    Assert-Exit 1 "$lab：不存在 → 退出码 1"
}

function Test-DeepDirAutoCreate {
    Reset-TestDirs
    $hp = Join-Path $hostDir 'a\b\c\deep.txt'
    Invoke-FileAction -Action 'new' -Targets @($hp) -LogName 'deep'
    $lab = "$($script:Arch)-$($script:Mode) 深层目录自动创建"
    Assert-Output ('\[test-app\] FILE_NEW ' + [regex]::Escape($hp) + ' OK') $lab
    Assert-Exit 0 $lab
    Assert-StoreContent $hp '^GVHD_TEST_NEW_\d+\r?\n?$' $lab
    if ($script:Mode -eq 'sandbox') {
        Assert-True (-not (Test-Path -LiteralPath (Join-Path $hostDir 'a'))) "$lab：host 未创建 a\b\c 目录树（零残留）"
    }
}

function Test-ReadThrough {
    Reset-TestDirs
    $hp = Join-Path $hostDir 'existing.txt'
    New-FileFixture $hp "HOST-ONLY-SEED`n"
    Invoke-FileAction -Action 'read' -Targets @($hp) -LogName 'readthrough'
    $lab = "$($script:Arch)-$($script:Mode) 读穿透 host 既有文件"
    Assert-Output ('\[test-app\] FILE_READ ' + [regex]::Escape($hp) + ' CONTENT=HOST-ONLY-SEED') $lab
    Assert-Exit 0 $lab
}

# --- 主流程：x64 与 x86 各跑一套 ------------------------------------------------

$archRuns = @(
    @{ Arch = 'x64'; App = $TestAppX64; Dll = $HookDllX64 },
    @{ Arch = 'x86'; App = $TestAppX86; Dll = $HookDllX86 }
)

foreach ($run in $archRuns) {
    $script:Arch = $run.Arch
    $script:Exe = $run.App

    if ($sandboxReady) {
        $script:Mode = 'sandbox'
        $script:Dll = $run.Dll
        Write-Step "Sandbox：$($run.Arch) file 经 runtime inject（W4T16 + W2T6 + W3T9）；写入重定向到 $mirrorDir"
    }
    else {
        $script:Mode = 'direct'
        Write-Step "SKIP sandbox：未提供 -RuntimeExe / -HookDllX64 / -HookDllX86（W4T16 未接线）；改为直跑基线，仅验证 test-app file 模式（W3T10）marker 契约。"
        Write-Step "基线：$($run.Arch) file 直跑（无注入）"
    }

    Reset-TestDirs

    Test-ActionNew 'save1.dat'
    Test-ActionOverwrite 'save2.dat'
    Test-ActionAppend 'save3.dat'
    Test-ActionRead 'save4.dat'
    Test-ActionList
    Test-ActionDelete 'save5.dat'
    Test-ActionRename 'old.dat' 'new.dat'
    Test-ActionExists 'save-exists.dat'
    Test-DeepDirAutoCreate

    if ($script:Mode -eq 'sandbox') {
        Test-ReadThrough
    }
}

Finish-Assertions
