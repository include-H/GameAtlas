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
    /// 当前平台不支持（非 Windows 桩）。
    UnsupportedPlatform,
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
        53 | 67 | 1203 => DiskErrorKind::SmbPathNotFound, // BAD_NET_PATH / BAD_NET_NAME / NO_NET_OR_BAD_PATH
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

/// VHD 文件格式（决定 CREATE_VIRTUAL_DISK_PARAMETERS 版本与 device_id）。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum VhdFormat {
    /// 传统 VHD：`VIRTUAL_STORAGE_TYPE_DEVICE_VHD`=2，参数 Version 1。
    Vhd,
    /// VHDX：`VIRTUAL_STORAGE_TYPE_DEVICE_VHDX`=3，参数 Version 2。
    Vhdx,
}

impl VhdFormat {
    /// 对应 `VIRTUAL_STORAGE_TYPE.DeviceId`。
    pub fn device_id(self) -> u32 {
        match self {
            VhdFormat::Vhd => 2,
            VhdFormat::Vhdx => 3,
        }
    }

    /// 按路径扩展名探测格式：`.vhd`（大小写不敏感）→ Vhd，其余 → Vhdx。
    pub fn detect(path: &str) -> VhdFormat {
        let lower = path.to_ascii_lowercase();
        if lower.ends_with(".vhd") && !lower.ends_with(".vhdx") {
            VhdFormat::Vhd
        } else {
            VhdFormat::Vhdx
        }
    }
}

/// `CREATE_VIRTUAL_DISK_PARAMETERS` 的字段规格（差分盘）。
/// 差分盘：大小/块/扇区继承 parent，仅 parent 路径与 parent 类型有效。
/// `version` 依格式：VHD=1 / VHDX=2（两者结构体布局不同，见 imp）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CreateDiffParamsSpec {
    pub format: VhdFormat,
    /// parent（基础盘）UNC 路径，如 `\\192.168.1.4\Game\base.vhd`。
    pub parent_path: String,
    /// parent 虚拟磁盘类型（VHDX=3 / VHD=2）。
    pub parent_device_id: u32,
    /// parent 厂商 GUID 前两字段（Microsoft vendor，其余固定 8 字节）。
    pub parent_vendor: [u32; 2],
}

