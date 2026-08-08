//! 重写规则表生成与协议字节序列化（W2T7）。
//!
//! 依据 `docs/injection_protocol.md` §3/§4 生成 gvhook 注入参数块与规则表：
//!
//! - 规则表采用**模式匹配**（PATTERN MATCHING）而非逐游戏枚举（用户确认的规格变更）：
//!   - 直通区（必须**先**匹配）：`C:\Windows\`、`C:\Program Files\`、`C:\Program Files (x86)\`；
//!   - 重写区（相对 `%USERPROFILE%`）：`Documents`、`Saved Games`、`AppData`（当
//!     `skip_cache_dirs = true` 时 **AppData 规则被排除** → 直通宿主，退出期宿主缓存
//!     清理由后续波次负责，此处仅记录警告）。
//! - `\\?\` 长路径前缀：每条规则同时生成**明文与 `\\?\` 两种变体**（`\\?\C:\...`）；
//!   重写目标（沙箱根 `<GameDataRoot>\Users\<u>\...`）路径较短、无需 `\\?\`，
//!   因此 `\\?\` 变体与明文变体共享同一 replacement（匹配掉前缀后余量自然不带 `\\?\`）。
//! - 序列化为协议字节布局：参数块 5280B + 每条规则 4104B
//!   （UTF-16LE WCHAR 数组 + u32 标志），与 C 头文件 `_Static_assert` 逐字节一致。
//!
//! 全部为纯逻辑（Windows 路径字符串在 Linux 上仅是字符串），无任何平台 cfg。

#![allow(dead_code)]

/// 协议 magic：磁盘字节 `47 56 48 44` = ASCII `G V H D`。
pub const GVHD_PARAM_MAGIC: u32 = 0x4448_5647;
/// 协议版本。
pub const GVHD_PROTOCOL_VERSION: u32 = 1;
/// 路径缓冲（WCHAR 数）。
pub const GVHD_PATH_MAX: usize = 512;
/// game_id 缓冲（WCHAR 数）。
pub const GVHD_GAME_ID_MAX: usize = 64;
/// 规则字符串缓冲（WCHAR 数）。
pub const GVHD_RULE_STRING_MAX: usize = 1024;
/// 最大规则条数。
pub const GVHD_RULE_MAX: u32 = 32;
/// 规则动作标志：命中 → 重写到 `replacement`。
pub const GVHD_RULE_FLAG_REWRITE: u32 = 0x0000_0001;
/// 参数 flags 中实际游戏 VHD 盘符的起始位（1=A，26=Z）。
pub const GVHD_PARAM_FLAG_GAME_DRIVE_SHIFT: u32 = 8;
/// 参数 flags 中实际游戏 VHD 盘符的掩码。
pub const GVHD_PARAM_FLAG_GAME_DRIVE_MASK: u32 = 0x1f << GVHD_PARAM_FLAG_GAME_DRIVE_SHIFT;
/// 规则动作标志：命中 → 不重写（直通宿主/原路径）。
pub const GVHD_RULE_FLAG_PASSTHROUGH: u32 = 0x0000_0002;
/// 参数块大小（字节）。
pub const PARAM_BLOCK_SIZE: usize = 5280;
/// 单条规则条目大小（字节）。
pub const RULE_ENTRY_SIZE: usize = 4104;

/// 参数块内各 WCHAR 字段的字节偏移（协议 §3）。
pub const OFFSET_HOOK_DLL_PATH: usize = 32;
pub const OFFSET_GAME_DATA_ROOT: usize = 1056;
pub const OFFSET_USER_PROFILE: usize = 2080;
pub const OFFSET_LOG_PATH: usize = 3104;
pub const OFFSET_REGISTRY_HIVE: usize = 4128;
pub const OFFSET_GAME_ID: usize = 5152;

/// 一条重写规则。
///
/// `prefix`：绝对路径匹配前缀，大小写不敏感、表序优先（第一条命中决定动作）。
/// `replacement`：REWRITE 时的替换前缀；PASSTHROUGH 时为空串。
/// `flags`：[`GVHD_RULE_FLAG_REWRITE`] 或 [`GVHD_RULE_FLAG_PASSTHROUGH`]。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Rule {
    pub prefix: String,
    pub replacement: String,
    pub flags: u32,
}

