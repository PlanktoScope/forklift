#!/bin/sh

echo "zig-cc args:" $@

zig cc $@

echo "workspace:" $WORK
ls -l $WORK/..
ls -lR $WORK
