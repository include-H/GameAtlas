<#
.SYNOPSIS
    GameVHD Runtime 断言框架公共库：Assert-True / Write-Step / Finish-Assertions /
    Require-Admin。被同目录 assert_*.ps1 dot-source。

.DESCRIPTION
    统一的验收断言基础设施：
      Assert-True <bool> <string>    断言；失败记录并继续（汇总后非零退出）
      Write-Step <string>            步骤标题输出（带分隔线）
      Finish-Assertions              收尾：打印通过/失败汇总，有失败则 exit 1
      Require-Admin                  非管理员直接 throw（挂载/注入需提权）

    用法（在断言脚本中）：
      . (Join-Path $PSScriptRoot 'assert_common.ps1')
      Require-Admin
      Write-Step "步骤标题"
      Assert-True (Test-Path $x) "描述"
      Finish-Assertions
#>

$script:__gvhd_failures = 0
$script:__gvhd_assertions = 0

function Assert-True {
    param(
        [Parameter(Mandatory = $true)][bool]$Condition,
        [Parameter(Mandatory = $true)][string]$Message
    )
    $script:__gvhd_assertions++
    if ($Condition) {
        Write-Host "  [PASS] $Message" -ForegroundColor Green
    }
    else {
        $script:__gvhd_failures++
        Write-Host "  [FAIL] $Message" -ForegroundColor Red
    }
}

function Write-Step {
    param([Parameter(Mandatory = $true)][string]$Title)
    Write-Host ""
    Write-Host "===== $Title =====" -ForegroundColor Cyan
}

function Finish-Assertions {
    Write-Host ""
    Write-Host ("断言汇总：{0} 项，失败 {1} 项" -f $script:__gvhd_assertions, $script:__gvhd_failures)
    if ($script:__gvhd_failures -gt 0) {
        Write-Host "验收未通过" -ForegroundColor Red
        exit 1
    }
    Write-Host "验收通过" -ForegroundColor Green
    exit 0
}

function Require-Admin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw '需要管理员权限运行（挂载 VHD / 注入需要提权）'
    }
}
