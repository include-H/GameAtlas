//! Windows 原生 API 薄封装 —— 归属：各阶段共享（W0T2 diskpart / W1 / W3 原生调用）。
//!
//! 职责：diskpart 挂载/卸载（阶段 0）、`CreateProcessW` / `VirtualAllocEx` /
//! `CreateRemoteThread`（阶段 1，inject.rs 的底层）、`RegLoadKey` / `RegUnLoadKey`
//! 与模板 hive（阶段 3，hive.rs 的底层）、进程快照（阶段 4，process.rs 的底层）。
//! 全部 `#[cfg(target_os = "windows")]` 语义由调用方保证；本模块函数在非 Windows
//! 上不得被调用（main.rs 已按平台分发）。
//! 本波只声明桩；后续任务实现，勿改 main.rs / Cargo.toml。
#![allow(dead_code, unused_variables)]

/// 通过 diskpart 挂载 VHD，等待盘符出现。
///
/// TODO(W0): diskpart mount — 生成脚本、执行、校验结果。
pub fn diskpart_mount(vhd_path: &str) -> Result<(), String> {
    todo!("W0: diskpart mount")
}

/// 通过 diskpart 卸载 VHD（幂等）。
///
/// TODO(W0): diskpart unmount — 生成脚本、执行、校验结果。
pub fn diskpart_unmount(vhd_path: &str) -> Result<(), String> {
    todo!("W0: diskpart unmount")
}
