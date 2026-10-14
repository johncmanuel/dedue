#!/usr/bin/env bash
# Usage: ./scripts/run_cluster.sh [num_servers] [base_port] [listen_addr] [strategy]
# Example: ./scripts/run_cluster.sh 3 9001 127.0.0.1:8080 random

# Exit on error, undefined variable, or pipe failure
set -euo pipefail

N="${1:-2}"
BASE_PORT="${2:-9001}"
LISTEN="${3:-127.0.0.1:8080}"
STRATEGY="${4:-round-robin}"

if ! [[ "$N" =~ ^[1-9][0-9]*$ ]]; then
  echo "Usage: $0 [num_servers] [base_port] [listen_addr] [strategy]" >&2
  exit 1
fi

LOG_DIR="./tmp"
mkdir -p "$LOG_DIR"
MOCK_BIN="$LOG_DIR/mock_backend"
go build -o "$MOCK_BIN" ./cmd/mockbackend

pids=()
backends=()

cleanup() {
  trap - EXIT INT TERM
  echo "Stopping backend servers..."
  for pid in "${pids[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# Start N backend servers on consecutive ports
for ((i = 0; i < N; i++)); do
  port=$((BASE_PORT + i))
  "$MOCK_BIN" "$port" "backend-$port" >"$LOG_DIR/server_$port.log" 2>&1 &
  pids+=($!)
  backends+=("http://127.0.0.1:$port")
  echo "Started backend on port $port (pid $!), log: $LOG_DIR/server_$port.log"
done

# Wait until every backend is accepting connections (max ~5s each)
for ((i = 0; i < N; i++)); do
  port=$((BASE_PORT + i))
  ready=0
  for _ in $(seq 1 50); do
    if (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null; then
      ready=1
      break
    fi
    sleep 0.1
  done
  if [[ $ready -eq 0 ]]; then
    echo "Backend on port $port failed to start" >&2
    exit 1
  fi
done

echo "Running: go run . -listen $LISTEN -strategy $STRATEGY ${backends[*]}"
go run . -listen "$LISTEN" -strategy "$STRATEGY" "${backends[@]}" 2>&1 | tee "$LOG_DIR/go.log"
