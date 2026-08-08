//! `run` 命令完整流程 —— 归属：阶段 5 启动器闭环（run 主流程）。
//!
//! 职责（§3.8 / §3.5 卸载顺序）：读取自身 footer manifest → 读 box.json →
//! 状态机 clean→running → 挂载 hive → 扫描/选择 exe → 构建参数块 →
//! 注入启动 → Job Object 等待整棵进程树退出 → 清理链（hive 卸载 +
//! 状态机回 clean）。被 main.rs 的 `run --drive <letter> --box <path>` 调用。
//!
//! 职责边界：run **假定磁盘层已由 `mount` 完成**（VHD 挂载到 `<drive>:`），
//! 磁盘卸载由 `unmount` 负责——CLI 分离 mount/run/unmount，run 只负责
//! 沙箱运行生命周期与 hive 清理。
//!
//! RAII 清理：中途任何错误经 [`win::RunGuard`] Drop 兜底——卸载 hive +
//! box.json 状态回滚 clean（崩溃恢复语义，绕过状态机校验）；Job Object
//! `KILL_ON_JOB_CLOSE` 保证 job 句柄关闭即强杀残留进程树（含注入失败时
//! 仍挂起的游戏主线程）。
//!
//! 跨平台：纯逻辑（路径拼接 / exe 选择 / hook 选择 / 日志路径）Linux 可测；
//! Windows 专属编排在 `#[cfg(target_os = "windows")]` 的 `win` 子模块。

#![allow(dead_code)]

use std::path::{Path, PathBuf};

use crate::scan;

/// 执行完整沙箱运行生命周期；返回时游戏已退出且清理完毕。
pub fn run_game(drive_letter: char, box_path: &str) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    {
        win::run_game_impl(drive_letter, box_path)
    }
    #[cfg(not(target_os = "windows"))]
    {
        let _ = (drive_letter, box_path);
        Err("run 需要 Windows（本二进制在 Linux 上仅用于测试）".into())
    }
}

// ---- 跨平台纯逻辑（Linux 可测；Windows 由 win 子模块消费） ----

/// 卷根：`E` → `E:\`。
fn volume_root_of(letter: char) -> String {
    format!("{}:\\", letter.to_ascii_uppercase())
}

/// Windows 路径是否已经是绝对路径（盘符或 UNC）。
fn is_windows_absolute_path(path: &str) -> bool {
    let trimmed = path.trim();
    let mut chars = trimmed.chars();
    match (chars.next(), chars.next(), chars.next()) {
        (Some(first), Some(':'), Some('\\' | '/')) if first.is_ascii_alphabetic() => true,
        _ => trimmed.starts_with(r"\\") || trimmed.starts_with("//"),
    }
}

/// box.json 相对卷根路径 → 绝对路径；已含盘符/UNC 前缀则原样返回。
/// 相对形式如 `GameData` / `GameData\Registry\user.dat`（v2 §3.9 路径相对卷根）。
fn abs_path(volume_root: &str, rel_or_abs: &str) -> String {
    let t = rel_or_abs.trim();
    if is_windows_absolute_path(t) {
        t.to_string()
    } else {
        let root = volume_root.trim_end_matches(['\\', '/']);
        format!("{root}\\{}", t.trim_start_matches(['\\', '/']))
    }
}

/// Windows 风格路径拼接。该模块在 Linux 上测试 Windows 字符串，不能使用
/// `Path::join`，否则会按宿主机的 `/` 规则处理。
fn join_windows_path(base: &str, child: &str) -> String {
    let base = base.trim().trim_end_matches(['\\', '/']);
    let child = child.trim().trim_start_matches(['\\', '/']);
    if child.is_empty() {
        base.to_string()
    } else {
        format!("{base}\\{child}")
    }
}

/// 取 Windows 路径最后一段，并去掉最后一个扩展名。
fn path_stem(path: &str) -> Option<String> {
    let leaf = path
        .trim()
        .trim_end_matches(['\\', '/'])
        .rsplit(['\\', '/'])
        .next()?;
    let stem = leaf.rsplit_once('.').map_or(leaf, |(prefix, _)| prefix).trim();
    (!stem.is_empty()).then(|| stem.to_string())
}

