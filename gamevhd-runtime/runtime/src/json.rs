//! 零依赖 JSON 子集解析器（阶段 0，本波交付；box.json 专用）。
//!
//! 只支持本模型所需的扁平对象 `{ "k": "v", ... }`（值为字符串或布尔字面量）：
//! 字符串转义 `\"` `\\` `\n` `\r` `\t` `\b` `\f` `\/` 与 `\uXXXX`（含代理对）；
//! 布尔字面量 `true` / `false` 以文本 `"true"` / `"false"` 返回（供 `skip_cache_dirs`
//! 等布尔字段消费）。超集 JSON（数组/数字/null/嵌套）按非法输入报错——box.json
//! 是我们自己写的文件，严格是特性。错误以 `String`（消息）返回，由调用方包装为领域错误。

// 本模块当前仅被 boxfile 与测试使用；后续 wave（run/cleanup 生命周期）将直接消费。
#![allow(dead_code)]

/// 序列化字符串值：转义 `"` `\` 与控制字符。
pub fn escape_json(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out
}

/// 解析 `{ "k": "v", ... }` 扁平对象，返回键值对列表（保持出现顺序）。
pub fn parse_json_object(s: &str) -> Result<Vec<(String, String)>, String> {
    let s = s.trim();
    if !s.starts_with('{') || !s.ends_with('}') {
        return Err("期望对象（花括号包裹）".into());
    }
    let mut rest = s[1..s.len() - 1].trim();
    let mut fields = Vec::new();
    if rest.is_empty() {
        return Ok(fields);
    }
    loop {
        rest = rest.trim_start();
        if rest.is_empty() {
            return Err("期望字段名".into());
        }
        let (key, after_key) = parse_json_string(rest)?;
        let after_colon = after_key
            .trim_start()
            .strip_prefix(':')
            .ok_or_else(|| "字段名后期望 ':'".to_string())?;
        let (value, after_value) = parse_json_value(after_colon.trim_start())?;
        fields.push((key, value));
        let tail = after_value.trim_start();
        if let Some(r) = tail.strip_prefix(',') {
            rest = r;
            continue;
        }
        if tail.is_empty() {
            break;
        }
        return Err("期望 ',' 或对象结尾".into());
    }
    Ok(fields)
}

/// 解析一个 JSON 值：字符串字面量或布尔字面量（本模型不需要数字/null/嵌套）。
/// 返回 (解码值, 剩余输入)。布尔返回文本 `"true"` / `"false"`。
fn parse_json_value(s: &str) -> Result<(String, &str), String> {
    if s.starts_with('"') {
        return parse_json_string(s);
    }
    for lit in ["true", "false"] {
        if s.starts_with(lit) {
            let rest = &s[lit.len()..];
            let boundary_ok = match rest.chars().next() {
                None => true,
                Some(c) => !c.is_alphanumeric(),
            };
            if boundary_ok {
                return Ok((lit.to_string(), rest));
            }
        }
    }
    // 整数（可选负号）：hoststate 的 generation/pid/started_at 使用。
    // 返回原样文本，由调用方 parse 成数值。
    let rest = s.trim_start();
    let mut idx = 0usize;
    if rest.starts_with('-') {
        idx = 1;
    }
    let digits_start = idx;
    let bytes = rest.as_bytes();
    while idx < bytes.len() && bytes[idx].is_ascii_digit() {
        idx += 1;
    }
    if idx > digits_start {
        return Ok((rest[..idx].to_string(), &rest[idx..]));
    }
    Err("期望字符串、布尔或整数".into())
}

