#!/bin/sh

log="$(mktemp)"

echo "zig-cc args:" $@ | tee $log

exit_code=0
zig cc $@ 2> $log || exit_code=$?
cat $log
exit $exit_code
