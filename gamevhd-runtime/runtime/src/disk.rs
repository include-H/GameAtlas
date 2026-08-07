//! 磁盘层（阶段 4：替代 diskpart，v2 定案 §3.2）。
//!
//! 职责：SMB 只读连接（`WNetAddConnection2`，失败重试 ×3 覆盖 NAS 休眠唤醒）→
//! 差分盘创建（`CreateVirtualDisk`，parent = UNC 基础盘）→ `AttachVirtualDisk` /
//! `DetachVirtualDisk` → `GetVirtualDiskPhysicalPath` + 卷枚举
//! （`FindFirstVolume` + `IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS` 匹配物理盘）→
//! `DefineDosDevice` 显式分配盘符（配置指定字母优先，否则取第一个空闲）。
//! 全部结构化错误码：无临时脚本、无文本解析、无 PowerShell。
//!
//! 结构：跨平台纯逻辑（参数构造 / 盘符选择 / 错误分类 / 重试编排，Linux 可测）
//! 与 Windows 专属 API 调用（`#[cfg(target_os = "windows")]` 子模块）分离。
//! 基础类型在本模块自包含（winffi.rs 整体 `cfg(windows)`，Linux 不可引用）。
//!
//! 启动器仅以 x64 运行（v2 定案 §6：launcher 单二进制 x64）；结构体布局按
//! MSVC x64 ABI 定义并经 `offset_of!`/`size_of` 单测锁定；i686 目标仅编译验证。

// 库模块：完整 API 面供阶段 5 run/生命周期消费；当前 wave 仅 mount/unmount
// CLI 路径被生产调用，底层 API（attach/detach/卷枚举等）留待后续，按模块豁免。
#![allow(dead_code)]

use std::fmt;

/// Win32 基础类型（与 winffi.rs 一致；此处自包含以便 Linux 可测纯逻辑）。
pub type DWORD = u32;
pub type BOOL = i32;
pub type HANDLE = usize;

// ---- 结构化错误（跨平台，可分类断言） ----

/// 磁盘层错误类别。Windows API 调用错误携带原始 Win32/HRESULT 码。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DiskErrorKind {
    /// SMB 连接失败且重试耗尽（携带最后一次 Win32 错误码）。
    SmbRetryExhausted(u32),
    /// SMB 目标共享不存在（ERROR_BAD_NET_PATH / ERROR_BAD_NET_NAME）。
    SmbPathNotFound,
    /// SMB 登录/访问被拒（ERROR_ACCESS_DENIED / ERROR_LOGON_FAILURE）。
    SmbAccessDenied,
    /// `CreateVirtualDisk` 失败（携带错误码）。
    VhdCreateFailed(u32),
    /// `OpenVirtualDisk` 失败（携带错误码）。
    VhdOpenFailed(u32),
    /// `AttachVirtualDisk` 失败（携带错误码）。
    VhdAttachFailed(u32),
    /// `DetachVirtualDisk` 失败（携带错误码）。
    VhdDetachFailed(u32),
    /// parent 差分链失配（ERROR_VHD_CHILD_PARENT_ID/TIMESTAMP/SIZE_MISMATCH）。
    ParentMismatch,
    /// 差分盘 parent 不存在（ERROR_VHD_PARENT_VHD_NOT_FOUND）。
    ParentNotFound,
    /// 差分盘 parent 访问被拒（ERROR_VHD_PARENT_VHD_ACCESS_DENIED）。
    ParentAccessDenied,
    /// VHD 文件损坏（footer/sparse/bitmap 等 ERROR_VHD_* 系列）。
    VhdCorrupt(u32),
    /// 重复挂载：VHD 已被本机或他机 attach（ERROR_VIRTDISK_DISK_ALREADY_OWNED）。
    AlreadyAttached,
    /// 差分盘文件已存在（ERROR_ALREADY_EXISTS，幂等跳过创建）。
    DiffAlreadyExists,
    /// 盘符分配失败或系统无空闲盘符。
    NoFreeDriveLetter,
    /// 卷枚举后未匹配到目标物理盘的卷。
    VolumeNotFound,
    /// `GetVirtualDiskPhysicalPath` 输出无法解析出盘号。
    BadPhysicalPath(String),
    /// 其他未知错误（携带错误码）。
    Other(u32),
}

/// 磁盘层操作错误（类别 + 可读消息）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DiskError {
    pub kind: DiskErrorKind,
    pub message: String,
}

impl DiskError {
    pub fn new(kind: DiskErrorKind, message: impl Into<String>) -> DiskError {
        DiskError {
            kind,
            message: message.into(),
        }
    }
}

impl fmt::Display for DiskError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{} ({:?})", self.message, self.kind)
    }
}

impl std::error::Error for DiskError {}

/// 按 Win32 错误码分类（跨平台纯函数，Linux 可测）。
/// 覆盖 virtdisk（0xC03Axxxx）与通用 Win32 码。
pub fn classify_win32_error(code: u32) -> DiskErrorKind {
    match code {
        // SMB / 网络（WNetAddConnection2 返回码）。
        53 | 67 => DiskErrorKind::SmbPathNotFound, // ERROR_BAD_NET_PATH / ERROR_BAD_NET_NAME
        5 | 1326 | 1219 => DiskErrorKind::SmbAccessDenied, // ACCESS_DENIED / LOGON_FAILURE / SESSION_CREDENTIAL_CONFLICT
        // virtdisk：parent 差分链。
        0xC03A_000E | 0xC03A_000F | 0xC03A_0017 => DiskErrorKind::ParentMismatch, // ID/TIMESTAMP/SIZE_MISMATCH
        0xC03A_000D => DiskErrorKind::ParentNotFound,
        0xC03A_0016 => DiskErrorKind::ParentAccessDenied,
        // virtdisk：损坏 / 状态。
        0xC03A_0001 | 0xC03A_0002 | 0xC03A_0003 | 0xC03A_0006 | 0xC03A_0008
        | 0xC03A_000A | 0xC03A_000C => DiskErrorKind::VhdCorrupt(code),
        0xC03A_001E => DiskErrorKind::AlreadyAttached, // ERROR_VIRTDISK_DISK_ALREADY_OWNED
        // 通用。
        183 => DiskErrorKind::DiffAlreadyExists, // ERROR_ALREADY_EXISTS
        85 | 87 => DiskErrorKind::Other(code),   // ERROR_ALREADY_ASSIGNED / DEVICE_ALREADY_REMEMBERED（SMB 幂等已连接）
        _ => DiskErrorKind::Other(code),
    }
}

