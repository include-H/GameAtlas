//! 尾部配置块 manifest（阶段 5 起步，v2 定案 §3.1）。
//!
//! 分发形态：`launcher.exe`（通用二进制）+ 游戏配置 JSON + footer
//! （magic 8B + u32le 长度）尾部追加。启动器从文件尾部自定位解析；
//! 无配置块（纯 launcher.exe）也能运行（`--selftest` / 命令分发）。
//!
//! footer 布局（小端）：
//! ```text
//! [ ... launcher.exe ... ][ config JSON bytes ][ GVHDCFG\0 ][ u32le json_len ]
//! └──────────────────────┘  ↑ json_start      ↑ json_end  ↑ footer
//! ```
//! 定位策略：优先取严格尾部 12B（magic+len）；若 magic 不匹配，回退扫描
//! 最后 [`SCAN_BACK`] 字节找 magic（容忍 exe 后额外填充/签名块）。
//!
//! 零依赖：JSON 解析委托 [`crate::json`]（扁平对象子集，字符串/布尔）。

// 库模块：`load_manifest_file`（磁盘读取）与 Io 错误留给阶段 5 run 主流程
// 消费（当前 wave 仅内存样本路径被 selftest/单测调用），按模块豁免。
#![allow(dead_code)]

use crate::json::{escape_json, parse_json_object};

/// footer magic（8B，含 NUL 终止符）。
pub const FOOTER_MAGIC: [u8; 8] = *b"GVHDCFG\0";
/// footer 固定开销：magic(8) + u32le 长度(4)。
pub const FOOTER_OVERHEAD: usize = 12;
/// 回退扫描窗口（尾部若干字节内找 magic，容忍签名/填充块）。
pub const SCAN_BACK: usize = 4096;
/// JSON 配置长度上限（防恶意文件声明超长长度）。
pub const MAX_CONFIG_LEN: usize = 64 * 1024;

/// 尾部配置（v2 §3.1 配置块内容）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Manifest {
    pub game_id: String,
    pub title: String,
    /// 基础盘 UNC 路径（只读共享内，如 `\\192.168.1.4\Game\base.vhdx`）。
    pub base_vhd: String,
    /// 差分盘文件名（落 `%LOCALAPPDATA%\GameAtlas\diff\<diff_name>`）。
    pub diff_name: String,
    /// SMB 凭据（明文 = 项目公认边界，README 声明）。
    pub smb_user: Option<String>,
    pub smb_pass: Option<String>,
    /// 可选 exe 提示（首启扫描排序的加分项）。
    pub exe_hint: Option<String>,
    /// 规则覆盖：AppData 直通宿主（默认 false）。
    pub skip_cache_dirs: bool,
}

/// manifest 解析错误。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ManifestError {
    /// 文件尾部找不到 footer magic。
    NoFooter,
    /// footer 长度字段非法（0 / 超上限）。
    BadLength(u32),
    /// JSON 解析失败（消息）。
    BadJson(String),
    /// 必填字段缺失（字段名）。
    MissingField(&'static str),
    /// 未知字段。
    UnknownField(String),
    /// 签名不匹配（消息；带 sig 的 manifest 被篡改或损坏）。
    BadSignature(String),
    /// 启动器 manifest 缺少签名。旧格式只允许通过显式 legacy API 读取。
    MissingSignature,
    /// 读取文件失败。
    Io(String),
}

impl std::fmt::Display for ManifestError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ManifestError::NoFooter => write!(f, "尾部无配置块（非卡带形态的通用启动器）"),
            ManifestError::BadLength(n) => write!(f, "配置长度非法: {n}"),
            ManifestError::BadJson(m) => write!(f, "配置 JSON 非法: {m}"),
            ManifestError::MissingField(k) => write!(f, "配置缺少必填字段 '{k}'"),
            ManifestError::UnknownField(k) => write!(f, "配置含未知字段 '{k}'"),
            ManifestError::BadSignature(m) => write!(f, "配置签名校验失败（可能被篡改）: {m}"),
            ManifestError::MissingSignature => write!(f, "配置缺少签名（仅兼容旧格式解析入口允许）"),
            ManifestError::Io(m) => write!(f, "读取失败: {m}"),
        }
    }
}

impl std::error::Error for ManifestError {}

/// manifest 签名密钥（审计 P2-10）：内置密钥防意外篡改/损坏；
/// 防恶意篡改需外部分发密钥（部署层决策，见已知局限文档）。
pub const MANIFEST_SIGN_KEY: &[u8] = b"gamevhd-atlas-v1";

