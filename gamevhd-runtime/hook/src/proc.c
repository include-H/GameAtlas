/*
 * gvhook — proc.c
 *
 * 归属波次：W2T6（阶段 2 任务 6：注入与 hook 骨架）。
 *
 * 依据 docs/injection_protocol.md §6 实现：
 *   - Hook NtCreateUserProcess / NtCreateProcessEx（MinHook）。
 *   - 钩子强制子进程 CREATE_SUSPENDED（NT 层即 PS_CREATE_SUSPENDED，
 *     Win32 CREATE_SUSPENDED(0x4) 的等价位），调用原函数后再注入。
 *   - gvhd_inject_child()：在子进程内重放注入协议（§6.2）——
 *     分配远程区域 → 写入参数块+规则表 → LoadLibraryW → 远程 gvhook_init
 *     →（由钩子）ResumeThread；成功写 CHILD_INJECTED，失败写 CHILD_INJECT_FAILED。
 *
 * 与 §6.2 的两个实现偏差（详见交付报告）：
 *   1. GetExitCodeThread 只能取回 DWORD，x64 下 HMODULE 高 32 位被截断，
 *      无法得到子进程 hook DLL 基址 → 改为读子进程 PEB.Ldr 模块链表求真实
 *      DllBase（x86 下另保留退出码兜底）。
 *   2. 是否 ResumeThread 由「钩子是否自行强加挂起」决定：若调用方本就请求了
 *      挂起（如编排 CreateProcessW CREATE_SUSPENDED），ResumeThread 归调用方，
 *      钩子不代劳；仅在钩子自己强加挂起、或注入失败时（§6.2 步骤 4 放行）才恢复。
 *
 * // allow: SIZE_OK — 注入协议 §12 将「注入与 hook 骨架」固定归属本文件
 * （proc.c 为单一职责单元：子进程注入）。约 440 纯代码行超出 250 行上限，
 * 但结构与波次由协议规定，拆分将偏离单一事实源的波次文件划分。
 */

#include <stdint.h>
#include <stddef.h>
#include <wchar.h>

#include <windows.h>
#include <winternl.h>

#include "hook_common.h"
#include "rules.h"
#include "internal.h"
#include "MinHook.h"

/* ================================================================ */
/* NT 函数签名（GetProcAddress 解析，不静态链接 ntdll）                  */
/* ================================================================ */

/* NtCreateUserProcess 的 ProcessFlags 参数（ntpsapi.h 未随 mingw 提供）：
 * PS_CREATE_SUSPENDED = Win32 CREATE_SUSPENDED(0x4) 的 NT 层等价位。 */
#ifndef PS_CREATE_SUSPENDED
#define PS_CREATE_SUSPENDED 0x00000001u
#endif

typedef NTSTATUS(NTAPI *P_NtCreateUserProcess)(
    PHANDLE ProcessHandle,
    PHANDLE ThreadHandle,
    ACCESS_MASK ProcessDesiredAccess,
    ACCESS_MASK ThreadDesiredAccess,
    POBJECT_ATTRIBUTES ProcessObjectAttributes,
    POBJECT_ATTRIBUTES ThreadObjectAttributes,
    ULONG ProcessFlags,
    ULONG ThreadFlags,
    PRTL_USER_PROCESS_PARAMETERS ProcessParameters,
    PVOID CreateInfo,
    PVOID AttributeList);

typedef NTSTATUS(NTAPI *P_NtCreateProcessEx)(
    PHANDLE ProcessHandle,
    ACCESS_MASK DesiredAccess,
    POBJECT_ATTRIBUTES ObjectAttributes,
    HANDLE ParentProcess,
    ULONG Flags,
    HANDLE SectionHandle,
    HANDLE DebugPort,
    HANDLE ExceptionPort,
    BOOLEAN InJob);

typedef NTSTATUS(NTAPI *P_NtQueryInformationProcess)(
    HANDLE ProcessHandle,
    ULONG ProcessInformationClass,
    PVOID ProcessInformation,
    ULONG ProcessInformationLength,
    PULONG ReturnLength);

/* 注入失败时记录到 CHILD_INJECT_FAILED status= 的 NTSTATUS 值
 * （ntstatus.h 等价常量；用 u 后缀字面量避免 long 溢出告警）。 */
