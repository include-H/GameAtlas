//! Windows 原生 API 薄封装（阶段 0，W1 注入 / W3 hive / W4 进程树共享底层）。
//!
//! 纯手写 FFI：`extern "system"`（x86 stdcall / x64 Win64 统一调用约定，与 WINAPI
//! 一致）+ 原始 `usize` 句柄/指针，零外部 crate。声明覆盖注入（CreateProcessW /
//! VirtualAllocEx / WriteProcessMemory / CreateRemoteThread）、Job Object
//! （KILL_ON_JOB_CLOSE 进程树）与 diskpart 挂载桩。
//!
//! `#![cfg(target_os = "windows")]`：本文件在非 Windows 上为空模块（main.rs 的
//! Windows 专属分发体也按 `#[cfg(target_os = "windows")]` 编译，互不引用）。

#![cfg(target_os = "windows")]
#![allow(non_snake_case, non_upper_case_globals, non_camel_case_types, dead_code)]

// ---- 基础类型（Windows 句柄/指针均为指针宽度）。 ----

pub type HANDLE = usize;
pub type HMODULE = usize;
pub type DWORD = u32;
pub type BOOL = i32;
pub type SIZE_T = usize;
pub type LPVOID = usize;

// ---- 常量。 ----

pub const CREATE_SUSPENDED: u32 = 0x0000_0004;
pub const MEM_COMMIT: u32 = 0x0000_1000;
pub const MEM_RESERVE: u32 = 0x0000_2000;
pub const MEM_RELEASE: u32 = 0x0000_8000;
pub const PAGE_READWRITE: u32 = 0x0000_0004;
pub const PAGE_EXECUTE_READWRITE: u32 = 0x0000_0040;
pub const INFINITE: u32 = 0xFFFF_FFFF;
pub const WAIT_OBJECT_0: u32 = 0;
pub const WAIT_TIMEOUT: u32 = 0x0000_0102;
pub const WAIT_FAILED: u32 = 0xFFFF_FFFF;
pub const INVALID_HANDLE_VALUE: usize = usize::MAX;

pub const JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE: u32 = 0x0000_2000;
pub const JobObjectExtendedLimitInformation: i32 = 9;

// ---- 注册表 hive 相关常量（W3：RegLoadKey / RegSaveKey / 特权启用）。 ----

/// `LSTATUS`（LONG）：Reg* API 返回码。
pub type LSTATUS = i32;

/// 预定义根键（HKEY 实为指针宽度的句柄值）。
pub const HKEY_CURRENT_USER: HANDLE = 0x8000_0001;
pub const HKEY_USERS: HANDLE = 0x8000_0003;

/// RegCreateKeyExW：所有访问权（模板键由本进程新建，无 ACL 顾虑）。
pub const KEY_ALL_ACCESS: u32 = 0x000F_003F;
/// RegOpenKeyExW：只查询已挂载 hive 是否存在。
pub const KEY_READ: u32 = 0x0002_0019;
/// RegCreateKeyExW：非易失键（落盘，随文件保存）。
pub const REG_OPTION_NON_VOLATILE: u32 = 0;

// Reg* API 返回码（LSTATUS）。
pub const ERROR_SUCCESS: LSTATUS = 0;
pub const ERROR_FILE_NOT_FOUND: LSTATUS = 2;
pub const ERROR_INVALID_PARAMETER: LSTATUS = 87;
pub const ERROR_BUSY: LSTATUS = 170;

// 特权启用（AdjustTokenPrivileges / GetLastError，DWORD 域）。
pub const SE_PRIVILEGE_ENABLED: u32 = 0x0000_0002;
pub const TOKEN_QUERY: u32 = 0x0000_0008;
pub const TOKEN_ADJUST_PRIVILEGES: u32 = 0x0000_0020;
/// 特权列表有项未被启用（AdjustTokenPrivileges 返回 TRUE 但仍需检查）。
pub const ERROR_NOT_ALL_ASSIGNED: u32 = 1300;

