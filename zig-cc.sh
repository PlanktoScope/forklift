#!/bin/bash -eux

sudo chown $USER -R $(pwd)
sudo chown $USER -R $ZIG_GLOBAL_CACHE_DIR
sudo chown $USER -R $ZIG_LOCAL_CACHE_DIR
ls -l $ZIG_LOCAL_CACHE_DIR/.. 1>&2
ls -l $ZIG_LOCAL_CACHE_DIR 1>&2
ls -l $(pwd)/.. 1>&2
ls -l $(pwd) 1>&2
zig env 1>&2
echo "zig cc $@" 1>&2
strace zig cc $@ 1>&2
exit $?
