#!/bin/bash -x

zig cc $@ 1>&2
exit $?
