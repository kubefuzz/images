#!/usr/bin/env sh

# Base images
docker build -t kubefuzz/afl           ./afl
docker build -t kubefuzz/afl-plus-plus ./afl-plus-plus

# Syncing and monitoring
docker build -t kubefuzz/afl-sync ./afl-sync
docker build -t kubefuzz/telegraf ./telegraf

# Targets
docker build -t kubefuzz/target-libjpeg-turbo ./target-libjpeg-turbo
docker build -t kubefuzz/target-libxml2       ./target-libxml2
docker build -t kubefuzz/target-woff2         ./target-woff2