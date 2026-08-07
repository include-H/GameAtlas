/*
 * gvhook — file.c
 *
 * 归属波次：W3T9（阶段 3 任务 9：文件系统重定向）——当前为 W2T6 阶段 1
 * 的「仅日志」桩：hook NtQueryAttributesFile，记录目标路径并调用原函数，
 * 用于验证拦截链路（实施计划阶段 1 验收）。不做任何重定向逻辑。
 *
 * 重定向全集（NtCreateFile/NtOpenFile/NtQueryDirectoryFile/NtDeleteFile/
 * NtSetInformationFile/NtQueryFullAttributesFile）在 W3T9 落地。
 */

#include <stdint.h>

#include <windows.h>
#include <winternl.h>

#include "hook_common.h"
#include "rules.h"
#include "internal.h"
#include "MinHook.h"

typedef NTSTATUS(NTAPI *P_NtQueryAttributesFile)(
    POBJECT_ATTRIBUTES ObjectAttributes,
    PFILE_BASIC_INFORMATION FileInformation);

/* __sys_* = ntdll 中真实函数地址（MH_CreateHook target） */
static LPVOID __sys_NtQueryAttributesFile = NULL;
/* 跳板：调用原函数 */
static P_NtQueryAttributesFile pfnOrig_NtQueryAttributesFile = NULL;

static NTSTATUS NTAPI Hook_NtQueryAttributesFile(
    POBJECT_ATTRIBUTES ObjectAttributes,
    PFILE_BASIC_INFORMATION FileInformation)
{
    /* 阶段 1 仅日志：详细日志开关（GVHD_PARAM_FLAG_LOG_VERBOSE）开启时
     * 记录目标路径，验证拦截链路；原函数恒被调用，行为不变。 */
    if ((gvhd_get_param()->flags & GVHD_PARAM_FLAG_LOG_VERBOSE) != 0 &&
        ObjectAttributes != NULL &&
        ObjectAttributes->ObjectName != NULL &&
        ObjectAttributes->ObjectName->Buffer != NULL) {
        const UNICODE_STRING *name = ObjectAttributes->ObjectName;

        gvhd_log_write(L"NtQueryAttributesFile %.*ls",
                       (int)(name->Length / sizeof(wchar_t)), name->Buffer);
    }
    return pfnOrig_NtQueryAttributesFile(ObjectAttributes, FileInformation);
}

uint32_t gvhd_install_file_hooks(void)
{
    HMODULE hNtdll;
    MH_STATUS mh;

    hNtdll = GetModuleHandleW(L"ntdll.dll");
    if (hNtdll == NULL) {
        return GVHD_INIT_ERR_HOOK;
    }
    __sys_NtQueryAttributesFile =
        (LPVOID)GetProcAddress(hNtdll, "NtQueryAttributesFile");
    if (__sys_NtQueryAttributesFile == NULL) {
        return GVHD_INIT_ERR_HOOK;
    }

    mh = MH_CreateHook(__sys_NtQueryAttributesFile,
                       (LPVOID)&Hook_NtQueryAttributesFile,
                       (LPVOID *)&pfnOrig_NtQueryAttributesFile);
    if (mh != MH_OK) {
        return GVHD_INIT_ERR_HOOK;
    }
    mh = MH_EnableHook(__sys_NtQueryAttributesFile);
    if (mh != MH_OK) {
        return GVHD_INIT_ERR_HOOK;
    }
    return 0;
}
