//! 盘符 exe 扫描与 box.json 记忆（阶段 0 + 阶段 5 增强；v2 定案 §3.7）。
//!
//! 递归收集 `.exe`（大小写不敏感），噪声过滤（`unins*` / `setup*` /
//! `install*` / `vcredist*` / `dxsetup` / `redist` 目录段等——安装器/卸载器/
//! 运行时组件不是游戏本体），排序（深度浅优先、体积大优先），
//! 逐文件标注位数（[`crate::pe::probe_file`]）。
//!
//! box.json 记忆：首次选择记入 `GameData\box.json` 的 `exe_relative`
//! （路径相对卷根，随卡带走）；记忆失效（exe 不存在）回退重扫重选。
//! 不写 VHD 标记、不改数据库。
//!
//! 纯 std 实现，Linux 上可对任意目录测试；单字母参数（如 `E`）解释为
//! 盘符根（`E:\`），否则按字面路径处理。深度上限 [`MAX_DEPTH`] 防失控递归。

// 库模块：记忆 API（remembered_exe_exists/resolve_remembered/to_relative）留给
// 阶段 5 run 主流程消费（当前 wave 仅 cmd_scan 与单测调用），按模块豁免。
#![allow(dead_code)]

use std::path::{Path, PathBuf};

use crate::pe;

/// 递归扫描深度上限（VHD 内目录树远小于此）。
pub const MAX_DEPTH: usize = 16;

/// 文件名前缀噪声（大小写不敏感；`setup` 需按路径段处理，见 [`is_noise_name`]）。
const NOISE_PREFIXES: [&str; 5] = ["unins", "setup", "install", "vcredist", "dxsetup"];

/// 噪声目录段名（路径中任一段匹配即过滤）。
const NOISE_DIRS: [&str; 6] = [
    "redist",
    "redistributable",
    "directx",
    "vcredist",
    "vc_redist",
    "support",
];

/// `scan <drive>` 命令入口，返回进程退出码（0 成功，1 目录不可读）。
pub fn cmd_scan(drive: &str) -> u8 {
    let root = resolve_root(drive);
    let exes = collect_exes(&root, MAX_DEPTH);
    if exes.is_empty() {
        println!("未找到候选 exe：{}", root.display());
        return 0;
    }
    for p in &exes {
        let kind = match pe::probe_file(p) {
            Ok(k) => k.as_str().to_string(),
            Err(_) => "unreadable".to_string(),
        };
        println!("[{kind}] {}", p.display());
    }
    println!("共 {} 个候选", exes.len());
    0
}

/// 单字母 → `X:\`；否则按字面路径。
fn resolve_root(drive: &str) -> PathBuf {
    let t = drive.trim();
    let mut chars = t.chars();
    if let (Some(c), None) = (chars.next(), chars.next()) {
        if c.is_ascii_alphabetic() {
            return PathBuf::from(format!("{}:\\", c.to_ascii_uppercase()));
        }
    }
    PathBuf::from(t)
}

/// 递归收集候选 exe（BFS，排序输出，忽略不可读目录）。
fn collect_exes(root: &Path, max_depth: usize) -> Vec<PathBuf> {
    let mut out = Vec::new();
    let mut stack = vec![(root.to_path_buf(), 0usize)];
    while let Some((dir, depth)) = stack.pop() {
        if depth > max_depth {
            continue;
        }
        let entries = match std::fs::read_dir(&dir) {
            Ok(e) => e,
            Err(_) => continue,
        };
        for entry in entries.flatten() {
            let p = entry.path();
            if p.is_dir() {
                if !is_noise_dir(&p) {
                    stack.push((p, depth + 1));
                }
            } else if is_candidate_exe(&p) {
                out.push(p);
            }
        }
    }
    out.sort_by(rank_candidate);
    out
}

/// 候选 exe：`.exe` 后缀（大小写不敏感）且文件名不含噪声前缀。
fn is_candidate_exe(p: &Path) -> bool {
    let name = match p.file_name().and_then(|n| n.to_str()) {
        Some(n) => n,
        None => return false,
    };
    let lower = name.to_ascii_lowercase();
    lower.ends_with(".exe")
        && !NOISE_PREFIXES
            .iter()
            .any(|pre| lower.starts_with(pre))
}

/// 噪声目录段：路径中任一段匹配（`redist` / `directx` 等）即跳过子树。
/// 按 `/` 与 `\` 双分隔符拆分所有段（跨平台：Linux 上也能识别 Windows 字面量，
/// 生产 Windows 上 file_name 语义一致）。
fn is_noise_dir(p: &Path) -> bool {
    let s = p.to_string_lossy();
    s.split(['/', '\\']).any(|seg| {
        let lower = seg.to_ascii_lowercase();
        NOISE_DIRS.iter().any(|d| *d == lower)
    })
}