/// 对象已存在（CreateMutexW 检测双开：已有同名互斥体）。
pub const ERROR_ALREADY_EXISTS: u32 = 183;

/// ASCII 字节串 → 等长 UTF-16 数组（const 上下文用；`array::map` 尚未 const 稳定）。
const fn utf16_of<const N: usize>(s: &[u8]) -> [u16; N] {
    let mut out = [0u16; N];
    let mut i = 0;
    while i < N {
        out[i] = s[i] as u16;
        i += 1;
    }
    out
}

/// `SeBackupPrivilege`（RegSaveKeyW 所需，NUL 结尾）。
pub const SE_BACKUP_NAME: [u16; 18] = utf16_of(b"SeBackupPrivilege\0");
/// `SeRestorePrivilege`（RegLoadKeyW / RegUnLoadKeyW 所需，NUL 结尾）。
pub const SE_RESTORE_NAME: [u16; 19] = utf16_of(b"SeRestorePrivilege\0");

// ---- 结构体（repr(C)，自然对齐；x86/x64 布局与 Windows 头文件一致）。 ----

/// `PROCESS_INFORMATION`：x86 = 16B，x64 = 24B。
#[repr(C)]
pub struct ProcessInformation {
    pub h_process: HANDLE,
    pub h_thread: HANDLE,
    pub dw_process_id: DWORD,
    pub dw_thread_id: DWORD,
}

/// `STARTUPINFOW`：x86 = 68B，x64 = 104B。
#[repr(C)]
pub struct StartupInfoW {
    pub cb: DWORD,
    pub lp_reserved: *mut u16,
    pub lp_desktop: *mut u16,
    pub lp_title: *mut u16,
    pub dw_x: DWORD,
    pub dw_y: DWORD,
    pub dw_x_size: DWORD,
    pub dw_y_size: DWORD,
    pub dw_x_count_chars: DWORD,
    pub dw_y_count_chars: DWORD,
    pub dw_fill_attribute: DWORD,
    pub dw_flags: DWORD,
    pub w_show_window: u16,
    pub cb_reserved2: u16,
    pub lp_reserved2: *mut u8,
    pub h_std_input: HANDLE,
    pub h_std_output: HANDLE,
    pub h_std_error: HANDLE,
}

/// `JOBOBJECT_BASIC_LIMIT_INFORMATION`（`limit_flags` 恒在偏移 16）。
#[repr(C)]
pub struct JobObjectBasicLimitInformation {
    pub per_process_user_time_limit: i64,
    pub per_job_user_time_limit: i64,
    pub limit_flags: u32,
    pub minimum_working_set_size: usize,
    pub maximum_working_set_size: usize,
    pub active_process_limit: u32,
    pub affinity: usize,
    pub priority_class: u32,
    pub scheduling_class: u32,
}

/// `IO_COUNTERS`。
#[repr(C)]
pub struct IoCounters {
    pub read_operation_count: u64,
    pub write_operation_count: u64,
    pub other_operation_count: u64,
    pub read_transfer_count: u64,
    pub write_transfer_count: u64,
    pub other_transfer_count: u64,
}

/// `JOBOBJECT_EXTENDED_LIMIT_INFORMATION`：x86 = 112B，x64 = 144B。
#[repr(C)]
pub struct JobObjectExtendedLimitInformation {
    pub basic_limit_information: JobObjectBasicLimitInformation,
    pub io_info: IoCounters,
    pub process_memory_limit: usize,
    pub job_memory_limit: usize,
    pub peak_process_memory_used: usize,
    pub peak_job_memory_used: usize,
}

/// `LUID`（LookupPrivilegeValueW 输出）。
#[repr(C)]
pub struct Luid {
    pub low_part: u32,
    pub high_part: i32,
}

/// `LUID_AND_ATTRIBUTES`（AdjustTokenPrivileges 特权项）。
#[repr(C)]
pub struct LuidAndAttributes {
    pub luid: Luid,
    pub attributes: u32,
}

/// `TOKEN_PRIVILEGES`（单特权版本；`Privileges` 为 C 的 ANYSIZE_ARRAY=1）。
#[repr(C)]
pub struct TokenPrivileges {
    pub privilege_count: u32,
    pub privileges: [LuidAndAttributes; 1],
}

