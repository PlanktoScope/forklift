#!/bin/bash -x

if ls | grep "fsevents" > /dev/null; then
  pwd 1>&2
  ls -l 1>&2
  ls -l /tmp 1>&2
fi

zig cc $@ 1>&2
exit_code=$?

if ls | grep "fsevents" > /dev/null; then
  ls -l /tmp 1>&2
fi

exit $exit_code
