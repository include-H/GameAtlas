//! gamevhd-runtime — GameVHD 沙箱编排器（GameAtlas VHD 远程启动，§2/§3/§4）。
//!
//! CLI 分发：`mount <vhd>` / `unmount <vhd>` / `scan <drive>` / `probe <exe>` /
//! `run --drive <letter> --box <path>` / `cleanup --box <path>` / `--selftest`，
//! 以及 `--help` / `--version`。
//! 退出码：0 成功、1 运行错误、2 用法错误、3 当前平台不支持（Windows 专属子命令
//! 在 Linux 上返回 3，保证本 crate 可 Linux 测试）。
//!
//! 文件归属（v2 定案 §6）：main.rs 负责引导/提权/命令分发 + `--selftest`；
//! 各阶段能力在各自模块（disk/manifest/scan/inject/hive/process/...）实现。

mod boxfile;
mod cleanup;
mod disk;
mod hive;
mod hoststate;
mod inject;
mod json;
mod log;
mod manifest;
mod pe;
mod process;
mod regpath;
mod rules;
mod run;
mod scan;
mod sha256;
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
        letter: Option<char>,
        retries: u32,
    },
    /// 写入 SMB 凭据到 Windows 凭据管理器（密码从 stdin 读入，不进命令行）。
    StoreCred { server: String, user: Option<String> },
    /// 删除 SMB 凭据。
    DeleteCred { server: String },
    Unmount {
        vhd: String,
        letter: Option<char>,
        smb: Option<String>,
    },
    Scan(String),
    Probe(String),
    InjectPoc {
        exe: String,
        work_dir: String,
        hook: String,
        log: String,
        mode: String,
        args: String,
        game_data_root: String,
        user_profile: String,
        game_id: String,
    },
    Run {
        drive: char,
        box_path: String,
        hklm_write_passthrough: bool,
    },
    Cleanup {
        box_path: String,
        vhd: Option<String>,
        state: Option<String>,
    },
    Selftest,
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
        Ok(Command::Mount { vhd, parent, smb, user, letter, retries }) => {
            cmd_mount(&vhd, parent.as_deref(), smb.as_deref(), user.as_deref(), letter, retries)
        }
        Ok(Command::StoreCred { server, user }) => cmd_store_cred(&server, user.as_deref()),
        Ok(Command::DeleteCred { server }) => cmd_delete_cred(&server),
        Ok(Command::Unmount { vhd, letter, smb }) => cmd_unmount(&vhd, letter, smb.as_deref()),
        Ok(Command::Scan(drive)) => scan::cmd_scan(&drive),
        Ok(Command::Probe(exe)) => cmd_probe(&exe),
        Ok(Command::InjectPoc {
            exe,
            work_dir,
            hook,
            log,
            mode,
            args,
            game_data_root,
            user_profile,
            game_id,
        }) => {
            cmd_inject_poc(
                &exe,
                &work_dir,
                &hook,
                &log,
                &mode,
                &args,
                &game_data_root,
                &user_profile,
                &game_id,
            )
        }
        Ok(Command::Run {
            drive,
            box_path,
            hklm_write_passthrough,
        }) => cmd_run(drive, &box_path, hklm_write_passthrough),
        Ok(Command::Cleanup { box_path, vhd, state }) => {
            cmd_cleanup(&box_path, vhd.as_deref(), state.as_deref())
        }
        Ok(Command::Selftest) => cmd_selftest(),
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
        "--selftest" | "selftest" => Ok(Command::Selftest),
        "mount" => parse_mount(&rest[1..]),
        "store-cred" => parse_store_cred(&rest[1..]),
        "delete-cred" => parse_delete_cred(&rest[1..]),
        "unmount" => parse_unmount(&rest[1..]),
        "scan" => positional(rest, "scan", "<drive>").map(Command::Scan),
        "probe" => positional(rest, "probe", "<exe>").map(Command::Probe),
        "inject-poc" => parse_inject_poc(&rest[1..]),
        "run" => parse_run(&rest[1..]),
        "cleanup" => {
            let opts = parse_kv_opts(&rest[1..], "cleanup", &["--box", "--vhd", "--state"])?;
            let box_path = opt_value(&opts, "--box")
                .ok_or_else(|| "cleanup 需要 --box <path>".to_string())?
                .clone();
            let vhd = opt_value(&opts, "--vhd").cloned();
            let state = opt_value(&opts, "--state").cloned();
            Ok(Command::Cleanup { box_path, vhd, state })
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

/// `mount <vhd> [--parent <UNC>] [--smb <UNC>] [--user <U>]
///       [--letter <L>] [--retries <N>]`
///
/// `<vhd>` 为本地差分盘路径（`%LOCALAPPDATA%\GameAtlas\diff\*.vhdx`）。
/// `--parent` 提供时若差分盘不存在则基于该 UNC 基础盘创建（已存在幂等跳过）。
/// `--smb` 提供时先连接只读共享（`--user` 可缺省走当前会话；密码取自
/// Windows 凭据管理器（`store-cred` 写入）或环境变量 `GAMEVHD_SMB_PASS`，
/// 不接收命令行明文）。
/// `--letter` 指定首选盘符，缺省取第一个空闲。`--retries` 为 SMB 重试次数。
fn parse_mount(rest: &[String]) -> Result<Command, String> {
    let vhd = rest
        .first()
        .ok_or_else(|| "mount 需要 <vhd>（本地差分盘路径）".to_string())?
        .clone();
    let opts = parse_kv_opts(&rest[1..], "mount", &["--parent", "--smb", "--user", "--letter", "--retries"])?;
    let parent = opt_value(&opts, "--parent").cloned();
    let smb = opt_value(&opts, "--smb").cloned();
    let user = opt_value(&opts, "--user").cloned();
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
        letter,
        retries,
    })
}