/// 判断错误是否为"差分盘已存在"（幂等跳过创建的依据）。
pub fn is_diff_already_exists(code: u32) -> bool {
    classify_win32_error(code) == DiskErrorKind::DiffAlreadyExists
}

/// 判断错误是否为"重复挂载"（attach 幂等 / 自愈复用的依据）。
pub fn is_already_attached(code: u32) -> bool {
    classify_win32_error(code) == DiskErrorKind::AlreadyAttached
}

/// 判断错误是否为 parent 失配族（删除重建决策的依据，v2 定案 §3.8）。
pub fn is_parent_mismatch(code: u32) -> bool {
    matches!(
        classify_win32_error(code),
        DiskErrorKind::ParentMismatch | DiskErrorKind::ParentNotFound
    )
}

// ---- 跨平台参数构造 ----

/// `NETRESOURCEW` 字段规格（构造结果可跨平台断言，实际 FFI 填充见 imp）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NetResourceSpec {
    pub scope: u32,
    pub type_: u32,
    pub display_type: u32,
    pub usage: u32,
    /// 本地盘符（None = 不映射本地盘符，仅建立会话）。
    pub local_name: Option<String>,
    pub remote_name: String,
}

/// 构造只读 SMB 连接的 `NETRESOURCEW` 规格。
/// scope=RESOURCE_CONNECTED(1)、type=RESOURCETYPE_DISK(1)、
/// display=RESOURCEDISPLAYTYPE_SHARE(3)、usage=RESOURCEUSAGE_CONNECTABLE(1)、
/// local=None（不占盘符，差分盘走 virtdisk 层）。
pub fn build_net_resource(remote_unc: &str) -> NetResourceSpec {
    NetResourceSpec {
        scope: 1,
        type_: 1,
        display_type: 3,
        usage: 1,
        local_name: None,
        remote_name: remote_unc.to_string(),
    }
}

/// `CREATE_VIRTUAL_DISK_PARAMETERS` Version 2 的字段规格（差分盘）。
/// 差分盘：大小/块/扇区继承 parent，仅 parent 路径与 parent 类型有效。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CreateDiffParamsSpec {
    pub version: u32,
    /// parent（基础盘）UNC 路径，如 `\\192.168.1.4\Game\base.vhdx`。
    pub parent_path: String,
    /// parent 虚拟磁盘类型（VHDX=3 / VHD=2）。
    pub parent_device_id: u32,
    /// parent 厂商 GUID 前两字段（Microsoft vendor，其余固定 8 字节）。
    pub parent_vendor: [u32; 2],
}

/// 构造差分盘创建参数（Version 2，VHDX，Microsoft 厂商）。
pub fn build_create_diff_params(parent_unc: &str) -> CreateDiffParamsSpec {
    // VIRTUAL_STORAGE_TYPE_VENDOR_MICROSOFT = {EC984AEC-A0F9-47e9-901F-71415A66345B}
    CreateDiffParamsSpec {
        version: 2, // CREATE_VIRTUAL_DISK_VERSION_2
        parent_path: parent_unc.to_string(),
        parent_device_id: 3, // VIRTUAL_STORAGE_TYPE_DEVICE_VHDX
        parent_vendor: [0xEC98_4AEC, 0xA0F9_47E9],
    }
}

/// 盘符选择（跨平台纯函数）：
/// 优先 `preferred`（若未被占用且合法 A-Z）；否则取第一个空闲（跳过 A/B，
/// 从 D 起——C 通常是系统盘）。返回 `None` 表示无空闲盘符。
pub fn pick_drive_letter(used: &[char], preferred: Option<char>) -> Option<char> {
    let is_used = |c: char| used.iter().any(|&u| u.eq_ignore_ascii_case(&c));
    if let Some(p) = preferred {
        let p = p.to_ascii_uppercase();
        if p.is_ascii_alphabetic() && !is_used(p) {
            return Some(p);
        }
    }
    ('D'..='Z').find(|c| !is_used(*c))
}

/// 解析 `GetVirtualDiskPhysicalPath` 输出（`\\?\PhysicalDrive2`）→ 物理盘号。
pub fn parse_physical_drive(phys_path: &str) -> Option<u32> {
    let trimmed = phys_path.trim_end_matches('\\');
    let idx = trimmed.rfind("Drive")?;
    trimmed[idx + "Drive".len()..].trim().parse::<u32>().ok()
}

/// SMB 连接重试编排（跨平台纯逻辑，注入单次连接尝试）。
/// `attempt` 返回 Ok=成功 / Err(Win32 码)=失败；除 ERROR_ALREADY_ASSIGNED /
/// DEVICE_ALREADY_REMEMBERED（幂等已连接，视为成功）外均重试 `attempts` 次。
pub fn smb_connect_with_retry<F>(attempts: u32, mut attempt: F) -> Result<(), DiskError>
where
    F: FnMut() -> Result<(), u32>,
{
    if attempts == 0 {
        return Err(DiskError::new(
            DiskErrorKind::Other(0),
            "SMB 重试次数必须 ≥ 1",
        ));
    }
    let mut last_code = 0u32;
    for i in 0..attempts {
        match attempt() {
            Ok(()) => return Ok(()),
            Err(code) => {
                if code == 85 || code == 87 {
                    // 已连接（幂等场景：残留会话复用），视为成功。
                    return Ok(());
                }
                last_code = code;
                crate::log_warn!(
                    "SMB 连接第 {}/{} 次失败（Win32 错误码 {}），{}",
                    i + 1,
                    attempts,
                    code,
                    if i + 1 < attempts { "重试中" } else { "放弃" }
                );
            }
        }
    }
    Err(DiskError::new(
        classify_win32_error(last_code),
        format!("SMB 连接重试 {attempts} 次后仍失败"),
    ))
}