/// 解析一个 JSON 字符串字面量，返回 (解码值, 剩余输入)。
fn parse_json_string(s: &str) -> Result<(String, &str), String> {
    if !s.starts_with('"') {
        return Err("期望字符串".into());
    }
    let bytes = s.as_bytes();
    let mut out = String::new();
    let mut i = 1usize;
    while i < bytes.len() {
        match bytes[i] {
            b'"' => return Ok((out, &s[i + 1..])),
            b'\\' => {
                i += 1;
                if i >= bytes.len() {
                    return Err("转义序列截断".into());
                }
                match bytes[i] {
                    b'"' => out.push('"'),
                    b'\\' => out.push('\\'),
                    b'/' => out.push('/'),
                    b'n' => out.push('\n'),
                    b'r' => out.push('\r'),
                    b't' => out.push('\t'),
                    b'b' => out.push('\u{0008}'),
                    b'f' => out.push('\u{000c}'),
                    b'u' => {
                        let end = i + 5;
                        if end > bytes.len() {
                            return Err("\\u 转义截断".into());
                        }
                        let hex = &s[i + 1..end];
                        let code =
                            u32::from_str_radix(hex, 16).map_err(|_| "\\u 非十六进制".to_string())?;
                        i = end - 1;
                        if (0xd800..=0xdbff).contains(&code) {
                            // 高位代理：要求紧跟低位代理 \uDC00-\uDFFF。
                            let rest = &s[i + 1..];
                            if rest.starts_with("\\u") {
                                if let Ok(low) = u32::from_str_radix(&rest[2..6], 16) {
                                    if (0xdc00..=0xdfff).contains(&low) {
                                        let cp = 0x10000
                                            + ((code - 0xd800) << 10)
                                            + (low - 0xdc00);
                                        out.push(char::from_u32(cp)
                                            .ok_or_else(|| "\\u 码点非法".to_string())?);
                                        // rest 从 i+1 起，低位转义恰好 6 字节。
                                        i += 7;
                                        continue;
                                    }
                                }
                            }
                            return Err("孤立高位代理".into());
                        }
                        out.push(
                            char::from_u32(code).ok_or_else(|| "\\u 码点非法".to_string())?,
                        );
                    }
                    _ => return Err("未知转义序列".into()),
                }
            }
            _ => {
                // 非 ASCII 起始字节：按 UTF-8 字符整体拷贝。
                let c = s[i..]
                    .chars()
                    .next()
                    .ok_or_else(|| "UTF-8 截断".to_string())?;
                out.push(c);
                i += c.len_utf8();
                continue;
            }
        }
        i += 1;
    }
    Err("字符串未闭合".into())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn escape_backslash_and_quote() {
        assert_eq!(escape_json(r#"a\b"c"#), r#"a\\b\"c"#);
        assert_eq!(escape_json("tab\there"), r"tab\there");
        assert_eq!(escape_json("plain"), "plain");
    }

    #[test]
    fn escape_control_characters() {
        assert_eq!(escape_json("\u{1}"), r"\u0001");
        assert_eq!(escape_json("\u{1f}"), r"\u001f");
    }

    #[test]
    fn string_round_trip_via_escapes() {
        let raw = r#"C:\Users\Hao\Documents"#;
        let json = format!("\"{}\"", escape_json(raw));
        let (decoded, rest) = parse_json_string(&json).unwrap();
        assert_eq!(decoded, raw);
        assert_eq!(rest, "");
    }

    #[test]
    fn parses_escaped_unicode_and_controls() {
        let json = r#""snow\u2603\ud83d\ude00\t\n""#;
        let (s, rest) = parse_json_string(json).unwrap();
        assert_eq!(s, "snow\u{2603}\u{1f600}\t\n");
        assert_eq!(rest, "");
    }

    #[test]
    fn rejects_isolated_surrogate() {
        assert!(parse_json_string(r#""\ud83d""#).is_err());
        assert!(parse_json_string(r#""\ud83d\ud800""#).is_err(), "高位+高位");
    }

    #[test]
    fn parses_integer_values() {
        let (v, rest) = parse_json_value("42,").unwrap();
        assert_eq!(v, "42");
        assert_eq!(rest, ",");
        let (v, _) = parse_json_value("-7").unwrap();
        assert_eq!(v, "-7");
        let (v, _) = parse_json_value("true").unwrap();
        assert_eq!(v, "true");
        // 混合对象：字符串+整数。
        let fields = parse_json_object(r#"{"generation": 3, "pid": 1234, "state": "running"}"#).unwrap();
        assert_eq!(fields[0], ("generation".into(), "3".into()));
        assert_eq!(fields[1], ("pid".into(), "1234".into()));
        assert_eq!(fields[2], ("state".into(), "running".into()));
    }

    #[test]
    fn rejects_bad_strings() {
        let cases = [
            r#""unterminated"#,
            r#""\q""#,
            r#""\u12""#,
            r#""\uZZZZ""#,
            r#"not-a-string"#,
        ];
        for c in cases {
            assert!(parse_json_string(c).is_err(), "应失败: {c}");
        }
    }

    #[test]
    fn object_parse_basic() {
        let fields = parse_json_object(r#"{ "a": "1", "b" : "2" }"#).unwrap();
        assert_eq!(fields, vec![("a".into(), "1".into()), ("b".into(), "2".into())]);
    }

    #[test]
    fn object_parse_empty() {
        assert!(parse_json_object("{}").unwrap().is_empty());
    }

    #[test]
    fn object_parse_boolean_literals() {
        assert_eq!(
            parse_json_object(r#"{ "skip": true, "no": false }"#).unwrap(),
            vec![
                ("skip".into(), "true".into()),
                ("no".into(), "false".into())
            ]
        );
        assert!(parse_json_object(r#"{"a": truex}"#).is_err(), "truex 不是布尔");
        assert!(parse_json_object(r#"{"a": null}"#).is_err(), "null 不支持");
    }

    #[test]
    fn object_parse_rejects_malformed() {
        let cases = [
            r#"not json"#,
            r#"{"a": }"#,     // 缺值
            r#"{"a": "1",}"#, // 尾逗号
            r#"{"a" "1"}"#,   // 缺冒号
            r#"{"a": "1" "b": "2"}"#, // 缺逗号
            r#"{"a": "1", "a": "2"}"#, // 重复键（解析层放行，语义层决定）
        ];
        // 注意：重复键在解析层放行（返回两对），由上层（boxfile）拒绝。
        // 数字是合法 JSON 值（hoststate 使用），不再作为非法用例。
        for c in cases.iter().take(5) {
            assert!(parse_json_object(c).is_err(), "应失败: {c}");
        }
        assert_eq!(
            parse_json_object(r#"{"a": "1", "a": "2"}"#).unwrap().len(),
            2
        );
    }
}
