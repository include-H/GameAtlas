<#
.SYNOPSIS
    GameVHD Runtime 阶段 1 注入验收断言：test-app 自检 marker + 子进程链全覆盖。

.DESCRIPTION
    Windows-only。真实注入依赖 CreateRemoteThread 与管理员权限；本机（Linux，无 wine）
    无法运行，本脚本由评审做 parse-check，待 W4T16 接线 `runtime inject` 后在
    Windows 11 上真跑。

    断言内容（每项通过 Assert-True 上报）：
      1. 负向：test-app（x64/x86）直接运行、不经注入 →
         stdout 含 [test-app] NOT_INJECTED、不含 [test-app] HOOK_DLL_PRESENT、
         退出码 1。
      2. 正向 inject 模式：经 `runtime inject` 启动 test-app（args "inject"）→
         test-app stdout 含 INJECT_SELFCHECK_START / PID / HOOK_DLL_PRESENT；
         钩子日志含 HOOK_DLL_PRESENT 与 RULES_LOADED N（N > 0）。
      3. 正向 child 链：`runtime inject --args "child --depth 2"` →
         stdout 恰有 (depth+1) = 3 个 [test-app] HOOK_DLL_PRESENT（3 层全注入）、
         2 个 SPAWN_CHILD；钩子日志含 ≥ 2 行 CHILD_INJECTED；runtime 退出码 0
         （最深层自检码 0 逐层上抛）。

.DEPENDENCIES
      - 同目录 assert_common.ps1（dot-source）。须提供：
          Assert-True <bool> <string>    断言，失败计入汇总
          Write-Step <string>            步骤标题输出
          Finish-Assertions              收尾：汇总 + 失败时非零退出
          Require-Admin                  非管理员直接失败
      - runtime `inject` 子命令（W4T16 接线，契约见 .CONTRACT）。
      - gvhook-x64.dll / gvhook-x86.dll（W2T6 构建）。
      - test-app-x64.exe / test-app-x86.exe（本仓库 make 产物）。
      未提供 -RuntimeExe/-HookDllX64/-HookDllX86 时正向断言块整体 SKIP
      （负向断言块仍然执行）；脚本本身即正向断言的验收契约。

.CONTRACT（W4T16 必须实现的 runtime 子命令，契约文本以脚本头为准）
      gamevhd-runtime.exe inject --exe <path> --dll <gvhook.dll> --log <hook-log> --args <string>

      --exe   目标 PE 绝对路径（位数须与 --dll 一致，协议 §6.3）
      --dll   gvhook DLL 绝对路径（同位数：gvhook-x64.dll / gvhook-x86.dll）
      --log   钩子日志绝对路径。按协议 §7，runtime 在会话开始时【清空】该文件；
              hook 以追加方式写（脚本亦会先行清理，双保险）
      --args  目标程序参数字符串，单个参数，如 "inject" 或 "child --depth 2"；
              runtime 需按空格切分（双引号感知）后作为目标 argv[1..]

      行为要求：
        1. CreateProcessW(CREATE_SUSPENDED) → 协议 §5.2 注入序列 → ResumeThread；
        2. 参数块必须含 ≥ 1 条规则（保证钩子日志出现 RULES_LOADED N 且 N ≥ 1）；
        3. 必须等待目标进程退出，并以目标退出码作为 runtime 退出码；
        4. 目标继承 runtime 的 stdio 与 CWD（断言从 runtime stdout 解析 test-app marker）；
        5. 注入任一步失败走 §5.3 失败路径：TerminateProcess + 非零退出码。

.PARAMETER TestAppX64
    必需。test-app-x64.exe 路径。
.PARAMETER TestAppX86
    必需。test-app-x86.exe 路径。
.PARAMETER RuntimeExe
    gamevhd-runtime.exe 路径。未提供 → 正向断言块 SKIP（W4T16 前置依赖）。
.PARAMETER HookDllX64
    gvhook-x64.dll 路径。
.PARAMETER HookDllX86
    gvhook-x86.dll 路径。
.PARAMETER HookLogDir
    日志临时目录；默认 %TEMP%\gvhook-assert-<guid>。
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$TestAppX64,

    [Parameter(Mandatory = $true)]
    [string]$TestAppX86,

    [string]$RuntimeExe,
    [string]$HookDllX64,
    [string]$HookDllX86,
    [string]$HookLogDir
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2

# --- 框架 ----------------------------------------------------------------
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
. (Join-Path $scriptDir 'assert_common.ps1')
Require-Admin

# --- 参数校验与准备 --------------------------------------------------------
foreach ($p in @($TestAppX64, $TestAppX86)) {
    if (-not (Test-Path -LiteralPath $p -PathType Leaf)) {
        throw "test-app 不存在: $p"
    }
}
if (-not $HookLogDir) {
    $HookLogDir = Join-Path ([IO.Path]::GetTempPath()) ('gvhook-assert-' + [guid]::NewGuid().ToString('N'))
}
New-Item -ItemType Directory -Force -Path $HookLogDir | Out-Null

$positiveReady = $false
if ($RuntimeExe -or $HookDllX64 -or $HookDllX86) {
    foreach ($p in @($RuntimeExe, $HookDllX64, $HookDllX86)) {
        if (-not $p) {
            throw "正向断言需要同时提供 -RuntimeExe / -HookDllX64 / -HookDllX86"
        }
        if (-not (Test-Path -LiteralPath $p -PathType Leaf)) {
            throw "runtime/hook 文件不存在: $p"
        }
    }
    $positiveReady = $true
}

