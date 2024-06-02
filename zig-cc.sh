#!/bin/sh

echo "$ZIG_LOCAL_CACHE_DIR" >&2
exit 1
zig cc $@
