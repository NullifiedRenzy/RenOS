#!/usr/bin/env bash
# ============================================================================
# RenOS 1.0.0 — Benchmark Runner
# ============================================================================
# Usage:  ./run-benchmarks.sh [--profile] [--quick] [--output-dir PATH]
#
# Flags:
#   --profile      Capture pprof CPU / heap profiles after each run
#   --quick        Run reduced iterations (CI smoke test mode)
#   --output-dir   Output directory (default: ./results)
#
# Dependencies: curl, jq (optional for pretty JSON; falls back gracefully)
# Optional: hey, wrk, k6, pprof (each benchmark degrades gracefully if absent)
# ============================================================================

set -euo pipefail

# ─── Config ─────────────────────────────────────────────────────────────────
BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
WS_URL="${WS_URL:-ws://127.0.0.1:8080/ws}"
CORE_BIN="${CORE_BIN:-./bin/renos-core}"
CORE_CONFIG="${CORE_CONFIG:-./tests/benchmark/benchmark-config.json}"

OUTPUT_DIR=""
QUICK=false
PROFILE=false

# Parse flags
while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)   PROFILE=true; shift ;;
    --quick)     QUICK=true;  shift ;;
    --output-dir) OUTPUT_DIR="$2"; shift 2 ;;
    *) echo "Unknown flag: $1"; exit 1 ;;
  esac
done

if [[ -z "$OUTPUT_DIR" ]]; then
  OUTPUT_DIR="./results/benchmark-$(date +%Y%m%d-%H%M%S)"
fi
mkdir -p "$OUTPUT_DIR/raw"
RAW_DIR="$OUTPUT_DIR/raw"

ITER=$($QUICK && echo 2 || echo 5)
DURATION=$($QUICK && echo "10s" || echo "60s")
CONCURRENCY=$($QUICK && echo "10 25" || echo "10 50 100 250 500")

echo "=== RenOS Benchmark Runner ==="
echo "  Base URL:      $BASE_URL"
echo "  Output:        $OUTPUT_DIR"
echo "  Duration:      $DURATION"
echo "  Iterations:    $ITER"
echo "  Profile:       $PROFILE"
echo ""

# ─── Discovery ──────────────────────────────────────────────────────────────
HAVE_HEY=false; command -v hey  &>/dev/null && HAVE_HEY=true
HAVE_WRK=false; command -v wrk &>/dev/null && HAVE_WRK=true
HAVE_K6=false;  command -v k6  &>/dev/null && HAVE_K6=true
HAVE_PPROF=false; command -v pprof &>/dev/null && HAVE_PPROF=true
HAVE_JQ=false;  command -v jq   &>/dev/null && HAVE_JQ=true

log()  { printf "[%s] %s\n" "$(date +%H:%M:%S)" "$*"; }
die()  { log "FATAL: $*"; exit 1; }
json() { $HAVE_JQ && jq -c . "$1" 2>/dev/null || cat "$1"; }

check_endpoint() {
  local ep="$1"
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 3 "$BASE_URL$ep" 2>/dev/null || echo "000")
  [[ "$code" == "200" || "$code" == "401" ]] && return 0
  return 1
}

# ─── Build per-epoch header for concurrency benchmarks ──────────────────────
timestamp() { date -u '+%Y-%m-%dT%H:%M:%SZ'; }

# ─── Helpers: HTTP load generators ─────────────────────────────────────────-
run_http_bench() {
  local endpoint="$1"
  local concurrency="$2"
  local duration="$3"
  local outfile="$4"
  local label="$5"

  # Strip leading slash for filenames
  local safe_name="${endpoint//\//_}"
  safe_name="${safe_name/#_/}"

  log "Bench: $label ($concurrency conn, $duration)"

  if $HAVE_HEY; then
    hey -n "$((concurrency * 100))" -c "$concurrency" -z "$duration" \
        -o csv "$BASE_URL$endpoint" \
        > "$outfile" 2>"$outfile.err" || true
  elif $HAVE_WRK; then
    wrk -t "$(( concurrency < 4 ? concurrency : 4 ))" \
        -c "$concurrency" -d "$duration" \
        --latency "$BASE_URL$endpoint" \
        > "$outfile" 2>"$outfile.err" || true
  else
    log "  hey/wrk not found — falling back to curl sequential"
    {
      echo "fallback,latency_ms,code"
      local end_time
      end_time=$(( $(date +%s) + ${duration%s} ))
      while [[ $(date +%s) -lt $end_time ]]; do
        # Fire up to $concurrency requests in parallel via background curls
        for _ in $(seq 1 "$concurrency"); do
          (
            local start_ms end_ms elapsed
            start_ms=$(date +%s%3N)
            code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$BASE_URL$endpoint")
            end_ms=$(date +%s%3N)
            elapsed=$(( end_ms - start_ms ))
            echo "$(timestamp),$elapsed,$code"
          ) &
        done
        wait
      done
    } > "$outfile" 2>"$outfile.err"
  fi
}

