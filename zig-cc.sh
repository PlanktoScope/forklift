#!/bin/bash -x

ls -l /tmp 1>&2

zig env 1>&2
ls -l $ZIG_LOCAL_CACHE_DIR 1>&2
zig cc $@ 1>&2