/// 单条规则条目大小：4104 字节（协议 §4）。
pub fn rule_entry_size() -> usize {
    RULE_ENTRY_SIZE
}

/// 参数块大小：5280 字节（协议 §3）。
pub fn param_block_size() -> usize {
    PARAM_BLOCK_SIZE
}

/// 按 SPEC CHANGE 的模式匹配表生成规则。
///
/// 顺序：直通系统区在前（避免被任何重写规则抢先命中），重写区在后；
/// 每条规则的 `\\?\` 变体紧跟其明文版本之后。
///
/// - `user_profile`：如 `C:\Users\Hao`（保留原大小写；hook 比较大小写不敏感）。
/// - `game_data_root`：如 `E:\GameData`。
/// - `skip_cache`：为 `true` 时排除 AppData 重写规则（直通宿主）。
///
/// 默认（`skip_cache = false`）返回 12 条；`skip_cache = true` 返回 10 条。
pub fn generate_rules(user_profile: &str, game_data_root: &str, skip_cache: bool) -> Vec<Rule> {
    let mut rules = Vec::new();

    // 1. 直通系统区（表序最前）。
    for zone in ["C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)"] {
        push_with_variants(
            &mut rules,
            &format!("{zone}\\"),
            "",
            GVHD_RULE_FLAG_PASSTHROUGH,
        );
    }

    // 缺少重写锚点时不生成重写规则（只保留直通区）。
    if user_profile.trim().is_empty() || game_data_root.trim().is_empty() {
        return rules;
    }

    let up = ensure_trailing_sep(user_profile);
    let root = ensure_trailing_sep(game_data_root);
    let user = last_segment(user_profile);

    // 2. 重写区（相对 %USERPROFILE%）。
    for zone in ["Documents", "Saved Games", "AppData"] {
        if skip_cache && zone == "AppData" {
            crate::log_warn!(
                "skip_cache_dirs=true：AppData 规则被排除（直通宿主）；\
                 退出期宿主缓存的清理由 runtime 负责"
            );
            continue;
        }
        let prefix = format!("{up}{zone}\\");
        let replacement = format!("{root}Users\\{user}\\{zone}\\");
        push_with_variants(&mut rules, &prefix, &replacement, GVHD_RULE_FLAG_REWRITE);
    }

    rules
}

/// 追加一条规则 + 其 `\\?\` 变体（变体紧跟在明文之后）。
fn push_with_variants(rules: &mut Vec<Rule>, prefix: &str, replacement: &str, flags: u32) {
    rules.push(Rule {
        prefix: prefix.to_string(),
        replacement: replacement.to_string(),
        flags,
    });
    rules.push(Rule {
        prefix: format!(r"\\?\{prefix}"),
        replacement: replacement.to_string(),
        flags,
    });
}

/// 保证路径以单个 `\` 结尾（去掉所有尾部反斜杠后补一个）。
fn ensure_trailing_sep(p: &str) -> String {
    let mut s = p.trim_end_matches('\\').to_string();
    s.push('\\');
    s
}

/// 取 `user_profile` 的最后一段（用户名，如 `C:\Users\Hao` → `Hao`）。
fn last_segment(p: &str) -> &str {
    p.trim_end_matches('\\')
        .rsplit('\\')
        .next()
        .unwrap_or(p)
}

/// 把字符串编码为 UTF-16LE 码元序列（不含 NUL）。
fn utf16_units(s: &str) -> Vec<u16> {
    s.encode_utf16().collect()
}