/// 候选排序：深度浅优先，同深度体积大优先（v2 定案 §3.7）。
/// 深度 = 相对卷根的路径段数（根=0）；体积取文件长度。
fn rank_candidate(a: &PathBuf, b: &PathBuf) -> std::cmp::Ordering {
    depth_of(a)
        .cmp(&depth_of(b))
        .then_with(|| size_of(b).cmp(&size_of(a)))
}

/// 路径深度：非盘符段数（`C:\`=0、`C:\Game`=1、`C:\Game\x.exe`=2）。
/// 按 `/` 与 `\` 双分隔符拆分所有段并排除盘符段（跨平台：Linux 上也识别
/// Windows 字面量）。文件与目录同规则，仅用于排序（相对序正确即可）。
fn depth_of(p: &Path) -> usize {
    let s = p.to_string_lossy();
    s.split(['/', '\\'])
        .filter(|seg| !seg.is_empty())
        .filter(|seg| !seg.ends_with(':')) // 盘符段（如 `C:` / `E:`）不计
        .count()
}

/// 文件大小（读 metadata；失败按 0 处理，不影响排序稳定性）。
fn size_of(p: &Path) -> u64 {
    std::fs::metadata(p).map(|m| m.len()).unwrap_or(0)
}

// ---- box.json 记忆（v2 定案 §3.7） ----

/// 记忆的 exe 相对路径是否仍存在（卷根 + 相对路径）。
/// `exe_relative` 形如 `Game\HorizonZeroDawn.exe`，一律相对卷根。
/// 卷根尾随分隔符与相对路径内的 `\`/`/` 均规范化后拼接（跨平台可测）。
pub fn remembered_exe_exists(volume_root: &Path, exe_relative: &str) -> bool {
    join_volume_path(volume_root, exe_relative).is_file()
}

/// 卷根 + 相对路径拼接（`\` → 平台分隔符，防 Linux 测试与 Windows 语义分歧）。
fn join_volume_path(volume_root: &Path, exe_relative: &str) -> PathBuf {
    let normalized = exe_relative.replace('\\', "/");
    volume_root.join(normalized)
}

/// 从候选列表解析记忆指向的 exe（精确匹配相对路径）。
/// 记忆失效（列表中不存在）→ `None`，调用方回退重扫重选。
pub fn resolve_remembered<'a>(
    candidates: &'a [PathBuf],
    volume_root: &Path,
    exe_relative: &str,
) -> Option<&'a PathBuf> {
    let target = join_volume_path(volume_root, exe_relative);
    candidates.iter().find(|p| **p == target)
}

