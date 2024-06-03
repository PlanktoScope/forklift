#!/bin/bash -x

if ls "$WORK" | grep "fsevents" > /dev/null; then
  pwd 1>&2
  ls -l $WORK 1>&2
  cat /tmp/cgo-gcc-input-* 1>&2
  ls /home/runner/go/pkg/mod/github.com/fsnotify/fsevents@v0.2.0/
fi

if ls "$WORK" | grep "fsevents" > /dev/null; then
  zig env 1>&2
  zig cc $@
  ls -l $WORK 1>&2
fi