impl Manifest {
    /// 序列化为配置 JSON（2 空格缩进，风格与 box.json 一致）。
    pub fn to_json(&self) -> String {
        let mut out = String::from("{\n");
        out.push_str(&format!("  \"game_id\": \"{}\",\n", escape_json(&self.game_id)));
        out.push_str(&format!("  \"title\": \"{}\",\n", escape_json(&self.title)));
        out.push_str(&format!("  \"base_vhd\": \"{}\",\n", escape_json(&self.base_vhd)));
        out.push_str(&format!("  \"diff_name\": \"{}\",\n", escape_json(&self.diff_name)));
        if let Some(u) = &self.smb_user {
            out.push_str(&format!("  \"smb_user\": \"{}\",\n", escape_json(u)));
        }
        if let Some(p) = &self.smb_pass {
            out.push_str(&format!("  \"smb_pass\": \"{}\",\n", escape_json(p)));
        }
        if let Some(h) = &self.exe_hint {
            out.push_str(&format!("  \"exe_hint\": \"{}\",\n", escape_json(h)));
        }
        out.push_str(&format!(
            "  \"skip_cache_dirs\": {}\n}}",
            if self.skip_cache_dirs { "true" } else { "false" }
        ));
        out
    }

    /// 序列化为「可追加到 exe 尾部的完整配置块」（JSON + magic + len）。
    pub fn to_footer_bytes(&self) -> Vec<u8> {
        let json = self.to_json();
        let json_bytes = json.as_bytes();
        let mut out = Vec::with_capacity(json_bytes.len() + FOOTER_OVERHEAD);
        out.extend_from_slice(json_bytes);
        out.extend_from_slice(&FOOTER_MAGIC);
        out.extend_from_slice(&(json_bytes.len() as u32).to_le_bytes());
        out
    }

    /// 序列化为带签名配置块（审计 P2-10）：JSON 尾部追加 `,"sig":"<hmac>"`。
    pub fn to_signed_footer_bytes(&self) -> Vec<u8> {
        let json = self.to_json();
        let signed = sign_json(&json);
        let json_bytes = signed.as_bytes();
        let mut out = Vec::with_capacity(json_bytes.len() + FOOTER_OVERHEAD);
        out.extend_from_slice(json_bytes);
        out.extend_from_slice(&FOOTER_MAGIC);
        out.extend_from_slice(&(json_bytes.len() as u32).to_le_bytes());
        out
    }
}

/// 对配置 JSON 计算签名并追加 `,"sig":"<hex>"`（在结尾 `}` 之前插入）。
fn sign_json(json: &str) -> String {
    let sig = crate::sha256::hmac_hex(MANIFEST_SIGN_KEY, json.as_bytes());
    let body = json
        .strip_suffix('}')
        .expect("Manifest::to_json must return a JSON object");
    format!("{body},\n  \"sig\": \"{sig}\"\n}}")
}

/// 在字节切片中定位配置 JSON 的边界 `(start, end)`（end = magic 起始，即 JSON 结束）。
/// 优先严格尾部；回退扫描最后 [`SCAN_BACK`] 字节（容忍 exe 后签名/填充块）。
pub fn locate_config(data: &[u8]) -> Result<(usize, usize), ManifestError> {
    if data.len() < FOOTER_OVERHEAD {
        return Err(ManifestError::NoFooter);
    }
    // 严格尾部：最后 12B = magic + len。
    let tail = &data[data.len() - FOOTER_OVERHEAD..];
    if &tail[..8] == &FOOTER_MAGIC {
        let len = u32::from_le_bytes(tail[8..12].try_into().unwrap()) as usize;
        if len == 0 || len > MAX_CONFIG_LEN {
            return Err(ManifestError::BadLength(len as u32));
        }
        let json_end = data.len() - FOOTER_OVERHEAD;
        let Some(start) = json_end.checked_sub(len) else {
            return Err(ManifestError::BadLength(len as u32));
        };
        return Ok((start, json_end));
    }
    // 回退：扫描尾部窗口找 magic，位置即 JSON 结束。
    let window = &data[data.len().saturating_sub(SCAN_BACK)..];
    let base = data.len() - window.len();
    let json_end = window
        .windows(8)
        .position(|w| w == FOOTER_MAGIC)
        .map(|i| base + i)
        .ok_or(ManifestError::NoFooter)?;
    if json_end + FOOTER_OVERHEAD > data.len() {
        return Err(ManifestError::NoFooter);
    }
    let len = u32::from_le_bytes(
        data[json_end + 8..json_end + FOOTER_OVERHEAD]
            .try_into()
            .unwrap(),
    ) as usize;
    if len == 0 || len > MAX_CONFIG_LEN || json_end < len {
        return Err(ManifestError::BadLength(len as u32));
    }
    Ok((json_end - len, json_end))
}

