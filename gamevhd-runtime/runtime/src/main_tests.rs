//! main.rs 的 CLI 解析/分发单元测试（本波交付）。
//! 与 main.rs 分离以保证其体积在审查上限内；`super::*` 访问 crate 根私有项。

use super::*;

fn args(v: &[&str]) -> Vec<String> {
    v.iter().map(|s| s.to_string()).collect()
}

#[test]
fn parse_help_and_version() {
    assert_eq!(parse_args(&args(&["gv", "--help"])), Ok(Command::Help));
    assert_eq!(parse_args(&args(&["gv", "-h"])), Ok(Command::Help));
    assert_eq!(parse_args(&args(&["gv", "--version"])), Ok(Command::Version));
}

#[test]
fn parse_positional_subcommands() {
    let mount_base = Command::Mount {
        vhd: "test.vhd".into(),
        parent: None,
        smb: None,
        user: None,
        pass: None,
        letter: None,
        retries: 3,
    };
    assert_eq!(parse_args(&args(&["gv", "mount", "test.vhd"])), Ok(mount_base.clone()));
    let unmount_base = Command::Unmount {
        vhd: "t.vhd".into(),
        letter: None,
        smb: None,
    };
    assert_eq!(parse_args(&args(&["gv", "unmount", "t.vhd"])), Ok(unmount_base));
    assert_eq!(
        parse_args(&args(&["gv", "scan", "E"])),
        Ok(Command::Scan("E".into()))
    );
    assert_eq!(
        parse_args(&args(&["gv", "probe", "game.exe"])),
        Ok(Command::Probe("game.exe".into()))
    );
}

#[test]
fn parse_mount_full_options() {
    let want = Command::Mount {
        vhd: r"C:\diff\g.vhdx".into(),
        parent: Some(r"\\192.168.1.4\Game\base.vhdx".into()),
        smb: Some(r"\\192.168.1.4\Game".into()),
        user: Some("rom".into()),
        pass: Some("secret".into()),
        letter: Some('G'),
        retries: 5,
    };
    assert_eq!(
        parse_args(&args(&[
            "gv", "mount", r"C:\diff\g.vhdx",
            "--parent", r"\\192.168.1.4\Game\base.vhdx",
            "--smb", r"\\192.168.1.4\Game",
            "--user", "rom", "--pass", "secret",
            "--letter", "g", "--retries", "5",
        ])),
        Ok(want)
    );
}

#[test]
fn parse_mount_partial_options() {
    // 只给 --letter：其余缺省。
    let want = Command::Mount {
        vhd: "a.vhdx".into(),
        parent: None,
        smb: None,
        user: None,
        pass: None,
        letter: Some('D'),
        retries: 3,
    };
    assert_eq!(
        parse_args(&args(&["gv", "mount", "a.vhdx", "--letter", "D"])),
        Ok(want)
    );
}

#[test]
fn parse_mount_bad_args() {
    assert!(parse_args(&args(&["gv", "mount"])).is_err(), "mount 缺 vhd");
    assert!(
        parse_args(&args(&["gv", "mount", "a.vhdx", "--letter", "EE"])).is_err(),
        "盘符必须单字母"
    );
    assert!(
        parse_args(&args(&["gv", "mount", "a.vhdx", "--retries", "0"])).is_err(),
        "retries 必须 ≥ 1"
    );
    assert!(
        parse_args(&args(&["gv", "mount", "a.vhdx", "--retries", "x"])).is_err(),
        "retries 必须为数字"
    );
    assert!(
        parse_args(&args(&["gv", "mount", "a.vhdx", "--parent"])).is_err(),
        "--parent 缺值"
    );
}

#[test]
fn parse_unmount_options() {
    let want = Command::Unmount {
        vhd: "b.vhdx".into(),
        letter: Some('E'),
        smb: Some(r"\\192.168.1.4\Game".into()),
    };
    assert_eq!(
        parse_args(&args(&[
            "gv", "unmount", "b.vhdx", "--letter", "e", "--smb", r"\\192.168.1.4\Game"
        ])),
        Ok(want)
    );
    assert!(parse_args(&args(&["gv", "unmount"])).is_err(), "unmount 缺 vhd");
}

