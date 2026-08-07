//! 线程安全日志器（阶段 0，本波交付）。
//!
//! 级别：INFO / WARN / ERROR；行格式：`[YYYY-MM-DD HH:MM:SS.mmm] [LEVEL] 消息`。
//! 输出：始终写 stderr；若环境变量 `GVHD_LOG` 指向一个文件路径，则同时追加写该文件
//! （父目录自动创建，打开失败降级为仅 stderr 并在 stderr 上提示）。
//!
//! 惰性初始化：`log()` 系列函数在未调用 [`init`] 前只写 stderr（测试友好）；
//! [`init`] 由 main 启动时调用一次（OnceLock，全局唯一）。
//! 纯 std 实现：`Mutex` + `io::Write` + `SystemTime`，无外部 crate。

use std::fs::OpenOptions;
use std::io::Write;
use std::path::PathBuf;
use std::sync::{Mutex, OnceLock};
use std::time::{SystemTime, UNIX_EPOCH};

/// 日志级别。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LogLevel {
    // Linux 非测试构建中 INFO 尚无调用方（Windows 分支与测试使用）。
    #[cfg_attr(not(target_os = "windows"), allow(dead_code))]
    Info,
    Warn,
    Error,
}

impl LogLevel {
    pub fn as_str(self) -> &'static str {
        match self {
            LogLevel::Info => "INFO",
            LogLevel::Warn => "WARN",
            LogLevel::Error => "ERROR",
        }
    }
}

struct Logger {
    /// 追加写目标；None = 仅 stderr（GVHD_LOG 未设置或打开失败）。
    file: Mutex<Option<std::fs::File>>,
}

static LOGGER: OnceLock<Logger> = OnceLock::new();

/// 初始化全局日志器：读取 `GVHD_LOG` 打开日志文件。幂等，重复调用只生效第一次。
pub fn init() {
    LOGGER.get_or_init(Logger::new);
}

impl Logger {
    fn new() -> Self {
        let file = match std::env::var("GVHD_LOG") {
            Ok(path) if !path.is_empty() => {
                let p = PathBuf::from(&path);
                if let Some(parent) = p.parent() {
                    if !parent.as_os_str().is_empty() {
                        let _ = std::fs::create_dir_all(parent);
                    }
                }
                match OpenOptions::new().create(true).append(true).open(&p) {
                    Ok(f) => Some(f),
                    Err(e) => {
                        eprintln!(
                            "log: 无法打开 GVHD_LOG 文件 '{path}': {e}；降级为仅 stderr"
                        );
                        None
                    }
                }
            }
            _ => None,
        };
        Logger {
            file: Mutex::new(file),
        }
    }
}

fn log(level: LogLevel, msg: &str) {
    let line = format!("[{}] [{}] {}\n", timestamp(), level.as_str(), msg);
    let mut stderr = std::io::stderr().lock();
    let _ = stderr.write_all(line.as_bytes());
    if let Some(logger) = LOGGER.get() {
        if let Ok(mut guard) = logger.file.lock() {
            if let Some(f) = guard.as_mut() {
                let _ = f.write_all(line.as_bytes());
            }
        }
    }
}

/// 记录一条 INFO。
#[cfg_attr(not(target_os = "windows"), allow(dead_code))]
pub fn info(msg: &str) {
    log(LogLevel::Info, msg);
}

/// 记录一条 WARN。
pub fn warn(msg: &str) {
    log(LogLevel::Warn, msg);
}

/// 记录一条 ERROR。
pub fn error(msg: &str) {
    log(LogLevel::Error, msg);
}

/// 带格式化参数的便捷宏（内部 `format!`）。
#[macro_export]
macro_rules! log_info {
    ($($arg:tt)*) => {
        $crate::log::info(&format!($($arg)*))
    };
}

/// 带格式化参数的便捷宏（内部 `format!`）。
#[macro_export]
macro_rules! log_warn {
    ($($arg:tt)*) => {
        $crate::log::warn(&format!($($arg)*))
    };
}

/// 带格式化参数的便捷宏（内部 `format!`）。
#[macro_export]
macro_rules! log_error {
    ($($arg:tt)*) => {
        $crate::log::error(&format!($($arg)*))
    };
}

/// 本地时间 `YYYY-MM-DD HH:MM:SS.mmm`（Howard Hinnant civil_from_days 算法，无外部 crate）。
fn timestamp() -> String {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default();
    let secs = now.as_secs() as i64;
    let millis = now.subsec_millis();
    let days = secs.div_euclid(86_400);
    let tod = secs.rem_euclid(86_400);
    let (y, mo, d) = civil_from_days(days);
    let (h, mi, s) = (tod / 3600, (tod % 3600) / 60, tod % 60);
    format!("{y:04}-{mo:02}-{d:02} {h:02}:{mi:02}:{s:02}.{millis:03}")
}

/// 天数（自 1970-01-01）→ (年, 月, 日)。Hinnant 算法。
fn civil_from_days(z: i64) -> (i64, u32, u32) {
    let z = z + 719_468;
    let era = if z >= 0 { z } else { z - 146_096 } / 146_097;
    let doe = z - era * 146_097;
    let yoe = (doe - doe / 1460 + doe / 36_524 - doe / 146_096) / 365;
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    (if m <= 2 { y + 1 } else { y }, m as u32, d as u32)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn tmp_log_path(name: &str) -> PathBuf {
        std::env::temp_dir().join(format!("gamevhd_log_test_{name}_{}.log", std::process::id()))
    }

    /// 仅此测试调用 init()（OnceLock 全局唯一），其他测试不触碰日志器。
    #[test]
    fn log_writes_to_file_with_timestamp_and_level() {
        let path = tmp_log_path("file");
        let _ = fs::remove_file(&path);
        std::env::set_var("GVHD_LOG", &path);
        init();

        crate::log_info!("hello {}", 42);
        crate::log_warn!("careful");
        crate::log_error!("boom");

        let content = fs::read_to_string(&path).expect("日志文件应已创建");
        let lines: Vec<&str> = content.lines().collect();
        // 并行测试中其他 run_cli 测试可能向全局日志器追加行，故只断言本测试三行存在
        // （格式与内容不变），而非精确行数。
        assert!(
            lines
                .iter()
                .any(|l| l.starts_with('[') && l.contains("[INFO] hello 42")),
            "应有 INFO 行：\n{content}"
        );
        assert!(
            lines.iter().any(|l| l.contains("[WARN] careful")),
            "应有 WARN 行：\n{content}"
        );
        assert!(
            lines.iter().any(|l| l.contains("[ERROR] boom")),
            "应有 ERROR 行：\n{content}"
        );

        let _ = fs::remove_file(&path);
        std::env::remove_var("GVHD_LOG");
    }

    #[test]
    fn timestamp_shape_is_valid() {
        let ts = timestamp();
        let bytes = ts.as_bytes();
        assert_eq!(bytes.len(), 23, "YYYY-MM-DD HH:MM:SS.mmm：'{ts}'");
        assert_eq!(&ts[4..5], "-");
        assert_eq!(&ts[10..11], " ");
        assert_eq!(&ts[19..20], ".");
        assert!(ts.starts_with("2026-"), "今年（2026）：{ts}");
    }
}