// ---- Win32 API。 ----

extern "system" {
    pub fn CreateProcessW(
        lp_application_name: *const u16,
        lp_command_line: *mut u16,
        lp_process_attributes: *const u8,
        lp_thread_attributes: *const u8,
        b_inherit_handles: BOOL,
        dw_creation_flags: DWORD,
        lp_environment: *const u8,
        lp_current_directory: *const u16,
        lp_startup_info: *const StartupInfoW,
        lp_process_information: *mut ProcessInformation,
    ) -> BOOL;

    pub fn VirtualAllocEx(
        h_process: HANDLE,
        lp_address: LPVOID,
        dw_size: SIZE_T,
        fl_allocation_type: DWORD,
        fl_protect: DWORD,
    ) -> LPVOID;

    pub fn VirtualFreeEx(
        h_process: HANDLE,
        lp_address: LPVOID,
        dw_size: SIZE_T,
        dw_free_type: DWORD,
    ) -> BOOL;

    pub fn WriteProcessMemory(
        h_process: HANDLE,
        lp_base_address: LPVOID,
        lp_buffer: *const u8,
        n_size: SIZE_T,
        lp_number_of_bytes_written: *mut SIZE_T,
    ) -> BOOL;

    pub fn ReadProcessMemory(
        h_process: HANDLE,
        lp_base_address: LPVOID,
        lp_buffer: *mut u8,
        n_size: SIZE_T,
        lp_number_of_bytes_read: *mut SIZE_T,
    ) -> BOOL;

    pub fn CreateRemoteThread(
        h_process: HANDLE,
        lp_thread_attributes: *const u8,
        dw_stack_size: SIZE_T,
        lp_start_address: LPVOID,
        lp_parameter: LPVOID,
        dw_creation_flags: DWORD,
        lp_thread_id: *mut DWORD,
    ) -> HANDLE;

    pub fn GetExitCodeThread(h_thread: HANDLE, lp_exit_code: *mut DWORD) -> BOOL;

    pub fn WaitForSingleObject(h_handle: HANDLE, dw_milliseconds: DWORD) -> DWORD;

    pub fn CloseHandle(h_object: HANDLE) -> BOOL;

    pub fn ResumeThread(h_thread: HANDLE) -> DWORD;

    pub fn QueueUserAPC(
        pfn_apc: LPVOID,
        h_thread: HANDLE,
        dw_data: usize,
    ) -> DWORD;

    pub fn Sleep(milliseconds: DWORD);

    pub fn GetModuleHandleW(lp_module_name: *const u16) -> HMODULE;

    pub fn LoadLibraryW(lp_lib_file_name: *const u16) -> HMODULE;

    pub fn GetProcAddress(h_module: HMODULE, lp_proc_name: *const u8) -> LPVOID;

    pub fn GetLastError() -> DWORD;

    pub fn CreateMutexW(
        lp_mutex_attributes: *const u8,
        b_initial_owner: BOOL,
        lp_name: *const u16,
    ) -> HANDLE;

    pub fn TerminateProcess(h_process: HANDLE, u_exit_code: u32) -> BOOL;

    pub fn TerminateJobObject(h_job: HANDLE, u_exit_code: u32) -> BOOL;

    pub fn CreateJobObjectW(lp_job_attributes: *const u8, lp_name: *const u16) -> HANDLE;

    pub fn AssignProcessToJobObject(h_job: HANDLE, h_process: HANDLE) -> BOOL;

    pub fn SetInformationJobObject(
        h_job: HANDLE,
        job_object_information_class: i32,
        lp_job_object_information: *const u8,
        cb_job_object_information_length: DWORD,
    ) -> BOOL;

    pub fn OpenJobObjectW(
        dw_desired_access: DWORD,
        b_inherit_handle: BOOL,
        lp_name: *const u16,
    ) -> HANDLE;
}

