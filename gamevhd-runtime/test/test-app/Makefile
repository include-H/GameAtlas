# GameVHD Runtime — test-app (Windows test program, x64 + x86)
#
# Builds the same C source set twice with the two mingw-w64 toolchains.
# Outputs PE32+ (test-app-x64.exe) and PE32 (test-app-x86.exe).

CC_X64 := x86_64-w64-mingw32-gcc
CC_X86 := i686-w64-mingw32-gcc

CFLAGS := -Wall -Wextra -O2 -static

SRCS := main.c test_app_log.c inject_mode.c file_mode.c reg_mode.c child_mode.c
HDRS := test_app_log.h

.PHONY: all clean

all: test-app-x64.exe test-app-x86.exe

test-app-x64.exe: $(SRCS) $(HDRS)
	$(CC_X64) $(CFLAGS) -o $@ $(SRCS)

test-app-x86.exe: $(SRCS) $(HDRS)
	$(CC_X86) $(CFLAGS) -o $@ $(SRCS)

clean:
	rm -f test-app-x64.exe test-app-x86.exe test-app.log
