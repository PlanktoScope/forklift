#!/bin/bash -x

mkdir -p zig-build
ZIG_GLOBAL_CACHE_DIR="$(pwd)/zig-build"
ZIG_LOCAL_CACHE_DIR="$(pwd)/zig-build"
zig cc $@ 1>&2
