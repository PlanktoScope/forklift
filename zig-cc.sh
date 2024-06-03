#!/bin/bash -x

ls -l /tmp 1>&2

zig cc $@ 1>&2
exit_code=$?

ls -l /tmp 1>&2

exit $exit_code