// ---- Windows 专属实现（FFI + 完整挂载/卸载流程） ----

#[cfg(target_os = "windows")]
mod imp {
    use super::*;
    use crate::winffi::{CloseHandle, GetLastError, INVALID_HANDLE_VALUE};

    // ---- 常量（与 mingw 头文件 / MS Learn 对齐） ----

    /// SMB：临时连接（不写入注册表持久化）。
    pub const CONNECT_TEMPORARY: DWORD = 0x0000_0004;
    /// `WNetCancelConnection2`：强制断开。
    pub const CONNECT_UPDATE_PROFILE: DWORD = 0x0000_0001;

    /// virtdisk：创建标志无特殊位（动态差分盘）。
    pub const CREATE_VIRTUAL_DISK_FLAG_NONE: u32 = 0;
    /// virtdisk：打开标志无特殊位。
    pub const OPEN_VIRTUAL_DISK_FLAG_NONE: u32 = 0;
    /// virtdisk：PERMANENT_LIFETIME（0x4）——句柄关闭后 VHD 仍保持挂载，
    /// 直到显式 `DetachVirtualDisk`。启动器 mount 后立即关句柄，游戏运行期间
    /// 挂载必须存活，故 attach 必须带此标志（否则句柄关闭即自动分离）。
    pub const ATTACH_VIRTUAL_DISK_FLAG_PERMANENT_LIFETIME: u32 = 0x0000_0004;
    /// virtdisk：detach 无特殊位。
    pub const DETACH_VIRTUAL_DISK_FLAG_NONE: u32 = 0;
    /// virtdisk：`VIRTUAL_DISK_ACCESS_NONE`（创建/打开时）。
    pub const VIRTUAL_DISK_ACCESS_NONE: u32 = 0;
    /// virtdisk：`VIRTUAL_DISK_ACCESS_ALL`。
    pub const VIRTUAL_DISK_ACCESS_ALL: u32 = 0x003F_0000;

    /// `IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS`（winioctl.h，METHOD_BUFFERED）。
    pub const IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS: DWORD = 0x0056_0000;

    /// `DefineDosDevice`：移除盘符定义。
    pub const DDD_REMOVE_DEFINITION: DWORD = 0x0000_0002;

    /// `CreateFileW`：打开卷句柄（卷名如 `\\?\Volume{GUID}\`）。
    pub const GENERIC_READ: DWORD = 0x8000_0000;
    pub const FILE_SHARE_READ: DWORD = 0x0000_0001;
    pub const FILE_SHARE_WRITE: DWORD = 0x0000_0002;
    pub const OPEN_EXISTING: DWORD = 3;
    pub const FILE_ATTRIBUTE_NORMAL: DWORD = 0x0000_0080;

    /// `FindFirstVolumeW` 缓冲长度（盘符/卷名上限，含 NUL）。
    pub const VOLUME_NAME_MAX: DWORD = 256;

    /// `CREATE_VIRTUAL_DISK_PARAMETERS` Version 2 的 `_pad` 字段
    /// （union 对齐到 8，ULONGLONG MaximumSize 所致；见布局单测）。
    #[allow(dead_code)]
    fn _union_align_pad_doc() {}

    // ---- 结构体（repr(C)，MSVC x64 布局；启动器仅 x64 运行） ----

    #[repr(C)]
    pub struct Guid {
        pub data1: u32,
        pub data2: u16,
        pub data3: u16,
        pub data4: [u8; 8],
    }

    impl Guid {
        pub const fn new(data1: u32, data2: u16, data3: u16, data4: [u8; 8]) -> Guid {
            Guid { data1, data2, data3, data4 }
        }
        pub const fn zero() -> Guid {
            Guid { data1: 0, data2: 0, data3: 0, data4: [0; 8] }
        }
    }

    /// `VIRTUAL_STORAGE_TYPE`（20B：DeviceId u32 + VendorId Guid）。
    #[repr(C)]
    pub struct VirtualStorageType {
        pub device_id: u32,
        pub vendor_id: Guid,
    }

    /// Microsoft VHDX 厂商 GUID。
    pub const VENDOR_MICROSOFT: Guid =
        Guid::new(0xEC98_4AEC, 0xA0F9, 0x47E9, [0x90, 0x1F, 0x71, 0x41, 0x5A, 0x66, 0x34, 0x5B]);

    /// `CREATE_VIRTUAL_DISK_PARAMETERS`，仅展开 Version 2 分支（VHDX）。
    /// 布局：Version(4) + pad(4) [union 对齐 8] + Version2 字段序列。
    /// x64 总大小 128B，各字段偏移见 `layout_*` 单测。
    #[repr(C)]
    #[allow(dead_code)]
    pub struct CreateVirtualDiskParameters {
        pub version: u32,
        _pad: u32,
        pub unique_id: Guid, // 8
        pub maximum_size: u64, // 24（差分盘继承 parent，填 0）
        pub block_size: u32, // 32
        pub sector_size: u32, // 36
        pub physical_sector_size: u32, // 40
        _pad2: u32,          // 44（指针对齐）
        pub parent_path: *const u16, // 48
        pub source_path: *const u16, // 56
        pub open_flags: u32, // 64
        pub parent_vst: VirtualStorageType, // 68
        pub source_vst: VirtualStorageType, // 88
        pub resiliency_guid: Guid, // 108
    } // size 124 → align 8 → 128

    /// `ATTACH_VIRTUAL_DISK_PARAMETERS`（Version 1：Reserved）。
    #[repr(C)]
    #[allow(dead_code)]
    pub struct AttachVirtualDiskParameters {
        pub version: u32,
        pub reserved: u32,
    }

    /// `DISK_EXTENT`（20B）。
    #[repr(C)]
    pub struct DiskExtent {
        pub disk_number: DWORD,
        pub starting_offset: i64,
        pub extent_length: i64,
    }

