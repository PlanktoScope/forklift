#!/bin/sh

log="$(mktemp)"

zig cc $@ 2&>1 $log
exit_code=$?
cat $log
exit $exit_code
