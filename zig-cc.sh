#!/bin/sh

echo "zig-cc args: $@" 1>&2
go env 1>&2

zig cc $@