/// 将字符串写入预清零缓冲区的 WCHAR 字段：NUL 结尾、超长截断（保留结尾 NUL）。
/// `capacity_wchars` 为字段的 WCHAR 容量；最多写 `capacity-1` 个码元。
/// 返回实际写入的码元数（不含 NUL）。
fn write_utf16_field(buf: &mut [u8], byte_offset: usize, capacity_wchars: usize, s: &str) -> usize {
    let mut units = s.encode_utf16();
    let mut written = 0usize;
    while written < capacity_wchars.saturating_sub(1) {
        let Some(u) = units.next() else { break };
        let off = byte_offset + written * 2;
        buf[off..off + 2].copy_from_slice(&u.to_le_bytes());
        written += 1;
    }
    // 缓冲区已预清零；显式写 NUL 以保证截断路径也满足"NUL 结尾"。
    let nul = byte_offset + written * 2;
    if nul + 1 < buf.len() {
        buf[nul] = 0;
        buf[nul + 1] = 0;
    }
    written
}

/// 将规则表序列化为协议规则条目字节流（每条 4104B：prefix@0、replacement@2048、
/// flags u32@4096、reserved u32@4100，均零填充 + NUL 结尾）。
pub fn to_rule_bytes(rules: &[Rule]) -> Vec<u8> {
    let mut out = Vec::with_capacity(rules.len() * RULE_ENTRY_SIZE);
    for rule in rules {
        let mut entry = vec![0u8; RULE_ENTRY_SIZE];
        write_utf16_field(&mut entry, 0, GVHD_RULE_STRING_MAX, &rule.prefix);
        write_utf16_field(
            &mut entry,
            GVHD_RULE_STRING_MAX * 2,
            GVHD_RULE_STRING_MAX,
            &rule.replacement,
        );
        entry[4096..4100].copy_from_slice(&rule.flags.to_le_bytes());
        // reserved [4100..4104] 保持零。
        out.extend_from_slice(&entry);
    }
    out
}

/// 构建完整注入参数块：`5280 + n*4104` 字节。
///
/// 布局（协议 §3/§5.1）：先整体清零，再依次写入 magic/version/flags/rule_count/
/// rule_table_offset(=5280)/game_id_len/reserved，各路径 WCHAR 字段按协议偏移
/// （UTF-16LE，截断到 511 码元 + NUL；game_id 截断到 63 码元），随后是规则条目。
pub fn param_block_with(
    hook_dll_path: &str,
    game_data_root: &str,
    user_profile: &str,
    log_path: &str,
    registry_hive: &str,
    game_id: &str,
    rule_table: &[Rule],
) -> Vec<u8> {
    param_block_with_drive(
        hook_dll_path,
        game_data_root,
        user_profile,
        log_path,
        registry_hive,
        game_id,
        rule_table,
        '\0',
    )
}

/// 构建参数块并编码实际游戏 VHD 盘符。
///
/// 外部状态根可能位于宿主 C:，不能再从 `game_data_root` 推断游戏盘符；
/// 盘符编码放在 flags 的扩展位中，不改变参数块的固定布局。
pub fn param_block_with_drive(
    hook_dll_path: &str,
    game_data_root: &str,
    user_profile: &str,
    log_path: &str,
    registry_hive: &str,
    game_id: &str,
    rule_table: &[Rule],
    game_drive: char,
) -> Vec<u8> {
    let rule_count = rule_table.len();
    debug_assert!(rule_count as u32 <= GVHD_RULE_MAX);
    let mut buf = vec![0u8; PARAM_BLOCK_SIZE + rule_count * RULE_ENTRY_SIZE];

    // 标量头部。
    buf[0..4].copy_from_slice(&GVHD_PARAM_MAGIC.to_le_bytes());
    buf[4..8].copy_from_slice(&GVHD_PROTOCOL_VERSION.to_le_bytes());
    let flags = encode_game_drive_flag(game_drive);
    buf[8..12].copy_from_slice(&flags.to_le_bytes());
    buf[12..16].copy_from_slice(&(rule_count as u32).to_le_bytes());
    buf[16..20].copy_from_slice(&(PARAM_BLOCK_SIZE as u32).to_le_bytes());
    // game_id_len = 实际写入的码元数（不含 NUL）。
    let game_id_len = write_utf16_field(&mut buf, OFFSET_GAME_ID, GVHD_GAME_ID_MAX, game_id);
    buf[20..24].copy_from_slice(&(game_id_len as u32).to_le_bytes());
    // reserved[2] @24..32 保持零。

    // 路径字段。
    write_utf16_field(&mut buf, OFFSET_HOOK_DLL_PATH, GVHD_PATH_MAX, hook_dll_path);
    write_utf16_field(&mut buf, OFFSET_GAME_DATA_ROOT, GVHD_PATH_MAX, game_data_root);
    write_utf16_field(&mut buf, OFFSET_USER_PROFILE, GVHD_PATH_MAX, user_profile);
    write_utf16_field(&mut buf, OFFSET_LOG_PATH, GVHD_PATH_MAX, log_path);
    write_utf16_field(&mut buf, OFFSET_REGISTRY_HIVE, GVHD_PATH_MAX, registry_hive);

    // 规则条目（紧跟参数块）。
    let rule_bytes = to_rule_bytes(rule_table);
    buf[PARAM_BLOCK_SIZE..].copy_from_slice(&rule_bytes);

    buf
}

