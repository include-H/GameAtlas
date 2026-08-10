//! 宿主权威运行状态 `state.json`（审计 E 类，P1-4）。
//!
//! box.json 位于 VHD 差分盘（游戏可写区），其状态字段可被游戏篡改；
//! 本模块在宿主受保护位置（`%LOCALAPPDATA%\GameAtlas\state.json`）记录
//! 权威运行状态，并携带单调递增的 `generation`：任何清理动作只作用于
//! 自己启动的那一代，防止旧实例/旧清理误杀新运行实例。
//!
//! 跨平台：文件读写/解析为纯逻辑（Linux 可测）；PID 存活检查
//! Windows 用 OpenProcess+STILL_ACTIVE，Linux 用 /proc。

#![allow(dead_code)]

use std::fs;
use std::path::Path;

use crate::json::{escape_json, parse_json_object};

/// state.json 的默认文件名（宿主 `%LOCALAPPDATA%\GameAtlas\` 下）。
pub const HOST_STATE_FILE_NAME: &str = "state.json";

/// Put each game's authority record in its own file.  The game id is hashed
/// instead of used as a path component, so a tampered box cannot escape the
/// host state directory or make two ids alias through separators/case.
pub fn state_path_for_game(local_app_data: &Path, game_id: &str) -> std::path::PathBuf {
    local_app_data
        .join("GameAtlas")
        .join("state")
        .join(format!("state_{:016x}.json", fnv1a64(game_id.as_bytes())))
}

/// Normalize a box path for authority comparisons and resource locking.  An
/// existing path is canonicalized so relative/absolute spellings and symlink
/// aliases cannot select different locks; the fallback still normalizes slash
/// direction when the file has already disappeared.
pub fn canonical_box_path(path: &str) -> String {
    let normalized = fs::canonicalize(Path::new(path))
        .map(|p| p.to_string_lossy().into_owned())
        .unwrap_or_else(|_| path.to_string())
        .replace('/', "\\");
    #[cfg(target_os = "windows")]
    {
        normalized.to_ascii_lowercase()
    }
    #[cfg(not(target_os = "windows"))]
    {
        normalized
    }
}

fn fnv1a64(bytes: &[u8]) -> u64 {
    let mut hash = 0xcbf29ce484222325u64;
    for byte in bytes {
        hash ^= u64::from(*byte);
        hash = hash.wrapping_mul(0x100000001b3);
    }
    hash
}

/// 宿主权威运行状态。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct HostState {
    /// 单调递增的运行代数。每次 begin_run 取旧记录 +1。
    pub generation: u64,
    pub game_id: String,
    pub box_path: String,
    /// "running" / "clean"。clean 表示无活动运行。
    pub state: String,
    pub pid: u32,
    /// 开始时间（Unix 秒）。
    pub started_at: u64,
    /// OS-specific process creation identity.  Zero means an old state file
    /// without this field; such a live PID is treated conservatively as busy.
    pub process_identity: u64,
}

impl HostState {
    pub fn running(&self) -> bool {
        self.state == "running"
    }
}

/// 读取宿主状态；文件不存在或解析失败返回 None（视为无历史记录）。
pub fn load(path: &Path) -> Option<HostState> {
    let s = fs::read_to_string(path).ok()?;
    from_json(&s).ok()
}

/// 开始一次运行：读取旧记录 → 若同 game_id 且仍在运行（PID 存活）则拒绝；
/// 否则 generation+1 写入 running。返回写入后的状态（调用方持有 generation
/// 用于结束时的 mark_clean）。
pub fn begin_run(path: &Path, game_id: &str, box_path: &str) -> Result<HostState, String> {
    let prior = load(path);
    if let Some(p) = &prior {
        if p.running() && p.game_id == game_id && pid_is_same_live_process(p) {
            return Err(format!(
                "宿主状态显示游戏 '{game_id}' 已在运行（pid={}，generation={}）",
                p.pid, p.generation
            ));
        }
    }
    let state = HostState {
        generation: prior.map_or(0, |p| p.generation).saturating_add(1),
        game_id: game_id.to_string(),
        box_path: canonical_box_path(box_path),
        state: "running".to_string(),
        pid: std::process::id(),
        started_at: now_secs(),
        process_identity: process_identity(std::process::id()).unwrap_or(0),
    };
    save(path, &state).map_err(|e| format!("写宿主状态失败: {e}"))?;
    Ok(state)
}