    /// `VOLUME_DISK_EXTENTS`（24B：计数 + 首个 extent）。
    #[repr(C)]
    pub struct VolumeDiskExtents {
        pub number_of_disk_extents: DWORD,
        pub extents: [DiskExtent; 1],
    }

    /// `NETRESOURCEW`（x64 48B）。
    #[repr(C)]
    #[allow(dead_code)]
    pub struct NetResourceW {
        pub scope: DWORD,
        pub type_: DWORD,
        pub display_type: DWORD,
        pub usage: DWORD,
        pub local_name: *mut u16,
        pub remote_name: *mut u16,
        pub comment: *mut u16,
        pub provider: *mut u16,
    }

    // ---- FFI 声明 ----

    // mpr.dll：SMB 会话管理。
    #[link(name = "mpr")]
    extern "system" {
        pub fn WNetAddConnection2W(
            lp_net_resource: *const NetResourceW,
            lp_password: *const u16,
            lp_user_name: *const u16,
            dw_flags: DWORD,
        ) -> DWORD;
        pub fn WNetCancelConnection2W(
            lp_name: *const u16,
            dw_flags: DWORD,
            f_force: BOOL,
        ) -> DWORD;
    }

    // virtdisk.dll：虚拟磁盘生命周期。
    #[link(name = "virtdisk")]
    extern "system" {
        pub fn CreateVirtualDisk(
            virtual_storage_type: *const VirtualStorageType,
            path: *const u16,
            virtual_disk_access_mask: u32,
            security_descriptor: *const u8,
            flags: u32,
            provider_specific_flags: u32,
            parameters: *const CreateVirtualDiskParameters,
            overlapped: *const u8,
            handle: *mut HANDLE,
        ) -> DWORD;
        pub fn OpenVirtualDisk(
            virtual_storage_type: *const VirtualStorageType,
            path: *const u16,
            virtual_disk_access_mask: u32,
            flags: u32,
            parameters: *const u8,
            handle: *mut HANDLE,
        ) -> DWORD;
        pub fn AttachVirtualDisk(
            virtual_disk_handle: HANDLE,
            security_descriptor: *const u8,
            flags: u32,
            provider_specific_flags: u32,
            parameters: *const u8,
            overlapped: *const u8,
        ) -> DWORD;
        pub fn DetachVirtualDisk(
            virtual_disk_handle: HANDLE,
            flags: u32,
            provider_specific_flags: u32,
        ) -> DWORD;
        pub fn GetVirtualDiskPhysicalPath(
            virtual_disk_handle: HANDLE,
            disk_path_size_in_bytes: *mut DWORD,
            disk_path: *mut u16,
        ) -> DWORD;
    }

    // kernel32：卷枚举 + 盘符分配 + 通用句柄。
    extern "system" {
        pub fn FindFirstVolumeW(
            lpsz_volume_name: *mut u16,
            cch_buffer_length: DWORD,
        ) -> HANDLE;
        pub fn FindNextVolumeW(
            h_find_volume: HANDLE,
            lpsz_volume_name: *mut u16,
            cch_buffer_length: DWORD,
        ) -> BOOL;
        pub fn FindVolumeClose(h_find_volume: HANDLE) -> BOOL;
        pub fn CreateFileW(
            lp_file_name: *const u16,
            dw_desired_access: DWORD,
            dw_share_mode: DWORD,
            lp_security_attributes: *const u8,
            dw_creation_disposition: DWORD,
            dw_flags_and_attributes: DWORD,
            h_template_file: HANDLE,
        ) -> HANDLE;
        pub fn DeviceIoControl(
            h_device: HANDLE,
            dw_io_control_code: DWORD,
            lp_in_buffer: *const u8,
            n_in_buffer_size: DWORD,
            lp_out_buffer: *mut u8,
            n_out_buffer_size: DWORD,
            lp_bytes_returned: *mut DWORD,
            lp_overlapped: *const u8,
        ) -> BOOL;
        pub fn DefineDosDeviceW(
            dw_flags: DWORD,
            lp_device_name: *const u16,
            lp_target_path: *const u16,
        ) -> BOOL;
        /// 位图：bit i 置位表示盘符 (A + i) 已占用。
        pub fn GetLogicalDrives() -> DWORD;
    }

    // ---- UTF-16 工具 ----

    /// &str → NUL 结尾 UTF-16（追加 NUL，供 *const u16 参数）。
    pub fn to_wide(s: &str) -> Vec<u16> {
        s.encode_utf16().chain(std::iter::once(0)).collect()
    }

    /// NUL 结尾 UTF-16 → String（丢弃尾 NUL）。
    pub unsafe fn from_wide(ptr: *const u16) -> String {
        if ptr.is_null() {
            return String::new();
        }
        let mut v = Vec::new();
        let mut p = ptr;
        while *p != 0 {
            v.push(*p);
            p = p.add(1);
        }
        String::from_utf16_lossy(&v)
    }

    // ---- 挂载/卸载参数 ----

    /// 完整挂载参数（SMB 可选：None = 跳过 SMB，要求 diff 已可访问）。
    #[derive(Debug, Clone)]
    pub struct MountParams {
        /// 本地差分盘路径（`%LOCALAPPDATA%\GameAtlas\diff\<name>.vhdx`）。
        pub diff_path: String,
        /// parent（基础盘）UNC 路径；Some 时创建差分盘（已存在则幂等跳过）。
        pub parent_unc: Option<String>,
        /// SMB 共享 UNC（`\\server\share`）；None = 跳过连接。
        pub smb_remote: Option<String>,
        pub smb_user: Option<String>,
        pub smb_pass: Option<String>,
        /// 首选盘符；None = 取第一个空闲。
        pub preferred_letter: Option<char>,
        /// SMB 连接重试次数（覆盖 NAS 休眠唤醒）。
        pub smb_retries: u32,
    }