/// 由候选路径生成相对卷根的路径（`E:\Game\x.exe` → `Game\x.exe`）。
/// 基于文本前缀匹配（卷根可能带尾分隔符；Linux 测试时路径无盘符，前缀按
/// 文本比较即可）。候选不在卷根下时返回原路径（防御：不应发生）。
pub fn to_relative(volume_root: &Path, exe: &Path) -> String {
    let root_s = volume_root.to_string_lossy();
    let root_norm = root_s.trim_end_matches(['\\', '/']);
    let exe_s = exe.to_string_lossy();
    if let Some(rest) = exe_s.strip_prefix(root_norm) {
        let rest = rest.trim_start_matches(['\\', '/']);
        if !rest.is_empty() {
            return rest.replace('/', "\\");
        }
    }
    exe_s.into_owned()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn tmp_tree(name: &str) -> PathBuf {
        let d = std::env::temp_dir().join(format!("gamevhd_scan_{name}_{}", std::process::id()));
        let _ = fs::remove_dir_all(&d);
        fs::create_dir_all(d.join("sub/deep")).unwrap();
        fs::create_dir_all(d.join("redist")).unwrap();
        d
    }

    fn write(d: &Path, rel: &str, content: &[u8]) {
        fs::write(d.join(rel), content).unwrap();
    }

    fn write_bytes(d: &Path, rel: &str, content: Vec<u8>) {
        fs::write(d.join(rel), content).unwrap();
    }

    #[test]
    fn candidate_filter_table() {
        let cases = [
            ("game.exe", true),
            ("Game.EXE", true),
            ("horizon.exe", true),
            ("unins000.exe", false),
            ("uninstaller.exe", false),
            ("install_game.exe", false),
            ("setup_game.exe", false),
            ("setup.exe", false),
            ("vcredist_x64.exe", false),
            ("dxsetup.exe", false),
            ("DXSETUP.EXE", false),
            ("notes.txt", false),
            ("noext", false),
        ];
        for (name, want) in cases {
            let p = PathBuf::from(name);
            assert_eq!(is_candidate_exe(&p), want, "{name}");
        }
    }

    #[test]
    fn noise_dir_filter_table() {
        let cases = [
            (r"C:\Game\redist", true),
            (r"C:\Game\Redistributable", true),
            (r"C:\Game\directx", true),
            (r"C:\Game\vcredist", true),
            (r"C:\Game\support", true),
            (r"C:\Game\bin", false),
            (r"C:\Game\Redemption", false), // 前缀匹配不得误杀
        ];
        for (name, want) in cases {
            let p = PathBuf::from(name);
            assert_eq!(is_noise_dir(&p), want, "{name}");
        }
    }

    #[test]
    fn collect_exes_recurses_filters_and_sorts() {
        let d = tmp_tree("collect");
        // 体积大优先：deep/app 大文件应排在同深度小文件前。
        write_bytes(&d, "game.exe", vec![b'M'; 100]);
        write(&d, "unins000.exe", b"MZ");
        write(&d, "install_x64.exe", b"MZ");
        write(&d, "setup_game.exe", b"MZ");
        write(&d, "redist/helper.exe", b"MZ"); // 噪声目录子树跳过
        write_bytes(&d, "sub/small.exe", vec![b'M'; 10]);
        write_bytes(&d, "sub/big.exe", vec![b'M'; 500]);
        write_bytes(&d, "sub/deep/tool.exe", vec![b'M'; 300]);
        write(&d, "sub/readme.txt", b"hi");

        let found = collect_exes(&d, MAX_DEPTH);
        let rels: Vec<String> = found
            .iter()
            .map(|p| p.strip_prefix(&d).unwrap().to_string_lossy().into_owned())
            .collect();
        // 期望：game.exe(深度0) → sub/big(深度1, 500B) → sub/small(深度1, 10B)
        // → sub/deep/tool(深度2)。redist/helper 被目录噪声过滤。
        assert_eq!(
            rels,
            vec!["game.exe", "sub/big.exe", "sub/small.exe", "sub/deep/tool.exe"]
        );

        let _ = fs::remove_dir_all(&d);
    }

    #[test]
    fn collect_exes_tolerates_missing_root() {
        let missing = std::env::temp_dir().join(format!(
            "gamevhd_scan_nonexistent_{}",
            std::process::id()
        ));
        assert!(collect_exes(&missing, MAX_DEPTH).is_empty());
    }

    #[test]
    fn resolve_root_letter_vs_path() {
        assert_eq!(resolve_root("E"), PathBuf::from(r"E:\"));
        assert_eq!(resolve_root("e"), PathBuf::from(r"E:\"));
        assert_eq!(resolve_root("/tmp/x"), PathBuf::from("/tmp/x"));
        assert_eq!(resolve_root(r"E:\Games"), PathBuf::from(r"E:\Games"));
        // 非单字母（含冒号）按字面处理：盘符路径由调用方保证存在。
        assert_eq!(resolve_root("E:"), PathBuf::from("E:"));
    }

    #[test]
    fn remembered_exe_exists_and_resolve() {
        let d = tmp_tree("memory");
        fs::create_dir_all(d.join("Game")).unwrap();
        write(&d, "Game/app.exe", b"MZ");
        write(&d, "Game/unins000.exe", b"MZ");

        let candidates = collect_exes(&d, MAX_DEPTH);
        assert!(remembered_exe_exists(&d, r"Game\app.exe"));
        assert!(!remembered_exe_exists(&d, r"Game\missing.exe"));

        let hit = resolve_remembered(&candidates, &d, r"Game\app.exe");
        assert!(hit.is_some(), "记忆指向的 exe 在候选中应命中");
        assert_eq!(hit.unwrap().file_name().unwrap().to_str(), Some("app.exe"));

        let miss = resolve_remembered(&candidates, &d, r"Game\gone.exe");
        assert!(miss.is_none(), "记忆失效（exe 不存在）应回退 None");

        let _ = fs::remove_dir_all(&d);
    }

    #[test]
    fn to_relative_and_depth() {
        let root = PathBuf::from(r"E:\");
        assert_eq!(to_relative(&root, Path::new(r"E:\Game\x.exe")), r"Game\x.exe");
        assert_eq!(depth_of(Path::new(r"E:\")), 0);
        assert_eq!(depth_of(Path::new(r"E:\Game")), 1);
        assert_eq!(depth_of(Path::new(r"E:\Game\x.exe")), 2);
    }
}