#[test]
fn parse_run_flags_in_any_order() {
    let want = Command::Run {
        drive: 'E',
        box_path: r"C:\box.json".into(),
    };
    assert_eq!(
        parse_args(&args(&["gv", "run", "--drive", "E", "--box", r"C:\box.json"])),
        Ok(want.clone())
    );
    assert_eq!(
        parse_args(&args(&["gv", "run", "--box", r"C:\box.json", "--drive", "e"])),
        Ok(want.clone()),
        "选项乱序 + 小写盘符应归一化"
    );
    // 未知裸标志告警忽略，不影响解析。
    assert_eq!(
        parse_args(&args(&["gv", "run", "--drive", "E", "--box", r"C:\box.json", "--verbose"])),
        Ok(want)
    );
}

#[test]
fn parse_cleanup() {
    assert_eq!(
        parse_args(&args(&["gv", "cleanup", "--box", "b.json"])),
        Ok(Command::Cleanup { box_path: "b.json".into() })
    );
    assert!(parse_args(&args(&["gv", "cleanup"])).is_err());
}

#[test]
fn parse_errors() {
    assert!(parse_args(&args(&["gv"])).is_err(), "无子命令");
    assert!(parse_args(&args(&["gv", "frobnicate"])).is_err(), "未知子命令");
    assert!(parse_args(&args(&["gv", "mount"])).is_err(), "mount 缺参");
    assert!(parse_args(&args(&["gv", "run"])).is_err(), "run 全缺");
    assert!(parse_args(&args(&["gv", "run", "--box", "b"])).is_err(), "run 缺 --drive");
    assert!(
        parse_args(&args(&["gv", "run", "--drive", "EE", "--box", "b"])).is_err(),
        "盘符必须单字母"
    );
    assert!(
        parse_args(&args(&["gv", "run", "--drive", "E"])).is_err(),
        "run 缺 --box 值"
    );
}

#[test]
fn exit_codes_for_help_version_and_usage() {
    assert_eq!(run_cli(&args(&["gv", "--help"])), 0);
    assert_eq!(run_cli(&args(&["gv", "--version"])), 0);
    assert_eq!(run_cli(&args(&["gv", "frobnicate"])), 2);
    assert_eq!(run_cli(&args(&["gv", "mount"])), 2);
}

/// Windows 专属子命令在 Linux 上必须返回 3（保证 Linux 可测）。
#[cfg(not(target_os = "windows"))]
#[test]
fn windows_only_commands_unsupported_on_linux() {
    assert_eq!(run_cli(&args(&["gv", "mount", "t.vhd"])), 3);
    assert_eq!(run_cli(&args(&["gv", "unmount", "t.vhd"])), 3);
    assert_eq!(
        run_cli(&args(&["gv", "run", "--drive", "E", "--box", "b"])),
        3
    );
    assert_eq!(run_cli(&args(&["gv", "cleanup", "--box", "b"])), 3);
}

#[test]
fn probe_cmd_exit_codes() {
    let path = std::env::temp_dir().join(format!("gamevhd_probe_cmd_{}.exe", std::process::id()));
    // 最小 PE32（x86）：MZ + e_lfanew + 签名 + COFF(machine 0x14c) + magic 0x10b
    let mut b = vec![0u8; 0x80 + 4 + 20 + 2];
    b[0..2].copy_from_slice(b"MZ");
    b[0x3c..0x40].copy_from_slice(&0x80u32.to_le_bytes());
    b[0x80..0x84].copy_from_slice(b"PE\0\0");
    b[0x84..0x86].copy_from_slice(&0x14cu16.to_le_bytes());
    b[0x98..0x9a].copy_from_slice(&0x10bu16.to_le_bytes());
    std::fs::write(&path, &b).unwrap();

    assert_eq!(run_cli(&args(&["gv", "probe", &path.to_string_lossy()])), 0);
    let missing = path.with_extension("missing");
    assert_eq!(run_cli(&args(&["gv", "probe", &missing.to_string_lossy()])), 1);

    let _ = std::fs::remove_file(&path);
}