    /// 卸载参数。
    #[derive(Debug, Clone)]
    pub struct UnmountParams {
        /// 已分配的盘符（移除 DosDevice 定义用）。
        pub drive_letter: char,
        /// 差分盘路径（detach 用，经 OpenVirtualDisk 打开）。
        pub diff_path: String,
        /// SMB 共享 UNC（断开用）；None = 跳过。
        pub smb_remote: Option<String>,
    }

    /// 挂载结果。
    #[derive(Debug, Clone)]
    pub struct MountResult {
        pub drive_letter: char,
        /// 卷 GUID 路径（`\\?\Volume{...}\`）。
        pub volume_guid: String,
        /// 物理盘路径（`\\?\PhysicalDriveN`）。
        pub physical_path: String,
    }

    // ---- 完整流程 ----

    /// 连接 SMB 只读共享（重试 ×N；已连接幂等视为成功）。
    pub fn smb_connect(remote: &str, user: Option<&str>, pass: Option<&str>, retries: u32) -> Result<(), DiskError> {
        let spec = build_net_resource(remote);
        let remote_wide = to_wide(remote);
        let user_wide = user.map(to_wide);
        let pass_wide = pass.map(to_wide);

        smb_connect_with_retry(retries, || {
            let nr = NetResourceW {
                scope: spec.scope,
                type_: spec.type_,
                display_type: spec.display_type,
                usage: spec.usage,
                local_name: std::ptr::null_mut(),
                remote_name: remote_wide.as_ptr() as *mut u16,
                comment: std::ptr::null_mut(),
                provider: std::ptr::null_mut(),
            };
            let code = unsafe {
                WNetAddConnection2W(
                    &nr as *const NetResourceW,
                    pass_wide.as_ref().map_or(std::ptr::null(), |v| v.as_ptr()),
                    user_wide.as_ref().map_or(std::ptr::null(), |v| v.as_ptr()),
                    CONNECT_TEMPORARY,
                )
            };
            if code == 0 {
                Ok(())
            } else {
                Err(code)
            }
        })
    }

    /// 断开 SMB 共享（幂等：未连接返回成功语义由调用方容忍）。
    pub fn smb_disconnect(remote: &str) -> Result<(), DiskError> {
        let remote_wide = to_wide(remote);
        let code = unsafe { WNetCancelConnection2W(remote_wide.as_ptr(), CONNECT_UPDATE_PROFILE, 1) };
        // ERROR_NOT_CONNECTED(2250) / ERROR_BAD_NET_NAME(67)：视为已断开。
        if code == 0 || code == 2250 || code == 67 {
            Ok(())
        } else {
            Err(DiskError::new(
                classify_win32_error(code),
                format!("WNetCancelConnection2W({remote}) 失败: {code}"),
            ))
        }
    }