#ifndef STATUS_SUCCESS
#define STATUS_SUCCESS                      ((NTSTATUS)0)
#endif
#ifndef STATUS_INVALID_PARAMETER
#define STATUS_INVALID_PARAMETER            ((NTSTATUS)0xC000000Du)
#endif
#define GVHD_STATUS_PROCEDURE_NOT_FOUND     ((NTSTATUS)0xC000007Au)
#define GVHD_STATUS_INVALID_IMAGE_FORMAT    ((NTSTATUS)0xC000007Bu)
#define GVHD_STATUS_INVALID_USER_BUFFER     ((NTSTATUS)0xC0000008u)
#define GVHD_STATUS_PROCESS_IS_TERMINATING  ((NTSTATUS)0xC000010Au)
#define GVHD_STATUS_INSUFFICIENT_RESOURCES  ((NTSTATUS)0xC000009Au)
#define GVHD_STATUS_DLL_NOT_FOUND           ((NTSTATUS)0xC0000135u)

/* ================================================================ */
/* 全局：原函数地址（MinHook target）与跳板（original）                  */
/* ================================================================ */

/* __sys_* = ntdll 中真实函数地址（GetProcAddress 解析；MH_CreateHook target） */
static LPVOID __sys_NtCreateUserProcess = NULL;
static LPVOID __sys_NtCreateProcessEx    = NULL;
/* 跳板：调用原函数 */
static P_NtCreateUserProcess pfnOrig_NtCreateUserProcess = NULL;
static P_NtCreateProcessEx    pfnOrig_NtCreateProcessEx    = NULL;

static P_NtQueryInformationProcess __sys_NtQueryInformationProcess = NULL;

/* ================================================================ */
/* 小工具                                                             */
/* ================================================================ */

/* 大小写不敏感（ASCII 区）宽字符串比较。 */
static BOOLEAN gvhd_wcs_ieq(const wchar_t *a, const wchar_t *b)
{
    for (;;) {
        wchar_t ca = gvhd_ascii_lower(*a);
        wchar_t cb = gvhd_ascii_lower(*b);
        if (ca != cb) {
            return FALSE;
        }
        if (ca == L'\0') {
            return TRUE;
        }
        ++a;
        ++b;
    }
}

/* 子进程位宽不变式（协议 §6.3）：CreateRemoteThread 注入仅在同位数地址空间
 * 对之间成立。此函数用 IsWow64Process2 探测子进程机器类型，异位数则跳过
 * 注入并记警告。IsWow64Process2 不可用（旧系统/32 位 Windows）时放行。 */
static BOOLEAN gvhd_child_same_bits(HANDLE hProcess)
{
    typedef BOOL(WINAPI *P_IsWow64Process2)(HANDLE, USHORT *, USHORT *);
    static P_IsWow64Process2 pIsWow64Process2 = NULL;
    USHORT pMachine = IMAGE_FILE_MACHINE_UNKNOWN;
    USHORT nMachine = 0;

    if (pIsWow64Process2 == NULL) {
        pIsWow64Process2 = (P_IsWow64Process2)(LPVOID)GetProcAddress(
            GetModuleHandleW(L"kernel32.dll"), "IsWow64Process2");
        if (pIsWow64Process2 == NULL) {
            return TRUE;   /* 无法探测 → 放行（本钩子正常情况下只见到同位数子进程） */
        }
    }
    if (!pIsWow64Process2(hProcess, &pMachine, &nMachine)) {
        return TRUE;       /* 32 位 Windows：无 WoW64 → 必然是 x86 子进程 */
    }
#if defined(_WIN64)
    return pMachine == IMAGE_FILE_MACHINE_UNKNOWN ||   /* 原生 x64 */
           pMachine == IMAGE_FILE_MACHINE_AMD64;
#else
    return pMachine == IMAGE_FILE_MACHINE_UNKNOWN ||   /* 原生 x86 */
           pMachine == IMAGE_FILE_MACHINE_I386;
#endif
}

/* ================================================================ */
/* 子进程 hook DLL 基址解析（PEB 模块链表遍历）                          */
/* ================================================================ */

