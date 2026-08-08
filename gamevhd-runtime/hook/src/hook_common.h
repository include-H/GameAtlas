/*
 * hook_common.h — GameVHD Runtime 注入协议（单一事实源）的 C 参考实现
 *
 * 权威文档：docs/injection_protocol.md
 * 双方（Rust 编排 gamevhd-runtime.exe + C 钩子 gvhook.dll）都以该文档为准；
 * 本头文件是其 C 侧参考实现，必须与文档逐字节一致。
 *
 * 协议变更规则：先改 docs/injection_protocol.md，再同步本头文件，
 * 并 bump GVHD_PROTOCOL_VERSION。
 *
 * ── 布局决策（为什么能用自然对齐、且 x86/x64 布局完全一致） ──────────
 *   1. 参数块不含任何指针字段：规则表用「相对字节偏移」而不是指针，
 *      因此 32/64 位进程使用同一份字节布局，杜绝 ABI 错位。
 *   2. 成员最大对齐为 4（uint32_t），wchar_t 对齐为 2；所有成员偏移天然
 *      对齐，编译器不插入任何 padding。不使用 #pragma pack。
 *   3. Windows 下 wchar_t 恒为 16 位（UTF-16LE），x86/x64 一致。
 *   4. 文件末尾的 _Static_assert 在两种架构下于编译期强制校验每个字段
 *      偏移与结构大小，任何布局漂移都会导致编译失败。
 *
 * 编译检查（两种架构都必须干净通过）：
 *   x86_64-w64-mingw32-gcc -fsyntax-only -std=c11 -Wall -Wextra hook_common.h
 *   i686-w64-mingw32-gcc    -fsyntax-only -std=c11 -Wall -Wextra hook_common.h
 */

#ifndef GVHD_HOOK_COMMON_H
#define GVHD_HOOK_COMMON_H

#include <stddef.h>  /* offsetof */
#include <stdint.h>  /* uint32_t */
#include <wchar.h>   /* wchar_t */