# --- 负向：无注入直接运行 ----------------------------------------------------
function Test-NegativeSelfCheck {
    param([string]$Exe, [string]$Label, [string]$LogFile)
    $out = & $Exe --log $LogFile inject | Out-String
    $code = $LASTEXITCODE
    Assert-True ($out -match '\[test-app\] NOT_INJECTED') "$Label: 无注入时打印 NOT_INJECTED"
    Assert-True ($out -notmatch '\[test-app\] HOOK_DLL_PRESENT') "$Label: 无注入时不打印 HOOK_DLL_PRESENT"
    Assert-True ($code -eq 1) "$Label: 未注入退出码为 1（实际 $code）"
}

Write-Step "负向：x64 无注入自检"
Test-NegativeSelfCheck -Exe $TestAppX64 -Label 'x64' -LogFile (Join-Path $HookLogDir 'neg-x64.log')

Write-Step "负向：x86 无注入自检"
Test-NegativeSelfCheck -Exe $TestAppX86 -Label 'x86' -LogFile (Join-Path $HookLogDir 'neg-x86.log')

# --- 正向：经 runtime inject（契约块，W4T16 接线后生效） -----------------------
function Clear-Log {
    param([string]$Path)
    Remove-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
}

function Test-InjectedInject {
    param([string]$Exe, [string]$Dll, [string]$HookLog, [string]$Label)
    Clear-Log $HookLog
    $out = & $RuntimeExe inject --exe $Exe --dll $Dll --log $HookLog --args 'inject' | Out-String
    $code = $LASTEXITCODE
    Assert-True ($code -eq 0) "$Label: runtime inject 退出码 0（实际 $code）"
    Assert-True ($out -match '\[test-app\] INJECT_SELFCHECK_START') "$Label: 自检起始 marker"
    Assert-True ($out -match '\[test-app\] PID \d+') "$Label: 打印 PID"
    Assert-True ($out -match '\[test-app\] HOOK_DLL_PRESENT') "$Label: GetModuleHandleW 探测到 gvhook"
    $hook = Get-Content -LiteralPath $HookLog -Raw -ErrorAction Stop
    Assert-True ($hook -match 'HOOK_DLL_PRESENT') "$Label: 钩子日志含 HOOK_DLL_PRESENT"
    $m = [regex]::Match($hook, 'RULES_LOADED\s+(\d+)')
    Assert-True $m.Success "$Label: 钩子日志含 RULES_LOADED N"
    if ($m.Success) {
        Assert-True ([int]$m.Groups[1].Value -gt 0) "$Label: RULES_LOADED N 且 N > 0（实际 $($m.Groups[1].Value)）"
    }
}

function Test-ChildChain {
    param([string]$Exe, [string]$Dll, [string]$HookLog, [int]$Depth, [string]$Label)
    Clear-Log $HookLog
    $out = & $RuntimeExe inject --exe $Exe --dll $Dll --log $HookLog --args "child --depth $Depth" | Out-String
    $code = $LASTEXITCODE
    $expected = $Depth + 1
    Assert-True ($code -eq 0) "$Label: child 链 runtime 退出码 0（实际 $code）"
    $present = ([regex]::Matches($out, '\[test-app\] HOOK_DLL_PRESENT')).Count
    Assert-True ($present -eq $expected) "$Label: $expected 层全部注入（HOOK_DLL_PRESENT 计数 $present）"
    $spawns = ([regex]::Matches($out, '\[test-app\] SPAWN_CHILD')).Count
    Assert-True ($spawns -eq $Depth) "$Label: $Depth 次子进程创建（SPAWN_CHILD 计数 $spawns）"
    $hook = Get-Content -LiteralPath $HookLog -Raw -ErrorAction Stop
    $children = ([regex]::Matches($hook, 'CHILD_INJECTED')).Count
    Assert-True ($children -ge $Depth) "$Label: 钩子日志 CHILD_INJECTED ≥ $Depth（实际 $children）"
}

if ($positiveReady) {
    Write-Step "正向：x64 inject 模式"
    Test-InjectedInject -Exe $TestAppX64 -Dll $HookDllX64 `
        -HookLog (Join-Path $HookLogDir 'hook-x64-inject.log') -Label 'x64'

    Write-Step "正向：x64 child 链 --depth 2"
    Test-ChildChain -Exe $TestAppX64 -Dll $HookDllX64 `
        -HookLog (Join-Path $HookLogDir 'hook-x64-child.log') -Depth 2 -Label 'x64'

    Write-Step "正向：x86 inject 模式"
    Test-InjectedInject -Exe $TestAppX86 -Dll $HookDllX86 `
        -HookLog (Join-Path $HookLogDir 'hook-x86-inject.log') -Label 'x86'

    Write-Step "正向：x86 child 链 --depth 2"
    Test-ChildChain -Exe $TestAppX86 -Dll $HookDllX86 `
        -HookLog (Join-Path $HookLogDir 'hook-x86-child.log') -Depth 2 -Label 'x86'
}
else {
    Write-Step "SKIP：正向注入断言需要 runtime inject 子命令（W4T16 接线）与 gvhook DLL；请提供 -RuntimeExe / -HookDllX64 / -HookDllX86。本脚本即该断言的验收契约。"
}

Finish-Assertions
