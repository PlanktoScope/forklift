#!/bin/sh

log="$(mktemp)"

echo "zig-cc args:" $@ | tee $log

exit_code=0
zig cc $@ 2>&1 || exit_code=$? | tee --append $log
exit $exit_code
