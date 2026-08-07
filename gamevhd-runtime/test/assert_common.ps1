<#
.SYNOPSIS
    GameVHD Runtime assertion framework common lib: Assert-True /
    Write-Step / Finish-Assertions / Require-Admin. Dot-sourced by
    assert_*.ps1 in the same directory.

.DESCRIPTION
    Unified acceptance assertion infrastructure:
      Assert-True <bool> <string>    assert; record failure, continue,
                                     non-zero exit at Finish-Assertions
      Write-Step <string>            step header output (with separator)
      Finish-Assertions              summary: pass/fail counts, exit 1 on fail
      Require-Admin                  throw if not elevated (mount/inject need it)

    NOTE: pure ASCII on purpose. PowerShell 5.1 decodes .ps1 by ANSI
    codepage; UTF-8 Chinese garbles parsing. Runtime markers are ASCII too.

    Usage (in an assert script):
      . (Join-Path $PSScriptRoot 'assert_common.ps1')
      Require-Admin
      Write-Step "step title"
      Assert-True (Test-Path $x) "description"
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
    Write-Host ("assertions: {0}, failures: {1}" -f $script:__gvhd_assertions, $script:__gvhd_failures)
    if ($script:__gvhd_failures -gt 0) {
        Write-Host "ACCEPTANCE FAILED" -ForegroundColor Red
        exit 1
    }
    Write-Host "ACCEPTANCE PASSED" -ForegroundColor Green
    exit 0
}

function Require-Admin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw 'administrator privileges required (VHD mount / inject need elevation)'
    }
}