#ifdef __cplusplus
extern "C" {
#endif

/* ================================================================ */
/* 1. 协议常量                                                       */
/* ================================================================ */

#define GVHD_PARAM_MAGIC            0x44485647u  /* 'G''V''H''D' 小端序读出的 u32 */
#define GVHD_PROTOCOL_VERSION       1u           /* 协议版本，ABI 冻结基准 */

/* 固定宽度字符串缓冲（单位：WCHAR 数，含结尾 NUL） */
#define GVHD_PATH_MAX               512u   /* 路径类字段（hook DLL / GameDataRoot / USERPROFILE / 日志 / hive） */
#define GVHD_RULE_STRING_MAX        1024u  /* 规则前缀 / 替换串 */
#define GVHD_GAME_ID_MAX            64u    /* 游戏 id */

#define GVHD_RULE_MAX               32u    /* 编排允许发送的最大规则条数 */

/* 参数块 flags（bit 字段，可组合） */
#define GVHD_PARAM_FLAG_CHILD_INJECT  0x00000001u  /* 启用子进程自动注入（proc.c） */
#define GVHD_PARAM_FLAG_LOG_VERBOSE   0x00000002u  /* 详细日志 */
/* 实际游戏 VHD 盘符编码：1=A ... 26=Z，0 表示旧协议未提供。 */
#define GVHD_PARAM_FLAG_GAME_DRIVE_SHIFT 8u
#define GVHD_PARAM_FLAG_GAME_DRIVE_MASK  (0x1Fu << GVHD_PARAM_FLAG_GAME_DRIVE_SHIFT)

/* gvhook_init 返回值（编排经 GetExitCodeThread 读取诊断） */
#define GVHD_INIT_OK           0u
#define GVHD_INIT_ERR_MAGIC    1u  /* magic 不匹配（参数块不是本协议） */
#define GVHD_INIT_ERR_VERSION  2u  /* version 不受支持 */
#define GVHD_INIT_ERR_MINHOOK  3u  /* MinHook 初始化失败 */
#define GVHD_INIT_ERR_HOOK     4u  /* 钩子安装失败 */
#define GVHD_INIT_ERR_LOG      5u  /* 日志文件无法创建/打开 */

/* 日志 marker 片段（断言脚本对完整日志行做子串匹配，因此此处只约定片段，
 * 完整行格式见 docs/injection_protocol.md §7） */
#define GVHD_MARKER_PRESENT        "HOOK_DLL_PRESENT"
#define GVHD_MARKER_RULES_LOADED   "RULES_LOADED"
#define GVHD_MARKER_CHILD_INJECTED "CHILD_INJECTED"

/* ================================================================ */
/* 2. 参数块 gvhd_param_block                                        */
/* ================================================================ */
/* 单一布局，x86/x64 完全一致；总大小 5280 字节，4 对齐。
 * 注：规则条目结构 gvhd_rule_entry 在 rules.h 中定义。 */

struct gvhd_param_block {
    uint32_t magic;              /* @0    恒为 GVHD_PARAM_MAGIC */
    uint32_t version;            /* @4    恒为 GVHD_PROTOCOL_VERSION */
    uint32_t flags;              /* @8    GVHD_PARAM_FLAG_* */
    uint32_t rule_count;         /* @12   规则条数，<= GVHD_RULE_MAX */
    uint32_t rule_table_offset;  /* @16   参数块基址到首条规则的字节偏移（须为 4 的倍数） */
    uint32_t game_id_len;        /* @20   game_id 的 wchar 长度（不含 NUL），0 = 无 */
    uint32_t reserved[2];        /* @24   必须为 0，扩展预留 */
    wchar_t  hook_dll_path[GVHD_PATH_MAX];  /* @32     gvhook DLL 绝对路径（也是 LoadLibraryW 的远程线程参数） */
    wchar_t  game_data_root[GVHD_PATH_MAX]; /* @1056  沙箱重定向根，如 E:\GameData */
    wchar_t  user_profile[GVHD_PATH_MAX];   /* @2080  宿主 USERPROFILE，如 C:\Users\Hao */
    wchar_t  log_path[GVHD_PATH_MAX];       /* @3104  日志文件绝对路径 */
    wchar_t  registry_hive[GVHD_PATH_MAX];  /* @4128  hive 文件绝对路径，如 E:\GameData\Registry\user.dat */
    wchar_t  game_id[GVHD_GAME_ID_MAX];     /* @5152  游戏 id（hive 键名后缀），无则空串 */
};

_Static_assert(sizeof(wchar_t) == 2, "GVHD: Windows wchar_t 必须为 16 位");
_Static_assert(sizeof(struct gvhd_param_block) == 5280, "GVHD: gvhd_param_block 大小");
_Static_assert(offsetof(struct gvhd_param_block, magic)             == 0,    "GVHD: param.magic 偏移");
_Static_assert(offsetof(struct gvhd_param_block, version)           == 4,    "GVHD: param.version 偏移");
_Static_assert(offsetof(struct gvhd_param_block, flags)             == 8,    "GVHD: param.flags 偏移");
_Static_assert(offsetof(struct gvhd_param_block, rule_count)        == 12,   "GVHD: param.rule_count 偏移");
_Static_assert(offsetof(struct gvhd_param_block, rule_table_offset) == 16,   "GVHD: param.rule_table_offset 偏移");
_Static_assert(offsetof(struct gvhd_param_block, game_id_len)       == 20,   "GVHD: param.game_id_len 偏移");
_Static_assert(offsetof(struct gvhd_param_block, reserved)          == 24,   "GVHD: param.reserved 偏移");
_Static_assert(offsetof(struct gvhd_param_block, hook_dll_path)     == 32,   "GVHD: param.hook_dll_path 偏移");
_Static_assert(offsetof(struct gvhd_param_block, game_data_root)    == 1056, "GVHD: param.game_data_root 偏移");
_Static_assert(offsetof(struct gvhd_param_block, user_profile)      == 2080, "GVHD: param.user_profile 偏移");
_Static_assert(offsetof(struct gvhd_param_block, log_path)          == 3104, "GVHD: param.log_path 偏移");
_Static_assert(offsetof(struct gvhd_param_block, registry_hive)     == 4128, "GVHD: param.registry_hive 偏移");
_Static_assert(offsetof(struct gvhd_param_block, game_id)           == 5152, "GVHD: param.game_id 偏移");

/* ================================================================ */
/* 3. 导出宏 / 调用约定                                               */
/* ================================================================ */
/* 钩子 DLL 构建时定义 GVHD_BUILD_DLL（W2T6 的 Makefile 负责）。
 * x86（WIN32 且非 WIN64）用 __stdcall 与 LoadLibraryW 等一致；x64 无差别。 */

#if defined(_MSC_VER) || defined(__MINGW32__)
#  if defined(GVHD_BUILD_DLL)
#    define GVHD_API __declspec(dllexport)
#  else
#    define GVHD_API __declspec(dllimport)
#  endif
#  if defined(_WIN32) && !defined(_WIN64)
#    define GVHD_CALL __stdcall
#  else
#    define GVHD_CALL
#  endif
#else
#  define GVHD_API
#  define GVHD_CALL
#endif

/* 前向声明（完整定义在 rules.h） */
struct gvhd_rule_entry;

/* ================================================================ */
/* 4. 注入入口与 hook 内部共享 API（协议文档 §5 / §6 / §7）            */
/* ================================================================ */

/* 注入入口：编排（首进程）或 proc.c（子进程）以远程线程方式调用；
 * param_block 指向目标进程内存中的参数块。返回 GVHD_INIT_*。
 * 命名用 gvhook_init（计划书中的 hook_init），避免通用符号名在任意
 * 游戏进程的导出命名空间中与其它符号冲突。 */
GVHD_API uint32_t GVHD_CALL gvhook_init(void *param_block);

/* 日志：向参数块 log_path 指向的文件追加一行
 * "[gvhook] YYYY-MM-DD HH:MM:SS.mmm <fmt>\n"。 */
void gvhd_log_write(const wchar_t *fmt, ...);

/* gvhook_init 保存的私有副本访问器（子进程注入 / 钩子查规则用） */
const struct gvhd_param_block *gvhd_get_param(void);
const struct gvhd_rule_entry *gvhd_get_rules(void);
uint32_t gvhd_get_rule_count(void);

/* 子进程注入（proc.c 实现）：h_process / h_thread 为已 CREATE_SUSPENDED
 * 的子进程句柄；成功返回 0 且子进程已恢复运行，失败返回非 0。 */
uint32_t gvhd_inject_child(void *h_process, void *h_thread);

#ifdef __cplusplus
}
#endif

#endif /* GVHD_HOOK_COMMON_H */
