/*
 * gvhook — proc.c
 *
 * 归属波次：W2T6（阶段 2 任务 6：注入与 hook 骨架）。
 *
 * 本文件将实现（当前为空桩，仅保证编译）：
 *   - NtCreateUserProcess / NtCreateProcessEx 钩子：按文档 §6 强制子进程
 *     CREATE_SUSPENDED → 调用 gvhd_inject_child() → ResumeThread。
 *   - gvhd_inject_child()：在子进程内重放注入协议（分配远程区域 → 写入
 *     参数块与规则表 → LoadLibraryW → 远程 gvhook_init → ResumeThread），
 *     成功后写日志 CHILD_INJECTED pid=<pid>。
 *
 * 相关原型声明见 hook_common.h。
 */
#include "hook_common.h"
#include "rules.h"
