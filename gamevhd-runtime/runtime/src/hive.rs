//! 注册表 hive 管理 —— 归属：阶段 3 注册表重定向（W3T1 模板复制 / W3T2 挂载卸载）。
//!
//! 职责（§3.5）：模板 hive（随包分发）首次复制为 `GameData\Registry\user.dat` 并轮换 `.bak`；
//! `RegLoadKey(HKUS, "GameVHD_<game_id>", ...)` 挂载 / `RegUnLoadKey` 卸载（含句柄释放
//! 等待重试与崩溃残留清理——启动时先幂等卸载）；hive 损坏时从 `.bak` 恢复并警告。
//! 本波只声明桩；后续任务实现，勿改 main.rs / Cargo.toml。
#![allow(dead_code, unused_variables)]

/// 挂载隔离 hive：`RegLoadKey(HKUS, "GameVHD_<game_id>", hive_path)`。
///
/// TODO(W3): RegLoadKey + 模板复制/轮换。
pub fn mount_hive(game_id: &str, hive_path: &str) -> Result<(), String> {
    todo!("W3: RegLoadKey + template copy")
}

/// 卸载隔离 hive：`RegUnLoadKey`，句柄未释放时等待重试。
///
/// TODO(W3): RegUnLoadKey with retry + 崩溃残留清理。
pub fn unmount_hive(game_id: &str) -> Result<(), String> {
    todo!("W3: RegUnLoadKey with retry")
}
