<#
.SYNOPSIS
    GameVHD Runtime Stage 4 disk layer acceptance: SMB -> create diff ->
    attach -> assign letter full flow.

.DESCRIPTION
    Windows-only (real machine, admin). Drives gamevhd-runtime.exe
    `mount` / `unmount` subcommands (virtdisk API, replaces diskpart).

    NOTE: This script is intentionally pure ASCII. PowerShell 5.1 decodes
    .ps1 files by ANSI codepage (GBK on zh-CN); UTF-8 Chinese would garble
    and break parsing. The runtime prints ASCII markers ([MOUNT-OK] E:)
    to stdout for machine parsing; log detail goes to stderr.

    Checks:
      1. Mount succeeds: exit 0, stdout contains [MOUNT-OK] <letter>:,
         drive letter accessible via Test-Path <letter>:\
      2. Error path: re-mount same diff -> AlreadyAttached (exit non-zero)
      3. Error path: unreachable parent -> mount fails (exit non-zero)
      4. Unmount: exit 0, [UNMOUNT-OK], drive letter gone

.DEPENDENCIES
      - assert_common.ps1 (dot-sourced): Assert-True / Write-Step /
        Finish-Assertions / Require-Admin.
      - gamevhd-runtime.exe (x64, stage 4 build).
      - Real env: NAS SMB share + base VHDX + writable diff dir.

.CONTRACT (stage 4 CLI, contract text is the script header)
      gamevhd-runtime.exe mount <vhd> [--parent <UNC>] [--smb <UNC>]
                        [--user <U>] [--letter <L>] [--retries <N>]
      gamevhd-runtime.exe unmount <vhd> [--letter <L>] [--smb <UNC>]

      mount success prints to stdout: [MOUNT-OK] <letter>:
      unmount success prints to stdout: [UNMOUNT-OK]
.PARAMETER RuntimeExe
    gamevhd-runtime.exe path (required).
.PARAMETER DiffVhd
    Local diff VHD path (required; may not exist - created from --parent).
.PARAMETER ParentUnc
    Base VHD UNC path (required).
.PARAMETER SmbUnc
    SMB share UNC (required).
.PARAMETER SmbUser
    SMB user (optional; defaults to current session).
.PARAMETER SmbPass
    SMB password (optional; paired with SmbUser).
.PARAMETER Letter
    Preferred drive letter (optional; auto if omitted).
.PARAMETER SmbRetries
    SMB retry count (optional; default 3).
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$RuntimeExe,
    [Parameter(Mandatory = $true)]
    [string]$DiffVhd,
    [Parameter(Mandatory = $true)]
    [string]$ParentUnc,
    [Parameter(Mandatory = $true)]
    [string]$SmbUnc,
    [string]$SmbUser,
    [string]$SmbPass,
    [string]$Letter,
    [int]$SmbRetries = 3
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
. (Join-Path $scriptDir 'assert_common.ps1')
Require-Admin

if (-not (Test-Path -LiteralPath $RuntimeExe -PathType Leaf)) {
    throw "runtime not found: $RuntimeExe"
}

if ($SmbPass) {
    if (-not $SmbUser) {
        throw 'SmbUser is required when SmbPass is supplied'
    }
    $credentialArgs = @('store-cred', $SmbUnc, '--user', $SmbUser)
    $credentialOutput = $SmbPass | & $RuntimeExe @credentialArgs 2>&1 | Out-String
    $credentialCode = $LASTEXITCODE
    Assert-True ($credentialCode -eq 0) "store-cred exit 0 (actual $credentialCode): $credentialOutput"
}

$LetterArg = @()
if ($Letter) {
    $LetterArg = @('--letter', $Letter)
}

function Invoke-Runtime {
    # ArgList (not Args: $Args is an automatic variable in PowerShell,
    # using it as a param name silently breaks argument expansion).
    param([string[]]$ArgList)
    $out = & $RuntimeExe @ArgList 2>&1 | Out-String
    return @{ Out = $out; Code = $LASTEXITCODE }
}

# --- 1. Mount full flow ------------------------------------------------------
Write-Step "Mount: SMB -> create diff -> attach -> assign letter"
$mountArgs = @(
    'mount', $DiffVhd,
    '--parent', $ParentUnc,
    '--smb', $SmbUnc,
    '--retries', "$SmbRetries"
) + $LetterArg
if ($SmbUser) { $mountArgs += @('--user', $SmbUser) }

$r = Invoke-Runtime -ArgList $mountArgs
Assert-True ($r.Code -eq 0) "mount exit 0 (actual $($r.Code)): $($r.Out)"
Assert-True ($r.Out -match '\[MOUNT-OK\]') "stdout has [MOUNT-OK] marker: $($r.Out)"

# Letter: preferred --letter, else parse from [MOUNT-OK] <letter>:
$m = [regex]::Match($r.Out, '\[MOUNT-OK\] ([A-Z]):')
if ($Letter) {
    $driveLetter = $Letter.ToUpper()
    Assert-True ($m.Success -and $m.Groups[1].Value -eq $driveLetter) "letter is requested $driveLetter (actual $($m.Groups[1].Value))"
}
else {
    Assert-True $m.Success "[MOUNT-OK] marker carries letter (got: $($r.Out))"
    $driveLetter = $m.Groups[1].Value
}

Assert-True (Test-Path "${driveLetter}:\") "drive ${driveLetter}: accessible"
$disk = Get-Disk | Where-Object { $_.Path -like '*Virtual*' -or $_.FriendlyName -like '*VHD*' }
Assert-True ($null -ne $disk) "a VHD virtual disk is present"

# --- 2. Error path: re-attach -> AlreadyAttached ----------------------------
Write-Step "Error path: re-mount should report AlreadyAttached"
$r2 = Invoke-Runtime -ArgList $mountArgs
Assert-True ($r2.Code -ne 0) "re-mount exit non-zero (actual $($r2.Code)): $($r2.Out)"
Assert-True ($r2.Out -match 'AlreadyAttached|ALREADY_OWNED|VhdAttachFailed') "re-mount error recognizable: $($r2.Out)"

# --- 3. Error path: unreachable parent -> fail ------------------------------
Write-Step "Error path: unreachable parent should fail"
$r3 = Invoke-Runtime -ArgList @(
    'mount', $DiffVhd,
    '--parent', '\\nonexistent-host\share\missing.vhdx'
) + $LetterArg
Assert-True ($r3.Code -ne 0) "bad parent mount exit non-zero (actual $($r3.Code)): $($r3.Out)"

# --- 4. Unmount full flow ----------------------------------------------------
Write-Step "Unmount: remove letter -> detach -> disconnect SMB"
$r4 = Invoke-Runtime -ArgList @(
    'unmount', $DiffVhd, '--letter', $driveLetter, '--smb', $SmbUnc
)
Assert-True ($r4.Code -eq 0) "unmount exit 0 (actual $($r4.Code)): $($r4.Out)"
Assert-True ($r4.Out -match '\[UNMOUNT-OK\]') "stdout has [UNMOUNT-OK]: $($r4.Out)"
Assert-True (-not (Test-Path "${driveLetter}:\")) "drive ${driveLetter}: removed"

Finish-Assertions
