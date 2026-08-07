//! 注册表路径重写 —— 归属：阶段 3 注册表重定向（W3T3 路径重写）。
//!
//! 职责（§3.5 / injection_protocol §6.5）：native 路径
//! `\REGISTRY\USER\<SID>\Software\X` → `\REGISTRY\USER\GameVHD_<game_id>\Software\X`；
//! 仅重写 HKCU 的 Software 子树；HKCU 其他根（Environment / Control Panel…）与
//! `.DEFAULT` 用户直通（None）；`\REGISTRY\MACHINE`（HKLM）直通宿主并记录警告。
//! 32 位游戏的 `Wow6432Node` 随 `Software` 子树自然重写。
//! 纯逻辑、无平台依赖，Linux 可单测。

#![allow(dead_code)] // main.rs 尚未接线（W4T16 拥有 run/cleanup 分发）

/// 注册表 native 路径分类（hook / 编排按类决策：重写 / 直通 / 记录警告）。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RegPathKind {
    /// `\REGISTRY\USER\<SID>\Software[...]` —— 重写目标。
    HkcuSoftware,
    /// HKCU 下非 Software 子树（Environment / Control Panel…）或 `.DEFAULT` 用户 —— 直通。
    HkcuOther,
    /// `\REGISTRY\MACHINE[...]` —— 直通宿主（记录警告不隔离）。
    Hklm,
    /// 非 `\REGISTRY\...` 前缀（如 NT 对象路径）—— 不处理。
    Other,
}

/// 前缀常量统一小写：只用于对 `to_ascii_lowercase()` 后的输入做大小写不敏感匹配。
const USER_PREFIX: &str = r"\registry\user\";
const MACHINE_PREFIX: &str = r"\registry\machine\";

/// 重写 HKCU Software 子树 native 路径到隔离 hive 键名：
/// `\REGISTRY\USER\<SID>\Software\X` → `\REGISTRY\USER\GameVHD_<game_id>\Software\X`。
///
/// 命中规则（全部满足才重写）：
/// - 前缀 `\REGISTRY\USER\`（大小写不敏感）；
/// - SID 段存在且不是 `.DEFAULT`（不同 SID 用户）；
/// - SID 后余段以 `Software`（大小写不敏感）开头或恰为 `Software`；
/// - `game_id` 非空。
/// 其余情况（HKLM / HKCU 非 Software / `.DEFAULT` / game_id 为空 / 非 REGISTRY 前缀）
/// 返回 `None`（直通）。命中时前缀改写为规范大写，余段保留原样大小写。
pub fn rewrite_native_reg_path(native: &str, game_id: &str) -> Option<String> {
    if game_id.is_empty() {
        return None;
    }
    let lower = native.to_ascii_lowercase();
    if !lower.starts_with(USER_PREFIX) {
        return None;
    }
    let rest = &native[USER_PREFIX.len()..];
    let sid_end = rest.find('\\')?;
    if rest[..sid_end].eq_ignore_ascii_case(".DEFAULT") {
        return None;
    }
    let remainder = &rest[sid_end + 1..];
    let rem_lower = remainder.to_ascii_lowercase();
    if rem_lower != "software" && !rem_lower.starts_with("software\\") {
        return None;
    }
    Some(format!(r"\REGISTRY\USER\GameVHD_{game_id}\{remainder}"))
}

