#!/bin/bash -x

sudo mkdir -p zig-build
sudo chown $USER -R zig-build
export ZIG_GLOBAL_CACHE_DIR="$(pwd)/zig-build"
export ZIG_LOCAL_CACHE_DIR="$(pwd)/zig-build"
zig env 1>&2
zig cc $@ 1>&2
