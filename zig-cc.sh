#!/bin/bash -eux

chmod +w $(pwd)
zig cc $@ 1>&2
exit_code=$?
ls -l $(pwd)
exit $exit_code
