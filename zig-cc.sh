#!/bin/bash -eux

sudo chown $USER -R $(pwd)
sudo chown $USER -R $ZIG_GLOBAL_CACHE_DIR
sudo chown $USER -R $ZIG_LOCAL_CACHE_DIR
if ls -l $(pwd) | grep "fsevents"; then
  ls -l $(pwd)/.. 1>&2
  ls -l $(pwd) 1>&2
  ls -l /tmp 1>&2
  sudo chown $USER /tmp/cgo-gcc-input-*.o
  sudo chmod a+w /tmp/cgo-gcc-input-*.o
  ls -l /tmp 1>&2
  echo "zig cc $@" 1>&2
  strace zig cc $@ 1>&2
  exit $?
fi

zig cc $@ 1>&2
exit $?