/// 从字节切片解析 manifest（定位 + 强制签名校验 + JSON 解析 + 字段校验）。
pub fn parse_manifest(data: &[u8]) -> Result<Manifest, ManifestError> {
    parse_manifest_with_policy(data, true)
}

/// 显式兼容旧版无签名 footer。正式启动器读取路径不调用此入口，避免
/// 删除 `sig` 后把篡改后的配置降级成“合法旧格式”。
pub fn parse_manifest_legacy(data: &[u8]) -> Result<Manifest, ManifestError> {
    parse_manifest_with_policy(data, false)
}

fn parse_manifest_with_policy(
    data: &[u8],
    require_signature: bool,
) -> Result<Manifest, ManifestError> {
    let (start, end) = locate_config(data)?;
    if start >= end {
        return Err(ManifestError::BadJson("空配置".into()));
    }
    let json = std::str::from_utf8(&data[start..end])
        .map_err(|e| ManifestError::BadJson(format!("非 UTF-8: {e}")))?;
    parse_manifest_json(json, require_signature)
}

/// 从配置 JSON 文本解析（字段严格校验：必填 4 项，未知字段报错）。
pub fn manifest_from_json(json: &str) -> Result<Manifest, ManifestError> {
    let fields = parse_json_object(json).map_err(ManifestError::BadJson)?;
    manifest_from_fields(fields)
}

/// Parse a manifest object and verify its optional signature.  Signature
/// extraction is structural: a `sig` key must be unique, and escaped JSON
/// spellings are decoded before the key is classified.  The HMAC covers the
/// canonical `Manifest::to_json()` representation, so whitespace and field
/// order cannot create a second signed meaning.
fn parse_manifest_json(json: &str, require_signature: bool) -> Result<Manifest, ManifestError> {
    let fields = parse_json_object(json).map_err(ManifestError::BadJson)?;
    let mut unsigned = Vec::with_capacity(fields.len());
    let mut signature = None;
    for (key, value) in fields {
        if key == "sig" {
            if signature.replace(value).is_some() {
                return Err(ManifestError::BadSignature(
                    "sig 字段重复".to_string(),
                ));
            }
        } else {
            unsigned.push((key, value));
        }
    }

    let manifest = manifest_from_fields(unsigned)?;
    let Some(actual) = signature else {
        if require_signature {
            return Err(ManifestError::MissingSignature);
        }
        // Legacy unsigned manifests remain accepted only by the explicit
        // compatibility parser.
        return Ok(manifest);
    };
    if actual.len() != 64 || !actual.bytes().all(|b| b.is_ascii_hexdigit()) {
        return Err(ManifestError::BadSignature("sig 字段格式非法".into()));
    }
    let expected = crate::sha256::hmac_hex(MANIFEST_SIGN_KEY, manifest.to_json().as_bytes());
    if !actual.eq_ignore_ascii_case(&expected) {
        return Err(ManifestError::BadSignature(format!(
            "期望 {expected}，实际 {actual}"
        )));
    }
    Ok(manifest)
}

