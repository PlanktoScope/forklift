#!/bin/bash -x

ls -l /tmp 1>&2

cat /tmp/cgo-gcc-input-*.c 1>&2 || true
zig cc $@ 1>&2