/* 模块名匹配：优先整路径大小写不敏感比较，退而求其次比较文件名。 */
static BOOLEAN gvhd_module_name_matches(const wchar_t *fullname,
                                        const wchar_t *want)
{
    const wchar_t *b1;
    const wchar_t *b2;

    if (gvhd_wcs_ieq(fullname, want)) {
        return TRUE;
    }
    b1 = wcsrchr(fullname, L'\\');
    b2 = wcsrchr(want, L'\\');
    b1 = (b1 != NULL) ? b1 + 1 : fullname;
    b2 = (b2 != NULL) ? b2 + 1 : want;
    return gvhd_wcs_ieq(b1, b2);
}

/* 读子进程 PEB.Ldr.InMemoryOrderModuleList，返回 hook DLL 的 DllBase
 * （完整 64 位地址；GetExitCodeThread 的 32 位退出码在 x64 下不可用）。
 * 找不到或读取失败返回 NULL。 */
static LPVOID gvhd_find_child_hook_module(HANDLE hProcess)
{
    PROCESS_BASIC_INFORMATION pbi;
    BYTE *pebLdrField;
    PPEB_LDR_DATA ldrAddr = NULL;
    PEB_LDR_DATA ldr;
    PVOID childHead;
    PVOID cur;
    const wchar_t *want = gvhd_get_param()->hook_dll_path;
    LPVOID found = NULL;
    unsigned iter;

    if (__sys_NtQueryInformationProcess == NULL ||
        __sys_NtQueryInformationProcess(hProcess, ProcessBasicInformation,
                                        &pbi, sizeof(pbi), NULL) < 0) {
        return NULL;
    }
    if (pbi.PebBaseAddress == NULL) {
        return NULL;
    }

    pebLdrField = (BYTE *)pbi.PebBaseAddress + offsetof(PEB, Ldr);
    if (!ReadProcessMemory(hProcess, pebLdrField, &ldrAddr,
                           sizeof(ldrAddr), NULL) ||
        ldrAddr == NULL) {
        return NULL;
    }
    if (!ReadProcessMemory(hProcess, ldrAddr, &ldr, sizeof(ldr), NULL)) {
        return NULL;
    }

    childHead = (BYTE *)ldrAddr + offsetof(PEB_LDR_DATA, InMemoryOrderModuleList);
    cur = ldr.InMemoryOrderModuleList.Flink;

    for (iter = 0; iter < 4096u && cur != NULL && cur != childHead; ++iter) {
        PLDR_DATA_TABLE_ENTRY te = (PLDR_DATA_TABLE_ENTRY)(
            (BYTE *)cur - offsetof(LDR_DATA_TABLE_ENTRY, InMemoryOrderLinks));
        LDR_DATA_TABLE_ENTRY ent;

        if (!ReadProcessMemory(hProcess, te, &ent, sizeof(ent), NULL)) {
            break;
        }
        if (ent.DllBase != NULL && ent.FullDllName.Buffer != NULL &&
            ent.FullDllName.Length > 0) {
            wchar_t namebuf[GVHD_PATH_MAX];
            SIZE_T chars = (SIZE_T)(ent.FullDllName.Length / sizeof(wchar_t));

            if (chars >= GVHD_PATH_MAX) {
                chars = GVHD_PATH_MAX - 1;
            }
            if (ReadProcessMemory(hProcess, ent.FullDllName.Buffer, namebuf,
                                  chars * sizeof(wchar_t), NULL)) {
                namebuf[chars] = L'\0';
                if (gvhd_module_name_matches(namebuf, want)) {
                    found = ent.DllBase;
                    break;
                }
            }
        }
        cur = ent.InMemoryOrderLinks.Flink;
    }
    return found;
}

/* ================================================================ */
/* 子进程注入实现（协议 §6.2）                                          */
/* ================================================================ */