fn manifest_from_fields(fields: Vec<(String, String)>) -> Result<Manifest, ManifestError> {
    let mut m = Manifest {
        game_id: String::new(),
        title: String::new(),
        base_vhd: String::new(),
        diff_name: String::new(),
        smb_user: None,
        smb_pass: None,
        exe_hint: None,
        skip_cache_dirs: false,
    };
    let mut seen = [false; 8];
    for (k, v) in fields {
        let idx = match k.as_str() {
            "game_id" => 0,
            "title" => 1,
            "base_vhd" => 2,
            "diff_name" => 3,
            "smb_user" => 4,
            "smb_pass" => 5,
            "exe_hint" => 6,
            "skip_cache_dirs" => 7,
            other => return Err(ManifestError::UnknownField(other.into())),
        };
        if seen[idx] {
            return Err(ManifestError::BadJson(format!("重复字段 '{k}'")));
        }
        seen[idx] = true;
        match idx {
            0 => m.game_id = v,
            1 => m.title = v,
            2 => m.base_vhd = v,
            3 => m.diff_name = v,
            4 => m.smb_user = Some(v),
            5 => m.smb_pass = Some(v),
            6 => m.exe_hint = Some(v),
            _ => {
                m.skip_cache_dirs = match v.as_str() {
                    "true" => true,
                    "false" => false,
                    other => return Err(ManifestError::BadJson(format!("非法 skip_cache_dirs '{other}'"))),
                }
            }
        }
    }
    const REQUIRED: [(&str, usize); 4] = [
        ("game_id", 0),
        ("title", 1),
        ("base_vhd", 2),
        ("diff_name", 3),
    ];
    for (name, i) in REQUIRED {
        if !seen[i] {
            return Err(ManifestError::MissingField(name));
        }
    }
    Ok(m)
}