run_ws_bench() {
  local num_conn="$1"
  local msgs_per_conn="$2"
  local outfile="$3"

  log "Bench: WebSocket ($num_conn connections, $msgs_per_conn msg each)"

  # Use a small Go or node oneliner if available; fall back to python3
  if command -v node &>/dev/null; then
    node -e "
    const WebSocket = require('ws');
    const conns = [];
    let done = 0;
    const start = Date.now();
    for (let i = 0; i < $num_conn; i++) {
      const ws = new WebSocket('$WS_URL');
      ws.on('open', () => {
        for (let j = 0; j < $msgs_per_conn; j++) {
          ws.send(JSON.stringify({type: 'ping', seq: j}));
        }
      });
      ws.on('message', () => {
        done++;
        if (done === $num_conn * $msgs_per_conn) {
          const elapsed = Date.now() - start;
          console.log(JSON.stringify({connections: $num_conn, messages: done, elapsed_ms: elapsed}));
          conns.forEach(c => c.close());
        }
      });
      conns.push(ws);
    }
    " 2>/dev/null > "$outfile" || echo '{"error":"node ws failed"}' > "$outfile"
  elif command -v python3 &>/dev/null; then
    python3 -c "
import asyncio, json, time, websockets
async def bench():
    uri = '$WS_URL'
    sent = 0; received = 0
    async def client(i):
        nonlocal sent, received
        async with websockets.connect(uri) as ws:
            for j in range($msgs_per_conn):
                await ws.send(json.dumps({'type':'ping','seq':j}))
                sent += 1
            for j in range($msgs_per_conn):
                await ws.recv()
                received += 1
    start = time.monotonic()
    await asyncio.gather(*[client(i) for i in range($num_conn)])
    elapsed = (time.monotonic() - start) * 1000
    print(json.dumps({'connections': $num_conn, 'messages': received, 'elapsed_ms': elapsed}))
asyncio.run(bench())
" 2>/dev/null > "$outfile" || echo '{"error":"python3 websockets failed"}' > "$outfile"
  else
    echo '{"error":"no ws runtime available (node or python3)"}' > "$outfile"
  fi
}

# ─── Profile capture ────────────────────────────────────────────────────────
capture_profile() {
  local label="$1"
  local pid="$2"
  local outdir="$3"

  if ! $HAVE_PPROF; then
    log "  pprof not available — skipping profile for $label"
    return
  fi

  local url="http://127.0.0.1:9090/debug/pprof"
  log "  Capturing pprof profile for $label (pid $pid) …"

  curl -s "${url}/profile?seconds=30" -o "$outdir/${label}-cpu.pprof" 2>/dev/null &
  curl -s "${url}/heap"               -o "$outdir/${label}-heap.pprof" 2>/dev/null &
  curl -s "${url}/goroutine?debug=2"  -o "$outdir/${label}-goroutines.txt" 2>/dev/null &
  wait
  log "  Profiles saved to $outdir/${label}-*.pprof"
}

# ─── Core server lifecycle ──────────────────────────────────────────────────
start_core() {
  local config="$1"
  local logfile="$2"

  if [[ ! -x "$CORE_BIN" ]]; then
    die "Core binary not found at $CORE_BIN — build with: make build-core"
  fi

  log "Starting Core server (config: $config) …"
  "$CORE_BIN" --config "$config" --pprof-addr "127.0.0.1:9090" \
    > "$logfile" 2>&1 &
  CORE_PID=$!
  log "Core PID: $CORE_PID"

  # Wait for readiness
  for i in $(seq 1 30); do
    if curl -s -o /dev/null -w "" --max-time 1 "$BASE_URL/healthz" 2>/dev/null; then
      log "Core ready after ${i}s"
      return 0
    fi
    sleep 1
  done
  die "Core failed to start within 30 s (see $logfile)"
}

stop_core() {
  local pid="$1"
  log "Stopping Core (pid $pid) …"
  kill -TERM "$pid" 2>/dev/null || true
  for i in $(seq 1 10); do
    kill -0 "$pid" 2>/dev/null || { log "Core stopped"; return 0; }
    sleep 1
  done
  log "Force-killing Core …"
  kill -KILL "$pid" 2>/dev/null || true
}

