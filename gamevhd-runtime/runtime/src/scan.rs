//! 盘符 exe 扫描（阶段 0，本波交付；`scan <drive>` 命令）。
//!
//! 递归收集 `.exe`（大小写不敏感），过滤 `unins*.exe` / `install*.exe`
//! （卸载器与安装器不是游戏本体，阶段 0 验收要求），逐文件标注位数
//! （[`crate::pe::probe_file`]）。纯 std 实现，Linux 上可对任意目录测试；
//! 单字母参数（如 `E`）解释为盘符根（`E:\`），否则按字面路径处理。
//! 深度上限 [`MAX_DEPTH`] 防止恶意/损坏目录树导致失控递归。

use std::path::{Path, PathBuf};

use crate::pe;

/// 递归扫描深度上限（VHD 内目录树远小于此）。
const MAX_DEPTH: usize = 16;

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
                stack.push((p, depth + 1));
            } else if is_candidate_exe(&p) {
                out.push(p);
            }
        }
    }
    out.sort();
    out
}

/// `.exe` 后缀（大小写不敏感）且不以 `unins` / `install` 开头。
fn is_candidate_exe(p: &Path) -> bool {
    let name = match p.file_name().and_then(|n| n.to_str()) {
        Some(n) => n,
        None => return false,
    };
    let lower = name.to_ascii_lowercase();
    lower.ends_with(".exe") && !lower.starts_with("unins") && !lower.starts_with("install")
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn tmp_tree(name: &str) -> PathBuf {
        let d = std::env::temp_dir().join(format!("gamevhd_scan_{name}_{}", std::process::id()));
        let _ = fs::remove_dir_all(&d);
        fs::create_dir_all(d.join("sub/deep")).unwrap();
        d
    }

    fn write(d: &Path, rel: &str, content: &[u8]) {
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
            ("setup.exe", true),
            ("notes.txt", false),
            ("noext", false),
        ];
        for (name, want) in cases {
            let p = PathBuf::from(name);
            assert_eq!(is_candidate_exe(&p), want, "{name}");
        }
    }

    #[test]
    fn collect_exes_recurses_and_filters() {
        let d = tmp_tree("collect");
        write(&d, "game.exe", b"MZ");
        write(&d, "unins000.exe", b"MZ");
        write(&d, "install_x64.exe", b"MZ");
        write(&d, "sub/app.exe", b"MZ");
        write(&d, "sub/deep/tool.exe", b"MZ");
        write(&d, "sub/readme.txt", b"hi");

        let found = collect_exes(&d, MAX_DEPTH);
        let rels: Vec<String> = found
            .iter()
            .map(|p| p.strip_prefix(&d).unwrap().to_string_lossy().into_owned())
            .collect();
        assert_eq!(rels, vec!["game.exe", "sub/app.exe", "sub/deep/tool.exe"]);

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
}
