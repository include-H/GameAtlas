//! 注册表 hive 管理 —— 归属：阶段 3 注册表重定向（W3T1 模板复制 / W3T2 挂载卸载）。
//!
//! 职责（§3.5）：hive 自举（文件缺失时在 Windows 侧现场生成空模板：RegCreateKeyExW +
//! RegSaveKeyW + RegDeleteKeyW，RegSaveKeyW 需 SE_BACKUP_NAME 特权）→ `.bak` 轮换 →
//! `RegLoadKey(HKUS, "GameVHD_<game_id>", ...)` 挂载 / `RegUnLoadKey` 卸载（ERROR_BUSY
//! 句柄等待重试）与崩溃残留清理（启动时先幂等卸载）；hive 损坏时从 `.bak` 恢复并警告。
//!
//! Windows 专属实现；Linux 上挂载/卸载/清理返回 [`HiveError::UnsupportedPlatform`]，
//! 键名校验为纯逻辑跨平台可测。main.rs 尚未接线（W4T16 拥有），故允许 dead_code。

#![allow(dead_code)]

use std::error::Error;
use std::fmt;

/// hive 挂载/卸载/清理错误。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum HiveError {
    /// 非 Windows 平台不支持 hive 挂载。
    UnsupportedPlatform,
    /// Win32 API 失败：op = 失败调用点，code = LSTATUS / GetLastError。
    Win32 { op: &'static str, code: u32 },
    /// game_id 不合法（注册表键名约束）。
    InvalidKeyName(String),
}

impl fmt::Display for HiveError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            HiveError::UnsupportedPlatform => {
                write!(f, "注册表 hive 挂载/卸载仅支持 Windows")
            }
            HiveError::Win32 { op, code } => write!(f, "{op} 失败：Win32 错误码 {code}"),
            HiveError::InvalidKeyName(reason) => write!(f, "无效的 game_id：{reason}"),
        }
    }
}

impl Error for HiveError {}

/// 隔离 hive 句柄：`key_name = GameVHD_<game_id>`，挂载于 `HKU\<key_name>`。
#[derive(Debug)]
pub struct Hive {
    key_name: String,
    hive_path: String,
}

/// 校验 game_id 可用于注册表键名后缀 `GameVHD_<game_id>`：拒绝空串、`\`、`/`、`*`、`?`
/// 与控制字符。纯逻辑、跨平台可测。
pub fn validate_key_name(game_id: &str) -> Result<(), HiveError> {
    if game_id.is_empty() {
        return Err(HiveError::InvalidKeyName("game_id 不能为空".into()));
    }
    for ch in game_id.chars() {
        if ch.is_control() || matches!(ch, '\\' | '/' | '*' | '?') {
            return Err(HiveError::InvalidKeyName(format!(
                "game_id '{game_id}' 含非法字符 {ch:?}（不允许 \\ / * ? 与控制字符）"
            )));
        }
    }
    Ok(())
}

/// 挂载隔离 hive：不存在时自举生成模板 → 轮换 `.bak` →
/// `RegLoadKey(HKUS, "GameVHD_<game_id>", hive_path)`；损坏时从 `.bak` 恢复重试一次。
pub fn mount_hive(game_id: &str, hive_path: &str) -> Result<Hive, HiveError> {
    #[cfg(target_os = "windows")]
    {
        win::mount_hive_impl(game_id, hive_path)
    }
    #[cfg(not(target_os = "windows"))]
    {
        let _ = (game_id, hive_path);
        Err(HiveError::UnsupportedPlatform)
    }
}

impl Hive {
    /// 卸载隔离 hive：`RegUnLoadKey`，ERROR_BUSY（句柄未释放）等待重试后报错。
    pub fn unmount_hive(&self) -> Result<(), HiveError> {
        #[cfg(target_os = "windows")]
        {
            win::unmount_hive_impl(&self.key_name)
        }
        #[cfg(not(target_os = "windows"))]
        {
            let _ = self;
            Err(HiveError::UnsupportedPlatform)
        }
    }
}

