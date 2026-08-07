/*
 * rules.h — 重写规则表条目与匹配辅助函数
 *
 * 权威文档：docs/injection_protocol.md §4
 * 规则表是「前缀匹配」表：对目标绝对路径按表序逐条比较，第一条命中的条目
 * 决定该路径的动作（REWRITE / PASSTHROUGH）。规则语义见文档 §4.3。
 *
 * 布局与 hook_common.h 相同：自然对齐（无 #pragma pack），不含指针字段，
 * x86/x64 布局完全一致；末尾 _Static_assert 在双架构下强制校验。
 */

#ifndef GVHD_RULES_H
#define GVHD_RULES_H

#include "hook_common.h"

#ifdef __cplusplus
extern "C" {
#endif

/* ================================================================ */
/* 1. 规则条目 gvhd_rule_entry                                       */
/* ================================================================ */

struct gvhd_rule_entry {
    wchar_t  prefix[GVHD_RULE_STRING_MAX];      /* @0    匹配前缀（绝对路径，大小写不敏感） */
    wchar_t  replacement[GVHD_RULE_STRING_MAX]; /* @2048 REWRITE 时的替换前缀 */
    uint32_t flags;                             /* @4096 GVHD_RULE_FLAG_* */
    uint32_t reserved;                          /* @4100 必须为 0 */
};

/* 单条目动作标志：两个动作位互斥，同时置位时 REWRITE 优先 */
#define GVHD_RULE_FLAG_REWRITE      0x00000001u
#define GVHD_RULE_FLAG_PASSTHROUGH  0x00000002u

/* 规则动作的强类型视图（gvhd_rule_action_of 的解码结果） */
enum gvhd_rule_action {
    GVHD_RULE_ACTION_NONE        = 0,
    GVHD_RULE_ACTION_REWRITE     = 1,
    GVHD_RULE_ACTION_PASSTHROUGH = 2
};

_Static_assert(sizeof(struct gvhd_rule_entry) == 4104, "GVHD: 规则条目大小");
_Static_assert(offsetof(struct gvhd_rule_entry, replacement) == 2048, "GVHD: 规则条目 replacement 偏移");
_Static_assert(offsetof(struct gvhd_rule_entry, flags)       == 4096, "GVHD: 规则条目 flags 偏移");
_Static_assert(offsetof(struct gvhd_rule_entry, reserved)    == 4100, "GVHD: 规则条目 reserved 偏移");

/* ================================================================ */
/* 2. 匹配辅助                                                       */
/* ================================================================ */

/* ASCII 范围大小写折叠。规则前缀均为 ASCII（盘符 / Users / Documents /
 * AppData / Saved Games），与 Windows 序数大小写不敏感语义在 ASCII 区一致；
 * 非 ASCII 的完整语义由实现层（W3T9）用 CompareStringOrdinal 补充。 */
static inline wchar_t gvhd_ascii_lower(wchar_t c)
{
    if (c >= L'A' && c <= L'Z') {
        return (wchar_t)(c + (L'a' - L'A'));
    }
    return c;
}

/* 前缀匹配（大小写不敏感）。命中：返回匹配到的前缀长度（wchar 数）；
 * 未命中：返回 0。空前缀永不匹配——0 同时表示「未命中」，调用方据此区分。
 * 注意返回的是「前缀长度」，不是剩余长度；重写公式见文档 §4.3。 */
static inline size_t gvhd_rule_match_prefix(const wchar_t *path,
                                            const wchar_t *prefix)
{
    size_t i;

    if (prefix[0] == L'\0') {
        return 0;
    }
    for (i = 0; prefix[i] != L'\0'; ++i) {
        if (path[i] == L'\0') {
            return 0;
        }
        if (gvhd_ascii_lower(path[i]) != gvhd_ascii_lower(prefix[i])) {
            return 0;
        }
    }
    return i;
}

/* 解码条目动作；两个动作位同时置位时 REWRITE 优先。 */
static inline enum gvhd_rule_action
gvhd_rule_action_of(const struct gvhd_rule_entry *entry)
{
    if ((entry->flags & GVHD_RULE_FLAG_REWRITE) != 0) {
        return GVHD_RULE_ACTION_REWRITE;
    }
    if ((entry->flags & GVHD_RULE_FLAG_PASSTHROUGH) != 0) {
        return GVHD_RULE_ACTION_PASSTHROUGH;
    }
    return GVHD_RULE_ACTION_NONE;
}

#ifdef __cplusplus
}
#endif

#endif /* GVHD_RULES_H */