/// 外部状态目录必须是单层目录名，避免配置把状态库逃逸到父目录之外。
fn validate_game_data_name(name: &str) -> Result<String, String> {
    let value = name.trim();
    if value.is_empty() || value == "." || value == ".." {
        return Err("game_data_name 不能为空或为 . / ..".into());
    }
    if value.ends_with('.') || value.ends_with(' ') {
        return Err(format!("game_data_name '{value}' 不能以点或空格结尾"));
    }
    if value.chars().any(|ch| {
        ch.is_control() || matches!(ch, '\\' | '/' | ':' | '*' | '?' | '"' | '<' | '>' | '|')
    }) {
        return Err(format!("game_data_name '{value}' 含非法 Windows 文件名字符"));
    }
    Ok(value.to_string())
}

/// 解析最终 GameData 根：显式 root > 外部父目录/游戏名 > VHD 内默认目录。
fn resolve_game_data_root(
    volume_root: &str,
    configured_root: &str,
    external_base: &str,
    configured_name: &str,
    fallback_name: &str,
) -> Result<String, String> {
    if !configured_root.trim().is_empty() {
        return Ok(abs_path(volume_root, configured_root));
    }
    if external_base.trim().is_empty() {
        return Ok(join_windows_path(volume_root, "GameData"));
    }

    let name = if configured_name.trim().is_empty() {
        fallback_name
    } else {
        configured_name
    };
    let name = validate_game_data_name(name)?;
    Ok(join_windows_path(&abs_path(volume_root, external_base), &name))
}

/// hive 未单独指定时，跟随最终 GameData 根，确保外部状态库完整可同步。
fn resolve_registry_hive(volume_root: &str, configured_hive: &str, game_data_root: &str) -> String {
    if configured_hive.trim().is_empty() {
        join_windows_path(game_data_root, r"Registry\user.dat")
    } else {
        abs_path(volume_root, configured_hive)
    }
}

/// 从基础 VHD 文件名推导外部状态目录名；失败时回退到稳定的 game_id。
fn derive_game_data_name(base_vhd: &str, game_id: &str) -> String {
    path_stem(base_vhd).unwrap_or_else(|| game_id.to_string())
}

/// exe 选择（v2 §3.7）：记忆命中 → exe_hint 命中 → 排序第一
/// （`collect_exes` 已按深度浅优先 + 体积大优先排序）。
fn pick_exe<'a>(
    candidates: &'a [PathBuf],
    volume_root: &Path,
    remembered: &str,
    exe_hint: Option<&str>,
) -> Option<&'a PathBuf> {
    if let Some(hit) = scan::resolve_remembered(candidates, volume_root, remembered) {
        return Some(hit);
    }
    if let Some(hint) = exe_hint {
        if let Some(hit) = candidates
            .iter()
            .find(|p| scan::to_relative(volume_root, p) == hint)
        {
            return Some(hit);
        }
    }
    candidates.first()
}

/// 按 exe 位数选 hook dll（启动器同目录 `gvhook-x64.dll` / `gvhook-x86.dll`）。
/// 位数不变式（协议 §6.3）：编排与 hook 同位数。
fn hook_dll_for(bits: &str, launcher_dir: &Path) -> Option<String> {
    let file = match bits {
        "x64" => "gvhook-x64.dll",
        "x86" => "gvhook-x86.dll",
        _ => return None,
    };
    Some(launcher_dir.join(file).to_string_lossy().into_owned())
}

/// 会话日志路径：`%LOCALAPPDATA%\GameAtlas\logs\<game_id>.log`（v2 §3.8）。
fn log_path_for(game_id: &str, local_app_data: &str) -> String {
    let base = local_app_data.trim_end_matches(['\\', '/']);
    format!("{base}\\GameAtlas\\logs\\{game_id}.log")
}

#[cfg(target_os = "windows")]
mod win {
    use super::*;
    use crate::boxfile::{BoxFile, BoxState};
    use crate::hive;
    use crate::inject;
    use crate::manifest::{load_manifest_file, Manifest};
    use crate::pe;
    use crate::process;
    use crate::rules;
    use crate::winffi;

    /// run 清理守卫：Drop 时兜底卸载 hive + box.json 状态回滚 clean。
    /// 成功路径显式 [`RunGuard::detach_hive`] 后 hive 为 None，Drop 幂等。
    struct RunGuard {
        box_path: PathBuf,
        hive: Option<hive::Hive>,
    }

    impl RunGuard {
        fn detach_hive(&mut self) -> Result<(), String> {
            match self.hive.take() {
                Some(h) => h.unmount_hive().map_err(|e| e.to_string()),
                None => Ok(()),
            }
        }
    }