/* 返回 0 成功；失败返回对应 NTSTATUS。resume 由调用方（钩子）决定。 */
static NTSTATUS gvhd_inject_child_impl(HANDLE hProcess)
{
    const struct gvhd_param_block *param = gvhd_get_param();
    const struct gvhd_rule_entry  *rules = gvhd_get_rules();
    uint32_t rule_count = gvhd_get_rule_count();
    SIZE_T   region_size =
        sizeof(struct gvhd_param_block) +
        (SIZE_T)rule_count * sizeof(struct gvhd_rule_entry);
    LPVOID  remote_base = NULL;
    LPVOID  pLoadLibraryW;   /* FARPROC 经 LPVOID 中转，避免 -Wcast-function-type */
    HANDLE  hThread1 = NULL;
    HANDLE  hThread2 = NULL;
    DWORD   thread1Result = 0;
    DWORD   initRc = 0;
    HMODULE hSelf = NULL;
    DWORD_PTR initRva = 0;
    LPVOID  remoteInit = NULL;
    LPVOID  childHmod = NULL;
    NTSTATUS status = GVHD_STATUS_PROCEDURE_NOT_FOUND;

    if (hProcess == NULL || hProcess == INVALID_HANDLE_VALUE) {
        return STATUS_INVALID_PARAMETER;
    }

    /* LoadLibraryW：kernel32 为 known-DLL，本进程解析地址即可（协议 §5.2 步骤 4） */
    pLoadLibraryW = (LPVOID)GetProcAddress(GetModuleHandleW(L"kernel32.dll"),
                                           "LoadLibraryW");
    if (pLoadLibraryW == NULL) {
        goto cleanup;
    }

    /* 1. 单块远程内存（MEM_COMMIT|MEM_RESERVE, PAGE_READWRITE） */
    remote_base = VirtualAllocEx(hProcess, NULL, region_size,
                                 MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE);
    if (remote_base == NULL) {
        status = GVHD_STATUS_INSUFFICIENT_RESOURCES;
        goto cleanup;
    }

    /* 2. 写入参数块 + 规则表：父进程私有副本整块逐字节复制，
     *    rule_table_offset 为相对偏移，无需地址翻译（§6.2 步骤 2）。 */
    if (!WriteProcessMemory(hProcess, remote_base, param,
                            sizeof(struct gvhd_param_block), NULL)) {
        status = GVHD_STATUS_INVALID_USER_BUFFER;
        goto cleanup;
    }
    if (rule_count > 0 &&
        !WriteProcessMemory(hProcess, (BYTE *)remote_base + param->rule_table_offset,
                            rules, (SIZE_T)rule_count * sizeof(struct gvhd_rule_entry),
                            NULL)) {
        status = GVHD_STATUS_INVALID_USER_BUFFER;
        goto cleanup;
    }

    /* 3. 线程①：LoadLibraryW(hook_dll_path)。路径就在参数块内（§5.2 步骤 4）。 */
    hThread1 = CreateRemoteThread(
        hProcess, NULL, 0, (LPTHREAD_START_ROUTINE)pLoadLibraryW,
        (LPVOID)((BYTE *)remote_base + offsetof(struct gvhd_param_block, hook_dll_path)),
        0, NULL);
    if (hThread1 == NULL) {
        status = GVHD_STATUS_PROCESS_IS_TERMINATING;
        goto cleanup;
    }
    WaitForSingleObject(hThread1, INFINITE);
    GetExitCodeThread(hThread1, &thread1Result);

    /* 4. 解析子进程内 gvhook_init 地址：RVA 换算（同 DLL 文件 → 同导出 RVA）。
     *    子进程 hmod 不能从线程退出码取（x64 截断）→ PEB 模块链表遍历；
     *    x86 下退出码完整，可作兜底。 */
    childHmod = gvhd_find_child_hook_module(hProcess);
#if !defined(_WIN64)
    if (childHmod == NULL && thread1Result != 0) {
        childHmod = (LPVOID)(DWORD_PTR)thread1Result;
    }
#endif
    if (childHmod == NULL) {
        status = (thread1Result == 0)
                     ? GVHD_STATUS_DLL_NOT_FOUND
                     : GVHD_STATUS_INVALID_IMAGE_FORMAT;
        goto cleanup;
    }

    if (!GetModuleHandleExW(
            GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS |
                GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
            (LPCWSTR)(uintptr_t)(void *)&gvhook_init, &hSelf)) {
        status = GVHD_STATUS_INVALID_IMAGE_FORMAT;
        goto cleanup;
    }
    initRva = (DWORD_PTR)(uintptr_t)(void *)&gvhook_init - (DWORD_PTR)(uintptr_t)hSelf;
    remoteInit = (LPVOID)((DWORD_PTR)(uintptr_t)childHmod + initRva);

    /* 5. 线程②：gvhook_init(子进程内参数块基址) */
    hThread2 = CreateRemoteThread(hProcess, NULL, 0,
                                  (LPTHREAD_START_ROUTINE)remoteInit,
                                  remote_base, 0, NULL);
    if (hThread2 == NULL) {
        status = GVHD_STATUS_PROCESS_IS_TERMINATING;
        goto cleanup;
    }
    WaitForSingleObject(hThread2, INFINITE);
    GetExitCodeThread(hThread2, &initRc);
    if (initRc != GVHD_INIT_OK) {
        status = (NTSTATUS)initRc;   /* 子进程 gvhook_init 返回码 */
        goto cleanup;
    }

    status = STATUS_SUCCESS;

cleanup:
    if (hThread1 != NULL) {
        CloseHandle(hThread1);
    }
    if (hThread2 != NULL) {
        CloseHandle(hThread2);
    }
    /* 子进程 hook 已在 gvhook_init 内保存私有副本（§6.1），可安全释放远程区域 */
    if (remote_base != NULL) {
        VirtualFreeEx(hProcess, remote_base, 0, MEM_RELEASE);
    }
    return status;
}

