//! 远程 DLL 注入三件套（W1T3 编排注入）。
//!
//! 按 `docs/injection_protocol.md` §5 双远程线程序列注入 gvhook：
//!
//! 1. `CreateProcessW(CREATE_SUSPENDED)` 挂起创建游戏进程；
//! 2. `VirtualAllocEx` 单块分配 [参数块 | 规则表 | DLL 路径 UTF-16]，两次
//!    `WriteProcessMemory` 写入；
//! 3. 线程① `CreateRemoteThread(LoadLibraryW, 参数块内 hook_dll_path)` → 等待 →
//!    `GetExitCodeThread` 得远程 HMODULE；
//! 4. 本进程 `LoadLibraryW` + `GetProcAddress` 换算 `gvhook_init` 导出 RVA
//!    （同 DLL 文件 → 同导出 RVA），远程地址 = 远程 hmod + RVA；
//! 5. 线程② `CreateRemoteThread(remote_gvhook_init, remote_base)` → 等待 →
//!    读返回码（0 = `GVHD_INIT_OK`）。
//!
//! 位宽不变式（协议 §6.3）：编排与 gvhook 同位数（发布 x64/x86 双版本）。
//! 失败路径：关闭已开线程句柄 + `VirtualFreeEx` 释放远程区域（协议 §5.3）。
//!
//! Windows 实现 `#[cfg(target_os = "windows")]`；Linux 提供返回
//! [`InjectError::UnsupportedPlatform`] 的桩，保证 crate 可 Linux 编译/测试。

#![allow(dead_code)]

use std::fmt;

/// 注入错误。
#[derive(Debug)]
pub enum InjectError {
    /// 当前平台不支持（非 Windows 桩）。
    UnsupportedPlatform,
    /// 参数非法。
    InvalidArgument(String),
    /// Win32 调用失败。
    Win32 { op: &'static str, code: u32 },
    /// `gvhook_init` 返回非零（协议 §10 错误码）。
    InitFailed(u32),
}

impl fmt::Display for InjectError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            InjectError::UnsupportedPlatform => write!(f, "注入仅支持 Windows"),
            InjectError::InvalidArgument(msg) => write!(f, "非法参数: {msg}"),
            InjectError::Win32 { op, code } => write!(f, "{op} 失败 (Win32 错误 {code})"),
            InjectError::InitFailed(code) => write!(f, "gvhook_init 返回非零: {code}"),
        }
    }
}

impl std::error::Error for InjectError {}

/// 以 `CREATE_SUSPENDED` 挂起创建游戏进程，返回 `(hProcess, hThread)`。
#[cfg(target_os = "windows")]
pub fn launch_suspended(
    exe_path: &str,
    work_dir: &str,
) -> Result<(crate::winffi::HANDLE, crate::winffi::HANDLE), InjectError> {
    use crate::winffi;

    let app = utf16_nul(exe_path);
    // 带引号命令行，保证 argv[0] 语义并兼容带空格路径。
    let mut cmd = utf16_nul(&format!("\"{exe_path}\""));
    let cwd_units = if work_dir.is_empty() {
        None
    } else {
        Some(utf16_nul(work_dir))
    };
    let cwd_ptr = cwd_units.as_ref().map_or(std::ptr::null(), |v| v.as_ptr());

    let mut si: winffi::StartupInfoW = unsafe { std::mem::zeroed() };
    si.cb = std::mem::size_of::<winffi::StartupInfoW>() as u32;
    let mut pi: winffi::ProcessInformation = unsafe { std::mem::zeroed() };

    let ok = unsafe {
        winffi::CreateProcessW(
            app.as_ptr(),
            cmd.as_mut_ptr(),
            std::ptr::null(),
            std::ptr::null(),
            0,
            winffi::CREATE_SUSPENDED,
            std::ptr::null(),
            cwd_ptr,
            &si,
            &mut pi,
        )
    };
    if ok == 0 {
        return Err(InjectError::Win32 {
            op: "CreateProcessW",
            code: last_error(),
        });
    }
    Ok((pi.h_process, pi.h_thread))
}