    /// 创建差分盘（parent = UNC）。diff 已存在 → Err(DiffAlreadyExists) 由调用方幂等跳过。
    pub fn create_diff_vhd(diff_path: &str, parent_unc: &str) -> Result<(), DiskError> {
        let spec = build_create_diff_params(parent_unc);
        let vst = VirtualStorageType {
            device_id: spec.parent_device_id,
            vendor_id: VENDOR_MICROSOFT,
        };
        let diff_wide = to_wide(diff_path);
        let parent_wide = to_wide(parent_unc);
        let params = CreateVirtualDiskParameters {
            version: spec.version,
            _pad: 0,
            unique_id: Guid::zero(),
            maximum_size: 0, // 差分盘继承 parent
            block_size: 0,
            sector_size: 0,
            physical_sector_size: 0,
            _pad2: 0,
            parent_path: parent_wide.as_ptr(),
            source_path: std::ptr::null(),
            open_flags: OPEN_VIRTUAL_DISK_FLAG_NONE,
            parent_vst: VirtualStorageType {
                device_id: spec.parent_device_id,
                vendor_id: VENDOR_MICROSOFT,
            },
            source_vst: VirtualStorageType {
                device_id: 0,
                vendor_id: Guid::zero(),
            },
            resiliency_guid: Guid::zero(),
        };
        let mut handle: HANDLE = 0;
        let code = unsafe {
            CreateVirtualDisk(
                &vst as *const VirtualStorageType,
                diff_wide.as_ptr(),
                VIRTUAL_DISK_ACCESS_NONE,
                std::ptr::null(),
                CREATE_VIRTUAL_DISK_FLAG_NONE,
                0,
                &params as *const CreateVirtualDiskParameters,
                std::ptr::null(),
                &mut handle,
            )
        };
        if code != 0 {
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("CreateVirtualDisk({diff_path}, parent={parent_unc}) 失败: {code:#x}"),
            ));
        }
        // 创建成功后句柄已打开；关闭（后续 attach 走 OpenVirtualDisk 单独打开）。
        unsafe { CloseHandle(handle) };
        Ok(())
    }

    /// 打开差分盘句柄（attach/detach 前置）。
    pub fn open_vhd(path: &str) -> Result<HANDLE, DiskError> {
        let vst = VirtualStorageType {
            device_id: 3, // VHDX
            vendor_id: VENDOR_MICROSOFT,
        };
        let path_wide = to_wide(path);
        let mut handle: HANDLE = 0;
        let code = unsafe {
            OpenVirtualDisk(
                &vst as *const VirtualStorageType,
                path_wide.as_ptr(),
                VIRTUAL_DISK_ACCESS_ALL,
                OPEN_VIRTUAL_DISK_FLAG_NONE,
                std::ptr::null(),
                &mut handle,
            )
        };
        if code != 0 {
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("OpenVirtualDisk({path}) 失败: {code:#x}"),
            ));
        }
        Ok(handle)
    }

    /// attach 差分盘。重复挂载（0xC03A001E）报 AlreadyAttached 供调用方决策。
    /// 必须带 PERMANENT_LIFETIME：句柄关闭后挂载保持（游戏运行期依赖），
    /// 卸载走显式 `DetachVirtualDisk`。
    pub fn attach_vhd(handle: HANDLE) -> Result<(), DiskError> {
        let code = unsafe {
            AttachVirtualDisk(
                handle,
                std::ptr::null(),
                ATTACH_VIRTUAL_DISK_FLAG_PERMANENT_LIFETIME,
                0,
                std::ptr::null(),
                std::ptr::null(),
            )
        };
        if code != 0 {
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("AttachVirtualDisk 失败: {code:#x}"),
            ));
        }
        Ok(())
    }

    /// detach 差分盘（幂等：调用方容忍 ERROR_VHD_NOT_ATTACHED / INVALID_STATE）。
    pub fn detach_vhd(handle: HANDLE) -> Result<(), DiskError> {
        let code = unsafe { DetachVirtualDisk(handle, DETACH_VIRTUAL_DISK_FLAG_NONE, 0) };
        if code != 0 && code != 0xC03A_001C {
            // ERROR_VHD_INVALID_STATE(0xC03A001C)：未挂载或已分离，视为幂等成功。
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("DetachVirtualDisk 失败: {code:#x}"),
            ));
        }
        Ok(())
    }

    /// 查询 attach 后 VHD 的物理盘路径（`\\?\PhysicalDriveN`）。
    pub fn physical_path(handle: HANDLE) -> Result<String, DiskError> {
        let mut size: DWORD = 0;
        let code = unsafe { GetVirtualDiskPhysicalPath(handle, &mut size, std::ptr::null_mut()) };
        if code != 0 || size == 0 {
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("GetVirtualDiskPhysicalPath(探测大小) 失败: {code:#x}"),
            ));
        }
        let mut buf = vec![0u16; (size / 2) as usize];
        let code = unsafe { GetVirtualDiskPhysicalPath(handle, &mut size, buf.as_mut_ptr()) };
        if code != 0 {
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("GetVirtualDiskPhysicalPath(读取) 失败: {code:#x}"),
            ));
        }
        let s = unsafe { from_wide(buf.as_ptr()) };
        if s.trim().is_empty() {
            return Err(DiskError::new(
                DiskErrorKind::BadPhysicalPath(String::new()),
                "GetVirtualDiskPhysicalPath 返回空路径",
            ));
        }
        Ok(s)
    }

    /// 枚举所有卷，匹配物理盘号 → 返回卷 GUID 路径（`\\?\Volume{...}\`）。
    pub fn find_volume_for_disk(disk_number: u32) -> Result<String, DiskError> {
        let mut buf = vec![0u16; VOLUME_NAME_MAX as usize];
        let handle = unsafe { FindFirstVolumeW(buf.as_mut_ptr(), VOLUME_NAME_MAX) };
        if handle == INVALID_HANDLE_VALUE {
            let code = unsafe { GetLastError() };
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("FindFirstVolumeW 失败: {code:#x}"),
            ));
        }
        let mut found: Option<String> = None;
        loop {
            let name = unsafe { from_wide(buf.as_ptr()) };
            if volume_has_disk(&name, disk_number) {
                found = Some(name);
                break;
            }
            let ok = unsafe { FindNextVolumeW(handle, buf.as_mut_ptr(), VOLUME_NAME_MAX) };
            if ok == 0 {
                break;
            }
        }
        unsafe { FindVolumeClose(handle) };
        found.ok_or_else(|| {
            DiskError::new(
                DiskErrorKind::VolumeNotFound,
                format!("未找到位于物理盘 {disk_number} 的卷"),
            )
        })
    }

    /// 打开卷句柄，IOCTL 查询其磁盘 extent，判断是否落在目标物理盘上。
    fn volume_has_disk(volume_guid: &str, disk_number: u32) -> bool {
        let volume_wide = to_wide(volume_guid);
        let h = unsafe {
            CreateFileW(
                volume_wide.as_ptr(),
                GENERIC_READ,
                FILE_SHARE_READ | FILE_SHARE_WRITE,
                std::ptr::null(),
                OPEN_EXISTING,
                FILE_ATTRIBUTE_NORMAL,
                0,
            )
        };
        if h == INVALID_HANDLE_VALUE {
            return false;
        }
        let mut extents = VolumeDiskExtents {
            number_of_disk_extents: 0,
            extents: [DiskExtent {
                disk_number: 0,
                starting_offset: 0,
                extent_length: 0,
            }],
        };
        let mut returned: DWORD = 0;
        let ok = unsafe {
            DeviceIoControl(
                h,
                IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS,
                std::ptr::null(),
                0,
                &mut extents as *mut VolumeDiskExtents as *mut u8,
                std::mem::size_of::<VolumeDiskExtents>() as DWORD,
                &mut returned,
                std::ptr::null(),
            )
        };
        unsafe { CloseHandle(h) };
        if ok == 0 {
            return false;
        }
        (0..extents.number_of_disk_extents)
            .any(|i| extents.extents[i as usize].disk_number == disk_number)
    }

    /// 用 `DefineDosDevice` 显式分配盘符（`E:` → `\\?\Volume{GUID}\`）。
    pub fn assign_drive_letter(volume_guid: &str, letter: char) -> Result<(), DiskError> {
        let device = format!("{}:", letter.to_ascii_uppercase());
        let target = to_wide(volume_guid);
        let device_wide = to_wide(&device);
        let ok = unsafe { DefineDosDeviceW(0, device_wide.as_ptr(), target.as_ptr()) };
        if ok == 0 {
            let code = unsafe { GetLastError() };
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("DefineDosDeviceW({device} → {volume_guid}) 失败: {code:#x}"),
            ));
        }
        Ok(())
    }

    /// 移除盘符定义（DDD_REMOVE_DEFINITION）。
    pub fn remove_drive_letter(letter: char) -> Result<(), DiskError> {
        let device = format!("{}:", letter.to_ascii_uppercase());
        let device_wide = to_wide(&device);
        let ok = unsafe {
            DefineDosDeviceW(DDD_REMOVE_DEFINITION, device_wide.as_ptr(), std::ptr::null())
        };
        if ok == 0 {
            let code = unsafe { GetLastError() };
            // ERROR_FILE_NOT_FOUND(2)：盘符本就不存在，幂等成功。
            if code != 2 {
                return Err(DiskError::new(
                    classify_win32_error(code),
                    format!("DefineDosDeviceW(移除 {device}) 失败: {code:#x}"),
                ));
            }
        }
        Ok(())
    }

    /// 查询系统已占用盘符（`GetLogicalDrives` bitmask → char 集合）。
    pub fn used_drive_letters() -> Vec<char> {
        let mask = unsafe { GetLogicalDrives() };
        let mut out = Vec::new();
        for i in 0..26 {
            if mask & (1u32 << i) != 0 {
                out.push((b'A' + i as u8) as char);
            }
        }
        out
    }

    /// 完整挂载流程：SMB → 建差分（幂等）→ attach → 物理路径 → 卷匹配 → 分配盘符。
    pub fn mount_vhd(params: &MountParams) -> Result<MountResult, DiskError> {
        // 1. SMB（可选）。
        if let Some(remote) = &params.smb_remote {
            smb_connect(
                remote,
                params.smb_user.as_deref(),
                params.smb_pass.as_deref(),
                params.smb_retries,
            )?;
        }

        // 2. 创建差分盘（parent 提供时；已存在幂等跳过）。
        if let Some(parent) = &params.parent_unc {
            match create_diff_vhd(&params.diff_path, parent) {
                Ok(()) => {}
                Err(e) if e.kind == DiskErrorKind::DiffAlreadyExists => {
                    crate::log_info!("差分盘已存在，跳过创建: {}", params.diff_path);
                }
                Err(e) => return Err(e),
            }
        }

        // 3. 打开 + attach。
        let handle = open_vhd(&params.diff_path)?;
        if let Err(e) = attach_vhd(handle) {
            unsafe { CloseHandle(handle) };
            return Err(e);
        }

        // 4. 物理盘路径 → 盘号。
        let phys = physical_path(handle)?;
        unsafe { CloseHandle(handle) };
        let disk_number = parse_physical_drive(&phys).ok_or_else(|| {
            DiskError::new(
                DiskErrorKind::BadPhysicalPath(phys.clone()),
                format!("无法从物理路径解析盘号: {phys}"),
            )
        })?;

        // 5. 卷枚举匹配。
        let volume_guid = find_volume_for_disk(disk_number)?;

        // 6. 分配盘符（优先 preferred，否则第一个空闲）。
        let used = used_drive_letters();
        let letter = pick_drive_letter(&used, params.preferred_letter).ok_or_else(|| {
            DiskError::new(DiskErrorKind::NoFreeDriveLetter, "系统无空闲盘符")
        })?;
        assign_drive_letter(&volume_guid, letter)?;

        Ok(MountResult {
            drive_letter: letter,
            volume_guid,
            physical_path: phys,
        })
    }

    /// 完整卸载流程：移除盘符 → detach → 断 SMB。
    pub fn unmount_vhd(params: &UnmountParams) -> Result<(), DiskError> {
        let _ = remove_drive_letter(params.drive_letter);

        if let Ok(handle) = open_vhd(&params.diff_path) {
            let _ = detach_vhd(handle);
            unsafe { CloseHandle(handle) };
        }

        if let Some(remote) = &params.smb_remote {
            let _ = smb_disconnect(remote);
        }
        Ok(())
    }
}