/* 对外入口（hook_common.h）：h_thread 参数保留（协议签名），恢复线程由
 * 钩子按「是否自行强加挂起」决定，本函数不做。返回 0 成功。 */
uint32_t gvhd_inject_child(void *h_process, void *h_thread)
{
    HANDLE hp = (HANDLE)h_process;
    NTSTATUS status;

    (void)h_thread;

    if (hp == NULL || hp == INVALID_HANDLE_VALUE) {
        gvhd_log_write(L"CHILD_INJECT_FAILED pid=0 status=0x%08lx",
                       (unsigned long)STATUS_INVALID_PARAMETER);
        return (uint32_t)STATUS_INVALID_PARAMETER;
    }

    status = gvhd_inject_child_impl(hp);
    if (status == STATUS_SUCCESS) {
        gvhd_log_write(L"CHILD_INJECTED pid=%lu",
                       (unsigned long)GetProcessId(hp));
        return 0;
    }
    gvhd_log_write(L"CHILD_INJECT_FAILED pid=%lu status=0x%08lx",
                   (unsigned long)GetProcessId(hp), (unsigned long)status);
    return (uint32_t)status;
}

/* ================================================================ */
/* 钩子函数                                                           */
/* ================================================================ */

static NTSTATUS NTAPI Hook_NtCreateUserProcess(
    PHANDLE ProcessHandle,
    PHANDLE ThreadHandle,
    ACCESS_MASK ProcessDesiredAccess,
    ACCESS_MASK ThreadDesiredAccess,
    POBJECT_ATTRIBUTES ProcessObjectAttributes,
    POBJECT_ATTRIBUTES ThreadObjectAttributes,
    ULONG ProcessFlags,
    ULONG ThreadFlags,
    PRTL_USER_PROCESS_PARAMETERS ProcessParameters,
    PVOID CreateInfo,
    PVOID AttributeList)
{
    ULONG flags = ProcessFlags;
    BOOLEAN weForcedSuspend = FALSE;
    NTSTATUS status;

    /* 强制子进程挂起创建，保证注入先于其任何用户代码（协议 §6.2 / §6.4） */
    if ((flags & PS_CREATE_SUSPENDED) == 0) {
        flags |= PS_CREATE_SUSPENDED;
        weForcedSuspend = TRUE;
    }

    status = pfnOrig_NtCreateUserProcess(
        ProcessHandle, ThreadHandle, ProcessDesiredAccess, ThreadDesiredAccess,
        ProcessObjectAttributes, ThreadObjectAttributes, flags, ThreadFlags,
        ProcessParameters, CreateInfo, AttributeList);

    if (NT_SUCCESS(status) && ProcessHandle != NULL && ThreadHandle != NULL &&
        *ProcessHandle != NULL && *ThreadHandle != NULL) {
        HANDLE hp = *ProcessHandle;
        HANDLE ht = *ThreadHandle;

        if (gvhd_child_same_bits(hp)) {
            uint32_t rc = gvhd_inject_child(hp, ht);

            /* 仅当钩子自己强加挂起时恢复（调用方请求的挂起归调用方）；
             * 注入失败按协议 §6.2 步骤 4 恢复放行，绝不让子进程永久冻结。 */
            if (rc != 0 || weForcedSuspend) {
                ResumeThread(ht);
            }
        } else {
            gvhd_log_write(L"CHILD_BITS_MISMATCH pid=%lu: injection skipped",
                           (unsigned long)GetProcessId(hp));
            if (weForcedSuspend) {
                ResumeThread(ht);
            }
        }
    }
    return status;
}

