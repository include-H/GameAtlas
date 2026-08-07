# gvhook — 便捷入口：在 hook/ 目录执行 `make` 即构建 x64/x86 两个 DLL。
# 真正的构建逻辑在 src/Makefile（产物输出到本目录）。
all:
	$(MAKE) -C src

clean:
	$(MAKE) -C src clean

.PHONY: all clean
