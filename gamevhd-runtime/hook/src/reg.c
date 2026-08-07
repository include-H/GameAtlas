/*
 * gvhook — reg.c
 *
 * 归属波次：W3T12（阶段 3 任务 12：注册表重定向）。
 *
 * 本文件将实现（当前为空桩，仅保证编译）——实施计划 §3.2「注册表」全集：
 *   - NtOpenKey(Ex) / NtCreateKey(Ex)（含 Transacted 变体）：
 *     \REGISTRY\USER\<SID>\Software → \REGISTRY\USER\GameVHD_<game_id>\Software
 *   - NtSetValueKey / NtDeleteValueKey：写入重定向
 *   - NtDeleteKey(Ex)：删除重定向（沙箱无 → NOT_FOUND，不动宿主）
 *   - NtQueryValueKey / NtQueryMultipleValueKey / NtEnumerateValueKey：
 *     读穿透 + 枚举合并
 *   - NtQueryKey / NtEnumerateKey：读穿透 + 枚举合并
 *   - NtRenameKey / NtNotifyChangeKey：先直通宿主（已知局限）
 *
 * 重写目标键名由参数块 game_id 派生（docs/injection_protocol.md §6.5）。
 */
#include "hook_common.h"
#include "rules.h"
