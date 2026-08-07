//! gamevhd-runtime — GameVHD 沙箱编排器（GameAtlas VHD 远程启动，§2/§3/§4 阶段 0）。
//!
//! CLI 分发：`mount <vhd>` / `unmount <vhd>` / `scan <drive>` / `probe <exe>` /
//! `run --drive <letter> --box <path>` / `cleanup --box <path>`，以及 `--help` / `--version`。
//! 退出码：0 成功、1 运行错误、2 用法错误、3 当前平台不支持（Windows 专属子命令
//! 在 Linux 上返回 3，保证本 crate 可 Linux 测试）。
//!
//! Windows 专属子命令（mount/unmount/run/cleanup）的分发体以
//! `#[cfg(target_os = "windows")]` 编译并委托给对应模块桩；Linux 分支编译为
//! "unsupported platform" 错误。后续阶段任务只填充各自模块文件，
//! **禁止再改本文件与 Cargo.toml**（文件归属契约）。

mod boxfile;
mod cleanup;
mod disk;
mod hive;
mod inject;
mod json;
mod log;
mod pe;
mod process;
mod regpath;
mod rules;
mod run;
mod scan;
mod winffi;

use std::path::Path;
use std::process::ExitCode;

/// 解析后的子命令。
#[derive(Debug, Clone, PartialEq, Eq)]
enum Command {
    Mount {
        vhd: String,
        parent: Option<String>,
        smb: Option<String>,
        user: Option<String>,
        pass: Option<String>,
        letter: Option<char>,
        retries: u32,
    },
    Unmount {
        vhd: String,
        letter: Option<char>,
        smb: Option<String>,
    },
    Scan(String),
    Probe(String),
    Run { drive: char, box_path: String },
    Cleanup { box_path: String },
    Help,
    Version,
}

fn main() -> ExitCode {
    log::init();
    let args: Vec<String> = std::env::args().collect();
    ExitCode::from(run_cli(&args))
}

/// CLI 主分发（纯函数，可测）：返回进程退出码。
fn run_cli(args: &[String]) -> u8 {
    match parse_args(args) {
        Ok(Command::Help) => {
            print_usage();
            0
        }
        Ok(Command::Version) => {
            println!("gamevhd-runtime {}", env!("CARGO_PKG_VERSION"));
            0
        }
        Ok(Command::Mount { vhd, parent, smb, user, pass, letter, retries }) => {
            cmd_mount(&vhd, parent.as_deref(), smb.as_deref(), user.as_deref(), pass.as_deref(), letter, retries)
        }
        Ok(Command::Unmount { vhd, letter, smb }) => cmd_unmount(&vhd, letter, smb.as_deref()),
        Ok(Command::Scan(drive)) => scan::cmd_scan(&drive),
        Ok(Command::Probe(exe)) => cmd_probe(&exe),
        Ok(Command::Run { drive, box_path }) => cmd_run(drive, &box_path),
        Ok(Command::Cleanup { box_path }) => cmd_cleanup(&box_path),
        Err(msg) => {
            crate::log_error!("{msg}");
            eprintln!("用法: gamevhd-runtime --help 查看帮助");
            2
        }
    }
}

/// 解析 argv（args[0] 为程序名）。未知子命令/缺参/坏参 → Err（退出码 2）。
fn parse_args(args: &[String]) -> Result<Command, String> {
    let rest = &args[1..];
    let Some(sub) = rest.first() else {
        return Err("缺少子命令（使用 --help 查看用法）".into());
    };
    match sub.as_str() {
        "--help" | "-h" => Ok(Command::Help),
        "--version" | "-V" => Ok(Command::Version),
        "mount" => parse_mount(&rest[1..]),
        "unmount" => parse_unmount(&rest[1..]),
        "scan" => positional(rest, "scan", "<drive>").map(Command::Scan),
        "probe" => positional(rest, "probe", "<exe>").map(Command::Probe),
        "run" => parse_run(&rest[1..]),
        "cleanup" => {
            let opts = parse_kv_opts(&rest[1..], "cleanup", &["--box"])?;
            let box_path = opt_value(&opts, "--box")
                .ok_or_else(|| "cleanup 需要 --box <path>".to_string())?
                .clone();
            Ok(Command::Cleanup { box_path })
        }
        other => Err(format!("未知子命令 '{other}'")),
    }
}

/// 位置参数子命令：取首个参数，多余参数告警后忽略（为后续扩展保留余量）。
fn positional(rest: &[String], cmd: &str, argname: &str) -> Result<String, String> {
    let value = rest
        .get(1)
        .ok_or_else(|| format!("{cmd} 需要 {argname}"))?
        .clone();
    if rest.len() > 2 {
        crate::log_warn!("{cmd}: 忽略多余参数 {:?}", &rest[2..]);
    }
    Ok(value)
}

/// 解析单个盘符参数（`E` / `e` → 'E'；非法 → Err）。
fn parse_letter(s: &str, what: &str) -> Result<char, String> {
    let mut cs = s.chars();
    match (cs.next(), cs.next()) {
        (Some(c), None) if c.is_ascii_alphabetic() => Ok(c.to_ascii_uppercase()),
        _ => Err(format!("非法的{what} '{s}'（需单个字母，如 E）")),
    }
}