/// 向已挂起进程注入 gvhook（双远程线程，协议 §5.2）。返回时钩子已安装完毕，
/// 游戏主线程仍挂起，由调用方随后 `ResumeThread`。
#[cfg(target_os = "windows")]
pub fn inject_into_process(
    hproc: crate::winffi::HANDLE,
    param_block: &[u8],
    hook_dll_path: &str,
) -> Result<(), InjectError> {
    use crate::winffi;

    let dll = utf16_nul(hook_dll_path);
    let dll_bytes = units_to_bytes(&dll);
    let alloc_size = param_block.len() + dll_bytes.len();

    let remote_base = unsafe {
        winffi::VirtualAllocEx(
            hproc,
            0,
            alloc_size,
            winffi::MEM_COMMIT | winffi::MEM_RESERVE,
            winffi::PAGE_READWRITE,
        )
    };
    if remote_base == 0 {
        return Err(InjectError::Win32 {
            op: "VirtualAllocEx",
            code: last_error(),
        });
    }

    let mut thread1: winffi::HANDLE = 0;
    let mut thread2: winffi::HANDLE = 0;

    let result = (|| -> Result<(), InjectError> {
        // 参数块（含规则表）与 DLL 路径 UTF-16 同块写入。
        if !write_remote(hproc, remote_base, param_block.as_ptr(), param_block.len()) {
            return Err(InjectError::Win32 {
                op: "WriteProcessMemory(param_block)",
                code: last_error(),
            });
        }
        let dll_addr = remote_base + param_block.len();
        if !write_remote(hproc, dll_addr, dll_bytes.as_ptr(), dll_bytes.len()) {
            return Err(InjectError::Win32 {
                op: "WriteProcessMemory(dll_path)",
                code: last_error(),
            });
        }

        // 线程①：LoadLibraryW(hook_dll_path)。kernel32 为 known-DLL 同基址。
        let kernel32_name = utf16_nul("kernel32.dll");
        let kernel32 = unsafe { winffi::GetModuleHandleW(kernel32_name.as_ptr()) };
        if kernel32 == 0 {
            return Err(InjectError::Win32 {
                op: "GetModuleHandleW(kernel32)",
                code: last_error(),
            });
        }
        let loadlib = unsafe { winffi::GetProcAddress(kernel32, b"LoadLibraryW\0".as_ptr()) };
        if loadlib == 0 {
            return Err(InjectError::Win32 {
                op: "GetProcAddress(LoadLibraryW)",
                code: last_error(),
            });
        }
        thread1 = unsafe {
            winffi::CreateRemoteThread(
                hproc,
                std::ptr::null(),
                0,
                loadlib,
                dll_addr,
                0,
                std::ptr::null_mut(),
            )
        };
        if thread1 == 0 {
            return Err(InjectError::Win32 {
                op: "CreateRemoteThread(LoadLibraryW)",
                code: last_error(),
            });
        }
        wait_infinite(thread1)?;
        let mut remote_hmod: winffi::DWORD = 0;
        if unsafe { winffi::GetExitCodeThread(thread1, &mut remote_hmod) } == 0 {
            return Err(InjectError::Win32 {
                op: "GetExitCodeThread",
                code: last_error(),
            });
        }
        if remote_hmod == 0 {
            return Err(InjectError::InvalidArgument(format!(
                "远程 LoadLibraryW 失败: {hook_dll_path}"
            )));
        }

        // RVA 换算：本进程加载同位数 gvhook，取导出相对基址偏移。
        let local_hmod = unsafe { winffi::LoadLibraryW(dll.as_ptr()) };
        if local_hmod == 0 {
            return Err(InjectError::Win32 {
                op: "LoadLibraryW(本地 gvhook)",
                code: last_error(),
            });
        }
        let local_fn = unsafe { winffi::GetProcAddress(local_hmod, b"gvhook_init\0".as_ptr()) };
        if local_fn == 0 {
            return Err(InjectError::InvalidArgument(
                "gvhook 未导出 gvhook_init".into(),
            ));
        }
        let remote_fn = remote_hmod as usize + (local_fn - local_hmod);

        // 线程②：gvhook_init(param_block)。
        thread2 = unsafe {
            winffi::CreateRemoteThread(
                hproc,
                std::ptr::null(),
                0,
                remote_fn,
                remote_base,
                0,
                std::ptr::null_mut(),
            )
        };
        if thread2 == 0 {
            return Err(InjectError::Win32 {
                op: "CreateRemoteThread(gvhook_init)",
                code: last_error(),
            });
        }
        wait_infinite(thread2)?;
        let mut init_code: winffi::DWORD = 0;
        if unsafe { winffi::GetExitCodeThread(thread2, &mut init_code) } == 0 {
            return Err(InjectError::Win32 {
                op: "GetExitCodeThread",
                code: last_error(),
            });
        }
        if init_code != 0 {
            return Err(InjectError::InitFailed(init_code));
        }
        Ok(())
    })();

    if thread1 != 0 {
        unsafe { winffi::CloseHandle(thread1) };
    }
    if thread2 != 0 {
        unsafe { winffi::CloseHandle(thread2) };
    }
    // hook 已在 gvhook_init 内保存私有副本（协议 §6.1），成功后也可释放。
    unsafe { winffi::VirtualFreeEx(hproc, remote_base, 0, winffi::MEM_RELEASE) };

    result
}

