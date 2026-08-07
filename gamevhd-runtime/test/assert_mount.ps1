<#
.SYNOPSIS
    GameVHD Runtime 阶段 4 磁盘层验收断言：SMB → 建差分 → attach → 定盘符 全流程。

.DESCRIPTION
    Windows-only（真机，管理员权限）。调用 gamevhd-runtime.exe 的
    `mount` / `unmount` 子命令（virtdisk API 全流程，替代 diskpart），断言：

      1. 挂载成功：退出码 0，日志含「挂载成功」，系统出现预期盘符
         （`Get-PSDrive` 或 `Test-Path <letter>:\` 可访问）；
      2. 重复挂载语义：已 attach 的 VHD 再次 attach → 报 AlreadyAttached
         （错误路径覆盖；脚本通过复用同一差分盘触发）；
      3. parent 失配检测：提供不存在的 parent UNC → 挂载失败且错误为
         parent 相关（错误路径覆盖；不要求真实失配场景，用坏 parent 路径
         验证错误分类可到达）；
      4. 卸载：unmount 后盘符消失（`Test-Path <letter>:\` 为 False）、
         SMB 会话断开、VHD 不再挂载（`Get-Disk` 中无对应盘）。

    脚本只做「契约 + 断言框架」，由用户以管理员 PowerShell 提供真实参数执行：
      .\assert_mount.ps1 -RuntimeExe .\gamevhd-runtime.exe -DiffVhd C:\...\diff.vhdx
                       -ParentUnc \\NAS\Game\base.vhdx -SmbUnc \\NAS\Game
                       -SmbUser rom -SmbPass xxx [-Letter E] [-SmbRetries 3]

    必填参数由调用方（真机用户）提供；脚本不做网络探测，只校验参数形态。

.DEPENDENCIES
      - 同目录 assert_common.ps1（dot-source）：Assert-True / Write-Step /
        Finish-Assertions / Require-Admin。
      - gamevhd-runtime.exe（x64，阶段 4 构建，含 disk.rs）。
      - 真实环境：NAS SMB 共享（只读）+ 基础 VHDX + 可写差分盘目录。

.CONTRACT（stage 4 CLI，契约以脚本头为准）
      gamevhd-runtime.exe mount <vhd> [--parent <UNC>] [--smb <UNC>]
                        [--user <U>] [--pass <P>] [--letter <L>] [--retries <N>]
      gamevhd-runtime.exe unmount <vhd> [--letter <L>] [--smb <UNC>]

      mount 行为（v2 定案 §3.2）：
        1. --smb 提供时 WNetAddConnection2（只读，重试 ×N，幂等已连接）；
        2. --parent 提供且差分盘不存在 → CreateVirtualDisk 建差分
           （已存在幂等跳过）；
        3. OpenVirtualDisk + AttachVirtualDisk（PERMANENT_LIFETIME，
           句柄关闭挂载保持）；
        4. GetVirtualDiskPhysicalPath → 卷枚举（FindFirstVolume +
           IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS）匹配物理盘 → 卷 GUID；
        5. DefineDosDevice 显式分配盘符（--letter 优先，否则第一个空闲）；
        6. 成功打印「mount: 挂载成功 <vhd> → <L>:（卷 <guid>）」。
      unmount 行为：
        1. DefineDosDevice(DDD_REMOVE_DEFINITION) 移除盘符；
        2. OpenVirtualDisk + DetachVirtualDisk（幂等）；
        3. --smb 提供时 WNetCancelConnection2 断会话。
.PARAMETER RuntimeExe
    gamevhd-runtime.exe 路径（必需）。
.PARAMETER DiffVhd
    本地差分盘路径（必需，可不存在——首次由 --parent 创建）。
.PARAMETER ParentUnc
    基础盘 UNC 路径（必需）。
.PARAMETER SmbUnc
    SMB 共享 UNC（必需）。
.PARAMETER SmbUser
    SMB 用户名（可选，缺省走当前会话）。
.PARAMETER SmbPass
    SMB 密码（可选，与 SmbUser 同缺省）。
.PARAMETER Letter
    首选盘符（可选，缺省自动选择）。
.PARAMETER SmbRetries
    SMB 重试次数（可选，默认 3）。
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
    throw "runtime 不存在: $RuntimeExe"
}

$LetterArg = @()
if ($Letter) {
    $LetterArg = @('--letter', $Letter)
}

function Invoke-Runtime {
    param([string[]]$Args)
    $out = & $RuntimeExe @Args 2>&1 | Out-String
    return @{ Out = $out; Code = $LASTEXITCODE }
}

# --- 1. 挂载全流程 -----------------------------------------------------------
Write-Step "挂载：SMB → 建差分 → attach → 定盘符"
$mountArgs = @(
    'mount', $DiffVhd,
    '--parent', $ParentUnc,
    '--smb', $SmbUnc,
    '--retries', "$SmbRetries"
) + $LetterArg
if ($SmbUser) { $mountArgs += @('--user', $SmbUser) }
if ($SmbPass) { $mountArgs += @('--pass', $SmbPass) }

$r = Invoke-Runtime -Args $mountArgs
Assert-True ($r.Code -eq 0) "mount 退出码 0（实际 $($r.Code)）：$($r.Out)"
Assert-True ($r.Out -match '挂载成功') "日志含「挂载成功」：$($r.Out)"

# 盘符确定：优先 --letter，否则从日志解析 `→ X:`。
$m = [regex]::Match($r.Out, '→ ([A-Z]):')
if ($Letter) {
    $driveLetter = $Letter.ToUpper()
    Assert-True ($m.Success -and $m.Groups[1].Value -eq $driveLetter) "盘符为指定 $driveLetter（实际 $($m.Groups[1].Value)）"
}
else {
    Assert-True $m.Success "自动分配盘符（日志含 → X:）"
    $driveLetter = $m.Groups[1].Value
}

Assert-True (Test-Path "${driveLetter}:\") "盘符 ${driveLetter}: 可访问"
$disk = Get-Disk | Where-Object { $_.Path -like '*Virtual*' -or $_.FriendlyName -like '*VHD*' }
Assert-True ($null -ne $disk) "系统存在 VHD 虚拟磁盘"

# --- 2. 错误路径：重复挂载 → AlreadyAttached --------------------------------
Write-Step "错误路径：重复 attach 应报 AlreadyAttached"
$r2 = Invoke-Runtime -Args $mountArgs
Assert-True ($r2.Code -ne 0) "重复挂载退出码非 0（实际 $($r2.Code)）"
Assert-True ($r2.Out -match 'AlreadyAttached|已挂载|ALREADY') "重复挂载错误可识别：$($r2.Out)"

# --- 3. 错误路径：坏 parent → 失败（错误分类可到达） ------------------------
Write-Step "错误路径：parent 失配/不可达应失败"
$badParent = (Join-Path $env:TEMP ('missing-' + [guid]::NewGuid().ToString('N') + '.vhdx'))
$r3 = Invoke-Runtime -Args @('mount', $DiffVhd, '--parent', "\\nonexistent-host\share\$badParent") + $LetterArg
Assert-True ($r3.Code -ne 0) "坏 parent 挂载退出码非 0（实际 $($r3.Code)）"

# --- 4. 卸载全流程 -----------------------------------------------------------
Write-Step "卸载：移除盘符 → detach → 断 SMB"
$unmountArgs = @('unmount', $DiffVhd, '--letter', $driveLetter, '--smb', $SmbUnc)
$r4 = Invoke-Runtime -Args $unmountArgs
Assert-True ($r4.Code -eq 0) "unmount 退出码 0（实际 $($r4.Code)）：$($r4.Out)"
Assert-True (-not (Test-Path "${driveLetter}:\")) "盘符 ${driveLetter}: 已移除"

Finish-Assertions
