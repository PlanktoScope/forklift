#!/bin/sh

echo "zig-cc args:" $@ | tee $log

zig cc $@
