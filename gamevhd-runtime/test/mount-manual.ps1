#requires -RunAsAdministrator
<#
.SYNOPSIS
    Manually attach a GameVHD diff without using Explorer's VHD handler.

.DESCRIPTION
    A differencing VHD may be opened by Explorer before its UNC parent SMB
    session exists. Windows can then show an initialize/format prompt even
    though the parent contains a valid partition and the runtime can mount it.
    This helper establishes the SMB session first, then delegates attach and
    drive-letter assignment to gamevhd-runtime.exe.

    Do not initialize or format a differencing VHD. Do not double-click it.

.PARAMETER RuntimeExe
    gamevhd-runtime.exe path.
.PARAMETER DiffVhd
    Local writable differencing VHD path.
.PARAMETER SmbUnc
    SMB share containing the parent VHD.
.PARAMETER ParentUnc
    Parent VHD UNC path. Optional when DiffVhd already exists.
.PARAMETER SmbUser
    SMB user. Optional when the current Windows session already has access.
.PARAMETER SmbPass
    SMB password. Optional when the current Windows session already has access.
.PARAMETER Letter
    Preferred drive letter. Omit to let Windows choose one.
.PARAMETER SmbRetries
    SMB connection retry count.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$RuntimeExe,
    [Parameter(Mandatory = $true)]
    [string]$DiffVhd,
    [Parameter(Mandatory = $true)]
    [string]$SmbUnc,
    [string]$ParentUnc,
    [string]$SmbUser,
    [string]$SmbPass,
    [ValidatePattern('^[A-Za-z]$')]
    [string]$Letter,
    [ValidateRange(1, 20)]
    [int]$SmbRetries = 3
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2

if (-not (Test-Path -LiteralPath $RuntimeExe -PathType Leaf)) {
    throw "runtime not found: $RuntimeExe"
}
if (-not (Test-Path -LiteralPath $DiffVhd -PathType Leaf) -and
    [string]::IsNullOrWhiteSpace($ParentUnc)) {
    throw "DiffVhd does not exist; ParentUnc is required to create it"
}

$runtimeArgs = @('mount', $DiffVhd, '--smb', $SmbUnc, '--retries', "$SmbRetries")
if (-not [string]::IsNullOrWhiteSpace($ParentUnc)) {
    $runtimeArgs += @('--parent', $ParentUnc)
}
if (-not [string]::IsNullOrWhiteSpace($SmbUser)) {
    $runtimeArgs += @('--user', $SmbUser)
}
if (-not [string]::IsNullOrWhiteSpace($SmbPass)) {
    $runtimeArgs += @('--pass', $SmbPass)
}
if (-not [string]::IsNullOrWhiteSpace($Letter)) {
    $runtimeArgs += @('--letter', $Letter.ToUpperInvariant())
}

Write-Host 'Connecting SMB parent before attaching the differencing VHD.'
Write-Host 'Do not initialize or format the disk if Windows displays that prompt.'
& $RuntimeExe @runtimeArgs
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    throw "GameVHD mount failed with exit code $exitCode"
}
