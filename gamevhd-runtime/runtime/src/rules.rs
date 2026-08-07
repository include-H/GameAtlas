//! 重写规则表生成/持久化 —— 归属：阶段 2 文件系统重定向（W2T7 规则表生成）。
//!
//! 职责（§3.3）：解析当前 `USERPROFILE`，生成文件路径重写规则表
//! （`%USERPROFILE%\Documents` / `AppData` / `Saved Games` → `GameDataRoot\Users\<u>\...`），
//! 供注入参数块传给 gvhook；规则表随 box.json 持久化，跨机器按新 USERPROFILE 重建。
//! 本波只声明桩；后续任务实现，勿改 main.rs / Cargo.toml。
#![allow(dead_code, unused_variables)]

/// 生成重写规则表：返回 (前缀, 替换前缀) 对列表。
///
/// TODO(W2T7): rules table generation — 解析 user_profile，输出 Documents/AppData/Saved Games 映射。
pub fn generate_rules(user_profile: &str, game_data_root: &str) -> Vec<(String, String)> {
    todo!("W2T7: rules table generation")
}