/// 结束一次运行：仅当现有记录的 generation 与 game_id 都匹配时才清除
/// （置 clean 并保留记录）。不匹配说明记录已被更新的运行接管，跳过。
pub fn mark_clean(path: &Path, generation: u64, game_id: &str) -> Result<(), String> {
    mark_clean_for_box(path, generation, game_id, None).map(|_| ())
}

/// Mark a generation clean only when it still owns the supplied box.  The
/// boolean is false for a newer generation or a different game/box.
pub fn mark_clean_for_box(
    path: &Path,
    generation: u64,
    game_id: &str,
    box_path: Option<&str>,
) -> Result<bool, String> {
    let Some(mut s) = load(path) else {
        return Ok(false);
    };
    if s.generation != generation
        || s.game_id != game_id
        || box_path.is_some_and(|p| !box_path_matches(&s.box_path, p))
    {
        return Ok(false);
    }
    s.state = "clean".to_string();
    save(path, &s).map_err(|e| format!("写宿主状态失败: {e}"))?;
    Ok(true)
}

/// Does the record still describe the same live process, rather than a PID
/// that has since been reused by an unrelated process?
pub fn owns_live_process(state: &HostState) -> bool {
    state.running() && pid_is_same_live_process(state)
}

/// Find the authority record that owns a box path.  The game id in box.json
/// is game-writable, so cleanup must be able to recover the state file without
/// deriving its filename from that untrusted value.
pub fn find_state_for_box(
    local_app_data: &Path,
    box_path: &str,
) -> Option<(std::path::PathBuf, HostState)> {
    let dir = local_app_data.join("GameAtlas").join("state");
    let wanted = canonical_box_path(box_path);
    let mut found: Option<(std::path::PathBuf, HostState)> = None;
    for entry in fs::read_dir(dir).ok()? {
        let Ok(entry) = entry else {
            continue;
        };
        let Ok(file_type) = entry.file_type() else {
            continue;
        };
        if !file_type.is_file() {
            continue;
        }
        let path = entry.path();
        let Some(state) = load(&path) else {
            continue;
        };
        if !box_path_matches(&state.box_path, &wanted) {
            continue;
        }
        let newer = found
            .as_ref()
            .map_or(true, |(_, current)| state.generation > current.generation);
        if newer {
            found = Some((path, state));
        }
    }
    found
}

fn pid_is_same_live_process(state: &HostState) -> bool {
    if !pid_alive(state.pid) {
        return false;
    }
    match state.process_identity {
        0 => true,
        expected => process_identity(state.pid) == Some(expected),
    }
}

pub fn box_path_matches(left: &str, right: &str) -> bool {
    let left = canonical_box_path(left);
    let right = canonical_box_path(right);
    #[cfg(target_os = "windows")]
    {
        left.eq_ignore_ascii_case(&right)
    }
    #[cfg(not(target_os = "windows"))]
    {
        left == right
    }
}

/// 进程是否存活。
#[cfg(target_os = "windows")]
pub fn pid_alive(pid: u32) -> bool {
    if pid == 0 {
        return false;
    }
    // PROCESS_QUERY_LIMITED_INFORMATION 即可查询退出状态。
    let handle = unsafe { crate::winffi::OpenProcess(0x1000, 0, pid) };
    if handle == 0 {
        return false;
    }
    let mut code: u32 = 0;
    let ok = unsafe { crate::winffi::GetExitCodeProcess(handle, &mut code) };
    unsafe { crate::winffi::CloseHandle(handle) };
    ok != 0 && code == 259 // STILL_ACTIVE
}