// 库 API 面：阶段 5 run/生命周期将消费这些底层函数；当前 CLI 仅用
// mount_vhd/unmount_vhd，其余按公开库 API 豁免 unused（与 `#![allow(dead_code)]` 同理）。
#[cfg(target_os = "windows")]
#[allow(unused_imports)]
pub use imp::{
    assign_drive_letter, attach_vhd, create_diff_vhd, detach_vhd, find_volume_for_disk,
    mount_vhd, open_vhd, physical_path, remove_drive_letter, smb_connect, smb_disconnect,
    unmount_vhd, used_drive_letters, MountParams, MountResult, UnmountParams,
};

// ---- 单测（跨平台纯逻辑；Windows 专属布局断言双 target 编译时同样生效） ----

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn classify_parent_mismatch_codes() {
        // ERROR_VHD_CHILD_PARENT_ID/TIMESTAMP/SIZE_MISMATCH。
        for code in [0xC03A_000Eu32, 0xC03A_000F, 0xC03A_0017] {
            assert_eq!(classify_win32_error(code), DiskErrorKind::ParentMismatch);
            assert!(is_parent_mismatch(code), "{code:#x}");
        }
        // PARENT_VHD_NOT_FOUND 也归为 parent 失配族。
        assert!(is_parent_mismatch(0xC03A_000D));
        assert!(!is_parent_mismatch(0xC03A_001E));
    }

    #[test]
    fn classify_already_attached() {
        // ERROR_VIRTDISK_DISK_ALREADY_OWNED。
        assert_eq!(
            classify_win32_error(0xC03A_001E),
            DiskErrorKind::AlreadyAttached
        );
        assert!(is_already_attached(0xC03A_001E));
        assert!(!is_already_attached(0xC03A_000D));
    }

    #[test]
    fn classify_diff_already_exists_and_smb() {
        assert_eq!(
            classify_win32_error(183),
            DiskErrorKind::DiffAlreadyExists
        );
        assert!(is_diff_already_exists(183));
        // SMB 网络路径错误。
        assert_eq!(classify_win32_error(53), DiskErrorKind::SmbPathNotFound);
        assert_eq!(classify_win32_error(67), DiskErrorKind::SmbPathNotFound);
        // SMB 访问拒绝。
        assert_eq!(classify_win32_error(5), DiskErrorKind::SmbAccessDenied);
        assert_eq!(classify_win32_error(1326), DiskErrorKind::SmbAccessDenied);
        assert_eq!(classify_win32_error(1219), DiskErrorKind::SmbAccessDenied);
        // 未知码 → Other。
        assert_eq!(classify_win32_error(0xDEAD_BEEF), DiskErrorKind::Other(0xDEAD_BEEF));
    }

    #[test]
    fn build_net_resource_spec() {
        let spec = build_net_resource(r"\\192.168.1.4\Game");
        assert_eq!(spec.scope, 1);
        assert_eq!(spec.type_, 1);
        assert_eq!(spec.display_type, 3);
        assert_eq!(spec.usage, 1);
        assert_eq!(spec.local_name, None);
        assert_eq!(spec.remote_name, r"\\192.168.1.4\Game");
    }

    #[test]
    fn build_create_diff_params_spec() {
        let spec = build_create_diff_params(r"\\192.168.1.4\Game\base.vhdx");
        assert_eq!(spec.version, 2);
        assert_eq!(spec.parent_path, r"\\192.168.1.4\Game\base.vhdx");
        assert_eq!(spec.parent_device_id, 3); // VHDX
        // Microsoft 厂商 GUID 前两字段。
        assert_eq!(spec.parent_vendor, [0xEC98_4AEC, 0xA0F9_47E9]);
    }

    #[test]
    fn pick_drive_letter_preferred_free() {
        assert_eq!(pick_drive_letter(&['C', 'D', 'E'], Some('G')), Some('G'));
        // 小写 preferred 归一为大写；'C' 空闲 → 直接采用。
        assert_eq!(pick_drive_letter(&['D', 'E'], Some('c')), Some('C'));
    }

    #[test]
    fn pick_drive_letter_preferred_taken_falls_back() {
        // D 是首个候选（跳过 A/B/C 语义由调用方 used 集合体现；此处 D 空闲）。
        assert_eq!(pick_drive_letter(&['C'], Some('C')), Some('D'));
        // 全占用 → 无空闲。
        let all: Vec<char> = ('A'..='Z').collect();
        assert_eq!(pick_drive_letter(&all, Some('Z')), None);
    }

    #[test]
    fn pick_drive_letter_invalid_preferred_ignored() {
        assert_eq!(pick_drive_letter(&['C'], Some('1')), Some('D'));
        assert_eq!(pick_drive_letter(&['C'], Some('\0')), Some('D'));
    }

    #[test]
    fn parse_physical_drive_variants() {
        assert_eq!(parse_physical_drive(r"\\?\PhysicalDrive2"), Some(2));
        assert_eq!(parse_physical_drive(r"\\?\PhysicalDrive10"), Some(10));
        assert_eq!(parse_physical_drive(r"\\?\PhysicalDrive0"), Some(0));
        assert_eq!(parse_physical_drive(r"\\?\PhysicalDrive2\"), Some(2));
        assert_eq!(parse_physical_drive("garbage"), None);
        assert_eq!(parse_physical_drive(r"\\?\PhysicalDrive"), None);
        assert_eq!(parse_physical_drive(""), None);
    }

    #[test]
    fn smb_connect_with_retry_succeeds() {
        // 前 2 次失败（NAS 休眠唤醒），第 3 次成功。
        let mut calls = 0;
        let r = smb_connect_with_retry(3, || {
            calls += 1;
            if calls < 3 {
                Err(53) // ERROR_BAD_NET_PATH
            } else {
                Ok(())
            }
        });
        assert_eq!(r, Ok(()));
        assert_eq!(calls, 3);
    }

    #[test]
    fn smb_connect_with_retry_exhausted() {
        let r = smb_connect_with_retry(3, || Err(53u32));
        assert!(r.is_err());
        assert_eq!(r.unwrap_err().kind, DiskErrorKind::SmbPathNotFound);
    }

    #[test]
    fn smb_connect_with_retry_already_connected_idempotent() {
        // ERROR_ALREADY_ASSIGNED(85)：残留会话复用，首次即视为成功。
        let mut calls = 0;
        let r = smb_connect_with_retry(3, || {
            calls += 1;
            Err(85)
        });
        assert_eq!(r, Ok(()));
        assert_eq!(calls, 1);
    }

    #[test]
    fn smb_connect_with_retry_zero_attempts_rejected() {
        let r = smb_connect_with_retry(0, || Ok(()));
        assert!(r.is_err());
        assert_eq!(r.unwrap_err().kind, DiskErrorKind::Other(0));
    }

    /// Windows 专属结构体布局锁定（x64 MSVC ABI；启动器仅 x64）。
    /// Linux 原生测试跑 x86_64，布局与 x64 Windows 一致。
    #[cfg(target_os = "windows")]
    #[test]
    fn layout_create_virtual_disk_parameters_x64() {
        use std::mem::{offset_of, size_of};
        let p = imp::CreateVirtualDiskParameters {
            version: 0,
            _pad: 0,
            unique_id: imp::Guid::zero(),
            maximum_size: 0,
            block_size: 0,
            sector_size: 0,
            physical_sector_size: 0,
            _pad2: 0,
            parent_path: std::ptr::null(),
            source_path: std::ptr::null(),
            open_flags: 0,
            parent_vst: imp::VirtualStorageType {
                device_id: 0,
                vendor_id: imp::Guid::zero(),
            },
            source_vst: imp::VirtualStorageType {
                device_id: 0,
                vendor_id: imp::Guid::zero(),
            },
            resiliency_guid: imp::Guid::zero(),
        };
        let _ = p;
        assert_eq!(size_of::<imp::CreateVirtualDiskParameters>(), 128);
        assert_eq!(offset_of!(imp::CreateVirtualDiskParameters, version), 0);
        assert_eq!(offset_of!(imp::CreateVirtualDiskParameters, unique_id), 8);
        assert_eq!(offset_of!(imp::CreateVirtualDiskParameters, maximum_size), 24);
        assert_eq!(offset_of!(imp::CreateVirtualDiskParameters, parent_path), 48);
        assert_eq!(offset_of!(imp::CreateVirtualDiskParameters, open_flags), 64);
        assert_eq!(offset_of!(imp::CreateVirtualDiskParameters, parent_vst), 68);
        assert_eq!(offset_of!(imp::CreateVirtualDiskParameters, resiliency_guid), 108);

        // NETRESOURCEW x64 = 48B；VIRTUAL_STORAGE_TYPE = 20B。
        assert_eq!(size_of::<imp::VirtualStorageType>(), 20);
        assert_eq!(size_of::<imp::Guid>(), 16);
    }
}
