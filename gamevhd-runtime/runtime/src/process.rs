//! Job Object 进程树管理（W4T1 进程树等待）。
//!
//! `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` 语义：job 句柄关闭时内核强杀 job 内全部进程；
//! `WaitForSingleObject(job, INFINITE)` 在 job 内**所有进程**退出后返回 —— 一次等待
//! 即覆盖整棵进程树（游戏 → ModManager → CrashReporter），无需逐 pid 枚举
//! （协议 §6.4 由 gvhook 子进程注入保证整树入 job）。
//!
//! 纯逻辑部分（job 命名）Linux 可测；Windows 调用 `#[cfg(target_os = "windows")]`
//! 隔离，Linux 提供返回 [`ProcError::UnsupportedPlatform`] 的桩。

#![allow(dead_code)]

use std::fmt;

/// 进程树错误。
#[derive(Debug)]
pub enum ProcError {
    /// 当前平台不支持（非 Windows 桩）。
    UnsupportedPlatform,
    /// Win32 调用失败。
    Win32 { op: &'static str, code: u32 },
}

impl fmt::Display for ProcError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ProcError::UnsupportedPlatform => write!(f, "进程树管理仅支持 Windows"),
            ProcError::Win32 { op, code } => write!(f, "{op} 失败 (Win32 错误 {code})"),
        }
    }
}

impl std::error::Error for ProcError {}

/// 生成 job 名称 `GameVHD_<game_id>`（无 game_id 时为 `GameVHD_`）。
pub fn job_name(game_id: &str) -> String {
    format!("GameVHD_{game_id}")
}

/// 创建带 KILL_ON_JOB_CLOSE 的命名 Job Object；返回 job 句柄。
#[cfg(target_os = "windows")]
pub fn create_kill_on_close_job(game_id: &str) -> Result<crate::winffi::HANDLE, ProcError> {
    use crate::winffi;

    let name = utf16_nul(&job_name(game_id));
    let job = unsafe { winffi::CreateJobObjectW(std::ptr::null(), name.as_ptr()) };
    if job == 0 {
        return Err(ProcError::Win32 {
            op: "CreateJobObjectW",
            code: last_error(),
        });
    }

    let mut info: winffi::JobObjectExtendedLimitInformation = unsafe { std::mem::zeroed() };
    info.basic_limit_information.limit_flags = winffi::JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
    let ok = unsafe {
        winffi::SetInformationJobObject(
            job,
            winffi::JobObjectExtendedLimitInformation,
            &info as *const _ as *const u8,
            std::mem::size_of::<winffi::JobObjectExtendedLimitInformation>() as u32,
        )
    };
    if ok == 0 {
        let code = last_error();
        unsafe { winffi::CloseHandle(job) };
        return Err(ProcError::Win32 {
            op: "SetInformationJobObject",
            code,
        });
    }
    Ok(job)
}

/// 把根进程划入 job（此后其全部子进程自动入 job）。
#[cfg(target_os = "windows")]
pub fn assign_to_job(
    job: crate::winffi::HANDLE,
    hproc: crate::winffi::HANDLE,
) -> Result<(), ProcError> {
    use crate::winffi;

    let ok = unsafe { winffi::AssignProcessToJobObject(job, hproc) };
    if ok == 0 {
        return Err(ProcError::Win32 {
            op: "AssignProcessToJobObject",
            code: last_error(),
        });
    }
    Ok(())
}

/// 阻塞等待一个进程退出。
#[cfg(target_os = "windows")]
pub fn wait_process(hproc: crate::winffi::HANDLE) -> Result<(), ProcError> {
    use crate::winffi;

    let r = unsafe { winffi::WaitForSingleObject(hproc, winffi::INFINITE) };
    if r != winffi::WAIT_OBJECT_0 {
        return Err(ProcError::Win32 {
            op: "WaitForSingleObject(process)",
            code: last_error(),
        });
    }
    Ok(())
}

