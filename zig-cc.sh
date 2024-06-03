#!/bin/sh

sudo chown $USER -R $ZIG_GLOBAL_CACHE_DIR
sudo chown $USER -R $ZIG_LOCAL_CACHE_DIR
zig cc $@ 1>&2
