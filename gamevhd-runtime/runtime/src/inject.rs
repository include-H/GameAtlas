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
    launch_suspended_with_args(exe_path, work_dir, "")
}

/// 以 CREATE_SUSPENDED 挂起创建进程，并附加一段已由调用方组装的命令行参数。
#[cfg(target_os = "windows")]
pub fn launch_suspended_with_args(
    exe_path: &str,
    work_dir: &str,
    args: &str,
) -> Result<(crate::winffi::HANDLE, crate::winffi::HANDLE), InjectError> {
    use crate::winffi;

    let app = utf16_nul(exe_path);
    // 带引号命令行，保证 argv[0] 语义并兼容带空格路径。
    let command_line = if args.trim().is_empty() {
        format!("\"{exe_path}\"")
    } else {
        format!("\"{exe_path}\" {args}")
    };
    let mut cmd = utf16_nul(&command_line);
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
    #[cfg(target_arch = "x86_64")]
    {
        return inject_into_process_x64(hproc, param_block, hook_dll_path);
    }

    #[cfg(not(target_arch = "x86_64"))]
    {
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
}

#[cfg(target_os = "windows")]
#[cfg(target_arch = "x86_64")]
fn x64_loader_init_stub() -> [u8; 59] {
    [
        0x41, 0x54,                         // push r12
        0x48, 0x83, 0xec, 0x20,             // sub rsp, 0x20
        0x49, 0x89, 0xcc,                   // mov r12, rcx
        0x48, 0x8b, 0x41, 0x10,             // mov rax, [rcx+0x10]
        0x48, 0x8b, 0x09,                   // mov rcx, [rcx]
        0xff, 0xd0,                         // call rax (LoadLibraryW)
        0x48, 0x85, 0xc0,                   // test rax, rax
        0x74, 0x13,                         // jz -> failure
        0x49, 0x03, 0x44, 0x24, 0x18,       // add rax, [r12+0x18]
        0x49, 0x8b, 0x4c, 0x24, 0x08,       // mov rcx, [r12+8]
        0xff, 0xd0,                         // call gvhook_init
        0x41, 0x89, 0x44, 0x24, 0x20,       // mov [r12+0x20], eax
        0xeb, 0x0a,                         // jmp -> cleanup
        0xb8, 0xfe, 0xff, 0xff, 0xff,       // mov eax, 0xfffffffe
        0x41, 0x89, 0x44, 0x24, 0x20,       // mov [r12+0x20], eax
        0x48, 0x83, 0xc4, 0x20,             // add rsp, 0x20
        0x41, 0x5c,                         // pop r12
        0xc3,                               // ret
    ]
}

/// x64 remote-thread bootstrap. `GetExitCodeThread` exposes only DWORD, so a
/// LoadLibraryW thread cannot safely carry a 64-bit HMODULE back to the injector.
/// The bootstrap keeps the full pointer in RAX and writes only the gvhook_init
/// return code to a remote u32 status cell.
#[cfg(target_os = "windows")]
#[cfg(target_arch = "x86_64")]
fn inject_into_process_x64(
    hproc: crate::winffi::HANDLE,
    param_block: &[u8],
    hook_dll_path: &str,
) -> Result<(), InjectError> {
    use crate::winffi;

    let dll = utf16_nul(hook_dll_path);
    let dll_bytes = units_to_bytes(&dll);
    let alloc_size = param_block.len() + dll_bytes.len();
    if alloc_size == 0 {
        return Err(InjectError::InvalidArgument(
            "remote 参数块和 DLL 路径不能为空".into(),
        ));
    }

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
            op: "VirtualAllocEx(param_block)",
            code: last_error(),
        });
    }

    let mut remote_context: winffi::LPVOID = 0;
    let mut remote_stub: winffi::LPVOID = 0;
    let mut remote_thread: winffi::HANDLE = 0;
    let result = (|| -> Result<(), InjectError> {
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
        let local_hmod = unsafe { winffi::LoadLibraryW(dll.as_ptr()) };
        if local_hmod == 0 {
            return Err(InjectError::Win32 {
                op: "LoadLibraryW(local gvhook)",
                code: last_error(),
            });
        }
        let local_fn = unsafe { winffi::GetProcAddress(local_hmod, b"gvhook_init\0".as_ptr()) };
        if local_fn == 0 || local_fn < local_hmod {
            return Err(InjectError::InvalidArgument(
                "gvhook 未导出可计算 RVA 的 gvhook_init".into(),
            ));
        }
        let init_rva = local_fn - local_hmod;

        let mut context = vec![0u8; 40];
        context[0..8].copy_from_slice(&(dll_addr as u64).to_le_bytes());
        context[8..16].copy_from_slice(&(remote_base as u64).to_le_bytes());
        context[16..24].copy_from_slice(&(loadlib as u64).to_le_bytes());
        context[24..32].copy_from_slice(&(init_rva as u64).to_le_bytes());
        context[32..36].copy_from_slice(&u32::MAX.to_le_bytes());

        remote_context = unsafe {
            winffi::VirtualAllocEx(
                hproc,
                0,
                context.len(),
                winffi::MEM_COMMIT | winffi::MEM_RESERVE,
                winffi::PAGE_READWRITE,
            )
        };
        if remote_context == 0 {
            return Err(InjectError::Win32 {
                op: "VirtualAllocEx(remote_context)",
                code: last_error(),
            });
        }
        if !write_remote(hproc, remote_context, context.as_ptr(), context.len()) {
            return Err(InjectError::Win32 {
                op: "WriteProcessMemory(remote_context)",
                code: last_error(),
            });
        }

        let stub = x64_loader_init_stub();
        remote_stub = unsafe {
            winffi::VirtualAllocEx(
                hproc,
                0,
                stub.len(),
                winffi::MEM_COMMIT | winffi::MEM_RESERVE,
                winffi::PAGE_EXECUTE_READWRITE,
            )
        };
        if remote_stub == 0 {
            return Err(InjectError::Win32 {
                op: "VirtualAllocEx(remote_stub)",
                code: last_error(),
            });
        }
        if !write_remote(hproc, remote_stub, stub.as_ptr(), stub.len()) {
            return Err(InjectError::Win32 {
                op: "WriteProcessMemory(remote_stub)",
                code: last_error(),
            });
        }

        remote_thread = unsafe {
            winffi::CreateRemoteThread(
                hproc,
                std::ptr::null(),
                0,
                remote_stub,
                remote_context,
                0,
                std::ptr::null_mut(),
            )
        };
        if remote_thread == 0 {
            return Err(InjectError::Win32 {
                op: "CreateRemoteThread(remote_stub)",
                code: last_error(),
            });
        }
        wait_infinite(remote_thread)?;

        let mut status_bytes = [0u8; 4];
        let mut read = 0usize;
        if unsafe {
            winffi::ReadProcessMemory(
                hproc,
                remote_context + 32,
                status_bytes.as_mut_ptr(),
                status_bytes.len(),
                &mut read,
            )
        } == 0
            || read != status_bytes.len()
        {
            return Err(InjectError::Win32 {
                op: "ReadProcessMemory(remote_status)",
                code: last_error(),
            });
        }
        let status = u32::from_le_bytes(status_bytes);
        if status == 0 {
            Ok(())
        } else if status == 0xffff_fffe {
            Err(InjectError::InvalidArgument(
                "remote bootstrap 中 LoadLibraryW 失败".into(),
            ))
        } else {
            Err(InjectError::InitFailed(status))
        }
    })();

    if remote_thread != 0 {
        unsafe { winffi::CloseHandle(remote_thread) };
    }
    if remote_stub != 0 {
        unsafe { winffi::VirtualFreeEx(hproc, remote_stub, 0, winffi::MEM_RELEASE) };
    }
    if remote_context != 0 {
        unsafe { winffi::VirtualFreeEx(hproc, remote_context, 0, winffi::MEM_RELEASE) };
    }
    unsafe { winffi::VirtualFreeEx(hproc, remote_base, 0, winffi::MEM_RELEASE) };
    result
}

