#!/bin/sh

log="$(mktemp)"

exit_code=0
zig cc $@ --verbose 2>&1 || exit_code=$? | tee $log
exit $exit_code
