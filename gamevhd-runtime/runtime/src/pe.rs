//! PE 位数探测（阶段 0，本波交付；阶段 0 真机验收、阶段 5 位数选择编排均依赖）。
//!
//! 解析顺序：`MZ` 魔数 → `e_lfanew`（DOS 头偏移 0x3C，u32 LE）→ `PE\0\0` 签名 →
//! COFF machine（0x8664 = x64，0x14c = x86）→ 可选头 magic（0x20b = PE32+，0x10b = PE32）。
//! machine 与 magic 必须配对，否则判 NotPe（非法组合）。
//! 纯字节解析，无 IO；文件读取在 [`probe_file`]（不可读文件 → `io::Error`）。

use std::io;
use std::path::Path;

/// PE 位数判定结果。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PeKind {
    X64,
    X86,
    NotPe,
}

impl PeKind {
    pub fn as_str(self) -> &'static str {
        match self {
            PeKind::X64 => "x64",
            PeKind::X86 => "x86",
            PeKind::NotPe => "not-pe",
        }
    }
}

/// 探测字节流是否为合法 PE 及其位数。
pub fn probe(bytes: &[u8]) -> PeKind {
    let len = bytes.len() as u64;
    if len < 2 || bytes[0] != b'M' || bytes[1] != b'Z' {
        return PeKind::NotPe;
    }
    if len < 0x40 {
        // DOS 头太短，读不到 e_lfanew。
        return PeKind::NotPe;
    }
    let e_lfanew = u32::from_le_bytes([bytes[0x3c], bytes[0x3d], bytes[0x3e], bytes[0x3f]]) as u64;
    // 规范要求 PE 头不得与 DOS 头重叠（e_lfanew >= 0x40），且须在文件内。
    if e_lfanew < 0x40 || e_lfanew + 4 > len {
        return PeKind::NotPe;
    }
    let pe_off = e_lfanew as usize;
    if &bytes[pe_off..pe_off + 4] != b"PE\0\0" {
        return PeKind::NotPe;
    }
    // COFF 头 20 字节，machine 在签名后偏移 4；可选头 magic 在 COFF 之后偏移 0。
    let opt_off = e_lfanew + 4 + 20;
    if opt_off + 2 > len {
        return PeKind::NotPe;
    }
    let machine = u16::from_le_bytes([bytes[pe_off + 4], bytes[pe_off + 5]]);
    let magic = u16::from_le_bytes([bytes[opt_off as usize], bytes[opt_off as usize + 1]]);
    match (machine, magic) {
        (0x8664, 0x020b) => PeKind::X64,
        (0x014c, 0x010b) => PeKind::X86,
        _ => PeKind::NotPe,
    }
}

/// 读取文件并探测位数；文件不可读 → `io::Error`。
pub fn probe_file(path: &Path) -> io::Result<PeKind> {
    let bytes = std::fs::read(path)?;
    Ok(probe(&bytes))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    const MACHINE_X64: u16 = 0x8664;
    const MACHINE_X86: u16 = 0x014c;
    const MAGIC_PE32_PLUS: u16 = 0x020b;
    const MAGIC_PE32: u16 = 0x010b;

    /// 在内存中构造最小合法 PE：MZ stub（0x80 字节）→ e_lfanew=0x80 →
    /// "PE\0\0" → COFF（machine）→ 可选头前 2 字节（magic）。
    fn craft_pe(machine: u16, magic: u16) -> Vec<u8> {
        const E_LFANEW: usize = 0x80;
        let mut b = vec![0u8; E_LFANEW + 4 + 20 + 2];
        b[0] = b'M';
        b[1] = b'Z';
        b[0x3c..0x40].copy_from_slice(&(E_LFANEW as u32).to_le_bytes());
        b[E_LFANEW..E_LFANEW + 4].copy_from_slice(b"PE\0\0");
        b[E_LFANEW + 4..E_LFANEW + 6].copy_from_slice(&machine.to_le_bytes());
        b[E_LFANEW + 24..E_LFANEW + 26].copy_from_slice(&magic.to_le_bytes());
        b
    }

    #[test]
    fn detects_x64_pe32_plus() {
        assert_eq!(probe(&craft_pe(MACHINE_X64, MAGIC_PE32_PLUS)), PeKind::X64);
    }

    #[test]
    fn detects_x86_pe32() {
        assert_eq!(probe(&craft_pe(MACHINE_X86, MAGIC_PE32)), PeKind::X86);
    }

    #[test]
    fn not_pe_for_empty_and_garbage() {
        assert_eq!(probe(&[]), PeKind::NotPe);
        assert_eq!(probe(b"hello world"), PeKind::NotPe);
        assert_eq!(probe(b"\0\0\0\0PE\0\0"), PeKind::NotPe); // 有签名无 MZ
    }

    #[test]
    fn not_pe_for_truncated_dos_stub() {
        assert_eq!(probe(b"MZ"), PeKind::NotPe);
        let mut short = vec![0u8; 0x40 - 1];
        short[0] = b'M';
        short[1] = b'Z';
        assert_eq!(probe(&short), PeKind::NotPe);
    }

    #[test]
    fn not_pe_when_e_lfanew_points_outside_file() {
        let mut b = craft_pe(MACHINE_X64, MAGIC_PE32_PLUS);
        b[0x3c..0x40].copy_from_slice(&0xffff_ffffu32.to_le_bytes()); // 远超文件长度
        assert_eq!(probe(&b), PeKind::NotPe);
        b[0x3c..0x40].copy_from_slice(&0u32.to_le_bytes()); // 指向 DOS 头内部（<0x40）
        assert_eq!(probe(&b), PeKind::NotPe);
    }

    #[test]
    fn not_pe_when_signature_missing() {
        let mut b = craft_pe(MACHINE_X64, MAGIC_PE32_PLUS);
        b[0x80..0x84].copy_from_slice(b"XXXX"); // 覆盖 PE 签名
        assert_eq!(probe(&b), PeKind::NotPe);
    }

    #[test]
    fn not_pe_when_truncated_before_optional_magic() {
        let full = craft_pe(MACHINE_X64, MAGIC_PE32_PLUS);
        let cut_at_magic = full.len() - 1; // 去掉 magic 最后一个字节
        assert_eq!(probe(&full[..cut_at_magic]), PeKind::NotPe);
        let cut_at_coff = full.len() - 3; // 截到 COFF 中途
        assert_eq!(probe(&full[..cut_at_coff]), PeKind::NotPe);
    }

    #[test]
    fn not_pe_for_mismatched_machine_and_magic() {
        assert_eq!(probe(&craft_pe(MACHINE_X64, MAGIC_PE32)), PeKind::NotPe);
        assert_eq!(probe(&craft_pe(MACHINE_X86, MAGIC_PE32_PLUS)), PeKind::NotPe);
        assert_eq!(probe(&craft_pe(0x01c0, MAGIC_PE32)), PeKind::NotPe); // ARM
    }

    #[test]
    fn probe_file_reads_disk_and_reports_unreadable() {
        let path = std::env::temp_dir()
            .join(format!("gamevhd_pe_test_{}.exe", std::process::id()));
        fs::write(&path, craft_pe(MACHINE_X86, MAGIC_PE32)).unwrap();
        assert_eq!(probe_file(&path).unwrap(), PeKind::X86);
        let _ = fs::remove_file(&path);

        let missing = path.with_extension("missing");
        assert!(probe_file(&missing).is_err(), "不存在的文件应报错");
    }
}
