//! 进程树等待/枚举 —— 归属：阶段 4 生命周期与崩溃恢复（W4T1 进程树等待）。
//!
//! 职责（§4）：`run` 结束后等待 root 游戏进程及其全部子进程退出（Toolhelp 快照枚举
//! 子进程），全部退出后才允许卸载 hive / 卸载 VHD；强杀场景配合超时与重试。
//! 本波只声明桩；后续任务实现，勿改 main.rs / Cargo.toml。
#![allow(dead_code, unused_variables)]

/// 等待 root 进程及其整棵进程树退出（阻塞）。
///
/// TODO(W4): wait process tree exit — 轮询子进程集合直至为空。
pub fn wait_process_tree(root_pid: u32) -> Result<(), String> {
    todo!("W4: wait process tree exit")
}

/// 枚举指定进程的全部子孙进程 PID。
///
/// TODO(W4): enumerate process tree — Toolhelp32 快照递归收集。
pub fn enumerate_children(root_pid: u32) -> Vec<u32> {
    todo!("W4: enumerate child processes")
}