#[cfg(target_os = "windows")]
fn process_identity(pid: u32) -> Option<u64> {
    let handle = unsafe { crate::winffi::OpenProcess(0x1000, 0, pid) };
    if handle == 0 {
        return None;
    }
    let mut created = crate::winffi::FileTime { low: 0, high: 0 };
    let mut exited = crate::winffi::FileTime { low: 0, high: 0 };
    let mut kernel = crate::winffi::FileTime { low: 0, high: 0 };
    let mut user = crate::winffi::FileTime { low: 0, high: 0 };
    let ok = unsafe {
        crate::winffi::GetProcessTimes(
            handle,
            &mut created,
            &mut exited,
            &mut kernel,
            &mut user,
        )
    };
    unsafe { crate::winffi::CloseHandle(handle) };
    (ok != 0).then_some((u64::from(created.high) << 32) | u64::from(created.low))
}

/// 进程是否存活（Linux：/proc/<pid> 存在）。
#[cfg(not(target_os = "windows"))]
pub fn pid_alive(pid: u32) -> bool {
    process_stat(pid).is_some_and(|fields| fields.first().is_some_and(|state| *state != "Z" && *state != "X"))
}

#[cfg(not(target_os = "windows"))]
fn process_identity(pid: u32) -> Option<u64> {
    let fields = process_stat(pid)?;
    fields.get(19)?.parse().ok()
}

#[cfg(not(target_os = "windows"))]
fn process_stat(pid: u32) -> Option<Vec<String>> {
    // The command name may contain spaces and parentheses; everything after
    // the final ')' has the stable `/proc/<pid>/stat` field layout.
    let raw = fs::read_to_string(format!("/proc/{pid}/stat")).ok()?;
    let tail = raw.rsplit_once(')')?.1;
    Some(tail.split_whitespace().map(str::to_string).collect())
}

fn now_secs() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

fn save(path: &Path, s: &HostState) -> Result<(), String> {
    if let Some(dir) = path.parent() {
        fs::create_dir_all(dir).map_err(|e| e.to_string())?;
    }
    let mut tmp = path.as_os_str().to_os_string();
    tmp.push(".tmp");
    let tmp = Path::new(&tmp);
    fs::write(tmp, to_json(s)).map_err(|e| e.to_string())?;
    match fs::rename(tmp, path) {
        Ok(()) => Ok(()),
        Err(_e) if path.exists() => {
            fs::remove_file(path).map_err(|e| e.to_string())?;
            fs::rename(tmp, path).map_err(|e| e.to_string())
        }
        Err(e) => Err(e.to_string()),
    }
}

fn to_json(s: &HostState) -> String {
    format!(
        "{{\n  \"generation\": {},\n  \"game_id\": \"{}\",\n  \"box_path\": \"{}\",\n  \"state\": \"{}\",\n  \"pid\": {},\n  \"started_at\": {},\n  \"process_identity\": {}\n}}",
        s.generation,
        escape_json(&s.game_id),
        escape_json(&s.box_path),
        escape_json(&s.state),
        s.pid,
        s.started_at,
        s.process_identity,
    )
}