/// 幂等清理崩溃残留：`RegUnLoadKey(HKUS, "GameVHD_<game_id>")`，忽略
/// ERROR_FILE_NOT_FOUND（本未挂载即视为已清理）。启动时调用一次。
pub fn cleanup_residue(game_id: &str) -> Result<(), HiveError> {
    #[cfg(target_os = "windows")]
    {
        win::cleanup_residue_impl(game_id)
    }
    #[cfg(not(target_os = "windows"))]
    {
        let _ = game_id;
        Err(HiveError::UnsupportedPlatform)
    }
}

#[cfg(target_os = "windows")]
mod win {
    use super::{Hive, HiveError, validate_key_name};
    use crate::winffi::{
        AdjustTokenPrivileges, CloseHandle, ERROR_BUSY, ERROR_FILE_NOT_FOUND, ERROR_INVALID_PARAMETER,
        ERROR_NOT_ALL_ASSIGNED, ERROR_SUCCESS, GetCurrentProcess, GetLastError, HANDLE,
        HKEY_CURRENT_USER, HKEY_USERS, KEY_ALL_ACCESS, LookupPrivilegeValueW, Luid,
        LuidAndAttributes, OpenProcessToken, RegCloseKey, RegCreateKeyExW, RegDeleteKeyW,
        RegLoadKeyW, RegSaveKeyW, RegUnLoadKeyW, SE_BACKUP_NAME, SE_PRIVILEGE_ENABLED,
        SE_RESTORE_NAME, TOKEN_ADJUST_PRIVILEGES, TOKEN_QUERY, TokenPrivileges,
        REG_OPTION_NON_VOLATILE,
    };

    /// RegUnLoadKey ERROR_BUSY 重试次数与间隔。
    const UNLOAD_RETRIES: u32 = 10;
    const UNLOAD_RETRY_MS: u64 = 500;

    /// 自举模板键名（HKCU 下临时键，保存成 hive 文件后删除）。
    const TEMPLATE_KEY: &str = "GVHD_HiveTemplate";

    /// UTF-16 NUL 结尾编码（Win32 宽字符入参）。
    fn to_wide(s: &str) -> Vec<u16> {
        s.encode_utf16().chain(std::iter::once(0)).collect()
    }

    pub(super) fn mount_hive_impl(game_id: &str, hive_path: &str) -> Result<Hive, HiveError> {
        validate_key_name(game_id)?;
        let key_name = format!("GameVHD_{game_id}");

        // RegLoadKeyW 每次都需要 SE_RESTORE_NAME（RegSaveKeyW 需 SE_BACKUP_NAME），
        // 且管理员令牌中该特权默认 disabled——必须在挂载前无条件启用
        // （bootstrap 只在文件缺失时走，但 RegLoadKeyW 与文件是否存在无关）。
        enable_backup_privilege()?;

        // 崩溃残留的模板键：无论本次是否自举都先幂等清理（自举中断可能留下它）。
        let _ = unsafe { RegDeleteKeyW(HKEY_CURRENT_USER, to_wide(TEMPLATE_KEY).as_ptr()) };

        if !std::path::Path::new(hive_path).exists() {
            // 自举前确保父目录存在（RegSaveKeyW 不自动建目录）。
            if let Some(parent) = std::path::Path::new(hive_path).parent() {
                if let Err(e) = std::fs::create_dir_all(parent) {
                    return Err(HiveError::Win32 {
                        op: "create_dir_all(hive 父目录)",
                        code: e.raw_os_error().unwrap_or(0) as u32,
                    });
                }
            }
            bootstrap_hive(hive_path)?;
        }
        rotate_backup(hive_path);

        let key_wide = to_wide(&key_name);
        let path_wide = to_wide(hive_path);
        let mut status = unsafe { RegLoadKeyW(HKEY_USERS, key_wide.as_ptr(), path_wide.as_ptr()) };
        if (status == ERROR_FILE_NOT_FOUND || status == ERROR_INVALID_PARAMETER)
            && restore_from_backup(hive_path)
        {
            crate::log_warn!("hive {hive_path} 损坏，已从 .bak 恢复并重试挂载");
            status = unsafe { RegLoadKeyW(HKEY_USERS, key_wide.as_ptr(), path_wide.as_ptr()) };
        }
        if status != ERROR_SUCCESS {
            return Err(HiveError::Win32 {
                op: "RegLoadKeyW",
                code: status as u32,
            });
        }
        Ok(Hive {
            key_name,
            hive_path: hive_path.to_string(),
        })
    }

