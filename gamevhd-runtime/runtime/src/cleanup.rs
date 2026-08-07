//! `cleanup` 命令 —— 归属：阶段 4 生命周期与崩溃恢复（W4T3 崩溃残留自检）。
//!
//! 职责（§4）：崩溃残留自检——残留 hive 挂载（幂等 `RegUnLoadKey`）→ 残留 VHD
//! 挂载（diskpart detach）→ box.json 状态回滚（非 clean 状态复位）；连续运行
//! 无累积污染。被 main.rs 的 `cleanup --box <path>` 调用。
//! 本波只声明桩；后续任务实现，勿改 main.rs / Cargo.toml。
#![allow(dead_code, unused_variables)]

/// 清理指定 box 的残留沙箱状态（幂等，可重复执行）。
///
/// TODO(W4): crash residue cleanup — hive/VHD 残留 + 状态回滚。
pub fn cleanup_box(box_path: &str) -> Result<(), String> {
    todo!("W4: crash residue cleanup")
}