/// 从磁盘文件读取并解析（自定位尾部配置）。
pub fn load_manifest_file(path: &std::path::Path) -> Result<Manifest, ManifestError> {
    let data = std::fs::read(path).map_err(|e| ManifestError::Io(e.to_string()))?;
    parse_manifest(&data)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample() -> Manifest {
        Manifest {
            game_id: "horizon-zero-dawn".into(),
            title: "地平线：零之曙光".into(),
            base_vhd: r"\\192.168.1.4\Game\base.vhdx".into(),
            diff_name: "horizon.vhdx".into(),
            smb_user: Some("rom".into()),
            smb_pass: Some("p@ss w\\ord".into()),
            exe_hint: Some("Game\\HorizonZeroDawn.exe".into()),
            skip_cache_dirs: true,
        }
    }

    #[test]
    fn footer_round_trip_strict_tail() {
        let m = sample();
        let mut blob = vec![0u8; 256]; // 模拟 launcher.exe 主体
        blob.extend_from_slice(&m.to_footer_bytes());
        let parsed = parse_manifest_legacy(&blob).unwrap();
        assert_eq!(parsed, m);
    }

    #[test]
    fn footer_round_trip_scan_back_finds_magic() {
        // 模拟 exe 尾部有额外填充（如签名块 64B），magic 不在严格尾部。
        let m = sample();
        let mut footer = m.to_footer_bytes();
        footer.extend_from_slice(&[0xAA; 64]); // 签名/填充块
        let mut blob = vec![0x00; 64];
        blob.extend_from_slice(&footer);
        let parsed = parse_manifest_legacy(&blob).unwrap();
        assert_eq!(parsed, m);
    }

    #[test]
    fn signed_footer_round_trip_and_tamper_rejection() {
        let m = sample();
        let signed = m.to_signed_footer_bytes();
        assert_eq!(parse_manifest(&signed).unwrap(), m);

        let mut tampered = signed.clone();
        let marker = b"horizon-zero-dawn";
        let pos = tampered
            .windows(marker.len())
            .position(|window| window == marker)
            .expect("signed game_id present");
        tampered[pos] = b"H"[0];
        assert!(matches!(
            parse_manifest(&tampered),
            Err(ManifestError::BadSignature(_))
        ));
    }

    #[test]
    fn signed_manifest_cannot_be_downgraded_by_removing_sig() {
        let m = sample();
        let signed = m.to_signed_footer_bytes();
        let (start, end) = locate_config(&signed).unwrap();
        let signed_json = String::from_utf8(signed[start..end].to_vec()).unwrap();
        let unsigned_json = signed_json
            .rsplit_once(",\n  \"sig\": \"")
            .map(|(body, _)| format!("{body}\n}}"))
            .expect("signed footer");
        let mut downgraded = signed.clone();
        downgraded.splice(start..end, unsigned_json.as_bytes().iter().copied());
        let n = downgraded.len();
        downgraded[n - 4..].copy_from_slice(&(unsigned_json.len() as u32).to_le_bytes());
        assert_eq!(
            parse_manifest(&downgraded),
            Err(ManifestError::MissingSignature)
        );
        assert_eq!(parse_manifest_legacy(&downgraded).unwrap(), m);
    }

    #[test]
    fn signed_sig_key_is_parsed_structurally() {
        let m = sample();
        let signed = m.to_signed_footer_bytes();
        let (start, end) = locate_config(&signed).unwrap();
        let signed_json = String::from_utf8(signed[start..end].to_vec()).unwrap();
        let escaped_key = signed_json.replace("\"sig\": \"", "\"\\u0073ig\": \"");
        let mut escaped_blob = signed.clone();
        escaped_blob.splice(start..end, escaped_key.as_bytes().iter().copied());
        let escaped_footer_len = escaped_blob.len();
        escaped_blob[escaped_footer_len - 4..]
            .copy_from_slice(&(escaped_key.len() as u32).to_le_bytes());
        assert_eq!(
            parse_manifest(&escaped_blob).unwrap(),
            m,
            "sig key escaping must not bypass verification"
        );

        let duplicate_json = signed_json.replace(
            "\n}",
            ",\n  \"sig\": \"0000000000000000000000000000000000000000000000000000000000000000\"\n}",
        );
        let mut duplicate_blob = signed.clone();
        duplicate_blob.splice(start..end, duplicate_json.as_bytes().iter().copied());
        let duplicate_footer_len = duplicate_blob.len();
        duplicate_blob[duplicate_footer_len - 4..]
            .copy_from_slice(&(duplicate_json.len() as u32).to_le_bytes());
        assert!(matches!(
            parse_manifest(&duplicate_blob),
            Err(ManifestError::BadSignature(_))
        ));
    }

    #[test]
    fn no_footer_rejected() {
        let blob = vec![0x00u8; 1024];
        assert_eq!(parse_manifest(&blob), Err(ManifestError::NoFooter));
        assert_eq!(locate_config(&[0u8; 4]), Err(ManifestError::NoFooter));
    }

    #[test]
    fn bad_length_rejected() {
        let m = sample();
        let mut footer = m.to_footer_bytes();
        let n = footer.len();
        // 篡改长度字段为超大值。
        footer[n - 4..].copy_from_slice(&0xFFFF_FFFFu32.to_le_bytes());
        assert_eq!(parse_manifest(&footer), Err(ManifestError::BadLength(0xFFFF_FFFF)));
    }

    #[test]
    fn short_footer_length_is_rejected_without_underflow() {
        let mut footer = Vec::from(FOOTER_MAGIC);
        footer.extend_from_slice(&100u32.to_le_bytes());
        assert_eq!(locate_config(&footer), Err(ManifestError::BadLength(100)));
    }

    #[test]
    fn missing_required_field() {
        let json = r#"{"game_id":"g","title":"t","base_vhd":"\\\\server\\share"}"#;
        assert_eq!(
            manifest_from_json(json),
            Err(ManifestError::MissingField("diff_name"))
        );
    }

    #[test]
    fn unknown_field_rejected() {
        let json = r#"{"game_id":"g","title":"t","base_vhd":"b","diff_name":"d","nope":"x"}"#;
        assert_eq!(
            manifest_from_json(json),
            Err(ManifestError::UnknownField("nope".into()))
        );
    }

    #[test]
    fn duplicate_field_rejected() {
        let json = r#"{"game_id":"a","game_id":"b","title":"t","base_vhd":"v","diff_name":"d"}"#;
        assert!(matches!(manifest_from_json(json), Err(ManifestError::BadJson(_))));
    }

    #[test]
    fn bool_and_optional_fields() {
        let json = r#"{"game_id":"g","title":"t","base_vhd":"v","diff_name":"d","skip_cache_dirs":true}"#;
        let m = manifest_from_json(json).unwrap();
        assert!(m.skip_cache_dirs);
        assert_eq!(m.smb_user, None);
        assert_eq!(m.exe_hint, None);
    }

    #[test]
    fn invalid_json_rejected() {
        assert!(matches!(manifest_from_json("not json"), Err(ManifestError::BadJson(_))));
        // 缺值 → 解析层拒绝（数字值是合法的，未知字段被语义层忽略）。
        assert!(matches!(manifest_from_json(r#"{"a": }"#), Err(ManifestError::BadJson(_))));
    }

    #[test]
    fn unicode_title_round_trip() {
        let m = sample();
        let parsed = manifest_from_json(&m.to_json()).unwrap();
        assert_eq!(parsed.title, "地平线：零之曙光");
    }

    #[test]
    fn escape_round_trip_smb_password() {
        let m = sample();
        let parsed = manifest_from_json(&m.to_json()).unwrap();
        assert_eq!(parsed.smb_pass.as_deref(), Some(r"p@ss w\ord"));
    }
}
