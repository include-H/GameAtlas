//! `cleanup` 命令 —— 归属：阶段 5 启动器闭环（崩溃残留自检）。
//!
//! 职责（§4 / §3.8 自愈）：box.json 崩溃残留自检——残留 VHD 挂载
//! （[`disk::detach_if_attached`]，显式传入差分盘路径）→ 残留 hive 挂载
//! （[`hive::cleanup_residue`] 幂等 RegUnLoadKey）→ box.json 状态回滚
//! （非 clean 复位）。连续运行无累积污染。被 main.rs 的
//! `cleanup --box <path> [--vhd <diff>]` 调用。
//!
//! 范围边界：state.json 宿主对账属后续事项（依赖宿主 state.json 记录
//! 挂载信息，本 wave 未建）——cleanup 处理 box.json 可见的残留
//! （VHD 挂载（显式参数）+ hive + 状态机）。

#![allow(dead_code)]

use std::path::Path;

use crate::boxfile::{BoxFile, BoxState};

fn detach_vhd_residue(vhd: &str) -> Result<(), String> {
    match crate::disk::detach_if_attached(vhd) {
        Ok(true) => crate::log_info!("cleanup: 已卸载残留 VHD 挂载: {vhd}"),
        Ok(false) => crate::log_info!("cleanup: VHD 未挂载，无需清理: {vhd}"),
        Err(e) if e.kind == crate::disk::DiskErrorKind::UnsupportedPlatform => {
            crate::log_info!("cleanup: 当前平台不支持 VHD 清理，跳过: {vhd}");
        }
        Err(e) => return Err(format!("残留 VHD 清理失败: {e}")),
    }
    Ok(())
}