/// 分类 native 路径：`HkcuSoftware` / `HkcuOther` / `Hklm` / `Other`。
pub fn classify_reg_path(native: &str) -> RegPathKind {
    let lower = native.to_ascii_lowercase();
    if lower.starts_with(MACHINE_PREFIX) {
        return RegPathKind::Hklm;
    }
    if !lower.starts_with(USER_PREFIX) {
        return RegPathKind::Other;
    }
    let rest = &native[USER_PREFIX.len()..];
    let Some(sid_end) = rest.find('\\') else {
        return RegPathKind::HkcuOther;
    };
    if rest[..sid_end].eq_ignore_ascii_case(".DEFAULT") {
        return RegPathKind::HkcuOther;
    }
    let remainder = &rest[sid_end + 1..];
    let rem_lower = remainder.to_ascii_lowercase();
    if rem_lower == "software" || rem_lower.starts_with("software\\") {
        RegPathKind::HkcuSoftware
    } else {
        RegPathKind::HkcuOther
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const SID: &str = "S-1-5-21-123-456-1001";

    #[test]
    fn rewrite_full_example() {
        let native = format!(r"\REGISTRY\USER\{SID}\Software\Bethesda\Test Game");
        assert_eq!(
            rewrite_native_reg_path(&native, "horizon-zero-dawn"),
            Some(
                r"\REGISTRY\USER\GameVHD_horizon-zero-dawn\Software\Bethesda\Test Game"
                    .to_string()
            )
        );
    }

    #[test]
    fn rewrite_exact_software_root_without_trailing_segments() {
        let native = format!(r"\REGISTRY\USER\{SID}\Software");
        assert_eq!(
            rewrite_native_reg_path(&native, "g1"),
            Some(r"\REGISTRY\USER\GameVHD_g1\Software".to_string())
        );
    }

    #[test]
    fn rewrite_case_insensitive_prefix_preserves_remainder_case() {
        let native = format!(r"\registry\user\{SID}\software\x");
        assert_eq!(
            rewrite_native_reg_path(&native, "g1"),
            Some(r"\REGISTRY\USER\GameVHD_g1\software\x".to_string())
        );
    }

    #[test]
    fn rewrite_sid_with_many_hyphens_parses_four_segments() {
        let native = r"\REGISTRY\USER\S-1-5-21-1234567890-987654321-55555\Software\X";
        assert_eq!(
            rewrite_native_reg_path(native, "g1"),
            Some(r"\REGISTRY\USER\GameVHD_g1\Software\X".to_string())
        );
    }

    #[test]
    fn non_software_hkcu_returns_none() {
        let native = format!(r"\REGISTRY\USER\{SID}\Environment\Path");
        assert_eq!(rewrite_native_reg_path(&native, "g1"), None);
        let native2 = format!(r"\REGISTRY\USER\{SID}\Control Panel\Desktop");
        assert_eq!(rewrite_native_reg_path(&native2, "g1"), None);
    }

    #[test]
    fn hklm_returns_none() {
        assert_eq!(
            rewrite_native_reg_path(r"\REGISTRY\MACHINE\SOFTWARE\Microsoft", "g1"),
            None
        );
    }

    #[test]
    fn default_user_returns_none() {
        assert_eq!(
            rewrite_native_reg_path(r"\REGISTRY\USER\.DEFAULT\Software\X", "g1"),
            None
        );
    }

    #[test]
    fn empty_game_id_returns_none() {
        let native = format!(r"\REGISTRY\USER\{SID}\Software\X");
        assert_eq!(rewrite_native_reg_path(&native, ""), None);
    }

    #[test]
    fn non_registry_prefix_returns_none() {
        assert_eq!(
            rewrite_native_reg_path(r"\Device\HarddiskVolume1\Software", "g1"),
            None
        );
        assert_eq!(rewrite_native_reg_path(r"C:\foo\Software", "g1"), None);
    }

    #[test]
    fn classify_path_kinds() {
        let sid = SID;
        assert_eq!(
            classify_reg_path(&format!(r"\REGISTRY\USER\{sid}\Software\Bethesda")),
            RegPathKind::HkcuSoftware
        );
        assert_eq!(
            classify_reg_path(&format!(r"\REGISTRY\USER\{sid}\Software")),
            RegPathKind::HkcuSoftware
        );
        assert_eq!(
            classify_reg_path(&format!(r"\REGISTRY\USER\{sid}\Environment")),
            RegPathKind::HkcuOther
        );
        assert_eq!(
            classify_reg_path(&format!(r"\REGISTRY\USER\{sid}")),
            RegPathKind::HkcuOther
        );
        assert_eq!(
            classify_reg_path(r"\REGISTRY\USER\.DEFAULT\Software"),
            RegPathKind::HkcuOther
        );
        assert_eq!(
            classify_reg_path(r"\REGISTRY\MACHINE\SOFTWARE\Microsoft"),
            RegPathKind::Hklm
        );
        assert_eq!(classify_reg_path(r"C:\foo\Software"), RegPathKind::Other);
        assert_eq!(classify_reg_path(r"\REGISTRY\USER"), RegPathKind::Other);
    }
}