/// 等待 job 内全部进程退出，允许调用方设置有限的宽限期。
#[cfg(target_os = "windows")]
pub fn wait_process_tree_timeout(
    job: crate::winffi::HANDLE,
    timeout_ms: crate::winffi::DWORD,
) -> Result<bool, ProcError> {
    use crate::winffi;

    let r = unsafe { winffi::WaitForSingleObject(job, timeout_ms) };
    match r {
        winffi::WAIT_OBJECT_0 => Ok(true),
        winffi::WAIT_TIMEOUT => Ok(false),
        _ => Err(ProcError::Win32 {
            op: "WaitForSingleObject(job)",
            code: last_error(),
        }),
    }
}

/// 终止 job 内仍存活的辅助进程；调用方随后应等待 job 句柄变为有信号。
#[cfg(target_os = "windows")]
pub fn terminate_job(job: crate::winffi::HANDLE) -> Result<(), ProcError> {
    use crate::winffi;

    let ok = unsafe { winffi::TerminateJobObject(job, 1) };
    if ok == 0 {
        return Err(ProcError::Win32 {
            op: "TerminateJobObject",
            code: last_error(),
        });
    }
    Ok(())
}

/// 阻塞等待 job 内全部进程退出（job 句柄置信号）。返回时整棵进程树已终结。
#[cfg(target_os = "windows")]
pub fn wait_process_tree(job: crate::winffi::HANDLE) -> Result<(), ProcError> {
    wait_process_tree_timeout(job, crate::winffi::INFINITE).map(|_| ())
}

#[cfg(target_os = "windows")]
fn utf16_nul(s: &str) -> Vec<u16> {
    let mut v: Vec<u16> = s.encode_utf16().collect();
    v.push(0);
    v
}

#[cfg(target_os = "windows")]
fn last_error() -> u32 {
    unsafe { crate::winffi::GetLastError() }
}

// ---- Linux 桩（保证 crate 可编译/测试）。 ----

#[cfg(not(target_os = "windows"))]
pub fn create_kill_on_close_job(_game_id: &str) -> Result<usize, ProcError> {
    Err(ProcError::UnsupportedPlatform)
}

#[cfg(not(target_os = "windows"))]
pub fn assign_to_job(_job: usize, _hproc: usize) -> Result<(), ProcError> {
    Err(ProcError::UnsupportedPlatform)
}

#[cfg(not(target_os = "windows"))]
pub fn wait_process_tree(_job: usize) -> Result<(), ProcError> {
    Err(ProcError::UnsupportedPlatform)
}

#[cfg(not(target_os = "windows"))]
pub fn wait_process(_hproc: usize) -> Result<(), ProcError> {
    Err(ProcError::UnsupportedPlatform)
}

#[cfg(not(target_os = "windows"))]
pub fn wait_process_tree_timeout(_job: usize, _timeout_ms: u32) -> Result<bool, ProcError> {
    Err(ProcError::UnsupportedPlatform)
}

#[cfg(not(target_os = "windows"))]
pub fn terminate_job(_job: usize) -> Result<(), ProcError> {
    Err(ProcError::UnsupportedPlatform)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn job_name_format() {
        assert_eq!(job_name("abc"), "GameVHD_abc");
        assert_eq!(job_name(""), "GameVHD_");
        assert_eq!(job_name("horizon-zero-dawn"), "GameVHD_horizon-zero-dawn");
    }

    #[test]
    fn error_display() {
        let e = ProcError::Win32 {
            op: "SetInformationJobObject",
            code: 87,
        };
        assert!(e.to_string().contains("SetInformationJobObject"));
        assert!(e.to_string().contains("87"));
    }

    #[cfg(not(target_os = "windows"))]
    #[test]
    fn linux_stubs_unsupported() {
        assert!(matches!(
            create_kill_on_close_job("x"),
            Err(ProcError::UnsupportedPlatform)
        ));
        assert!(matches!(
            assign_to_job(0, 0),
            Err(ProcError::UnsupportedPlatform)
        ));
        assert!(matches!(
            wait_process_tree(0),
            Err(ProcError::UnsupportedPlatform)
        ));
    }
}
