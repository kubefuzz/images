#!/usr/bin/env sh

set -e

# If you run the container in privileged mode, you can reconfigure the host system
# to have a better fuzzing performance by running the following command:
# afl-system-config

exec "$@"