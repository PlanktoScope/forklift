#!/bin/bash -eux

chmod +w $(pwd)
if ls -l $(pwd) | grep "fsevents"; then
  ls -l $(pwd)/.. 1>&2
  ls -l $(pwd) 1>&2
  echo "zig cc $@" 1>&2
  strace zig cc $@ 1>&2
  exit $?
fi

zig cc $@ 1>&2
exit $?
