/*
 * gvhook — init.c
 *
 * 归属波次：W2T6（阶段 2 任务 6：注入与 hook 骨架）。
 *
 * 本文件将实现（当前为空桩，仅保证编译）：
 *   - gvhook_init()：注入入口。按 docs/injection_protocol.md §5/§6 执行：
 *       校验参数块 magic/version → 复制私有副本（gvhd_get_param 等访问器）
 *       → 打开日志 → MinHook 初始化 → 安装 file/proc/reg 钩子 →
 *       写自检日志 HOOK_DLL_PRESENT / RULES_LOADED <n>。
 *   - 日志实现 gvhd_log_write()（格式见文档 §7）。
 *   - 参数块私有副本的存储与访问器。
 *
 * 相关原型声明见 hook_common.h。
 */
#include "hook_common.h"
#include "rules.h"
