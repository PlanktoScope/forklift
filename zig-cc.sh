#!/bin/sh

log="$(mktemp)"

echo $@

exit_code=0
zig cc $@ 2>&1 || exit_code=$? | tee $log
exit $exit_code