    /// hive 文件缺失时自举：RegCreateKeyExW(HKCU, GVHD_HiveTemplate) → RegSaveKeyW
    /// （需 SE_BACKUP_NAME）→ RegCloseKey → RegDeleteKeyW。生成的空 hive 几 KB，可挂载。
    /// 特权已在 [`mount_hive_impl`] 入口统一启用，此处不再重复。
    fn bootstrap_hive(hive_path: &str) -> Result<(), HiveError> {
        let template = to_wide(TEMPLATE_KEY);
        let mut hkey: HANDLE = 0;
        let mut disposition: u32 = 0;
        let status = unsafe {
            RegCreateKeyExW(
                HKEY_CURRENT_USER,
                template.as_ptr(),
                0,
                std::ptr::null_mut(),
                REG_OPTION_NON_VOLATILE,
                KEY_ALL_ACCESS,
                std::ptr::null(),
                &mut hkey,
                &mut disposition,
            )
        };
        if status != ERROR_SUCCESS {
            return Err(HiveError::Win32 {
                op: "RegCreateKeyExW",
                code: status as u32,
            });
        }
        let path = to_wide(hive_path);
        let save = unsafe { RegSaveKeyW(hkey, path.as_ptr(), std::ptr::null()) };
        unsafe { RegCloseKey(hkey) };
        if save != ERROR_SUCCESS {
            return Err(HiveError::Win32 {
                op: "RegSaveKeyW",
                code: save as u32,
            });
        }
        let del = unsafe { RegDeleteKeyW(HKEY_CURRENT_USER, template.as_ptr()) };
        if del != ERROR_SUCCESS && del != ERROR_FILE_NOT_FOUND {
            return Err(HiveError::Win32 {
                op: "RegDeleteKeyW",
                code: del as u32,
            });
        }
        crate::log_info!("hive 自举：已生成空 hive 文件 {hive_path}");
        Ok(())
    }

    /// 启用 SE_BACKUP_NAME / SE_RESTORE_NAME（RegSaveKeyW / RegLoadKeyW 分别所需）。
    fn enable_backup_privilege() -> Result<(), HiveError> {
        let mut token: HANDLE = 0;
        if unsafe {
            OpenProcessToken(
                GetCurrentProcess(),
                TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY,
                &mut token,
            )
        } == 0
        {
            return Err(HiveError::Win32 {
                op: "OpenProcessToken",
                code: unsafe { GetLastError() },
            });
        }
        let result =
            enable_privilege(token, &SE_BACKUP_NAME).and_then(|()| enable_privilege(token, &SE_RESTORE_NAME));
        unsafe { CloseHandle(token) };
        result
    }

    fn enable_privilege(token: HANDLE, name: &[u16]) -> Result<(), HiveError> {
        let mut luid = Luid {
            low_part: 0,
            high_part: 0,
        };
        if unsafe { LookupPrivilegeValueW(std::ptr::null(), name.as_ptr(), &mut luid) } == 0 {
            return Err(HiveError::Win32 {
                op: "LookupPrivilegeValueW",
                code: unsafe { GetLastError() },
            });
        }
        let tp = TokenPrivileges {
            privilege_count: 1,
            privileges: [LuidAndAttributes {
                luid,
                attributes: SE_PRIVILEGE_ENABLED,
            }],
        };
        let ok = unsafe {
            AdjustTokenPrivileges(token, 0, &tp, 0, std::ptr::null_mut(), std::ptr::null_mut())
        };
        let last = unsafe { GetLastError() };
        if ok == 0 || last == ERROR_NOT_ALL_ASSIGNED {
            return Err(HiveError::Win32 {
                op: "AdjustTokenPrivileges",
                code: last,
            });
        }
        Ok(())
    }