    impl Drop for RunGuard {
        fn drop(&mut self) {
            if let Err(e) = self.detach_hive() {
                crate::log_warn!("run 清理：hive 卸载失败（残留可 cleanup 处理）: {e}");
            }
            // 崩溃恢复语义：无论当前 state，直接归 clean（绕过状态机校验）。
            if let Ok(mut bf) = BoxFile::load(&self.box_path) {
                bf.state = BoxState::Clean;
                if let Err(e) = bf.save(&self.box_path) {
                    crate::log_warn!("run 清理：box.json 状态回滚失败: {e}");
                }
            }
        }
    }

    pub(super) fn run_game_impl(drive_letter: char, box_path: &str) -> Result<(), String> {
        // 1. 读取自身 footer manifest（卡带配置）。
        let launcher = std::env::current_exe().map_err(|e| format!("无法定位启动器自身路径: {e}"))?;
        let manifest: Manifest = load_manifest_file(&launcher)
            .map_err(|e| format!("启动器无卡带配置（非《title》.exe 形态）: {e}"))?;
        let launcher_dir = launcher.parent().unwrap_or(Path::new(".")).to_path_buf();

        // 2. 读取 box.json（VHD 内 `GameData\box.json`）。
        let box_path_buf = PathBuf::from(box_path);
        let mut bf = BoxFile::load(&box_path_buf).map_err(|e| format!("读取 box.json 失败: {e}"))?;
        if bf.state != BoxState::Clean {
            return Err(format!(
                "box.json 状态为 {}（上轮未干净退出，请先执行 cleanup --box）",
                bf.state
            ));
        }
        if bf.game_id != manifest.game_id {
            return Err(format!(
                "box.json game_id '{}' 与启动器配置 '{}' 不一致（卡带错配？）",
                bf.game_id, manifest.game_id
            ));
        }

        // 3. 状态机 clean → running（先行落盘：run 中断后 cleanup 可识别）。
        bf.transition(BoxState::Running).map_err(|e| e.to_string())?;
        bf.save(&box_path_buf).map_err(|e| format!("box.json 写 running 失败: {e}"))?;

        let volume_root = volume_root_of(drive_letter);
        let fallback_data_name = derive_game_data_name(&manifest.base_vhd, &bf.game_id);
        let game_data_root = resolve_game_data_root(
            &volume_root,
            &bf.game_data_root,
            &bf.game_data_base,
            &bf.game_data_name,
            &fallback_data_name,
        )?;
        let registry_hive_path = resolve_registry_hive(
            &volume_root,
            &bf.registry_hive,
            &game_data_root,
        );
        crate::log_info!(
            "run: GameData root={} registry_hive={}",
            game_data_root,
            registry_hive_path
        );

        // RAII 清理守卫：此后任何 Err 提前返回都走 Drop 兜底。
        let mut guard = RunGuard {
            box_path: box_path_buf.clone(),
            hive: None,
        };

        // 4. 挂载隔离 hive（RegLoadKeyW；损坏自动 .bak 恢复）。
        let h = hive::mount_hive(&bf.game_id, &registry_hive_path)
            .map_err(|e| format!("hive 挂载失败: {e}"))?;
        guard.hive = Some(h);

        // 5. 扫描卷内 exe 并选择（记忆 → hint → 排序第一）。
        let root = PathBuf::from(&volume_root);
        let candidates = scan::collect_exes(&root, scan::MAX_DEPTH);
        let chosen = pick_exe(&candidates, &root, &bf.exe_relative, manifest.exe_hint.as_deref())
            .ok_or_else(|| format!("卷 {} 下未找到候选 exe", volume_root))?;
        let exe_abs = chosen.clone();
        let rel = scan::to_relative(&root, &exe_abs);
        if rel != bf.exe_relative {
            crate::log_info!("exe 记忆更新：{} → {}", bf.exe_relative, rel);
            bf.exe_relative = rel;
            bf.save(&box_path_buf)
                .map_err(|e| format!("box.json 写 exe 记忆失败: {e}"))?;
        }

        // 6. exe 位数 → 同目录 hook dll（协议 §6.3 位数不变式）。
        let bits = pe::probe_file(&exe_abs).map_err(|e| format!("exe 探测失败: {e}"))?;
        let hook_dll = hook_dll_for(bits.as_str(), &launcher_dir)
            .ok_or_else(|| format!("exe 位数 '{}' 无对应 hook", bits.as_str()))?;
        if !Path::new(&hook_dll).is_file() {
            return Err(format!("缺少 hook dll：{hook_dll}（需与启动器同目录）"));
        }

        // 7. 会话日志：创建目录 + 清空（hook 追加写，编排负责清空，协议 §7）。
        let local_app_data = std::env::var("LOCALAPPDATA")
            .map_err(|_| "缺少 LOCALAPPDATA 环境变量（无法定位日志目录）".to_string())?;
        let log_path = log_path_for(&manifest.game_id, &local_app_data);
        if let Some(dir) = Path::new(&log_path).parent() {
            std::fs::create_dir_all(dir).map_err(|e| format!("创建日志目录失败: {e}"))?;
        }
        std::fs::write(&log_path, b"").map_err(|e| format!("清空会话日志失败: {e}"))?;

        // 8. 重写规则表 + 注入参数块（5280 + n×4104 字节）。
        let rule_table = rules::generate_rules(&bf.user_profile, &game_data_root, bf.skip_cache_dirs);
        let param_block = rules::param_block_with(
            &hook_dll,
            &game_data_root,
            &bf.user_profile,
            &log_path,
            &registry_hive_path,
            &bf.game_id,
            &rule_table,
        );

        // 9. 命名 Job Object（KILL_ON_JOB_CLOSE：句柄关闭强杀整树）。
        let job = process::create_kill_on_close_job(&bf.game_id)
            .map_err(|e| format!("创建 Job Object 失败: {e}"))?;

        // 10. 挂起启动 → 划入 job → 注入 hook → resume → 等待整棵树退出。
        let work_dir = exe_abs
            .parent()
            .map(|p| p.to_string_lossy().into_owned())
            .unwrap_or_default();
        let (hproc, hthread) =
            inject::launch_suspended(&exe_abs.to_string_lossy(), &work_dir)
                .map_err(|e| format!("启动游戏进程失败: {e}"))?;

        let run_result = (|| -> Result<(), String> {
            process::assign_to_job(job, hproc).map_err(|e| format!("进程划入 Job 失败: {e}"))?;
            if std::env::var("GAMEVHD_INJECT_MODE").as_deref() == Ok("early-bird") {
                crate::log_info!("run: 使用 Early-Bird APC 注入实验路径");
                inject::inject_into_process_early_bird(hproc, hthread, &param_block, &hook_dll)
                    .map_err(|e| format!("gvhook Early-Bird APC 注入失败: {e}"))?;
            } else {
                inject::inject_into_process(hproc, &param_block, &hook_dll)
                    .map_err(|e| format!("gvhook 注入失败: {e}"))?;
                let prev = unsafe { winffi::ResumeThread(hthread) };
                if prev == u32::MAX {
                    return Err("ResumeThread 失败".into());
                }
            }
            process::wait_process_tree(job).map_err(|e| format!("等待进程树退出失败: {e}"))?;
            Ok(())
        })();

        // 关闭句柄：KILL_ON_JOB_CLOSE 语义下，若注入失败（进程仍挂起），
        // 关 job 即强杀残留进程树——无需手动 TerminateProcess。
        unsafe { winffi::CloseHandle(hthread) };
        unsafe { winffi::CloseHandle(hproc) };
        unsafe { winffi::CloseHandle(job) };
        run_result?;

        // 11. 清理链（§3.5 卸载顺序）：进程树全退 → cleaning → hive 卸载 → clean。
        bf.transition(BoxState::Cleaning).map_err(|e| e.to_string())?;
        bf.save(&box_path_buf).map_err(|e| format!("box.json 写 cleaning 失败: {e}"))?;
        guard.detach_hive().map_err(|e| format!("hive 卸载失败: {e}"))?;
        bf.transition(BoxState::Clean).map_err(|e| e.to_string())?;
        bf.save(&box_path_buf).map_err(|e| format!("box.json 写 clean 失败: {e}"))?;

        crate::log_info!("run: 游戏已退出，沙箱清理完毕（{}）", manifest.title);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    #[test]
    fn volume_root_uppercases_letter() {
        assert_eq!(volume_root_of('e'), r"E:\");
        assert_eq!(volume_root_of('G'), r"G:\");
    }

    #[test]
    fn abs_path_joins_relative_and_passes_absolute() {
        assert_eq!(abs_path(r"E:\", "GameData"), r"E:\GameData");
        assert_eq!(
            abs_path(r"E:\", r"GameData\Registry\user.dat"),
            r"E:\GameData\Registry\user.dat"
        );
        assert_eq!(abs_path(r"E:\", "D:\\x"), "D:\\x");
        assert_eq!(abs_path(r"E:\", "/GameData"), r"E:\GameData");
        assert_eq!(abs_path(r"E:\", r"\\nas\share\GameData"), r"\\nas\share\GameData");
        assert_eq!(abs_path(r"E:", "GameData"), r"E:\GameData");
    }

    #[test]
    fn external_data_root_appends_vhd_stem() {
        assert_eq!(
            resolve_game_data_root(r"G:\", "", r"D:\GameAtlas", "", "地平线").unwrap(),
            r"D:\GameAtlas\地平线"
        );
        assert_eq!(
            resolve_game_data_root(
                r"G:\",
                "",
                r"D:\GameAtlas",
                "Horizon",
                "地平线"
            )
            .unwrap(),
            r"D:\GameAtlas\Horizon"
        );
        assert_eq!(
            resolve_game_data_root(r"G:\", r"G:\GameData", r"D:\GameAtlas", "地平线", "x")
                .unwrap(),
            r"G:\GameData"
        );
    }

    #[test]
    fn data_root_name_is_derived_and_validated() {
        assert_eq!(derive_game_data_name(r"\\nas\share\地平线.vhd", "fallback"), "地平线");
        assert_eq!(derive_game_data_name("", "horizon-zero-dawn"), "horizon-zero-dawn");
        assert!(validate_game_data_name(r"bad\name").is_err());
        assert!(validate_game_data_name("地平线").is_ok());
    }

    #[test]
    fn registry_hive_follows_external_root_when_unspecified() {
        assert_eq!(
            resolve_registry_hive(r"G:\", "", r"D:\GameAtlas\地平线"),
            r"D:\GameAtlas\地平线\Registry\user.dat"
        );
        assert_eq!(
            resolve_registry_hive(r"G:\", r"GameData\Registry\user.dat", r"D:\GameAtlas\地平线"),
            r"G:\GameData\Registry\user.dat"
        );
    }

    #[test]
    fn pick_exe_prefers_memory_then_hint_then_first() {
        let d = std::env::temp_dir().join(format!("gamevhd_run_pick_{}", std::process::id()));
        let _ = fs::remove_dir_all(&d);
        fs::create_dir_all(d.join("Game")).unwrap();
        fs::write(d.join("Game/a.exe"), b"MZ").unwrap();
        fs::write(d.join("Game/b.exe"), b"MZ").unwrap();
        fs::write(d.join("c.exe"), vec![b'M'; 300]).unwrap();
        let candidates = scan::collect_exes(&d, scan::MAX_DEPTH);

        // 记忆命中优先于 hint。
        let hit = pick_exe(&candidates, &d, r"Game\b.exe", Some(r"Game\a.exe")).unwrap();
        assert_eq!(hit.file_name().unwrap().to_str(), Some("b.exe"));

        // 记忆失效 → hint 命中。
        let hit = pick_exe(&candidates, &d, r"Game\gone.exe", Some(r"Game\a.exe")).unwrap();
        assert_eq!(hit.file_name().unwrap().to_str(), Some("a.exe"));

        // 记忆 + hint 均失效 → 排序第一（c.exe 深度 0 优先）。
        let hit = pick_exe(&candidates, &d, r"Game\gone.exe", None).unwrap();
        assert_eq!(hit.file_name().unwrap().to_str(), Some("c.exe"));

        // 空候选 → None。
        assert!(pick_exe(&[], &d, "", None).is_none());

        let _ = fs::remove_dir_all(&d);
    }

    #[test]
    fn hook_dll_matches_bitness() {
        let dir = Path::new(".");
        let dll64 = hook_dll_for("x64", dir).unwrap();
        assert!(
            dll64.ends_with("gvhook-x64.dll"),
            "x64 → gvhook-x64.dll，收到 {dll64}"
        );
        let dll86 = hook_dll_for("x86", dir).unwrap();
        assert!(
            dll86.ends_with("gvhook-x86.dll"),
            "x86 → gvhook-x86.dll，收到 {dll86}"
        );
        assert!(hook_dll_for("not-pe", dir).is_none());
        assert!(hook_dll_for("", dir).is_none());
    }

    #[test]
    fn log_path_joins_gameatlas_logs() {
        assert_eq!(
            log_path_for("horizon", r"C:\Users\Hao\AppData\Local"),
            r"C:\Users\Hao\AppData\Local\GameAtlas\logs\horizon.log"
        );
        assert_eq!(
            log_path_for("g", r"C:\Users\Hao\AppData\Local\"),
            r"C:\Users\Hao\AppData\Local\GameAtlas\logs\g.log"
        );
    }
}
