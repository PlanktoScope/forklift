#!/bin/sh

echo "workspace: $WORK"
ls -lR $WORK

echo "zig cache: $ZIG_GLOBAL_CACHE_DIR"
ls -l $ZIG_GLOBAL_CACHE_DIR

echo "zig-cc args: $@"

zig cc $@

echo "zig cache: $ZIG_GLOBAL_CACHE_DIR"
ls -l $ZIG_GLOBAL_CACHE_DIR

echo "workspace parent:"
ls -l $WORK/..
echo "workspace: $WORK"
ls -lR $WORK
