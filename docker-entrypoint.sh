#!/bin/bash
set -e

echo "Running $@"

if [ "$1" = "--" ]; then
  shift
  exec "$@"
else
  exec go-wrapper run "$@"
fi