/// 恢复主线程并等待根进程退出（进程树整体等待由 [`crate::process`] 的
/// Job Object 负责；此处仅占位集成）。
#[cfg(target_os = "windows")]
pub fn resume_and_wait(
    hproc: crate::winffi::HANDLE,
    hthread: crate::winffi::HANDLE,
) -> Result<(), InjectError> {
    use crate::winffi;

    let prev = unsafe { winffi::ResumeThread(hthread) };
    if prev == u32::MAX {
        return Err(InjectError::Win32 {
            op: "ResumeThread",
            code: last_error(),
        });
    }
    wait_infinite(hproc)
}

/// UTF-16LE（含结尾 NUL）。
#[cfg(target_os = "windows")]
fn utf16_nul(s: &str) -> Vec<u16> {
    let mut v: Vec<u16> = s.encode_utf16().collect();
    v.push(0);
    v
}

/// u16 码元序列 → 小端字节。
#[cfg(target_os = "windows")]
fn units_to_bytes(units: &[u16]) -> Vec<u8> {
    let mut v = Vec::with_capacity(units.len() * 2);
    for u in units {
        v.extend_from_slice(&u.to_le_bytes());
    }
    v
}

#[cfg(target_os = "windows")]
fn last_error() -> u32 {
    unsafe { crate::winffi::GetLastError() }
}

/// 单次远程内存写入。
#[cfg(target_os = "windows")]
fn write_remote(
    hproc: crate::winffi::HANDLE,
    base: crate::winffi::LPVOID,
    data: *const u8,
    len: usize,
) -> bool {
    unsafe {
        crate::winffi::WriteProcessMemory(hproc, base, data, len, std::ptr::null_mut()) != 0
    }
}

/// 无限等待句柄，返回 `WAIT_FAILED` 时出错。
#[cfg(target_os = "windows")]
fn wait_infinite(h: crate::winffi::HANDLE) -> Result<(), InjectError> {
    let r = unsafe { crate::winffi::WaitForSingleObject(h, crate::winffi::INFINITE) };
    if r == crate::winffi::WAIT_FAILED {
        Err(InjectError::Win32 {
            op: "WaitForSingleObject",
            code: last_error(),
        })
    } else {
        Ok(())
    }
}

// ---- Linux 桩（保证 crate 可编译/测试）。 ----

#[cfg(not(target_os = "windows"))]
pub fn launch_suspended(
    _exe_path: &str,
    _work_dir: &str,
) -> Result<(usize, usize), InjectError> {
    Err(InjectError::UnsupportedPlatform)
}

#[cfg(not(target_os = "windows"))]
pub fn inject_into_process(
    _hproc: usize,
    _param_block: &[u8],
    _hook_dll_path: &str,
) -> Result<(), InjectError> {
    Err(InjectError::UnsupportedPlatform)
}

#[cfg(not(target_os = "windows"))]
pub fn resume_and_wait(_hproc: usize, _hthread: usize) -> Result<(), InjectError> {
    Err(InjectError::UnsupportedPlatform)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn error_display() {
        let e = InjectError::Win32 {
            op: "CreateProcessW",
            code: 5,
        };
        assert!(e.to_string().contains("CreateProcessW"));
        assert!(e.to_string().contains("5"));
        let f = InjectError::InitFailed(4);
        assert!(f.to_string().contains("4"));
    }

    #[cfg(not(target_os = "windows"))]
    #[test]
    fn linux_stubs_unsupported() {
        assert!(matches!(
            launch_suspended("a.exe", ""),
            Err(InjectError::UnsupportedPlatform)
        ));
        assert!(matches!(
            inject_into_process(0, &[], "x.dll"),
            Err(InjectError::UnsupportedPlatform)
        ));
        assert!(matches!(
            resume_and_wait(0, 0),
            Err(InjectError::UnsupportedPlatform)
        ));
    }
}
