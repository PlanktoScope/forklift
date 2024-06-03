#!/bin/sh

echo "zig-cc args: $@" 1>&2
echo $CC
echo $CXX

zig cc $@
