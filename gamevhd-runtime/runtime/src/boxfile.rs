//! 沙箱状态文件 `box.json` 模型（阶段 0，本波交付；§3.6 定案）。
//!
//! 字段：`game_id`、`exe_relative`、`user_profile`、`state` 必填；
//! `game_data_root`、`registry_hive`、`skip_cache_dirs`、`game_data_base`、
//! `game_data_name` 可选。空的 `game_data_root` 表示使用 VHD 内的
//! `<drive>:\GameData`；设置 `game_data_base` 后，运行时会追加游戏目录名。
//! 状态机：`clean → running → cleaning → clean`，非法迁移报错。
//! 原子保存：写同目录临时文件 + `rename`；Windows 下目标已存在时先删后换。
//! JSON 编解码委托 [`crate::json`]（零依赖手写子集解析器）。

// 本模块由后续阶段（4：run/cleanup 生命周期）消费；当前 wave 无生产调用方，
// 全部 API 仅被测试使用，按模块整体豁免 dead_code。
#![allow(dead_code)]

use std::fmt;
use std::fs;
use std::io;
use std::path::Path;

use crate::json::{escape_json, parse_json_object};

/// box.json 的标准文件名（VHD 内 `GameData\box.json`）。
pub const BOX_FILE_NAME: &str = "box.json";

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BoxState {
    Clean,
    Running,
    Cleaning,
}

impl BoxState {
    pub fn as_str(self) -> &'static str {
        match self {
            BoxState::Clean => "clean",
            BoxState::Running => "running",
            BoxState::Cleaning => "cleaning",
        }
    }

    pub fn from_str(s: &str) -> Option<BoxState> {
        match s {
            "clean" => Some(BoxState::Clean),
            "running" => Some(BoxState::Running),
            "cleaning" => Some(BoxState::Cleaning),
            _ => None,
        }
    }
}

impl fmt::Display for BoxState {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.as_str())
    }
}

/// box.json 模型。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BoxFile {
    pub game_id: String,
    /// exe 相对 VHD 根（如 `Game\HorizonZeroDawn.exe`）。
    pub exe_relative: String,
    /// 沙箱重定向根（如 `E:\GameData`）；为空时由 runtime 解析默认根或外部根。
    pub game_data_root: String,
    /// 生成规则表时的 `%USERPROFILE%`（如 `C:\Users\Hao`）。
    pub user_profile: String,
    /// hive 路径；为空时放在解析后的 GameData 根下。
    pub registry_hive: String,
    /// 为 `true` 时规则表排除 AppData 重写（直通宿主缓存）。
    pub skip_cache_dirs: bool,
    pub state: BoxState,
    /// 外部状态库的父目录（如 `D:\GameAtlas`），为空时不启用外部状态库。
    pub game_data_base: String,
    /// 外部状态库的游戏目录名（如 `地平线`）；为空时由启动器配置推导。
    pub game_data_name: String,
    /// 外部状态库目录名附带用户标识（审计 P2-9 多用户隔离）；缺省 false 兼容旧布局。
    pub user_namespace: bool,
}

/// box.json 相关错误。
#[derive(Debug)]
pub enum BoxError {
    Io(io::Error),
    InvalidJson(String),
    InvalidTransition { from: BoxState, to: BoxState },
}

// `io::Error` 不实现 PartialEq（1.97 起移除），Io 变体按错误类别比较。
impl PartialEq for BoxError {
    fn eq(&self, other: &Self) -> bool {
        match (self, other) {
            (BoxError::Io(a), BoxError::Io(b)) => a.kind() == b.kind(),
            (BoxError::InvalidJson(a), BoxError::InvalidJson(b)) => a == b,
            (
                BoxError::InvalidTransition { from: af, to: at },
                BoxError::InvalidTransition { from: bf, to: bt },
            ) => af == bf && at == bt,
            _ => false,
        }
    }
}

impl Eq for BoxError {}

impl fmt::Display for BoxError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            BoxError::Io(e) => write!(f, "box.json I/O 错误: {e}"),
            BoxError::InvalidJson(msg) => write!(f, "box.json 非法: {msg}"),
            BoxError::InvalidTransition { from, to } => {
                write!(f, "非法状态迁移: {from} -> {to}")
            }
        }
    }
}

impl std::error::Error for BoxError {}

impl BoxFile {
    /// 状态机迁移。合法边：Clean→Running、Running→Cleaning、Cleaning→Clean。
    pub fn transition(&mut self, to: BoxState) -> Result<(), BoxError> {
        let from = self.state;
        let valid = matches!(
            (from, to),
            (BoxState::Clean, BoxState::Running)
                | (BoxState::Running, BoxState::Cleaning)
                | (BoxState::Cleaning, BoxState::Clean)
        );
        if valid {
            self.state = to;
            Ok(())
        } else {
            Err(BoxError::InvalidTransition { from, to })
        }
    }