/// `mount <vhd> [--parent <UNC>] [--smb <UNC>] [--user <U>] [--pass <P>]
///       [--letter <L>] [--retries <N>]`
///
/// `<vhd>` 为本地差分盘路径（`%LOCALAPPDATA%\GameAtlas\diff\*.vhdx`）。
/// `--parent` 提供时若差分盘不存在则基于该 UNC 基础盘创建（已存在幂等跳过）。
/// `--smb` 提供时先连接只读共享（`--user`/`--pass` 可缺省走当前会话）。
/// `--letter` 指定首选盘符，缺省取第一个空闲。`--retries` 为 SMB 重试次数。
fn parse_mount(rest: &[String]) -> Result<Command, String> {
    let vhd = rest
        .first()
        .ok_or_else(|| "mount 需要 <vhd>（本地差分盘路径）".to_string())?
        .clone();
    let opts = parse_kv_opts(&rest[1..], "mount", &["--parent", "--smb", "--user", "--pass", "--letter", "--retries"])?;
    let parent = opt_value(&opts, "--parent").cloned();
    let smb = opt_value(&opts, "--smb").cloned();
    let user = opt_value(&opts, "--user").cloned();
    let pass = opt_value(&opts, "--pass").cloned();
    let letter = opt_value(&opts, "--letter")
        .map(|v| parse_letter(v, "盘符"))
        .transpose()?;
    let retries = match opt_value(&opts, "--retries") {
        Some(v) => v
            .parse::<u32>()
            .map_err(|_| format!("mount: --retries 需为正整数，收到 '{v}'"))?,
        None => 3,
    };
    if retries == 0 {
        return Err("mount: --retries 必须 ≥ 1".into());
    }
    Ok(Command::Mount {
        vhd,
        parent,
        smb,
        user,
        pass,
        letter,
        retries,
    })
}

/// `unmount <vhd> [--letter <L>] [--smb <UNC>]`
fn parse_unmount(rest: &[String]) -> Result<Command, String> {
    let vhd = rest
        .first()
        .ok_or_else(|| "unmount 需要 <vhd>（本地差分盘路径）".to_string())?
        .clone();
    let opts = parse_kv_opts(&rest[1..], "unmount", &["--letter", "--smb"])?;
    let letter = opt_value(&opts, "--letter")
        .map(|v| parse_letter(v, "盘符"))
        .transpose()?;
    let smb = opt_value(&opts, "--smb").cloned();
    Ok(Command::Unmount { vhd, letter, smb })
}

/// `run --drive <letter> --box <path>`：选项乱序均可，未知 `--x` 告警后忽略。
fn parse_run(rest: &[String]) -> Result<Command, String> {
    let opts = parse_kv_opts(rest, "run", &["--drive", "--box"])?;
    let letter = opt_value(&opts, "--drive")
        .ok_or_else(|| "run 需要 --drive <letter>".to_string())?;
    let box_path = opt_value(&opts, "--box")
        .ok_or_else(|| "run 需要 --box <path>".to_string())?
        .clone();
    let mut cs = letter.chars();
    let drive = match (cs.next(), cs.next()) {
        (Some(c), None) if c.is_ascii_alphabetic() => c.to_ascii_uppercase(),
        _ => return Err(format!("非法的盘符 '{letter}'（需单个字母，如 E）")),
    };
    Ok(Command::Run {
        drive,
        box_path: box_path.clone(),
    })
}

/// 解析 `--key value` 对：已知键必须带值（缺值 → Err）；未知键按裸标志处理
/// （告警后跳过、不消费值，为后续 wave 扩展 `--no-hook` 类标志留余地）。
fn parse_kv_opts(rest: &[String], cmd: &str, known: &[&str]) -> Result<Vec<(String, String)>, String> {
    let mut out = Vec::new();
    let mut i = 0;
    while i < rest.len() {
        let token = &rest[i];
        if let Some(key) = token.strip_prefix("--") {
            let key = format!("--{key}");
            if known.contains(&key.as_str()) {
                let value = rest
                    .get(i + 1)
                    .filter(|v| !v.starts_with("--"))
                    .ok_or_else(|| format!("{cmd}: 选项 {key} 缺少值"))?;
                out.push((key, value.clone()));
                i += 2;
            } else {
                crate::log_warn!("{cmd}: 忽略未知选项 {key}");
                i += 1;
            }
        } else {
            return Err(format!("{cmd}: 意外的位置参数 '{token}'"));
        }
    }
    Ok(out)
}

/// 按键查找 `--key value` 解析结果。
fn opt_value<'a>(opts: &'a [(String, String)], key: &str) -> Option<&'a String> {
    opts.iter().find(|(k, _)| k == key).map(|(_, v)| v)
}