    /// 会话首次挂载前轮换 `.bak`：`hive_path` 存在且 `.bak` 不存在或更旧时复制（尽力而为）。
    fn rotate_backup(hive_path: &str) {
        use std::path::Path;
        let src = Path::new(hive_path);
        if !src.is_file() {
            return;
        }
        let bak_path = format!("{hive_path}.bak");
        let bak = Path::new(&bak_path);
        let stale = match (std::fs::metadata(bak), src.metadata()) {
            (Ok(b), Ok(s)) => b
                .modified()
                .ok()
                .zip(s.modified().ok())
                .map_or(false, |(b, s)| b < s),
            (Err(_), Ok(_)) => true,
            _ => false,
        };
        if stale {
            let _ = std::fs::copy(src, bak);
        }
    }

    /// 从 `.bak` 恢复损坏的 hive：返回是否成功复制。
    fn restore_from_backup(hive_path: &str) -> bool {
        std::fs::copy(format!("{hive_path}.bak"), hive_path).is_ok()
    }

    pub(super) fn unmount_hive_impl(key_name: &str) -> Result<(), HiveError> {
        let key_wide = to_wide(key_name);
        let mut status = unsafe { RegUnLoadKeyW(HKEY_USERS, key_wide.as_ptr()) };
        let mut retries = 0u32;
        while status == ERROR_BUSY && retries < UNLOAD_RETRIES {
            std::thread::sleep(std::time::Duration::from_millis(UNLOAD_RETRY_MS));
            retries += 1;
            status = unsafe { RegUnLoadKeyW(HKEY_USERS, key_wide.as_ptr()) };
        }
        if status != ERROR_SUCCESS {
            return Err(HiveError::Win32 {
                op: "RegUnLoadKeyW",
                code: status as u32,
            });
        }
        Ok(())
    }

    pub(super) fn cleanup_residue_impl(game_id: &str) -> Result<(), HiveError> {
        validate_key_name(game_id)?;
        let key_name = format!("GameVHD_{game_id}");
        let key_wide = to_wide(&key_name);
        let status = unsafe { RegUnLoadKeyW(HKEY_USERS, key_wide.as_ptr()) };
        match status {
            ERROR_SUCCESS | ERROR_FILE_NOT_FOUND => Ok(()),
            code => Err(HiveError::Win32 {
                op: "RegUnLoadKeyW",
                code: code as u32,
            }),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn validate_key_name_accepts_typical_game_ids() {
        assert!(validate_key_name("horizon-zero-dawn").is_ok());
        assert!(validate_key_name("God_of_War_Ragnarok").is_ok());
        assert!(validate_key_name("half_life2.ep1").is_ok());
    }

    #[test]
    fn validate_key_name_rejects_empty() {
        assert!(matches!(
            validate_key_name(""),
            Err(HiveError::InvalidKeyName(_))
        ));
    }

    #[test]
    fn validate_key_name_rejects_backslash() {
        assert!(validate_key_name("a\\b").is_err());
        assert!(validate_key_name(r"foo\bar").is_err());
    }

    #[test]
    fn validate_key_name_rejects_slash_star_question() {
        assert!(validate_key_name("a/b").is_err());
        assert!(validate_key_name("a*b").is_err());
        assert!(validate_key_name("a?b").is_err());
    }

    #[test]
    fn validate_key_name_rejects_control_chars() {
        assert!(validate_key_name("a\nb").is_err());
        assert!(validate_key_name("a\tb").is_err());
        assert!(validate_key_name("a\u{0}b").is_err());
        assert!(validate_key_name("a\u{1f}b").is_err());
    }

    #[test]
    fn non_windows_mount_unsupported() {
        #[cfg(not(target_os = "windows"))]
        {
            assert!(matches!(
                mount_hive("g1", r"E:\GameData\Registry\user.dat"),
                Err(HiveError::UnsupportedPlatform)
            ));
        }
    }
}
