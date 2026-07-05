#!/bin/bash
# 9R-S2 — NAS Whisper benchmark + OpenVINO iGPU eval (run ON the NAS)
#
# Usage:   ./whisper-nas-benchmark.sh "/path/to/a/real/episode.mkv" [slice_minutes]
# Output:  ./whisper-bench-results.txt (+ stdout)
#
# What it does:
#   0. Prints CPU model / threads / RAM / /dev/dri (iGPU) presence
#   1. Extracts a N-minute (default 10) 16kHz mono WAV slice from the episode (ffmpeg,
#      host binary if present else dockerized linuxserver/ffmpeg)
#   2. Benchmarks Speaches (faster-whisper, CPU, int8) with models base + small via the
#      OpenAI-compatible POST /v1/audio/transcriptions
#   3. If /dev/dri exists: benchmarks WhisperLive OpenVINO backend (Intel iGPU) via its
#      python client (dockerized python:3.11-slim, pip install whisper-live)
#   4. Reports xRT (audio_seconds / wall_seconds) and extrapolated minutes per 45-min episode
#
# Pass gate (spike S1... S2): measured min/episode for each engine; go/no-go on local default.
set -uo pipefail

MEDIA="${1:?usage: $0 /path/to/episode.mkv [slice_minutes]}"
SLICE_MIN="${2:-10}"
WORK="$(pwd)/whisper-bench-work"
OUT="$(pwd)/whisper-bench-results.txt"
AUDIO="$WORK/slice.wav"
SPEACHES_PORT=8971
WL_PORT=9090
mkdir -p "$WORK"

log() { echo "$@" | tee -a "$OUT"; }
: > "$OUT"

log "=== 9R-S2 Whisper NAS benchmark — $(date) ==="
log "--- system ---"
if [ -r /proc/cpuinfo ]; then
  log "CPU: $(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2- | sed 's/^ //')"
  log "Threads: $(nproc)"
else
  log "CPU: $(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo unknown)"
  log "Threads: $(sysctl -n hw.ncpu 2>/dev/null || echo unknown)"
fi
log "RAM: $(free -h 2>/dev/null | awk '/Mem:/{print $2}' || echo unknown)"
if [ -d /dev/dri ]; then log "iGPU: /dev/dri PRESENT ($(ls /dev/dri | tr '\n' ' '))"; else log "iGPU: /dev/dri ABSENT — OpenVINO iGPU test will be SKIPPED"; fi

log "--- audio slice (${SLICE_MIN} min, 16kHz mono) ---"
FF_ARGS=(-y -ss 300 -t $((SLICE_MIN*60)) -i "$MEDIA" -vn -ac 1 -ar 16000 -c:a pcm_s16le "$AUDIO")
if command -v ffmpeg >/dev/null 2>&1; then
  ffmpeg "${FF_ARGS[@]}" >/dev/null 2>&1
else
  docker run --rm -v "$(dirname "$MEDIA")":/in:ro -v "$WORK":/out linuxserver/ffmpeg \
    -y -ss 300 -t $((SLICE_MIN*60)) -i "/in/$(basename "$MEDIA")" -vn -ac 1 -ar 16000 -c:a pcm_s16le /out/slice.wav >/dev/null 2>&1
fi
[ -s "$AUDIO" ] || { log "FATAL: audio slice extraction failed"; exit 1; }
AUDIO_SEC=$((SLICE_MIN*60))
log "slice: $AUDIO ($(du -h "$AUDIO" | cut -f1))"

bench_speaches() {
  local model="$1"
  log "--- Speaches CPU / faster-whisper-$model (int8) ---"
  docker rm -f s2-speaches >/dev/null 2>&1
  docker run -d --name s2-speaches -p $SPEACHES_PORT:8000 \
    -v s2-hf-cache:/home/ubuntu/.cache/huggingface/hub \
    -e WHISPER__MODEL="Systran/faster-whisper-$model" \
    -e WHISPER__COMPUTE_TYPE=int8 \
    ghcr.io/speaches-ai/speaches:latest-cpu >/dev/null || { log "  container start failed"; return; }
  for i in $(seq 1 60); do curl -sf "http://localhost:$SPEACHES_PORT/health" >/dev/null 2>&1 && break; sleep 5; done
  # warm-up call downloads the model on first use; not timed
  curl -sf -X POST "http://localhost:$SPEACHES_PORT/v1/audio/transcriptions" \
    -F "file=@$AUDIO" -F "model=Systran/faster-whisper-$model" -F "language=en" \
    -o "$WORK/warmup-$model.json" >/dev/null 2>&1
  local t0 t1 wall
  t0=$(date +%s)
  curl -sf -X POST "http://localhost:$SPEACHES_PORT/v1/audio/transcriptions" \
    -F "file=@$AUDIO" -F "model=Systran/faster-whisper-$model" -F "language=en" \
    -o "$WORK/out-$model.json"
  t1=$(date +%s); wall=$((t1-t0))
  docker rm -f s2-speaches >/dev/null 2>&1
  if [ -s "$WORK/out-$model.json" ] && [ "$wall" -gt 0 ]; then
    local xrt ep
    xrt=$(python3 -c "print(f'{$AUDIO_SEC/$wall:.2f}')" 2>/dev/null || awk "BEGIN{printf \"%.2f\", $AUDIO_SEC/$wall}")
    ep=$(awk "BEGIN{printf \"%.1f\", 45*60/($AUDIO_SEC/$wall)/60}")
    log "  wall=${wall}s for ${AUDIO_SEC}s audio → xRT=${xrt} → ~${ep} min per 45-min episode"
    log "  text head: $(python3 -c "import json;print(json.load(open('$WORK/out-$model.json')).get('text','')[:80])" 2>/dev/null)"
  else
    log "  FAILED (no output or zero wall time)"
  fi
}

bench_speaches base
bench_speaches small

if [ -d /dev/dri ]; then
  log "--- WhisperLive OpenVINO (Intel iGPU) / small ---"
  docker rm -f s2-wl >/dev/null 2>&1
  docker run -d --name s2-wl --device=/dev/dri -p $WL_PORT:9090 ghcr.io/collabora/whisperlive-openvino >/dev/null \
    || { log "  container start failed"; }
  sleep 25
  t0=$(date +%s)
  docker run --rm --network host -v "$WORK":/w python:3.11-slim bash -lc \
    "pip install -q whisper-live >/dev/null 2>&1 && python3 -c \"
from whisper_live.client import TranscriptionClient
c = TranscriptionClient('localhost', $WL_PORT, lang='en', model='small')
c('/w/slice.wav')\"" >>"$OUT" 2>&1
  t1=$(date +%s); wall=$((t1-t0))
  docker rm -f s2-wl >/dev/null 2>&1
  log "  wall=${wall}s (incl. pip install overhead — see note) for ${AUDIO_SEC}s audio"
  awk "BEGIN{printf \"  xRT(raw)=%.2f → ~%.1f min per 45-min episode (upper bound)\n\", $AUDIO_SEC/$wall, 45*60/($AUDIO_SEC/$wall)/60}" | tee -a "$OUT"
else
  log "--- WhisperLive OpenVINO: SKIPPED (no /dev/dri) ---"
fi

log ""
log "=== done — paste $OUT back into the 9R-S2 spike doc ==="
