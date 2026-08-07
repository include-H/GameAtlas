//! 注入三件套 —— 归属：阶段 1 注入与 hook 骨架（W1T3 编排注入）。
//!
//! 职责（§3.1）：`CreateProcessW(CREATE_SUSPENDED)` → `VirtualAllocEx` 写参数块
//! （hook DLL 路径 + 重写规则表）→ `CreateRemoteThread(LoadLibraryW)` → `ResumeThread`。
//! 编排按游戏 exe 位数（[`crate::pe::probe`]）选择自身位数；子进程由 gvhook 内
//! `NtCreateUserProcess` hook 递归注入。
//! 本波只声明桩；后续任务实现，勿改 main.rs / Cargo.toml。
#![allow(dead_code, unused_variables)]

/// 向已挂起的游戏进程注入 hook DLL 并恢复执行。
///
/// TODO(W1): inject into suspended process — 三件套注入 + 参数块编码。
pub fn inject_into_process(process_id: u32, hook_dll_path: &str, rules: &[String]) -> Result<(), String> {
    todo!("W1: inject hook DLL into suspended game process")
}
