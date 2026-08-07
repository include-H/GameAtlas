/*
 * gvhook — file.c
 *
 * 归属波次：W3T9（阶段 3 任务 9：文件系统重定向）。
 *
 * 本文件将实现（当前为空桩，仅保证编译）——实施计划 §3.2「文件」全集：
 *   - NtCreateFile / NtOpenFile：路径重写 + write-copy 决策 + 父目录自动创建
 *   - NtQueryAttributesFile / NtQueryFullAttributesFile：读穿透（沙箱无 → 宿主）
 *   - NtQueryDirectoryFile：目录枚举合并（宿主 + 沙箱，去重，沙箱优先）
 *   - NtDeleteFile：删除重定向到沙箱路径
 *   - NtSetInformationFile：按句柄操作；FileRenameInformation 新路径重写
 *
 * 路径重写公式与规则匹配语义见 docs/injection_protocol.md §4.3 与 rules.h。
 */
#include "hook_common.h"
#include "rules.h"