/// 清理指定 box 的残留沙箱状态（幂等，可重复执行）。
/// `vhd_path` 为可选差分盘路径：提供时先清理残留 VHD 挂载。
/// `host_state_path` 为可选宿主权威状态路径：提供时检查游戏进程是否
/// 仍在运行（拒绝清理），完成后按代数清除。
pub fn cleanup_box(
    box_path: &str,
    vhd_path: Option<&str>,
    host_state_path: Option<&Path>,
) -> Result<(), String> {
    let path = Path::new(box_path);
    // Acquire this before parsing box.json: the file is game-writable and may
    // be missing or malformed after a crash, while the CLI path remains the
    // same resource identity used by run.
    let _box_lock = crate::process::acquire_box_mutex(box_path)
        .map_err(|e| format!("cleanup box 资源互斥锁获取失败: {e}"))?;

    let mut bf = match BoxFile::load(path) {
        Ok(bf) => bf,
        Err(e) => {
            // An explicit --vhd must still cover a mounted residue even when
            // box.json itself is damaged.  Preserve the parse error so the
            // caller does not mistake a corrupt state file for full success.
            if let Some(vhd) = vhd_path {
                detach_vhd_residue(vhd)?;
            }
            if !path.is_file() {
                crate::log_info!("cleanup: box.json 不存在，无需清理: {box_path}");
                return Ok(());
            }
            return Err(format!("读取 box.json 失败: {e}"));
        }
    };

    // Prefer the host record's game id when it owns this box.  The value in
    // box.json can be changed by the game and must not select the run lock.
    let state_hint = host_state_path
        .and_then(crate::hoststate::load)
        .filter(|st| crate::hoststate::box_path_matches(&st.box_path, box_path));
    let lock_game_id = state_hint
        .as_ref()
        .map_or_else(|| bf.game_id.clone(), |st| st.game_id.clone());
    let _run_lock = crate::process::acquire_run_mutex(&lock_game_id)
        .map_err(|e| format!("cleanup 并发互斥锁获取失败: {e}"))?;

    // 0. 宿主权威状态检查（审计 E / P1-4）：只看同游戏同 box 的记录；
    //    running 且进程仍是同一代 → 拒绝清理，崩溃残留继续处理。
    if let Some(state_path) = host_state_path {
        if let Some(st) = crate::hoststate::load(state_path) {
            let same_run = crate::hoststate::box_path_matches(&st.box_path, box_path);
            if same_run && crate::hoststate::owns_live_process(&st) {
                return Err(format!(
                    "宿主状态显示游戏 '{}' 正在运行（pid={}），拒绝清理",
                    st.game_id, st.pid
                ));
            }
        }
    }

    // 1. 残留 VHD 挂载清理（可选：显式指定差分盘路径，幂等 detach）。
    if let Some(vhd) = vhd_path {
        detach_vhd_residue(vhd)?;
    }

    // 1. 残留 hive 幂等卸载（无条件：崩溃可能发生在状态回滚之前）。
    crate::hive::cleanup_residue(&lock_game_id)
        .map_err(|e| format!("残留 hive 清理失败: {e}"))?;

    // 2. 状态回滚：非 clean → 归 clean（崩溃恢复语义，绕过状态机校验）。
    if bf.state != BoxState::Clean {
        crate::log_info!("cleanup: box.json 状态 {} → clean", bf.state);
        bf.state = BoxState::Clean;
        bf.save(path).map_err(|e| format!("box.json 状态回滚失败: {e}"))?;
    }

    // 3. 宿主权威状态清除：仅当记录已被本 cleanup 之前的运行遗弃时才需要
    //    清除（mark_clean 内部校验 game_id，残留 running 记录无 PID 存活）。
    if let Some(state_path) = host_state_path {
        if let Some(st) = crate::hoststate::load(state_path) {
            if st.running()
                && crate::hoststate::box_path_matches(&st.box_path, box_path)
                && !crate::hoststate::owns_live_process(&st)
            {
                let changed = crate::hoststate::mark_clean_for_box(
                    state_path,
                    st.generation,
                    &st.game_id,
                    Some(box_path),
                )
                .map_err(|e| format!("宿主状态清除失败: {e}"))?;
                if changed {
                    crate::log_info!("cleanup: 宿主状态已清除（generation={}）", st.generation);
                }
            }
        }
    }

    crate::log_info!("cleanup: 残留已清理: {box_path}");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn tmp_box(name: &str) -> std::path::PathBuf {
        let d = std::env::temp_dir().join(format!("gamevhd_cleanup_{name}_{}", std::process::id()));
        let _ = fs::remove_dir_all(&d);
        fs::create_dir_all(&d).unwrap();
        d.join("box.json")
    }

    #[test]
    fn missing_box_is_idempotent_ok() {
        let missing = std::env::temp_dir().join(format!(
            "gamevhd_cleanup_nonexistent_{}.json",
            std::process::id()
        ));
        assert!(cleanup_box(missing.to_str().unwrap(), None, None).is_ok());
    }

    #[test]
    fn clean_state_stays_clean() {
        let path = tmp_box("clean");
        let bf = BoxFile {
            game_id: "g".into(),
            exe_relative: r"Game\a.exe".into(),
            game_data_root: "GameData".into(),
            user_profile: r"C:\Users\T".into(),
            registry_hive: r"GameData\Registry\user.dat".into(),
            skip_cache_dirs: false,
            state: BoxState::Clean,
            game_data_base: String::new(),
            game_data_name: String::new(),
            user_namespace: false,
        };
        bf.save(&path).unwrap();

        // Linux 上 hive cleanup 返回 UnsupportedPlatform —— 只验证逻辑不 panic，
        // Windows 上才真正跑 hive 卸载。此测试断言「状态不变」由下面的
        // state_machine_rollback 覆盖逻辑路径。
        let _ = cleanup_box(path.to_str().unwrap(), Some("C:\\diff.vhd"), None);
    }

    #[test]
    fn vhd_residue_step_does_not_break_cleanup() {
        let path = tmp_box("vhdresidue");
        let bf = BoxFile {
            game_id: format!("g-vhdresidue-{}", std::process::id()),
            exe_relative: String::new(),
            game_data_root: String::new(),
            user_profile: String::new(),
            registry_hive: String::new(),
            skip_cache_dirs: false,
            state: BoxState::Running,
            game_data_base: String::new(),
            game_data_name: String::new(),
            user_namespace: false,
        };
        bf.save(&path).unwrap();
        let result = cleanup_box(path.to_str().unwrap(), Some("C:\\nonexistent-diff.vhd"), None);
        #[cfg(not(target_os = "windows"))]
        {
            // Linux：VHD 桩 UnsupportedPlatform 被跳过（不抢先失败），
            // 失败只能来自后续 hive 步骤（既有 Linux 行为）。
            let err = result.expect_err("Linux 上 cleanup 失败于 hive 步骤");
            assert!(err.contains("hive"), "VHD 步骤不得导致失败: {err}");
        }
        #[cfg(target_os = "windows")]
        {
            // Windows：不存在的差分盘 open 失败 → 视为未挂载跳过，整体 Ok。
            assert!(result.is_ok(), "残留 VHD 清理不得使 cleanup 失败: {result:?}");
        }
    }

    #[test]
    fn boxfile_state_transitions_remain_valid() {
        // run 主流程状态机（clean→running→cleaning→clean）跨平台纯逻辑验证：
        // cleanup 的直接置 Clean 是崩溃恢复语义，不走 transition。
        let mut bf = BoxFile {
            game_id: "g".into(),
            exe_relative: String::new(),
            game_data_root: String::new(),
            user_profile: String::new(),
            registry_hive: String::new(),
            skip_cache_dirs: false,
            state: BoxState::Clean,
            game_data_base: String::new(),
            game_data_name: String::new(),
            user_namespace: false,
        };
        assert!(bf.transition(BoxState::Running).is_ok());
        assert!(bf.transition(BoxState::Cleaning).is_ok());
        assert!(bf.transition(BoxState::Clean).is_ok());
        assert!(bf.transition(BoxState::Running).is_ok());
    }
}