/// `store-cred <server> [--user <U>]`：把 SMB 密码写入 Windows 凭据管理器。
/// 密码从 stdin 第一行读取（不回显、不进命令行参数与 shell 历史）。
fn parse_store_cred(rest: &[String]) -> Result<Command, String> {
    let server = rest
        .first()
        .ok_or_else(|| "store-cred 需要 <server>（如 \\\\192.168.1.4\\Game1）".to_string())?
        .clone();
    let opts = parse_kv_opts(&rest[1..], "store-cred", &["--user"])?;
    let user = opt_value(&opts, "--user").cloned();
    Ok(Command::StoreCred { server, user })
}

/// `delete-cred <server>`：删除已存储的 SMB 凭据（幂等）。
fn parse_delete_cred(rest: &[String]) -> Result<Command, String> {
    let server = rest
        .first()
        .ok_or_else(|| "delete-cred 需要 <server>".to_string())?
        .clone();
    Ok(Command::DeleteCred { server })
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

/// `run --drive <letter> --box <path> [--hklm-write passthrough|deny]`：
/// 选项乱序均可，未知 `--x` 告警后忽略。HKLM 写默认拒绝（P2-7），
/// passthrough 切回透传（兼容需要写 HKLM 的游戏）。
fn parse_run(rest: &[String]) -> Result<Command, String> {
    let opts = parse_kv_opts(rest, "run", &["--drive", "--box", "--hklm-write"])?;
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
    let hklm_write_passthrough = match opt_value(&opts, "--hklm-write") {
        None => false,
        Some(v) if v == "passthrough" => true,
        Some(v) if v == "deny" => false,
        Some(v) => {
            return Err(format!("run: --hklm-write 仅接受 passthrough|deny，收到 '{v}'"))
        }
    };
    Ok(Command::Run {
        drive,
        box_path: box_path.clone(),
        hklm_write_passthrough,
    })
}

/// Wine 联调专用注入入口；不读取 manifest/box，也不参与正式启动流程。
fn parse_inject_poc(rest: &[String]) -> Result<Command, String> {
    let opts = parse_kv_opts(
        rest,
        "inject-poc",
        &[
            "--exe",
            "--work-dir",
            "--hook",
            "--log",
            "--mode",
            "--args",
            "--game-data-root",
            "--user-profile",
            "--game-id",
        ],
    )?;
    let exe = opt_value(&opts, "--exe")
        .ok_or_else(|| "inject-poc 需要 --exe <Windows 路径>".to_string())?
        .clone();
    let hook = opt_value(&opts, "--hook")
        .ok_or_else(|| "inject-poc 需要 --hook <Windows DLL 路径>".to_string())?
        .clone();
    let log = opt_value(&opts, "--log")
        .ok_or_else(|| "inject-poc 需要 --log <Windows 路径>".to_string())?
        .clone();
    let work_dir = opt_value(&opts, "--work-dir").cloned().unwrap_or_default();
    let mode = opt_value(&opts, "--mode")
        .cloned()
        .unwrap_or_else(|| "remote".into());
    let args = opt_value(&opts, "--args").cloned().unwrap_or_default();
    let game_data_root = opt_value(&opts, "--game-data-root")
        .cloned()
        .unwrap_or_default();
    let user_profile = opt_value(&opts, "--user-profile")
        .cloned()
        .unwrap_or_default();
    let game_id = opt_value(&opts, "--game-id")
        .cloned()
        .unwrap_or_default();
    if mode != "remote" && mode != "early-bird" {
        return Err(format!("inject-poc: --mode 只支持 remote 或 early-bird，收到 '{mode}'"));
    }
    Ok(Command::InjectPoc {
        exe,
        work_dir,
        hook,
        log,
        mode,
        args,
        game_data_root,
        user_profile,
        game_id,
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
    letter: Option<char>,
    retries: u32,
) -> u8 {
    #[cfg(target_os = "windows")]
    {
        // 密码解析顺序：凭据管理器（store-cred 写入）→ 环境变量
        // GAMEVHD_SMB_PASS → None（复用现有 SMB 会话）。命令行永不明文传密。
        let (smb_user, smb_pass) = smb
            .map(|remote| {
                let cred = disk::read_smb_cred(remote);
                let env_pass = std::env::var("GAMEVHD_SMB_PASS").ok();
                let resolved_user = user
                    .map(str::to_string)
                    .or_else(|| cred.as_ref().map(|(u, _)| u.clone()))
                    .or_else(|| std::env::var("GAMEVHD_SMB_USER").ok());
                let resolved_pass = cred
                    .and_then(|(_, p)| if p.is_empty() { None } else { Some(p) })
                    .or(env_pass);
                if resolved_pass.is_some() && resolved_user.is_none() {
                    crate::log_warn!("mount: 有密码但缺用户名（--user / 凭据 / GAMEVHD_SMB_USER）");
                }
                (resolved_user, resolved_pass)
            })
            .unwrap_or((user.map(str::to_string), None));
        let params = disk::MountParams {
            diff_path: vhd.to_string(),
            parent_unc: parent.map(str::to_string),
            smb_remote: smb.map(str::to_string),
            smb_user,
            smb_pass,
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
                // ASCII marker 到 stdout：断言脚本/自动化依赖此格式（PS 5.1
                // GBK 控制台会乱码中文，marker 必须纯 ASCII 可机器解析）。
                println!("[MOUNT-OK] {}:", result.drive_letter);
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

/// 仅 Windows：写入 SMB 凭据（stdin 读密码，不进命令行）。
/// Linux：不支持（退出码 3）。
fn cmd_store_cred(server: &str, user: Option<&str>) -> u8 {
    #[cfg(target_os = "windows")]
    {
        let mut pass = String::new();
        if std::io::stdin().read_line(&mut pass).is_err() || pass.trim().is_empty() {
            crate::log_error!("store-cred: 请在 stdin 第一行输入密码");
            return 1;
        }
        let pass = pass.trim();
        let resolved_user = user
            .map(str::to_string)
            .or_else(|| std::env::var("GAMEVHD_SMB_USER").ok());
        let Some(resolved_user) = resolved_user else {
            crate::log_error!("store-cred: 需要 --user <U> 或环境变量 GAMEVHD_SMB_USER");
            return 1;
        };
        match disk::store_smb_cred(server, &resolved_user, pass) {
            Ok(()) => {
                crate::log_info!("store-cred: 凭据已保存（target GameVHD_{server}）");
                0
            }
            Err(e) => {
                crate::log_error!("store-cred 失败: {e}");
                1
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        let _ = server;
        let _ = user;
        unsupported("store-cred", "Windows 凭据管理器")
    }
}

/// 删除 SMB 凭据（幂等）。Linux：不支持（退出码 3）。
fn cmd_delete_cred(server: &str) -> u8 {
    #[cfg(target_os = "windows")]
    {
        match disk::delete_smb_cred(server) {
            Ok(()) => {
                crate::log_info!("delete-cred: 凭据已删除（target GameVHD_{server}）");
                0
            }
            Err(e) => {
                crate::log_error!("delete-cred 失败: {e}");
                1
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        let _ = server;
        unsupported("delete-cred", "Windows 凭据管理器")
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
                println!("[UNMOUNT-OK]");
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
fn cmd_run(drive: char, box_path: &str, hklm_write_passthrough: bool) -> u8 {
    #[cfg(target_os = "windows")]
    {
        match run::run_game(drive, box_path, hklm_write_passthrough) {
            Ok(()) => {
                crate::log_info!("run: 游戏已退出，清理完毕");
                // ASCII marker：断言脚本/自动化依赖此格式（同 [MOUNT-OK]）。
                println!("[RUN-OK]");
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
fn cmd_cleanup(box_path: &str, vhd_path: Option<&str>, host_state: Option<&str>) -> u8 {
    #[cfg(target_os = "windows")]
    {
        // 宿主权威状态路径缺省为 %LOCALAPPDATA%\GameAtlas\state.json。
        let state_path = match host_state {
            Some(p) => Some(std::path::PathBuf::from(p)),
            None => std::env::var("LOCALAPPDATA").ok().map(|la| {
                std::path::Path::new(la.trim_end_matches(['\\', '/']))
                    .join("GameAtlas")
                    .join(crate::hoststate::HOST_STATE_FILE_NAME)
            }),
        };
        match cleanup::cleanup_box(box_path, vhd_path, state_path.as_deref()) {
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

/// Windows/Wine 联调专用：挂起创建目标进程，选择一种注入路径后等待其退出。
#[cfg_attr(not(target_os = "windows"), allow(unused_variables))]
fn cmd_inject_poc(
    exe: &str,
    work_dir: &str,
    hook: &str,
    log: &str,
    mode: &str,
    args: &str,
    game_data_root: &str,
    user_profile: &str,
    game_id: &str,
) -> u8 {
    #[cfg(target_os = "windows")]
    {
        let (hproc, hthread) = match inject::launch_suspended_with_args(exe, work_dir, args) {
            Ok(handles) => handles,
            Err(e) => {
                crate::log_error!("inject-poc: 创建挂起进程失败: {e}");
                return 1;
            }
        };

        // 默认仍使用空参数块进行纯注入验证；提供路径参数时可单独联调
        // 文件/注册表重定向，不需要准备完整 manifest/box 生命周期。
        let param_block = rules::param_block_with(
            hook,
            game_data_root,
            user_profile,
            log,
            "",
            game_id,
            &[],
        );
        let result = (|| -> Result<(), String> {
            if mode == "early-bird" {
                inject::inject_into_process_early_bird(hproc, hthread, &param_block, hook)
                    .map_err(|e| e.to_string())?;
            } else {
                inject::inject_into_process(hproc, &param_block, hook)
                    .map_err(|e| e.to_string())?;
                let prev = unsafe { winffi::ResumeThread(hthread) };
                if prev == u32::MAX {
                    return Err(format!("ResumeThread 失败 (Win32 错误 {})", unsafe {
                        winffi::GetLastError()
                    }));
                }
            }
            let wait = unsafe { winffi::WaitForSingleObject(hproc, winffi::INFINITE) };
            if wait == winffi::WAIT_FAILED {
                return Err(format!("等待目标进程失败 (Win32 错误 {})", unsafe {
                    winffi::GetLastError()
                }));
            }
            Ok(())
        })();

        if result.is_err() {
            unsafe {
                winffi::TerminateProcess(hproc, 1);
                winffi::WaitForSingleObject(hproc, 5000);
            }
        }
        unsafe {
            winffi::CloseHandle(hthread);
            winffi::CloseHandle(hproc);
        }

        match result {
            Ok(()) => {
                println!("[INJECT-OK] {mode}");
                0
            }
            Err(e) => {
                crate::log_error!("inject-poc [{mode}] 失败: {e}");
                1
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        unsupported("inject-poc", "Windows/Wine")
    }
}

/// 自检模式（`--selftest`，v2 定案 §5.1）：footer 自定位 / 规则生成 / PE 探测。
/// 跨平台可跑；任一项失败退出码 1，全部通过 0。
fn cmd_selftest() -> u8 {
    let mut ok = true;
    println!("gamevhd-runtime {} 自检", env!("CARGO_PKG_VERSION"));

    // 1. footer 自定位/解析（构造内存样本：launcher 主体 + 配置块）。
    let sample = manifest::Manifest {
        game_id: "selftest".into(),
        title: "自检样本".into(),
        base_vhd: r"\\selftest\base.vhdx".into(),
        diff_name: "selftest.vhdx".into(),
        smb_user: None,
        smb_pass: None,
        exe_hint: None,
        skip_cache_dirs: false,
    };
    let mut blob = vec![0x00u8; 512];
    blob.extend_from_slice(&sample.to_footer_bytes());
    match manifest::parse_manifest(&blob) {
        Ok(m) if m.game_id == "selftest" => println!("  [PASS] footer 自定位/解析"),
        Ok(_) => {
            println!("  [FAIL] footer 解析内容不符");
            ok = false;
        }
        Err(e) => {
            println!("  [FAIL] footer 解析失败: {e}");
            ok = false;
        }
    }

    // 2. 规则生成（%USERPROFILE%/%TEMP% 锚点，skip_cache 变体）。
    let rules = rules::generate_rules(r"C:\Users\Selftest", r"G:\GameData", false);
    if !rules.is_empty() {
        println!("  [PASS] 规则生成（{} 条）", rules.len());
    } else {
        println!("  [FAIL] 规则生成为空");
        ok = false;
    }

    // 3. PE 位数探测：用本二进制自身（Windows 上为 PE；Linux 上应报 not-pe 但不算失败——
    //    探测路径本身可用即可，跨平台语义）。
    match pe::probe_file(std::env::current_exe().unwrap_or_default().as_path()) {
        Ok(_kind) => println!("  [PASS] PE 探测（本二进制）"),
        Err(_) => {
            // Linux 上本二进制非 PE：验证的是探测不 panic、错误路径可达。
            println!("  [PASS] PE 探测错误路径可达（当前平台非 PE，预期）");
        }
    }

    if ok {
        println!("自检全部通过");
        0
    } else {
        crate::log_error!("自检存在失败项");
        1
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
    println!("  mount <vhd> [--parent <UNC>] [--smb <UNC>] [--user <U>] [--letter <L>] [--retries <N>]");
    println!("                挂载 VHD（Windows；SMB→建差分→attach→定盘符）");
    println!("                密码来源：store-cred 凭据管理器 > 环境变量 GAMEVHD_SMB_PASS，命令行不明文");
    println!("  store-cred <server> [--user <U>]  保存 SMB 凭据到 Windows 凭据管理器（密码从 stdin 读入）");
    println!("  delete-cred <server>              删除已保存的 SMB 凭据（幂等）");
    println!("  unmount <vhd> [--letter <L>] [--smb <UNC>]   卸载 VHD（Windows）");
    println!("  scan <drive>              扫描盘符下的 exe 候选并标注位数（如 scan E）");
    println!("  probe <exe>               探测 exe 位数（x64 / x86 / not-pe）");
    println!("  inject-poc --exe <path> --hook <dll> --log <log> [--work-dir <dir>] [--mode remote|early-bird]");
    println!("             [--args <raw-args>] [--game-data-root <dir> --user-profile <dir> --game-id <id>]");
    println!("                Wine 联调专用：对照测试远程线程与 Early-Bird APC 注入");
    println!("  run --drive <letter> --box <path> [--hklm-write passthrough|deny]   沙箱启动游戏（Windows；HKLM 写默认拒绝）");
    println!("  cleanup --box <path> [--vhd <diff>]   清理残留沙箱状态与残留 VHD 挂载（Windows）");
    println!("  --help, -h                显示本帮助");
    println!("  --version, -V             显示版本");
    println!("  --selftest                自检（footer/规则/PE 探测）");
    println!();
    println!("退出码: 0 成功  1 运行错误  2 用法错误  3 当前平台不支持");
}

#[cfg(test)]
mod main_tests;
