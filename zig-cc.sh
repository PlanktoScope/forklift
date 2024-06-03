#!/bin/bash -eux

sudo chown $USER -R $(pwd)
sudo chown $USER -R $ZIG_GLOBAL_CACHE_DIR
sudo chown $USER -R $ZIG_LOCAL_CACHE_DIR
echo "zig cc $@" 1>&2
zig cc $@ 1>&2
exit $?