# ─── Metrics extraction ────────────────────────────────────────────────────
extract_hey_metrics() {
  local raw="$1"
  # hey CSV: response_time,status_code,error
  # Compute counts and percentiles with awk
  awk -F',' '
    NR>1 && $2 != "" {
      count++; sum += $1; codes[$2]++
      if ($3 != "") errors++
      vals[count] = $1
    }
    END {
      asort(vals)
      p50 = vals[int(count * 0.5)]
      p95 = vals[int(count * 0.95)]
      p99 = vals[int(count * 0.99)]
      p999 = vals[int(count * 0.999)]
      printf "requests %d\n", count
      printf "errors %d\n", errors
      printf "avg_ms %.2f\n", sum / count
      printf "p50_ms %.2f\n", p50
      printf "p95_ms %.2f\n", p95
      printf "p99_ms %.2f\n", p99
      printf "p999_ms %.2f\n", p999
      for (c in codes) printf "code_%s %d\n", c, codes[c]
    }
  ' "$raw" 2>/dev/null || echo "ERROR: failed to parse $raw"
}

extract_wrk_metrics() {
  local raw="$1"
  # wrk text output – parse key lines
  local req_s lat_p50 lat_p99 lat_p999
  req_s=$(grep 'Requests/sec:' "$raw" | awk '{print $2}')
  lat_p50=$(grep '50%' "$raw" | awk '{print $2}')
  lat_p99=$(grep '99%' "$raw" | awk '{print $2}')
  lat_p999=$(grep '99.9%' "$raw" | awk '{print $2}')
  echo "req_s $req_s"
  echo "p50_ms $lat_p50"
  echo "p99_ms $lat_p99"
  echo "p999_ms $lat_p999"
}

# ─── Generate summary markdown ──────────────────────────────────────────────
render_report() {
  local outdir="$1"
  local report="$outdir/benchmark-report.md"

  cat > "$report" <<- 'REPORT_HEADER'
# RenOS Benchmark Report

**Generated:** $(date -u '+%Y-%m-%d %H:%M:%S UTC')
**Version:** RenOS 1.0.0

## Summary

| Endpoint | Concurrency | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | p99.9 (ms) | Req/s | Errors |
|----------|------------|----------|----------|----------|----------|------------|-------|--------|
REPORT_HEADER

  for raw in "$outdir/raw"/http_*.csv "$outdir/raw"/http_*.txt; do
    [[ -f "$raw" ]] || continue

    local ep concurrency
    ep=$(basename "$raw" | sed -E 's/^http_(.+)_c[0-9]+\.(csv|txt)$/\1/')
    concurrency=$(basename "$raw" | sed -E 's/^http_.+_c([0-9]+)\..*$/\1/')

    if grep -q ',' "$raw" 2>/dev/null; then
      # hey CSV
      local metrics
      metrics=$(extract_hey_metrics "$raw")
      local avg p50 p95 p99 p999 req_s errors
      avg=$(echo "$metrics" | awk '/^avg_ms/ {printf "%.1f", $2}')
      p50=$(echo "$metrics" | awk '/^p50_ms/ {printf "%.1f", $2}')
      p95=$(echo "$metrics" | awk '/^p95_ms/ {printf "%.1f", $2}')
      p99=$(echo "$metrics" | awk '/^p99_ms/ {printf "%.1f", $2}')
      p999=$(echo "$metrics" | awk '/^p999_ms/ {printf "%.1f", $2}')
      errors=$(echo "$metrics" | awk '/^errors/ {print $2}')
      # Approximate req/s from count / duration
      req_s=$(echo "$metrics" | awk '/^requests/ {print $2}')
      echo "| /$ep | $concurrency | $avg | $p50 | $p95 | $p99 | $p999 | $req_s | $errors |" >> "$report"
    else
      # wrk text
      local metrics
      metrics=$(extract_wrk_metrics "$raw")
      local req_s p50 p99 p999
      req_s=$(echo "$metrics" | awk '/^req_s/ {print $2}')
      p50=$(echo "$metrics" | awk '/^p50_ms/ {print $2}')
      p99=$(echo "$metrics" | awk '/^p99_ms/ {print $2}')
      p999=$(echo "$metrics" | awk '/^p999_ms/ {print $2}')
      echo "| /$ep | $concurrency | — | $p50 | — | $p99 | $p999 | $req_s | — |" >> "$report"
    fi
  done

  # WebSocket section
  echo "" >> "$report"
  echo "## WebSocket" >> "$report"
  echo "| Connections | Messages | Elapsed (ms) | Msg/s |" >> "$report"
  echo "|------------|----------|-------------|-------|" >> "$report"
  for raw in "$outdir/raw"/ws_*.json; do
    [[ -f "$raw" ]] || continue
    local conn msg elapsed msgs
    conn=$(jq -r '.connections // 0' "$raw" 2>/dev/null || echo 0)
    msg=$(jq -r '.messages // 0' "$raw" 2>/dev/null || echo 0)
    elapsed=$(jq -r '.elapsed_ms // 0' "$raw" 2>/dev/null || echo 0)
    if [[ "$elapsed" -gt 0 ]]; then
      msgs=$(awk "BEGIN {printf \"%.0f\", $msg / ($elapsed / 1000)}")
    else
      msgs=0
    fi
    echo "| $conn | $msg | $elapsed | $msgs |" >> "$report"
  done

  echo "" >> "$report"
  echo "---" >> "$report"
  echo "_Report auto-generated by run-benchmarks.sh on $(date -u '+%Y-%m-%d %H:%M:%S UTC')_" >> "$report"
  echo "_Raw data: $outdir/raw/_" >> "$report"

  log "Report written to $report"
}