// 注册表 hive 与特权 API（advapi32）。与上方 kernel32 块分离，显式声明链接库。
#[link(name = "advapi32")]
extern "system" {
    pub fn RegLoadKeyW(h_key: HANDLE, lp_sub_key: *const u16, lp_file: *const u16) -> LSTATUS;

    pub fn RegUnLoadKeyW(h_key: HANDLE, lp_sub_key: *const u16) -> LSTATUS;

    pub fn RegOpenKeyExW(
        h_key: HANDLE,
        lp_sub_key: *const u16,
        ul_options: DWORD,
        sam_desired: u32,
        phk_result: *mut HANDLE,
    ) -> LSTATUS;

    pub fn RegSaveKeyW(
        h_key: HANDLE,
        lp_file: *const u16,
        lp_security_attributes: *const u8,
    ) -> LSTATUS;

    pub fn RegCreateKeyExW(
        h_key: HANDLE,
        lp_sub_key: *const u16,
        reserved: DWORD,
        lp_class: *mut u16,
        dw_options: DWORD,
        sam_desired: u32,
        lp_security_attributes: *const u8,
        phk_result: *mut HANDLE,
        lpdw_disposition: *mut DWORD,
    ) -> LSTATUS;

    pub fn RegDeleteKeyW(h_key: HANDLE, lp_sub_key: *const u16) -> LSTATUS;

    pub fn RegCloseKey(h_key: HANDLE) -> LSTATUS;

    pub fn OpenProcessToken(
        h_process: HANDLE,
        desired_access: DWORD,
        token_handle: *mut HANDLE,
    ) -> BOOL;

    pub fn AdjustTokenPrivileges(
        token_handle: HANDLE,
        disable_all_privileges: BOOL,
        new_state: *const TokenPrivileges,
        buffer_length: DWORD,
        previous_state: *mut TokenPrivileges,
        return_length: *mut DWORD,
    ) -> BOOL;

    pub fn LookupPrivilegeValueW(
        lp_system_name: *const u16,
        lp_name: *const u16,
        lp_luid: *mut Luid,
    ) -> BOOL;

    pub fn GetCurrentProcess() -> HANDLE;

    pub fn CredReadW(
        target_name: *const u16,
        type_: DWORD,
        flags: DWORD,
        credential: *mut *mut CredentialW,
    ) -> BOOL;

    pub fn CredWriteW(credential: *const CredentialW, flags: DWORD) -> BOOL;

    pub fn CredDeleteW(target_name: *const u16, type_: DWORD, flags: DWORD) -> BOOL;

    pub fn CredFree(buffer: *mut u8);
}

/// Windows 凭据管理器条目（CREDENTIALW 最小子集：读/写 SMB 密码所需字段）。
#[repr(C)]
pub struct CredentialW {
    pub flags: DWORD,
    pub type_: DWORD,
    pub target_name: *mut u16,
    pub comment: *mut u16,
    pub last_written: u64,
    pub credential_blob_size: DWORD,
    pub credential_blob: *mut u8,
    pub persist: DWORD,
    pub attribute_count: DWORD,
    pub attributes: *mut u8,
    pub target_alias: *mut u16,
    pub user_name: *mut u16,
}

/// 凭据类型：通用凭据。
pub const CRED_TYPE_GENERIC: DWORD = 1;
/// 持久化：本机（下次登录仍可用）。
pub const CRED_PERSIST_LOCAL_MACHINE: DWORD = 2;
/// 凭据不存在（CredReadW 失败码）。
pub const ERROR_NOT_FOUND: DWORD = 1168;

// ---- diskpart 桩（main.rs 的 mount/unmount 在 Windows 分支调用）。 ----

/// 通过 diskpart 挂载 VHD，等待盘符出现。
pub fn diskpart_mount(_vhd_path: &str) -> Result<(), String> {
    Err("diskpart mount: 未实现（后续波次）".into())
}

/// 通过 diskpart 卸载 VHD（幂等）。
pub fn diskpart_unmount(_vhd_path: &str) -> Result<(), String> {
    Err("diskpart unmount: 未实现（后续波次）".into())
}