    /// 序列化为 box.json 文本（2 空格缩进，与 §3.6 示例一致）。
    pub fn to_json(&self) -> String {
        format!(
            "{{\n  \"game_id\": \"{}\",\n  \"exe_relative\": \"{}\",\n  \"game_data_root\": \"{}\",\n  \"user_profile\": \"{}\",\n  \"registry_hive\": \"{}\",\n  \"skip_cache_dirs\": {},\n  \"state\": \"{}\",\n  \"game_data_base\": \"{}\",\n  \"game_data_name\": \"{}\",\n  \"user_namespace\": {}\n}}",
            escape_json(&self.game_id),
            escape_json(&self.exe_relative),
            escape_json(&self.game_data_root),
            escape_json(&self.user_profile),
            escape_json(&self.registry_hive),
            if self.skip_cache_dirs { "true" } else { "false" },
            escape_json(self.state.as_str()),
            escape_json(&self.game_data_base),
            escape_json(&self.game_data_name),
            if self.user_namespace { "true" } else { "false" },
        )
    }

    /// 从 box.json 文本解析；必填字段缺失/未知/重复均报错。
    /// 路径覆盖字段和 `skip_cache_dirs` 可省略，省略时使用默认策略。
    pub fn from_json(s: &str) -> Result<BoxFile, BoxError> {
        let mut bf = BoxFile {
            game_id: String::new(),
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
        let mut seen = [false; 10];
        for (key, value) in parse_json_object(s).map_err(BoxError::InvalidJson)? {
            let idx = match key.as_str() {
                "game_id" => 0,
                "exe_relative" => 1,
                "game_data_root" => 2,
                "user_profile" => 3,
                "registry_hive" => 4,
                "skip_cache_dirs" => 5,
                "state" => 6,
                "game_data_base" => 7,
                "game_data_name" => 8,
                "user_namespace" => 9,
                other => return Err(BoxError::InvalidJson(format!("未知字段 '{other}'"))),
            };
            if seen[idx] {
                return Err(BoxError::InvalidJson(format!("重复字段 '{key}'")));
            }
            seen[idx] = true;
            match idx {
                0 => bf.game_id = value,
                1 => bf.exe_relative = value,
                2 => bf.game_data_root = value,
                3 => bf.user_profile = value,
                4 => bf.registry_hive = value,
                5 => {
                    bf.skip_cache_dirs = match value.as_str() {
                        "true" => true,
                        "false" => false,
                        other => {
                            return Err(BoxError::InvalidJson(format!(
                                "非法 skip_cache_dirs 值 '{other}'"
                            )))
                        }
                    }
                }
                6 => {
                    bf.state = BoxState::from_str(&value).ok_or_else(|| {
                        BoxError::InvalidJson(format!("非法 state 值 '{value}'"))
                    })?
                }
                7 => bf.game_data_base = value,
                8 => bf.game_data_name = value,
                9 => {
                    bf.user_namespace = match value.as_str() {
                        "true" => true,
                        "false" => false,
                        other => {
                            return Err(BoxError::InvalidJson(format!(
                                "非法 user_namespace 值 '{other}'"
                            )))
                        }
                    }
                }
                _ => unreachable!("box.json 字段索引超出范围"),
            }
        }
        const REQUIRED: [(&str, usize); 4] = [
            ("game_id", 0),
            ("exe_relative", 1),
            ("user_profile", 3),
            ("state", 6),
        ];
        for (name, i) in REQUIRED {
            if !seen[i] {
                return Err(BoxError::InvalidJson(format!("缺少字段 '{name}'")));
            }
        }
        Ok(bf)
    }

    /// 原子保存：写 `<path>.tmp` 后 rename。Windows 下目标存在时先删再换
    /// （`fs::rename` 不覆盖已存在文件），牺牲瞬时原子性换取可用性。
    pub fn save(&self, path: &Path) -> Result<(), BoxError> {
        let mut tmp = path.as_os_str().to_os_string();
        tmp.push(".tmp");
        let tmp = Path::new(&tmp);
        fs::write(tmp, self.to_json()).map_err(BoxError::Io)?;
        match fs::rename(tmp, path) {
            Ok(()) => Ok(()),
            Err(_e) if path.exists() => {
                // 目标已存在（Windows 语义）：删旧再换。
                fs::remove_file(path).map_err(BoxError::Io)?;
                fs::rename(tmp, path).map_err(BoxError::Io)
            }
            Err(e) => Err(BoxError::Io(e)),
        }
    }

    /// 从磁盘读取并解析。
    pub fn load(path: &Path) -> Result<BoxFile, BoxError> {
        let s = fs::read_to_string(path).map_err(BoxError::Io)?;
        BoxFile::from_json(&s)
    }
}

#[cfg(test)]
#[path = "boxfile_tests.rs"]
mod boxfile_tests;
