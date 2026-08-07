//! 注册表路径重写 —— 归属：阶段 3 注册表重定向（W3T3 路径重写）。
//!
//! 职责（§3.5）：native 路径 `\REGISTRY\USER\<当前SID>\Software\X` →
//! `\REGISTRY\USER\GameVHD_<game_id>\Software\X`；仅匹配当前用户 SID；
//! `\REGISTRY\MACHINE`（HKLM）直通宿主（记录警告不隔离）；
//! 32 位游戏的 `Wow6432Node` 随 `Software` 子树自然重写。
//! 本波只声明桩；后续任务实现，勿改 main.rs / Cargo.toml。
#![allow(dead_code, unused_variables)]

/// 重写 HKCU native 路径；非当前用户 / HKLM 等不重写场景返回 None。
///
/// TODO(W3): HKCU path rewrite — 仅当前 SID 前缀命中。
pub fn rewrite_registry_path(native_path: &str, game_id: &str, user_sid: &str) -> Option<String> {
    todo!("W3: HKCU path rewrite")
}