static NTSTATUS NTAPI Hook_NtCreateProcessEx(
    PHANDLE ProcessHandle,
    ACCESS_MASK DesiredAccess,
    POBJECT_ATTRIBUTES ObjectAttributes,
    HANDLE ParentProcess,
    ULONG Flags,
    HANDLE SectionHandle,
    HANDLE DebugPort,
    HANDLE ExceptionPort,
    BOOLEAN InJob)
{
    NTSTATUS status = pfnOrig_NtCreateProcessEx(
        ProcessHandle, DesiredAccess, ObjectAttributes, ParentProcess, Flags,
        SectionHandle, DebugPort, ExceptionPort, InJob);

    /* NtCreateProcessEx 不创建初始线程（线程由随后的 NtCreateThreadEx 创建），
     * 因此没有可挂起/恢复的线程句柄；注入为尽力而为——子进程在调用方创建
     * 线程之前不会运行任何用户代码，注入通常先完成。 */
    if (NT_SUCCESS(status) && ProcessHandle != NULL && *ProcessHandle != NULL) {
        HANDLE hp = *ProcessHandle;

        if (gvhd_child_same_bits(hp)) {
            gvhd_inject_child(hp, NULL);
        } else {
            gvhd_log_write(L"CHILD_BITS_MISMATCH pid=%lu: injection skipped",
                           (unsigned long)GetProcessId(hp));
        }
    }
    return status;
}

/* ================================================================ */
/* 安装入口（init.c 调用）                                             */
/* ================================================================ */

uint32_t gvhd_install_process_hooks(void)
{
    HMODULE hNtdll;
    MH_STATUS mh;

    hNtdll = GetModuleHandleW(L"ntdll.dll");
    if (hNtdll == NULL) {
        return GVHD_INIT_ERR_HOOK;
    }

    __sys_NtCreateUserProcess = (LPVOID)GetProcAddress(hNtdll, "NtCreateUserProcess");
    __sys_NtCreateProcessEx = (LPVOID)GetProcAddress(hNtdll, "NtCreateProcessEx");
    __sys_NtQueryInformationProcess = (P_NtQueryInformationProcess)(LPVOID)GetProcAddress(
        hNtdll, "NtQueryInformationProcess");
    if (__sys_NtCreateUserProcess == NULL || __sys_NtCreateProcessEx == NULL ||
        __sys_NtQueryInformationProcess == NULL) {
        return GVHD_INIT_ERR_HOOK;
    }

    mh = MH_CreateHook(__sys_NtCreateUserProcess,
                       (LPVOID)&Hook_NtCreateUserProcess,
                       (LPVOID *)&pfnOrig_NtCreateUserProcess);
    if (mh != MH_OK) {
        return GVHD_INIT_ERR_HOOK;
    }
    mh = MH_CreateHook(__sys_NtCreateProcessEx,
                       (LPVOID)&Hook_NtCreateProcessEx,
                       (LPVOID *)&pfnOrig_NtCreateProcessEx);
    if (mh != MH_OK) {
        return GVHD_INIT_ERR_HOOK;
    }

    mh = MH_EnableHook(__sys_NtCreateUserProcess);
    if (mh != MH_OK) {
        return GVHD_INIT_ERR_HOOK;
    }
    mh = MH_EnableHook(__sys_NtCreateProcessEx);
    if (mh != MH_OK) {
        return GVHD_INIT_ERR_HOOK;
    }
    return 0;
}