fn encode_game_drive_flag(game_drive: char) -> u32 {
    let upper = game_drive.to_ascii_uppercase();
    if upper.is_ascii_uppercase() {
        ((upper as u32 - 'A' as u32 + 1) << GVHD_PARAM_FLAG_GAME_DRIVE_SHIFT)
            & GVHD_PARAM_FLAG_GAME_DRIVE_MASK
    } else {
        0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// 手工 UTF-16LE 编码（测试自用）。
    fn utf16le(s: &str) -> Vec<u8> {
        let mut v = Vec::new();
        for u in s.encode_utf16() {
            v.extend_from_slice(&u.to_le_bytes());
        }
        v
    }

    fn u32at(b: &[u8], off: usize) -> u32 {
        u32::from_le_bytes(b[off..off + 4].try_into().unwrap())
    }

    #[test]
    fn size_constants() {
        assert_eq!(rule_entry_size(), 4104);
        assert_eq!(param_block_size(), 5280);
        assert_eq!(GVHD_PARAM_MAGIC, u32::from_le_bytes(*b"GVHD"));
    }

    #[test]
    fn generate_rules_default_12() {
        let rules = generate_rules(r"C:\Users\Hao", r"E:\GameData", false);
        assert_eq!(rules.len(), 12, "默认：3 直通 ×2 变体 + 3 重写 ×2 变体");

        // 第一条是 C:\Windows 直通（表序最前）。
        assert_eq!(rules[0].prefix, r"C:\Windows\");
        assert_eq!(rules[0].flags, GVHD_RULE_FLAG_PASSTHROUGH);
        assert_eq!(rules[0].replacement, "");
        // 变体紧跟明文。
        assert_eq!(rules[1].prefix, r"\\?\C:\Windows\");
        assert_eq!(rules[3].prefix, r"\\?\C:\Program Files\");
        assert_eq!(rules[5].prefix, r"\\?\C:\Program Files (x86)\");
        // 重写区从第 6 条开始。
        assert_eq!(rules[6].prefix, r"C:\Users\Hao\Documents\");
        assert_eq!(rules[6].replacement, r"E:\GameData\Users\Hao\Documents\");
        assert_eq!(rules[6].flags, GVHD_RULE_FLAG_REWRITE);
        assert_eq!(rules[7].prefix, r"\\?\C:\Users\Hao\Documents\");
        assert_eq!(rules[7].replacement, r"E:\GameData\Users\Hao\Documents\");
        // 最后一条是 \\?\ AppData 重写。
        let last = rules.last().unwrap();
        assert_eq!(last.prefix, r"\\?\C:\Users\Hao\AppData\");
        assert_eq!(last.flags, GVHD_RULE_FLAG_REWRITE);
        // 含 AppData 重写。
        assert!(rules
            .iter()
            .any(|r| r.flags == GVHD_RULE_FLAG_REWRITE && r.prefix.contains("AppData")));
        // 全部直通在前、重写在后。
        let first_rewrite = rules
            .iter()
            .position(|r| r.flags == GVHD_RULE_FLAG_REWRITE)
            .unwrap();
        assert!(rules[..first_rewrite]
            .iter()
            .all(|r| r.flags == GVHD_RULE_FLAG_PASSTHROUGH));
    }

    #[test]
    fn generate_rules_skip_cache_excludes_appdata() {
        let rules = generate_rules(r"C:\Users\Hao", r"E:\GameData", true);
        assert_eq!(rules.len(), 10);
        assert!(
            !rules.iter().any(|r| r.prefix.contains("AppData")),
            "skip_cache=true 不得含 AppData 规则"
        );
        // 其余重写区仍在。
        assert!(rules
            .iter()
            .any(|r| r.prefix == r"C:\Users\Hao\Documents\"));
        assert!(rules
            .iter()
            .any(|r| r.prefix == r"C:\Users\Hao\Saved Games\"));
        assert!(rules
            .iter()
            .any(|r| r.prefix == r"C:\Users\Hao\Saved Games\"));
    }

    #[test]
    fn generate_rules_normalizes_trailing_seps_and_other_drive() {
        let rules = generate_rules(r"C:\Users\Hao\", r"D:\Data\", false);
        assert_eq!(rules.len(), 12);
        let doc = rules
            .iter()
            .find(|r| r.prefix == r"C:\Users\Hao\Documents\")
            .unwrap();
        assert_eq!(doc.replacement, r"D:\Data\Users\Hao\Documents\");
    }

    #[test]
    fn generate_rules_empty_anchors_yield_passthrough_only() {
        let rules = generate_rules("", r"E:\GameData", false);
        assert_eq!(rules.len(), 6);
        assert!(rules.iter().all(|r| r.flags == GVHD_RULE_FLAG_PASSTHROUGH));
    }

    #[test]
    fn to_rule_bytes_entry_layout() {
        let rules = vec![
            Rule {
                prefix: "X".into(),
                replacement: "Y".into(),
                flags: GVHD_RULE_FLAG_REWRITE,
            },
            Rule {
                prefix: "Z".into(),
                replacement: String::new(),
                flags: GVHD_RULE_FLAG_PASSTHROUGH,
            },
        ];
        let b = to_rule_bytes(&rules);
        assert_eq!(b.len(), 2 * RULE_ENTRY_SIZE);

        // 第 0 条：prefix UTF-16 在 [0..]，replacement 在 2048，flags 在 4096。
        assert_eq!(&b[0..2], &[b'X', 0]);
        assert_eq!(b[2], 0, "前缀区余量须为 NUL 填充");
        assert_eq!(&b[2048..2050], &[b'Y', 0]);
        assert_eq!(u32at(&b, 4096), GVHD_RULE_FLAG_REWRITE);
        assert_eq!(u32at(&b, 4100), 0, "reserved 须为零");

        // 第 1 条在 4104 偏移（stride 验证）。
        let base = RULE_ENTRY_SIZE;
        assert_eq!(&b[base..base + 2], &[b'Z', 0]);
        assert_eq!(u32at(&b, base + 4096), GVHD_RULE_FLAG_PASSTHROUGH);
        assert_eq!(u32at(&b, base + 4100), 0);
    }

    #[test]
    fn to_rule_bytes_truncates_long_strings_with_nul() {
        let long = "A".repeat(2000);
        let rules = vec![Rule {
            prefix: long,
            replacement: "R".into(),
            flags: GVHD_RULE_FLAG_REWRITE,
        }];
        let b = to_rule_bytes(&rules);
        for i in 0..1023 {
            assert_eq!(&b[i * 2..i * 2 + 2], &[b'A', 0], "prefix 码元 {i}");
        }
        assert_eq!(&b[1023 * 2..1023 * 2 + 2], &[0, 0], "截断后 NUL 结尾");
        assert_eq!(&b[2048..2050], &[b'R', 0]);
        assert_eq!(u32at(&b, 4096), GVHD_RULE_FLAG_REWRITE);
    }

    #[test]
    fn param_block_layout_exact() {
        let rules = generate_rules(r"C:\Users\Hao", r"E:\GameData", false);
        let n = rules.len();
        let block = param_block_with(
            r"C:\tools\gvhook-x64.dll",
            r"E:\GameData",
            r"C:\Users\Hao",
            r"E:\GameData\logs\gvhook.log",
            r"E:\GameData\Registry\user.dat",
            "horizon-zero-dawn",
            &rules,
        );

        // 总长 = 5280 + n*4104。
        assert_eq!(block.len(), PARAM_BLOCK_SIZE + n * RULE_ENTRY_SIZE);

        // 标量头部。
        assert_eq!(&block[0..4], b"GVHD");
        assert_eq!(u32at(&block, 0), 0x4448_5647);
        assert_eq!(u32at(&block, 4), GVHD_PROTOCOL_VERSION);
        assert_eq!(u32at(&block, 8), 0, "flags 恒为 0");
        assert_eq!(u32at(&block, 12), n as u32, "rule_count");
        assert_eq!(u32at(&block, 16), PARAM_BLOCK_SIZE as u32, "rule_table_offset");
        assert_eq!(u32at(&block, 20), 17, "game_id_len = 'horizon-zero-dawn' 17 码元");
        assert_eq!(u32at(&block, 24), 0, "reserved[0]");
        assert_eq!(u32at(&block, 28), 0, "reserved[1]");

        // 路径字段（精确偏移，UTF-16LE）。
        let hp = utf16le(r"C:\tools\gvhook-x64.dll");
        assert_eq!(&block[OFFSET_HOOK_DLL_PATH..OFFSET_HOOK_DLL_PATH + hp.len()], &hp[..]);
        let gd = utf16le(r"E:\GameData");
        assert_eq!(&block[OFFSET_GAME_DATA_ROOT..OFFSET_GAME_DATA_ROOT + gd.len()], &gd[..]);
        let up = utf16le(r"C:\Users\Hao");
        assert_eq!(&block[OFFSET_USER_PROFILE..OFFSET_USER_PROFILE + up.len()], &up[..]);
        let lp = utf16le(r"E:\GameData\logs\gvhook.log");
        assert_eq!(&block[OFFSET_LOG_PATH..OFFSET_LOG_PATH + lp.len()], &lp[..]);
        let rg = utf16le(r"E:\GameData\Registry\user.dat");
        assert_eq!(&block[OFFSET_REGISTRY_HIVE..OFFSET_REGISTRY_HIVE + rg.len()], &rg[..]);
        let gid = utf16le("horizon-zero-dawn");
        assert_eq!(&block[OFFSET_GAME_ID..OFFSET_GAME_ID + gid.len()], &gid[..]);

        // 规则条目逐条校验。
        for (i, rule) in rules.iter().enumerate() {
            let base = PARAM_BLOCK_SIZE + i * RULE_ENTRY_SIZE;
            let p = utf16le(&rule.prefix);
            assert_eq!(&block[base..base + p.len()], &p[..], "rule {i} prefix");
            if !rule.replacement.is_empty() {
                let rp = utf16le(&rule.replacement);
                assert_eq!(
                    &block[base + 2048..base + 2048 + rp.len()],
                    &rp[..],
                    "rule {i} replacement"
                );
            }
            assert_eq!(u32at(&block, base + 4096), rule.flags, "rule {i} flags");
            assert_eq!(u32at(&block, base + 4100), 0, "rule {i} reserved");
        }
    }

    #[test]
    fn param_block_rule_table_offset_is_4_aligned() {
        let block = param_block_with("d", "e", "u", "l", "h", "g", &[]);
        assert_eq!(block.len(), PARAM_BLOCK_SIZE);
        assert_eq!(u32at(&block, 16) % 4, 0);
    }

    #[test]
    fn param_block_encodes_actual_game_drive_without_changing_layout() {
        let block = param_block_with_drive(
            "d",
            r"C:\GameData\Horizon",
            "u",
            "l",
            "h",
            "g",
            &[],
            'g',
        );
        assert_eq!(u32at(&block, 8), 7 << GVHD_PARAM_FLAG_GAME_DRIVE_SHIFT);
        assert_eq!(u32at(&block, 8) & !GVHD_PARAM_FLAG_GAME_DRIVE_MASK, 0);
        assert_eq!(block.len(), PARAM_BLOCK_SIZE);
    }
}
