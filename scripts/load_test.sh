#!/usr/bin/env bash
set -euo pipefail

URL="${1:-http://127.0.0.1:8080/}"
DURATION="${2:-30s}"
if (( $# > 2 )); then
  RATES=("${@:3}")
else
  RATES=(100 500 1000)
fi

for tool in vegeta jq; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Missing required command: $tool" >&2
    exit 1
  fi
done

RESULT_DIR="tmp/load-test-$(date +%Y%m%d-%H%M%S)-$$"
mkdir -p "$RESULT_DIR"
echo "Saving Vegeta results to $RESULT_DIR"

failed=0
for rate in "${RATES[@]}"; do
  if ! [[ "$rate" =~ ^(0|[1-9][0-9]*)$ ]]; then
    echo "Rate must be a non-negative integer: $rate" >&2
    exit 1
  fi

  result="$RESULT_DIR/$rate.gob"
  metrics="$RESULT_DIR/$rate.json"
  attack_options=(-rate="$rate")
  if [[ "$rate" == 0 ]]; then
    workers="${VEGETA_WORKERS:-20}"
    if ! [[ "$workers" =~ ^[1-9][0-9]*$ ]]; then
      echo "VEGETA_WORKERS must be a positive integer: $workers" >&2
      exit 1
    fi
    attack_options+=(-workers="$workers" -max-workers="$workers")
    label="$workers workers at unlimited rate"
  else
    label="$rate requests/s"
  fi
  echo "Testing $URL ($label) for $DURATION"
  printf 'GET %s\n' "$URL" | vegeta attack "${attack_options[@]}" -duration="$DURATION" -output="$result"
  vegeta report "$result"
  vegeta report -type=json "$result" > "$metrics"

  if jq -e '.requests > 0 and .success == 1 and .status_codes["200"] == .requests and ((.errors // []) | length) == 0' "$metrics" >/dev/null; then
    echo "PASS: $label"
  else
    echo "FAIL: $label (see $metrics)" >&2
    failed=1
  fi
done

exit "$failed"
