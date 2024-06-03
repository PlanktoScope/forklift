#!/bin/bash -x

if ls | grep "fsevents" > /dev/null; then
  pwd 1>&2
  ls -l 1>&2
  cat /tmp/cgo-gcc-input-* 1>&2
  ls /home/runner/go/pkg/mod/github.com/fsnotify/fsevents@v0.2.0/
fi

zig cc $@

if ls | grep "fsevents" > /dev/null; then
  zig env 1>&2
  ls -l 1>&2
fi