# ─── Main ───────────────────────────────────────────────────────────────────

# If --profile, check for pprof endpoint requirement
if $PROFILE && ! $HAVE_PPROF; then
  log "WARNING: --profile requested but pprof not found in PATH; will capture raw pprof data anyway"
fi

# 1. Start Core
CORE_LOG="$OUTPUT_DIR/core-server.log"
start_core "$CORE_CONFIG" "$CORE_LOG"
trap 'stop_core $CORE_PID' EXIT

# 2. Health check
log "Verifying endpoints …"
for ep in /healthz /api/v1/devices /api/v1/auth/login; do
  check_endpoint "$ep" || die "Endpoint $ep not reachable"
done
log "All endpoints OK"

# 3. HTTP benchmarks
ENDPOINTS=(
  "/healthz"
  "/api/v1/devices"
  "/api/v1/devices/sample-id"
  "/api/v1/auth/login"
)

for endpoint in "${ENDPOINTS[@]}"; do
  safe_name="${endpoint//\//_}"
  safe_name="${safe_name/#_/}"
  for concurrency in $CONCURRENCY; do
    rawfile="$RAW_DIR/http_${safe_name}_c${concurrency}.csv"
    for iter in $(seq 1 "$ITER"); do
      run_http_bench "$endpoint" "$concurrency" "$DURATION" "$rawfile" "$endpoint (c=$concurrency, iter $iter/$ITER)"
    done
    # Keep only the last run (simplification; CI would aggregate)
  done
done

# 4. WebSocket benchmarks
WS_CONNS=$($QUICK && echo "10" || echo "100 500 1000")
WS_MSGS=$($QUICK && echo "5" || echo "100")
for conns in $WS_CONNS; do
  rawfile="$RAW_DIR/ws_${conns}.json"
  run_ws_bench "$conns" "$WS_MSGS" "$rawfile"
done

# 5. Profile capture (under sustained load)
if $PROFILE; then
  log "Starting sustained load for profile capture …"
  # Generate background load
  if $HAVE_HEY; then
    hey -n 100000 -c 100 -z "60s" "$BASE_URL/api/v1/devices" > /dev/null 2>&1 &
    LOAD_PID=$!
  fi
  sleep 5
  capture_profile "core" "$CORE_PID" "$OUTPUT_DIR/profiles"
  if [[ -n "${LOAD_PID:-}" ]]; then
    wait "$LOAD_PID" 2>/dev/null || true
  fi
  log "Profile capture complete"
fi

# 6. System resource snapshot
log "Capturing system resource snapshot …"
{
  echo "=== Memory (free -m) ==="
  free -m
  echo ""
  echo "=== CPU (mpstat -P ALL 1 3) ==="
  mpstat -P ALL 1 3 2>/dev/null || sar -P ALL 1 3 2>/dev/null || echo "mpstat/sar not available"
  echo ""
  echo "=== Disk IO (iostat -x 1 3) ==="
  iostat -x 1 3 2>/dev/null || echo "iostat not available"
  echo ""
  echo "=== Core process stats ==="
  ps -o pid,%cpu,%mem,rss,vsz,args -p "$CORE_PID" 2>/dev/null || true
  echo ""
  echo "=== Open file descriptors ==="
  ls /proc/"$CORE_PID"/fd 2>/dev/null | wc -l || echo "N/A"
} > "$OUTPUT_DIR/system-resources.txt"

# 7. Generate report
render_report "$OUTPUT_DIR"

# 8. Archive raw data
log "Archiving raw data …"
tar -czf "$OUTPUT_DIR/raw-data.tar.gz" -C "$OUTPUT_DIR" raw/

log "=== Benchmark run complete ==="
log "Results: $OUTPUT_DIR"
log "Report:  $OUTPUT_DIR/benchmark-report.md"
log "Raw:     $OUTPUT_DIR/raw-data.tar.gz"