fn from_json(s: &str) -> Result<HostState, String> {
    let mut out = HostState {
        generation: 0,
        game_id: String::new(),
        box_path: String::new(),
        state: String::new(),
        pid: 0,
        started_at: 0,
        process_identity: 0,
    };
    let mut seen = [false; 7];
    for (key, value) in parse_json_object(s).map_err(|e| format!("非法 JSON: {e}"))? {
        let idx = match key.as_str() {
            "generation" => 0,
            "game_id" => 1,
            "box_path" => 2,
            "state" => 3,
            "pid" => 4,
            "started_at" => 5,
            "process_identity" => 6,
            other => return Err(format!("未知字段 '{other}'")),
        };
        if seen[idx] {
            return Err(format!("重复字段 '{key}'"));
        }
        seen[idx] = true;
        match idx {
            0 => out.generation = value.parse().map_err(|_| "非法 generation".to_string())?,
            1 => out.game_id = value,
            2 => out.box_path = value,
            3 => out.state = value,
            4 => out.pid = value.parse().map_err(|_| "非法 pid".to_string())?,
            5 => out.started_at = value.parse().map_err(|_| "非法 started_at".to_string())?,
            6 => out.process_identity = value
                .parse()
                .map_err(|_| "非法 process_identity".to_string())?,
            _ => unreachable!(),
        }
    }
    if !seen[0] || !seen[1] || !seen[3] || !seen[4] {
        return Err("缺少必填字段".to_string());
    }
    Ok(out)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn tmp_state(name: &str) -> std::path::PathBuf {
        let d = std::env::temp_dir().join(format!("gamevhd_hoststate_{name}_{}", std::process::id()));
        let _ = fs::remove_dir_all(&d);
        fs::create_dir_all(&d).unwrap();
        d.join(HOST_STATE_FILE_NAME)
    }

    #[test]
    fn missing_state_is_none_and_begin_creates() {
        let p = tmp_state("missing");
        assert!(load(&p).is_none());
        let s = begin_run(&p, "g1", r"C:\box.json").unwrap();
        assert_eq!(s.generation, 1);
        assert!(s.running());
        assert_eq!(s.pid, std::process::id());
        let loaded = load(&p).unwrap();
        assert_eq!(loaded.generation, 1);
        assert_eq!(loaded.game_id, "g1");
    }

    #[test]
    fn second_begin_rejects_when_self_alive() {
        let p = tmp_state("reject");
        let _ = begin_run(&p, "g1", "b.json").unwrap();
        // 本测试进程自己还活着 → 第二次 begin_run 同 game_id 被拒。
        let err = begin_run(&p, "g1", "b.json").unwrap_err();
        assert!(err.contains("已在运行"), "{err}");
        // 不同 game_id 不受影响。
        let s2 = begin_run(&p, "g2", "b2.json").unwrap();
        assert_eq!(s2.generation, 2);
    }

    #[test]
    fn stale_running_record_is_reclaimed() {
        let p = tmp_state("stale");
        // 写入一个死 PID 的 running 记录（模拟崩溃残留）。
        fs::write(
            &p,
            "{\"generation\": 7, \"game_id\": \"g1\", \"box_path\": \"b.json\", \"state\": \"running\", \"pid\": 99999999, \"started_at\": 0}",
        )
        .unwrap();
        let s = begin_run(&p, "g1", "b.json").unwrap();
        assert_eq!(s.generation, 8, "死 PID 记录被接管并 generation+1");
    }

    #[test]
    fn mark_clean_only_matches_own_generation() {
        let p = tmp_state("clean");
        let s = begin_run(&p, "g1", "b.json").unwrap();
        // 代数不匹配（旧清理）→ 不动。
        mark_clean(&p, s.generation - 1, "g1").unwrap();
        assert!(load(&p).unwrap().running());
        // 代数匹配 → 置 clean。
        mark_clean(&p, s.generation, "g1").unwrap();
        assert!(!load(&p).unwrap().running());
        // 再 begin 又是新代数。
        let s2 = begin_run(&p, "g1", "b.json").unwrap();
        assert_eq!(s2.generation, s.generation + 1);
    }

    #[test]
    fn game_state_path_is_stable_and_cannot_escape() {
        let root = Path::new(r"C:\Users\Hao\AppData\Local");
        let a = state_path_for_game(root, r"a\..\b");
        let b = state_path_for_game(root, r"a\..\b");
        assert_eq!(a, b);
        assert_eq!(a.parent().unwrap().file_name().unwrap(), "state");
        assert!(a.file_name().unwrap().to_string_lossy().starts_with("state_"));
    }

    #[test]
    fn state_lookup_uses_authoritative_box_path() {
        let root = std::env::temp_dir().join(format!("gamevhd_state_lookup_{}", std::process::id()));
        let state_path = state_path_for_game(&root, "authoritative-game");
        let state = HostState {
            generation: 4,
            game_id: "authoritative-game".into(),
            box_path: canonical_box_path("box.json"),
            state: "clean".into(),
            pid: 0,
            started_at: 0,
            process_identity: 0,
        };
        save(&state_path, &state).unwrap();
        let found = find_state_for_box(&root, "box.json").expect("state found");
        assert_eq!(found.0, state_path);
        assert_eq!(found.1.game_id, "authoritative-game");
        let _ = fs::remove_dir_all(root);
    }
}
