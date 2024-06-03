#!/bin/sh

echo "workspace: $WORK" 1>&2
ls -lR $WORK 1>&2

echo "zig cache: $ZIG_GLOBAL_CACHE_DIR" 1>&2
ls -l $ZIG_GLOBAL_CACHE_DIR 1>&2

echo "zig-cc args: $@" 1>&2

zig cc $@

echo "zig cache: $ZIG_GLOBAL_CACHE_DIR" 1>&2
ls -l $ZIG_GLOBAL_CACHE_DIR 1>&2

echo "workspace parent:" 1>&2
ls -l $WORK/.. 1>&2
echo "workspace: $WORK" 1>&2
ls -lR $WORK 1>&2
