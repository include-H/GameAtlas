# gvhook — GameVHD Runtime hook DLL (x64 + x86)，基于 MinHook。
#
# 用法：在 hook/ 目录执行 `make`（经 hook/Makefile 转发 `make -C src`）或
# 直接 `make -C src`。本 Makefile 内路径均以 src/ 为基准。
# 产物输出到 hook/（上一级）：gvhook-x64.dll / gvhook-x86.dll。
# 中间 .o 落在 src/build/{x64,x86}/。
#
# 来源：*.c（init/proc/file/reg）+ ../deps/minhook/src/{hook,buffer,trampoline}.c
#       + ../deps/minhook/src/hde/*.c（$(wildcard) 通配收集；hde32.c / hde64.c
#       各自按架构自守卫，见 hde/hde32.c 与 hde/hde64.c 顶部的 #if，编入任一
#       DLL 均安全）。
# 导出：定义 GVHD_BUILD_DLL（hook_common.h 的 GVHD_API → __declspec(dllexport)）。

CC64 := x86_64-w64-mingw32-gcc
CC32 := i686-w64-mingw32-gcc

CFLAGS := -std=c11 -Wall -Wextra -O2 -DGVHD_BUILD_DLL
CFLAGS_DLL := -shared -static-libgcc
# 第三方 MinHook 源：buffer.c 有一个已知的无害未用参数告警，用 flags 压制，
# 不改 vendored 代码。我们自己的源保持完整 -Wall -Wextra。
CFLAGS_THIRDPARTY := -Wno-unused-parameter
INCLUDES := -I. -I../deps/minhook/include

# x86 下 __stdcall 导出名会被装饰为 _gvhook_init@4，而编排用
# GetProcAddress("gvhook_init") 解析（协议 §5.2 步骤 6）→ 必须剥掉 @N 装饰。
# --kill-at 对 x64 无副作用（x64 无 stdcall 装饰）。
LDFLAGS32 := -Wl,--kill-at

HOOK_SRCS := $(wildcard *.c)
MINHOOK_SRCS := ../deps/minhook/src/hook.c ../deps/minhook/src/buffer.c ../deps/minhook/src/trampoline.c
HDE_SRCS := $(wildcard ../deps/minhook/src/hde/*.c)
THIRD_PARTY_SRCS := $(MINHOOK_SRCS) $(HDE_SRCS)

# eval 展开的编译规则先于 all: 出现，会抢占默认目标；显式固定默认目标。
.DEFAULT_GOAL := all

# 全部源 → 扁平化到各架构 build 目录的对象（各文件基名唯一，映射无歧义）
OBJS64 := $(addprefix build/x64/,$(notdir $(HOOK_SRCS:.c=.o)) $(notdir $(THIRD_PARTY_SRCS:.c=.o)))
OBJS32 := $(addprefix build/x86/,$(notdir $(HOOK_SRCS:.c=.o)) $(notdir $(THIRD_PARTY_SRCS:.c=.o)))

# 逐对象编译规则（foreach/eval 展开）；第三个参数为编译器，第四个为附加 flags
define COMPILE_template
$(2)/$(notdir $(1:.c=.o)): $(1) Makefile
	@mkdir -p $(2)
	$(3) $(CFLAGS) $(INCLUDES) $(4) -c $(1) -o $$@
endef

$(foreach s,$(HOOK_SRCS),$(eval $(call COMPILE_template,$(s),build/x64,$(CC64),)))
$(foreach s,$(THIRD_PARTY_SRCS),$(eval $(call COMPILE_template,$(s),build/x64,$(CC64),$(CFLAGS_THIRDPARTY))))
$(foreach s,$(HOOK_SRCS),$(eval $(call COMPILE_template,$(s),build/x86,$(CC32),)))
$(foreach s,$(THIRD_PARTY_SRCS),$(eval $(call COMPILE_template,$(s),build/x86,$(CC32),$(CFLAGS_THIRDPARTY))))

all: ../gvhook-x64.dll ../gvhook-x86.dll

../gvhook-x64.dll: $(OBJS64)
	$(CC64) $(CFLAGS_DLL) $(OBJS64) -o $@

../gvhook-x86.dll: $(OBJS32)
	$(CC32) $(CFLAGS_DLL) $(LDFLAGS32) $(OBJS32) -o $@

clean:
	rm -rf build
	rm -f ../gvhook-x64.dll ../gvhook-x86.dll

.PHONY: all clean