/// Early-Bird APC 注入实验路径。
///
/// 远程线程注入依赖目标进程已经完成一部分初始化；此路径在主线程仍挂起时，
/// 把一个很小的 x64 bootstrap 放进目标进程并排入主线程 APC 队列。bootstrap
/// 依次调用 LoadLibraryW 和 gvhook_init，再把返回码写回远程上下文。这样 DLL
/// 的实际初始化仍在普通进程线程上下文中进行，同时避开首个 CreateRemoteThread。
///
/// 这是独立实验入口，当前只实现 x64。超时会终止仍处于实验状态的目标进程，
/// 确保释放远程代码前不会留下悬空 APC。
#[cfg(target_os = "windows")]
pub fn inject_into_process_early_bird(
    hproc: crate::winffi::HANDLE,
    hthread: crate::winffi::HANDLE,
    param_block: &[u8],
    hook_dll_path: &str,
) -> Result<(), InjectError> {
    #[cfg(not(target_arch = "x86_64"))]
    {
        let _ = (hproc, hthread, param_block, hook_dll_path);
        return Err(InjectError::InvalidArgument(
            "Early-Bird APC 实验路径当前仅支持 x64".into(),
        ));
    }

    #[cfg(target_arch = "x86_64")]
    {
        use crate::winffi;

        let dll = utf16_nul(hook_dll_path);
        let dll_bytes = units_to_bytes(&dll);
        let alloc_size = param_block.len() + dll_bytes.len();
        if alloc_size == 0 {
            return Err(InjectError::InvalidArgument(
                "Early-Bird 参数块和 DLL 路径不能为空".into(),
            ));
        }

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
                op: "VirtualAllocEx(param_block)",
                code: last_error(),
            });
        }

        let mut remote_context: winffi::LPVOID = 0;
        let mut remote_stub: winffi::LPVOID = 0;
        let result = (|| -> Result<(), InjectError> {
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

            // 由本进程加载同一 DLL，得到 gvhook_init 的导出 RVA；目标进程的
            // LoadLibraryW 返回 HMODULE 后，bootstrap 用同一 RVA 计算远程地址。
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
            let local_hmod = unsafe { winffi::LoadLibraryW(dll.as_ptr()) };
            if local_hmod == 0 {
                return Err(InjectError::Win32 {
                    op: "LoadLibraryW(local gvhook)",
                    code: last_error(),
                });
            }
            let local_fn = unsafe { winffi::GetProcAddress(local_hmod, b"gvhook_init\0".as_ptr()) };
            if local_fn == 0 || local_fn < local_hmod {
                return Err(InjectError::InvalidArgument(
                    "gvhook 未导出可计算 RVA 的 gvhook_init".into(),
                ));
            }
            let init_rva = local_fn - local_hmod;

            // [dll_addr, param_addr, LoadLibraryW, init_rva, status]，全部按
            // x64 usize/u32 的协议写入，status 初始为等待哨兵。
            let mut context = vec![0u8; 40];
            context[0..8].copy_from_slice(&(dll_addr as u64).to_le_bytes());
            context[8..16].copy_from_slice(&(remote_base as u64).to_le_bytes());
            context[16..24].copy_from_slice(&(loadlib as u64).to_le_bytes());
            context[24..32].copy_from_slice(&(init_rva as u64).to_le_bytes());
            context[32..36].copy_from_slice(&u32::MAX.to_le_bytes());

            remote_context = unsafe {
                winffi::VirtualAllocEx(
                    hproc,
                    0,
                    context.len(),
                    winffi::MEM_COMMIT | winffi::MEM_RESERVE,
                    winffi::PAGE_READWRITE,
                )
            };
            if remote_context == 0 {
                return Err(InjectError::Win32 {
                    op: "VirtualAllocEx(early_bird_context)",
                    code: last_error(),
                });
            }
            if !write_remote(
                hproc,
                remote_context,
                context.as_ptr(),
                context.len(),
            ) {
                return Err(InjectError::Win32 {
                    op: "WriteProcessMemory(early_bird_context)",
                    code: last_error(),
                });
            }

            // x64 Windows ABI callback:
            //   push r12; shadow space; LoadLibraryW(ctx->dll); call
            //   (hmod + ctx->init_rva)(ctx->param); store EAX; return.
            let stub = x64_loader_init_stub();
            remote_stub = unsafe {
                winffi::VirtualAllocEx(
                    hproc,
                    0,
                    stub.len(),
                    winffi::MEM_COMMIT | winffi::MEM_RESERVE,
                    winffi::PAGE_EXECUTE_READWRITE,
                )
            };
            if remote_stub == 0 {
                return Err(InjectError::Win32 {
                    op: "VirtualAllocEx(early_bird_stub)",
                    code: last_error(),
                });
            }
            if !write_remote(hproc, remote_stub, stub.as_ptr(), stub.len()) {
                return Err(InjectError::Win32 {
                    op: "WriteProcessMemory(early_bird_stub)",
                    code: last_error(),
                });
            }

            if unsafe { winffi::QueueUserAPC(remote_stub, hthread, remote_context) } == 0 {
                return Err(InjectError::Win32 {
                    op: "QueueUserAPC(early_bird_stub)",
                    code: last_error(),
                });
            }
            let prev = unsafe { winffi::ResumeThread(hthread) };
            if prev == u32::MAX {
                unsafe { winffi::TerminateProcess(hproc, 1) };
                let _ = wait_infinite(hproc);
                return Err(InjectError::Win32 {
                    op: "ResumeThread(early-bird)",
                    code: last_error(),
                });
            }

            // 等待 bootstrap 写回结果。只等待注入完成，不接管游戏进程的
            // 生命周期，调用方仍负责 Job Object / 进程树等待。
            let status_addr = remote_context + 32;
            for _ in 0..500 {
                let mut status_bytes = [0u8; 4];
                let mut read = 0usize;
                if unsafe {
                    winffi::ReadProcessMemory(
                        hproc,
                        status_addr,
                        status_bytes.as_mut_ptr(),
                        status_bytes.len(),
                        &mut read,
                    )
                } != 0
                    && read == status_bytes.len()
                {
                    let status = u32::from_le_bytes(status_bytes);
                    if status != u32::MAX {
                        if status == 0 {
                            return Ok(());
                        }
                        if status == 0xffff_fffe {
                            return Err(InjectError::InvalidArgument(
                                "Early-Bird APC 中 LoadLibraryW 失败".into(),
                            ));
                        }
                        return Err(InjectError::InitFailed(status));
                    }
                }
                unsafe { winffi::Sleep(10) };
            }

            unsafe { winffi::TerminateProcess(hproc, 1) };
            let _ = wait_infinite(hproc);
            Err(InjectError::InvalidArgument(
                "Early-Bird APC 未在 5 秒内完成".into(),
            ))
        })();

        if remote_stub != 0 {
            unsafe { winffi::VirtualFreeEx(hproc, remote_stub, 0, winffi::MEM_RELEASE) };
        }
        if remote_context != 0 {
            unsafe { winffi::VirtualFreeEx(hproc, remote_context, 0, winffi::MEM_RELEASE) };
        }
        unsafe { winffi::VirtualFreeEx(hproc, remote_base, 0, winffi::MEM_RELEASE) };
        result
    }
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
pub fn launch_suspended_with_args(
    _exe_path: &str,
    _work_dir: &str,
    _args: &str,
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