/// 仅 Windows：挂载 VHD（阶段 4：virtdisk API 全流程，替代 diskpart）。
/// Linux：不支持（退出码 3）。
#[cfg_attr(not(target_os = "windows"), allow(unused_variables))]
fn cmd_mount(
    vhd: &str,
    parent: Option<&str>,
    smb: Option<&str>,
    user: Option<&str>,
    pass: Option<&str>,
    letter: Option<char>,
    retries: u32,
) -> u8 {
    #[cfg(target_os = "windows")]
    {
        let params = disk::MountParams {
            diff_path: vhd.to_string(),
            parent_unc: parent.map(str::to_string),
            smb_remote: smb.map(str::to_string),
            smb_user: user.map(str::to_string),
            smb_pass: pass.map(str::to_string),
            preferred_letter: letter,
            smb_retries: retries,
        };
        match disk::mount_vhd(&params) {
            Ok(result) => {
                crate::log_info!(
                    "mount: 挂载成功 {} → {}:（卷 {}）",
                    vhd,
                    result.drive_letter,
                    result.volume_guid
                );
                0
            }
            Err(e) => {
                crate::log_error!("mount 失败: {e}");
                1
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        unsupported("mount", "Windows + virtdisk API")
    }
}

/// 仅 Windows：卸载 VHD（阶段 4：detach + 断 SMB，替代 diskpart）。
/// Linux：不支持（退出码 3）。
#[cfg_attr(not(target_os = "windows"), allow(unused_variables))]
fn cmd_unmount(vhd: &str, letter: Option<char>, smb: Option<&str>) -> u8 {
    #[cfg(target_os = "windows")]
    {
        let params = disk::UnmountParams {
            drive_letter: letter.unwrap_or('E'),
            diff_path: vhd.to_string(),
            smb_remote: smb.map(str::to_string),
        };
        match disk::unmount_vhd(&params) {
            Ok(()) => {
                crate::log_info!("unmount: VHD 已卸载: {vhd}");
                0
            }
            Err(e) => {
                crate::log_error!("unmount 失败: {e}");
                1
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        unsupported("unmount", "Windows + virtdisk API")
    }
}

/// 仅 Windows：沙箱运行游戏。Linux：不支持（退出码 3）。
#[cfg_attr(not(target_os = "windows"), allow(unused_variables))]
fn cmd_run(drive: char, box_path: &str) -> u8 {
    #[cfg(target_os = "windows")]
    {
        match run::run_game(drive, box_path) {
            Ok(()) => {
                crate::log_info!("run: 游戏已退出，清理完毕");
                0
            }
            Err(e) => {
                crate::log_error!("run 失败: {e}");
                1
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        unsupported("run", "Windows")
    }
}

/// 仅 Windows：清理残留沙箱状态。Linux：不支持（退出码 3）。
#[cfg_attr(not(target_os = "windows"), allow(unused_variables))]
fn cmd_cleanup(box_path: &str) -> u8 {
    #[cfg(target_os = "windows")]
    {
        match cleanup::cleanup_box(box_path) {
            Ok(()) => {
                crate::log_info!("cleanup: 残留已清理: {box_path}");
                0
            }
            Err(e) => {
                crate::log_error!("cleanup 失败: {e}");
                1
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        unsupported("cleanup", "Windows")
    }
}

/// 跨平台（纯逻辑）：探测 exe 位数并打印。
fn cmd_probe(exe: &str) -> u8 {
    match pe::probe_file(Path::new(exe)) {
        Ok(kind) => {
            println!("{}: {}", exe, kind.as_str());
            0
        }
        Err(e) => {
            crate::log_error!("无法读取文件 '{exe}': {e}");
            1
        }
    }
}

/// 平台不支持错误（退出码 3）。
#[cfg_attr(target_os = "windows", allow(dead_code))]
fn unsupported(cmd: &str, need: &str) -> u8 {
    crate::log_error!("'{cmd}' 需要 {need}；当前平台不支持（本二进制在 Linux 上仅用于测试与交叉编译验证）");
    3
}

fn print_usage() {
    println!(
        "gamevhd-runtime {} — GameVHD 沙箱编排器（GameAtlas VHD 远程启动）",
        env!("CARGO_PKG_VERSION")
    );
    println!();
    println!("用法: gamevhd-runtime <子命令> [参数]");
    println!();
    println!("子命令:");
    println!("  mount <vhd> [--parent <UNC>] [--smb <UNC>] [--user <U>] [--pass <P>] [--letter <L>] [--retries <N>]");
    println!("                挂载 VHD（Windows；SMB→建差分→attach→定盘符）");
    println!("  unmount <vhd> [--letter <L>] [--smb <UNC>]   卸载 VHD（Windows）");
    println!("  scan <drive>              扫描盘符下的 exe 候选并标注位数（如 scan E）");
    println!("  probe <exe>               探测 exe 位数（x64 / x86 / not-pe）");
    println!("  run --drive <letter> --box <path>   沙箱启动游戏（Windows）");
    println!("  cleanup --box <path>      清理残留沙箱状态（Windows）");
    println!("  --help, -h                显示本帮助");
    println!("  --version, -V             显示版本");
    println!();
    println!("退出码: 0 成功  1 运行错误  2 用法错误  3 当前平台不支持");
}

#[cfg(test)]
mod main_tests;
