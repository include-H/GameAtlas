/*
 * gvhook — internal.h
 *
 * 跨翻译单元的私有声明（hook 各 .c 文件之间共享，不属于对外 ABI）。
 * 对外的注入协议 ABI 全部在 hook_common.h / rules.h（冻结，勿改）。
 */

#ifndef GVHD_HOOK_INTERNAL_H
#define GVHD_HOOK_INTERNAL_H

#include <windows.h>

#include "hook_common.h"
#include "rules.h"

/* 钩子安装入口，由 gvhd_init() 在 MH_Initialize() 成功后调用。
 * 返回 0 成功；失败返回 GVHD_INIT_ERR_HOOK。 */
uint32_t gvhd_install_process_hooks(void);   /* proc.c */
uint32_t gvhd_install_file_hooks(void);      /* file.c */
uint32_t gvhd_install_registry_hooks(void);  /* reg.c */

/* 当前进程 id（诊断辅助）。 */
DWORD gvhd_current_pid(void);                /* init.c */

#endif /* GVHD_HOOK_INTERNAL_H */
