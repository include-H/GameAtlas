//! `run` 命令完整流程 —— 归属：阶段 4 生命周期与崩溃恢复（W4T2 run 闭环）。
//!
//! 职责（§4、§3.5 卸载顺序）：挂载 hive → 扫描确认 exe → 注入启动游戏 →
//! 进程树等待（root + 全部子进程退出）→ 卸载 hive（句柄重试）→ box.json 状态
//! 落盘（clean）。被 main.rs 的 `run --drive <letter> --box <path>` 调用。
//! 本波只声明桩；后续任务实现，勿改 main.rs / Cargo.toml。
#![allow(dead_code, unused_variables)]

/// 执行完整沙箱运行生命周期；返回时游戏已退出且清理完毕。
///
/// TODO(W4): full run lifecycle — 见模块注释中的顺序。
pub fn run_game(drive_letter: char, box_path: &str) -> Result<(), String> {
    todo!("W4: full run lifecycle")
}