/// 构造差分盘创建参数（按 parent 格式选版本，Microsoft 厂商）。
pub fn build_create_diff_params(parent_unc: &str) -> CreateDiffParamsSpec {
    // VIRTUAL_STORAGE_TYPE_VENDOR_MICROSOFT = {EC984AEC-A0F9-47e9-901F-71415A66345B}
    let format = VhdFormat::detect(parent_unc);
    CreateDiffParamsSpec {
        format,
        parent_path: parent_unc.to_string(),
        parent_device_id: format.device_id(),
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
    /// attach 时禁止系统自动分配盘符（盘符由启动器显式 DefineDosDevice）。
    pub const ATTACH_VIRTUAL_DISK_FLAG_NO_DRIVE_LETTER: u32 = 0x0000_0002;
    /// virtdisk：detach 无特殊位。
    pub const DETACH_VIRTUAL_DISK_FLAG_NONE: u32 = 0;
    /// virtdisk：`VIRTUAL_DISK_ACCESS_NONE`（V2/VHDX 创建时，文档强制）。
    pub const VIRTUAL_DISK_ACCESS_NONE: u32 = 0;
    /// virtdisk：`VIRTUAL_DISK_ACCESS_CREATE`（V1/VHD 创建时，官方示例用）。
    pub const VIRTUAL_DISK_ACCESS_CREATE: u32 = 0x0010_0000;
    /// virtdisk：`VIRTUAL_DISK_ACCESS_ALL`。
    pub const VIRTUAL_DISK_ACCESS_ALL: u32 = 0x003F_0000;

    /// `IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS`（winioctl.h，METHOD_BUFFERED）。
    pub const IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS: DWORD = 0x0056_0000;

    /// `CreateFileW`：打开卷句柄（卷名如 `\\?\Volume{GUID}\`，desired access 恒 0）。

    /// `CreateFileW`：打开卷句柄（卷名如 `\\?\Volume{GUID}\`，desired access 恒 0）。
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
    /// x64 总大小 128B；关键偏移见下方编译期断言（布局错 → 编译失败）。
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

    /// `CREATE_VIRTUAL_DISK_PARAMETERS`，仅展开 Version 1 分支（VHD）。
    /// 布局（真机 layout_probe 实测，mingw x64）：
    /// Version(4) + pad(4) [union 对齐 8] + Version1 字段序列：
    ///   unique_id@8, maximum_size@24, block_size@32, sector_size@36,
    ///   parent_path@40, source_path@48（SectorSize 结束于 40 恰为 8 倍数，
    ///   **无 padding**——V2 才有 pad 因 PhysicalSectorSize 结束于 44）。
    /// 注意：union 整体大小由最大成员 V2(128B) 决定；本结构独立 size = 56。
    #[repr(C)]
    #[allow(dead_code)]
    pub struct CreateVirtualDiskParametersV1 {
        pub version: u32,
        _pad: u32,
        pub unique_id: Guid, // 8
        pub maximum_size: u64, // 24（差分盘继承 parent，填 0）
        pub block_size: u32, // 32
        pub sector_size: u32, // 36
        pub parent_path: *const u16, // 40（无 pad！40 恰为 8 倍数）
        pub source_path: *const u16, // 48
    } // size 56（union 内成员；整体结构为 V2 大小 128）

    // 编译期布局锁定（x64 启动器；与真机 layout_probe 实测一致）。
    // 布局错即编译失败。真机教训：V1 曾多加 _pad2 致 parent_path@48 错位，
    // CreateVirtualDisk 读到错误指针 → 0x5 ACCESS_DENIED。
    #[cfg(target_pointer_width = "64")]
    const _: () = {
        assert!(std::mem::size_of::<CreateVirtualDiskParameters>() == 128);
        assert!(std::mem::size_of::<CreateVirtualDiskParametersV1>() == 56);
        assert!(std::mem::offset_of!(CreateVirtualDiskParameters, unique_id) == 8);
        assert!(std::mem::offset_of!(CreateVirtualDiskParameters, parent_path) == 48);
        assert!(std::mem::offset_of!(CreateVirtualDiskParametersV1, parent_path) == 40);
        assert!(std::mem::offset_of!(CreateVirtualDiskParametersV1, source_path) == 48);
    };

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
            // VHD→Version1 结构 / VHDX→Version2 结构，统一裸指针传入。
            parameters: *const u8,
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
        /// 给卷分配盘符/挂载点（`SetVolumeMountPointW("E:\\", "\\?\Volume{GUID}\\")`）。
        /// 给卷分配盘符的正确 API（DefineDosDevice 只定义进程命名空间设备名，
        /// 文件资源管理器不可见——v2 定案 §3.2 的实现偏差，真机实证）。
        pub fn SetVolumeMountPointW(
            lpsz_volume_mount_point: *const u16,
            lpsz_volume_name: *const u16,
        ) -> BOOL;
        /// 移除卷挂载点（`DeleteVolumeMountPointW("E:\\")`）。
        pub fn DeleteVolumeMountPointW(lpsz_volume_mount_point: *const u16) -> BOOL;
        /// 查询卷的所有挂载路径（盘符/挂载点），输出多字符串 `X:\0Y:\0\0`。
        pub fn GetVolumePathNamesForVolumeNameW(
            lpsz_volume_name: *const u16,
            lpsz_volume_path_names: *mut u16,
            cch_buffer_length: DWORD,
            lpcch_return_length: *mut DWORD,
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

    /// 读取 SMB 凭据（Windows 凭据管理器，target `GameVHD_<server>`）。
    /// 返回 (用户名, 密码)；未存储返回 None。
    pub fn read_smb_cred(server: &str) -> Option<(String, String)> {
        use crate::winffi::{CredReadW, CRED_TYPE_GENERIC};

        let target = cred_target(server);
        let target_wide = to_wide(&target);
        let mut cred_ptr: *mut crate::winffi::CredentialW = std::ptr::null_mut();
        let ok = unsafe { CredReadW(target_wide.as_ptr(), CRED_TYPE_GENERIC, 0, &mut cred_ptr) };
        if ok == 0 {
            return None;
        }
        let cred = unsafe { &*cred_ptr };
        let user = unsafe { from_wide(cred.user_name) };
        let pass = if cred.credential_blob_size > 0 && !cred.credential_blob.is_null() {
            let bytes = unsafe { std::slice::from_raw_parts(cred.credential_blob, cred.credential_blob_size as usize) };
            String::from_utf8_lossy(bytes).into_owned()
        } else {
            String::new()
        };
        unsafe { crate::winffi::CredFree(cred_ptr as *mut u8) };
        Some((user, pass))
    }

    /// 写入 SMB 凭据（Windows 凭据管理器，target `GameVHD_<server>`）。
    pub fn store_smb_cred(server: &str, user: &str, pass: &str) -> Result<(), DiskError> {
        use crate::winffi::{CredWriteW, CRED_PERSIST_LOCAL_MACHINE, CRED_TYPE_GENERIC, CredentialW};

        let target = cred_target(server);
        let target_wide = to_wide(&target);
        let user_wide = to_wide(user);
        let mut blob: Vec<u8> = pass.as_bytes().to_vec();
        let cred = CredentialW {
            flags: 0,
            type_: CRED_TYPE_GENERIC,
            target_name: target_wide.as_ptr() as *mut u16,
            comment: std::ptr::null_mut(),
            last_written: 0,
            credential_blob_size: blob.len() as u32,
            credential_blob: blob.as_mut_ptr(),
            persist: CRED_PERSIST_LOCAL_MACHINE,
            attribute_count: 0,
            attributes: std::ptr::null_mut(),
            target_alias: std::ptr::null_mut(),
            user_name: user_wide.as_ptr() as *mut u16,
        };
        let code = unsafe { CredWriteW(&cred as *const CredentialW, 0) };
        // blob 在 CredWriteW 返回后由本函数栈持有至函数退出，无需提前清零。
        if code == 0 {
            return Err(DiskError::new(
                classify_win32_error(unsafe { crate::winffi::GetLastError() }),
                format!("CredWriteW({target}) 失败"),
            ));
        }
        Ok(())
    }

    /// 删除 SMB 凭据（幂等：不存在视为成功）。
    pub fn delete_smb_cred(server: &str) -> Result<(), DiskError> {
        use crate::winffi::{CredDeleteW, CRED_TYPE_GENERIC};

        let target = cred_target(server);
        let target_wide = to_wide(&target);
        let ok = unsafe { CredDeleteW(target_wide.as_ptr(), CRED_TYPE_GENERIC, 0) };
        if ok == 0 {
            let code = unsafe { crate::winffi::GetLastError() };
            if code == crate::winffi::ERROR_NOT_FOUND {
                return Ok(());
            }
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("CredDeleteW({target}) 失败: {code}"),
            ));
        }
        Ok(())
    }

    /// 凭据 target 名：`GameVHD_<server>`，server 去尾部反斜杠（`\\srv\share\` → `\\srv\share`）。
    pub fn cred_target(server: &str) -> String {
        let trimmed = server.trim_end_matches('\\');
        format!("GameVHD_{trimmed}")
    }

    /// 创建差分盘（parent = UNC，格式按 parent 扩展名 VHD/VHDX 自适应）。
    /// diff 已存在 → Err(DiffAlreadyExists) 由调用方幂等跳过。
    pub fn create_diff_vhd(diff_path: &str, parent_unc: &str) -> Result<(), DiskError> {
        let spec = build_create_diff_params(parent_unc);
        let vst = VirtualStorageType {
            device_id: spec.parent_device_id,
            vendor_id: VENDOR_MICROSOFT,
        };
        let diff_wide = to_wide(diff_path);
        let parent_wide = to_wide(parent_unc);
        // 参数版本依格式：VHD→Version 1（64B），VHDX→Version 2（128B）。
        // 两结构布局不同，分别构造后按裸指针传入（FFI 参数均为 *const）。
        let params_v1: CreateVirtualDiskParametersV1;
        let params_v2: CreateVirtualDiskParameters;
        let params_ptr: *const u8 = match spec.format {
            VhdFormat::Vhd => {
                params_v1 = CreateVirtualDiskParametersV1 {
                    version: 1,
                    _pad: 0,
                    unique_id: Guid::zero(),
                    maximum_size: 0, // 差分盘继承 parent
                    block_size: 0,   // DEFAULT_BLOCK_SIZE（2MB）
                    sector_size: 0x200, // 512——VHD V1 结构文档硬性要求，0 会拒绝
                    parent_path: parent_wide.as_ptr(),
                    source_path: std::ptr::null(),
                };
                &params_v1 as *const CreateVirtualDiskParametersV1 as *const u8
            }
            VhdFormat::Vhdx => {
                params_v2 = CreateVirtualDiskParameters {
                    version: 2,
                    _pad: 0,
                    unique_id: Guid::zero(),
                    maximum_size: 0,
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
                &params_v2 as *const CreateVirtualDiskParameters as *const u8
            }
        };
        // VIRTUAL_DISK_ACCESS_MASK：文档规定 V2（VHDX）必须 NONE；微软官方
        // 示例（CppVhdAPI）V1（VHD）创建用 VIRTUAL_DISK_ACCESS_CREATE。
        let access_mask = match spec.format {
            VhdFormat::Vhd => VIRTUAL_DISK_ACCESS_CREATE,
            VhdFormat::Vhdx => VIRTUAL_DISK_ACCESS_NONE,
        };
        let mut handle: HANDLE = 0;
        let code = unsafe {
            CreateVirtualDisk(
                &vst as *const VirtualStorageType,
                diff_wide.as_ptr(),
                access_mask,
                std::ptr::null(),
                CREATE_VIRTUAL_DISK_FLAG_NONE,
                0,
                params_ptr,
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

    /// 打开差分盘句柄（attach/detach 前置）。类型按路径扩展名自适应
    /// （VHD=2 / VHDX=3）；access mask 用 ALL（含 DETACH 位 0x40000，
    /// 打开永久挂载盘必需，OpenVirtualDisk 文档要求）。
    pub fn open_vhd(path: &str) -> Result<HANDLE, DiskError> {
        let vst = VirtualStorageType {
            device_id: VhdFormat::detect(path).device_id(),
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
    /// 标志：
    /// - PERMANENT_LIFETIME：句柄关闭后挂载保持（游戏运行期依赖），卸载走显式 detach
    /// - `preferred.is_some()` 时加 NO_DRIVE_LETTER：禁止系统自动分配，盘符由
    ///   启动器 SetVolumeMountPoint 强挂到指定字母（显式控制能力）
    /// - `preferred.is_none()` 时不加：让系统自动分配（零干预、无注册表残留），
    ///   挂载后 GetVolumePathNamesForVolumeNameW 读回实际字母
    pub fn attach_vhd(handle: HANDLE, preferred: Option<char>) -> Result<(), DiskError> {
        let flags = if preferred.is_some() {
            ATTACH_VIRTUAL_DISK_FLAG_PERMANENT_LIFETIME | ATTACH_VIRTUAL_DISK_FLAG_NO_DRIVE_LETTER
        } else {
            ATTACH_VIRTUAL_DISK_FLAG_PERMANENT_LIFETIME
        };
        let code = unsafe {
            AttachVirtualDisk(
                handle,
                std::ptr::null(),
                flags,
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

    /// 若指定 VHD 已挂载则 detach（幂等，供 cleanup 清理崩溃残留挂载）。
    /// 返回是否实际执行了 detach；VHD 无法打开视为未挂载（Ok(false)）。
    pub fn detach_if_attached(vhd_path: &str) -> Result<bool, DiskError> {
        let handle = match open_vhd(vhd_path) {
            Ok(h) => h,
            Err(_) => return Ok(false),
        };
        let mut buf = [0u16; 260];
        let mut size: DWORD = (buf.len() * 2) as DWORD;
        let rc = unsafe { GetVirtualDiskPhysicalPath(handle, &mut size, buf.as_mut_ptr()) };
        if rc == 0 {
            let code = unsafe { DetachVirtualDisk(handle, DETACH_VIRTUAL_DISK_FLAG_NONE, 0) };
            unsafe { CloseHandle(handle) };
            if code != 0 && code != 0xC03A_001C {
                return Err(DiskError::new(
                    classify_win32_error(code),
                    format!("DetachVirtualDisk({vhd_path}) 失败: {code:#x}"),
                ));
            }
            return Ok(true);
        }
        unsafe { CloseHandle(handle) };
        Ok(false)
    }

    /// 查询 attach 后 VHD 的物理盘路径（`\\?\PhysicalDriveN`）。
    /// 单次调用 + MAX_PATH 缓冲：该 API 无"传 NULL 探测大小"语义（真机
    /// 实测两段式首调用返回 0x7a ERROR_INSUFFICIENT_BUFFER=122）。
    pub fn physical_path(handle: HANDLE) -> Result<String, DiskError> {
        let mut buf = vec![0u16; 260]; // MAX_PATH
        let mut size: DWORD = (buf.len() * 2) as DWORD;
        let code = unsafe { GetVirtualDiskPhysicalPath(handle, &mut size, buf.as_mut_ptr()) };
        if code != 0 {
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("GetVirtualDiskPhysicalPath 失败: {code:#x}"),
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
            match volume_has_disk(&name, disk_number) {
                Ok(true) => {
                    found = Some(name);
                    break;
                }
                Ok(false) => {
                    crate::log_warn!("卷 {name} 不在物理盘 {disk_number} 上，跳过");
                }
                Err(reason) => {
                    crate::log_warn!("卷 {name} 查询失败: {reason}");
                }
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
    /// 返回 Err(原因) 区分「无法查询」与「不在该盘」（诊断用）。
    /// 卷设备句柄：desired access=0（GENERIC_READ 被卷拒绝 → VolumeNotFound）、
    /// dwFlagsAndAttributes=0（设备句柄不适用文件属性）、share 含 FILE_SHARE_WRITE
    /// （CreateFile 文档 Physical Disks and Volumes 节硬性要求）。
    /// 卷名必须去尾反斜杠（`\\?\Volume{GUID}\` → `\\?\Volume{GUID}`）：
    /// FindFirstVolume 返回带尾斜杠，但 CreateFile 打开卷设备路径不带尾斜杠，
    /// 否则所有卷 CreateFileW 0x3 ERROR_PATH_NOT_FOUND（真机诊断实证）。
    fn volume_has_disk(volume_guid: &str, disk_number: u32) -> Result<bool, String> {
        let trimmed = volume_guid.trim_end_matches('\\');
        let volume_wide = to_wide(trimmed);
        let h = unsafe {
            CreateFileW(
                volume_wide.as_ptr(),
                0, // 卷设备：desired access 必须为 0
                FILE_SHARE_READ | FILE_SHARE_WRITE,
                std::ptr::null(),
                OPEN_EXISTING,
                0, // 设备句柄：flags 必须为 0，FILE_ATTRIBUTE_NORMAL 不适用
                0,
            )
        };
        if h == INVALID_HANDLE_VALUE {
            let code = unsafe { GetLastError() };
            return Err(format!("CreateFileW 失败: {code:#x}"));
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
            let code = unsafe { GetLastError() };
            return Err(format!("IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS 失败: {code:#x}"));
        }
        let on_disk = (0..extents.number_of_disk_extents)
            .any(|i| extents.extents[i as usize].disk_number == disk_number);
        Ok(on_disk)
    }

    /// 用 `SetVolumeMountPointW` 给卷分配盘符（`E:\` → `\\?\Volume{GUID}\`）。
    /// 这是给卷分配盘符的正确 API；`DefineDosDevice` 只定义进程命名空间的
    /// 设备名，文件资源管理器不可见（真机实证：DefineDosDevice "成功"但无盘符）。
    /// 卷名带尾反斜杠（SetVolumeMountPoint 文档要求 `\\?\Volume{GUID}\`）。
    pub fn assign_drive_letter(volume_guid: &str, letter: char) -> Result<(), DiskError> {
        let mount_point = format!("{}:\\", letter.to_ascii_uppercase());
        let mount_wide = to_wide(&mount_point);
        let target_wide = to_wide(volume_guid);
        let ok = unsafe { SetVolumeMountPointW(mount_wide.as_ptr(), target_wide.as_ptr()) };
        if ok == 0 {
            let code = unsafe { GetLastError() };
            return Err(DiskError::new(
                classify_win32_error(code),
                format!("SetVolumeMountPointW({mount_point} → {volume_guid}) 失败: {code:#x}"),
            ));
        }
        Ok(())
    }

    /// 移除盘符挂载点（`DeleteVolumeMountPointW("E:\\")`）。
    pub fn remove_drive_letter(letter: char) -> Result<(), DiskError> {
        let mount_point = format!("{}:\\", letter.to_ascii_uppercase());
        let mount_wide = to_wide(&mount_point);
        let ok = unsafe { DeleteVolumeMountPointW(mount_wide.as_ptr()) };
        if ok == 0 {
            let code = unsafe { GetLastError() };
            // ERROR_FILE_NOT_FOUND(2)：盘符本就不存在，幂等成功。
            if code != 2 {
                return Err(DiskError::new(
                    classify_win32_error(code),
                    format!("DeleteVolumeMountPointW({mount_point}) 失败: {code:#x}"),
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

    /// 读取系统实际分配给卷的盘符（`GetVolumePathNamesForVolumeNameW`）。
    /// 输出是多字符串 `X:\0Y:\0\0`（双 NUL 结尾），取首个单字母盘符路径。
    /// 系统自动分配模式（preferred=None）下用于获知实际盘符。
    fn read_assigned_letter(volume_guid: &str) -> Option<char> {
        let volume_wide = to_wide(volume_guid);
        let mut buf = vec![0u16; 512];
        let mut len: DWORD = 0;
        let ok = unsafe {
            GetVolumePathNamesForVolumeNameW(
                volume_wide.as_ptr(),
                buf.as_mut_ptr(),
                buf.len() as DWORD,
                &mut len,
            )
        };
        if ok == 0 {
            return None;
        }
        // 多字符串：逐段解析，找 `X:\` 形态（单字母盘符）的首个。
        let mut i = 0usize;
        while i < buf.len() && buf[i] != 0 {
            let seg: Vec<u16> = buf[i..].iter().take_while(|&&c| c != 0).copied().collect();
            let s = String::from_utf16_lossy(&seg);
            let mut cs = s.chars();
            if let (Some(c), Some(':')) = (cs.next(), cs.next()) {
                if c.is_ascii_alphabetic() {
                    return Some(c.to_ascii_uppercase());
                }
            }
            i += seg.len() + 1;
        }
        None
    }

    /// 完整挂载流程：SMB → 建差分（幂等）→ attach → 物理路径 → 卷匹配 → 分配盘符。
    /// 各步骤打 ASCII `[STEP-n]` 到 stdout：真机崩溃（0xC0000005）时用于定位
    /// 崩溃点（GBK 控制台会乱码中文日志，ASCII marker 可机器解析）。
    pub fn mount_vhd(params: &MountParams) -> Result<MountResult, DiskError> {
        println!("[STEP-1] smb-connect");
        // 1. SMB（可选）。
        if let Some(remote) = &params.smb_remote {
            smb_connect(
                remote,
                params.smb_user.as_deref(),
                params.smb_pass.as_deref(),
                params.smb_retries,
            )?;
        }

        println!("[STEP-2] ensure-diff-dir");
        // 2. 确保 diff 父目录存在（CreateVirtualDisk 不自动建目录；
        //    首次运行 diff 根缺失会返回 ERROR_PATH_NOT_FOUND=3）。
        if let Some(parent_dir) = std::path::Path::new(&params.diff_path).parent() {
            if !parent_dir.as_os_str().is_empty() {
                std::fs::create_dir_all(parent_dir).map_err(|e| {
                    DiskError::new(
                        DiskErrorKind::Other(0),
                        format!("创建 diff 父目录 {} 失败: {e}", parent_dir.display()),
                    )
                })?;
            }
        }

        println!("[STEP-3] create-diff");
        // 3. 创建差分盘（parent 提供时；已存在幂等跳过）。
        if let Some(parent) = &params.parent_unc {
            match create_diff_vhd(&params.diff_path, parent) {
                Ok(()) => {}
                Err(e) if e.kind == DiskErrorKind::DiffAlreadyExists => {
                    crate::log_info!("差分盘已存在，跳过创建: {}", params.diff_path);
                }
                Err(e) => return Err(e),
            }
        }

        println!("[STEP-4] open-attach");
        // 4. 打开 + attach（preferred 指定时禁系统分配，否则系统自动分配）。
        let handle = open_vhd(&params.diff_path)?;
        if let Err(e) = attach_vhd(handle, params.preferred_letter) {
            unsafe { CloseHandle(handle) };
            return Err(e);
        }

        println!("[STEP-5] physical-path");
        // 5. 物理盘路径 → 盘号。
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

        // 6. 盘符：preferred 指定 → SetVolumeMountPoint 强挂；未指定 → 读系统分配。
        let letter = match params.preferred_letter {
            Some(letter) => {
                assign_drive_letter(&volume_guid, letter)?;
                letter
            }
            None => read_assigned_letter(&volume_guid).ok_or_else(|| {
                DiskError::new(DiskErrorKind::NoFreeDriveLetter, "系统未分配盘符")
            })?,
        };

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
    assign_drive_letter, attach_vhd, create_diff_vhd, cred_target, delete_smb_cred,
    detach_if_attached, detach_vhd, find_volume_for_disk, mount_vhd, open_vhd, physical_path,
    read_smb_cred, remove_drive_letter, smb_connect, smb_disconnect, store_smb_cred, unmount_vhd,
    used_drive_letters, MountParams, MountResult, UnmountParams,
};

#[cfg(not(target_os = "windows"))]
pub fn detach_if_attached(_vhd_path: &str) -> Result<bool, DiskError> {
    Err(DiskError::new(
        DiskErrorKind::UnsupportedPlatform,
        "VHD detach 仅支持 Windows",
    ))
}

#[cfg(not(target_os = "windows"))]
pub fn read_smb_cred(_server: &str) -> Option<(String, String)> {
    None
}

#[cfg(not(target_os = "windows"))]
pub fn store_smb_cred(_server: &str, _user: &str, _pass: &str) -> Result<(), DiskError> {
    Err(DiskError::new(
        DiskErrorKind::UnsupportedPlatform,
        "凭据管理器仅支持 Windows",
    ))
}

#[cfg(not(target_os = "windows"))]
pub fn delete_smb_cred(_server: &str) -> Result<(), DiskError> {
    Err(DiskError::new(
        DiskErrorKind::UnsupportedPlatform,
        "凭据管理器仅支持 Windows",
    ))
}

#[cfg(not(target_os = "windows"))]
pub fn cred_target(server: &str) -> String {
    format!("GameVHD_{}", server.trim_end_matches('\\'))
}

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
        assert_eq!(classify_win32_error(1203), DiskErrorKind::SmbPathNotFound);
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
        // VHDX parent → Version 2 结构 + device_id 3。
        let spec = build_create_diff_params(r"\\192.168.1.4\Game\base.vhdx");
        assert_eq!(spec.format, VhdFormat::Vhdx);
        assert_eq!(spec.parent_path, r"\\192.168.1.4\Game\base.vhdx");
        assert_eq!(spec.parent_device_id, 3); // VHDX
        // Microsoft 厂商 GUID 前两字段。
        assert_eq!(spec.parent_vendor, [0xEC98_4AEC, 0xA0F9_47E9]);
    }

    #[test]
    fn vhd_format_detect_by_extension() {
        assert_eq!(VhdFormat::detect(r"\\nas\Game\base.vhd"), VhdFormat::Vhd);
        assert_eq!(VhdFormat::detect(r"\\nas\Game\base.VHD"), VhdFormat::Vhd);
        assert_eq!(VhdFormat::detect(r"\\nas\Game\base.vhdx"), VhdFormat::Vhdx);
        assert_eq!(VhdFormat::detect(r"\\nas\Game\base"), VhdFormat::Vhdx);
        assert_eq!(VhdFormat::detect(r"\\nas\Game\base.vhd.bak"), VhdFormat::Vhdx);
        assert_eq!(VhdFormat::detect(""), VhdFormat::Vhdx);
        assert_eq!(VhdFormat::Vhd.device_id(), 2);
        assert_eq!(VhdFormat::Vhdx.device_id(), 3);
    }

    #[test]
    fn build_diff_params_vhd_parent_uses_version1() {
        let spec = build_create_diff_params(r"\\192.168.1.4\Game\base.vhd");
        assert_eq!(spec.format, VhdFormat::Vhd);
        assert_eq!(spec.parent_device_id, 2); // VHD
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
}
