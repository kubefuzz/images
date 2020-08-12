#!/usr/bin/env sh

# This file draws inspiration from "afl-whatsup".
# https://github.com/AFLplusplus/AFLplusplus/blob/2.63c/afl-whatsup

set -e

# Configuration
AFL_SYNC_DIR=${AFL_SYNC_DIR:-"/fuzz/sync"}
FUZZER_STATS_TMP=`mktemp -t .afl-whatsup-XXXXXXXX` || exit 1
CURRENT_TIME=`date +%s`

# Exit if the directory doesn't exist
if [ ! -d "$AFL_SYNC_DIR" ]
then
  exit 0
fi

# Summary metrics defaults
ALIVE_COUNT=0
TOTAL_TIME=0
TOTAL_EXECS=0
TOTAL_EPS=0  # execs/sec
TOTAL_CRASHES=0

for FUZZER_STATS in `find ${AFL_SYNC_DIR} -maxdepth 2 -iname fuzzer_stats | sort`
do
  NAME=$(echo ${FUZZER_STATS} | cut -d '/' -f 4)
  # Prefix the line with AFL fuzzing name. Information is used by telegraf.
  # The output will look like this:
  #   master01 start_time        : 1587325610
  #   master01 execs_done        : 117750
  #   [...]
  sed -e "s/^/${NAME} /" ${FUZZER_STATS}

  # Format and read "fuzzer_stats" information
  # https://github.com/AFLplusplus/AFLplusplus/blob/2.65c/afl-whatsup#L129-L130
  sed 's/[ ]*:[ ]*/="/;s/$/"/' ${FUZZER_STATS} > "$FUZZER_STATS_TMP"
  . $FUZZER_STATS_TMP

  RUN_UNIX=$((CURRENT_TIME - start_time))

  # Convert float to int (by simply removing the decimals)
  # Read: `cut --delimiter '.' --fields 1`
  EXECS_PER_SEC_INT=$(echo $execs_per_sec | cut -d '.' -f 1)

  # Aggregate summary metrics
  ALIVE_COUNT=$((ALIVE_COUNT + 1))
  TOTAL_TIME=$((TOTAL_TIME + RUN_UNIX))
  TOTAL_EXECS=$((TOTAL_EXECS + execs_done))
  TOTAL_EPS=$((TOTAL_EPS + EXECS_PER_SEC_INT))
  TOTAL_CRASHES=$((TOTAL_CRASHES + unique_crashes))
done

# Only output a summary if at least one fuzzer is alive.
if [[ $ALIVE_COUNT -ne 0 ]]
then
  # Output summary so telegraf can read it
  echo "whatsup fuzzers_alive     : ${ALIVE_COUNT}"
  echo "whatsup total_time        : ${TOTAL_TIME}"
  echo "whatsup total_execs       : ${TOTAL_EXECS}"
  echo "whatsup cumulative_speed  : ${TOTAL_EPS}"
  echo "whatsup average_speed     : $((TOTAL_EPS / ALIVE_COUNT))"
  echo "whatsup crashes_found     : ${TOTAL_CRASHES}"
fi

# Clean up
if [ -f "$FUZZER_STATS_TMP" ]
then
  rm -f "$FUZZER_STATS_TMP"
fi