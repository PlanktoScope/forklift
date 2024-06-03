#!/bin/bash -x

ls -lR $WORK 1>&2

zig env

zig cc $@

ls -lR $WORK 1>&2
