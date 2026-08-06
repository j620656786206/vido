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

MEDIA="${1:?usage: $0 /path/to/episode.mkv [slice_minutes] [start_seconds]}"
SLICE_MIN="${2:-10}"
# Sample from DEEP inside the file, not the first minutes: at -ss 300 the F1
# (2025) test file transcribed to an empty string because the opening is score
# and effects with no dialogue — a speech-free slice makes the benchmark measure
# VAD skipping rather than transcription (found 2026-08-06 on the first real run).
START_SEC="${3:-1200}"
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
FF_ARGS=(-y -ss "$START_SEC" -t $((SLICE_MIN*60)) -i "$MEDIA" -vn -ac 1 -ar 16000 -c:a pcm_s16le "$AUDIO")
if command -v ffmpeg >/dev/null 2>&1; then
  ffmpeg "${FF_ARGS[@]}" >/dev/null 2>&1
else
  # $WORK MUST be world-writable: the linuxserver image runs s6-overlay, which
  # drops to an unprivileged user (observed: `nobody`) INSIDE the container and
  # therefore ignores `--user 0:0` — a root-owned $WORK yields "Permission
  # denied" on the output file. The original silent `2>/dev/null` hid this as a
  # bare "extraction failed" (found 2026-08-06 on the first real NAS run).
  chmod 0777 "$WORK"
  docker run --rm -v "$(dirname "$MEDIA")":/in:ro -v "$WORK":/out linuxserver/ffmpeg \
    -y -ss "$START_SEC" -t $((SLICE_MIN*60)) -i "/in/$(basename "$MEDIA")" -vn -ac 1 -ar 16000 -c:a pcm_s16le /out/slice.wav >"$WORK/ffmpeg.log" 2>&1
fi
[ -s "$AUDIO" ] || { log "FATAL: audio slice extraction failed — see $WORK/ffmpeg.log"; tail -5 "$WORK/ffmpeg.log" 2>/dev/null | sed 's/^/  /' | tee -a "$OUT"; exit 1; }
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
  # Current Speaches does NOT auto-download on first transcription — it answers
  # "Model '...' is not installed locally. You can download the model using
  # POST /v1/models". WHISPER__MODEL only preloads a model that is already on
  # disk. Download explicitly, then warm up. (found 2026-08-06 on the first real
  # NAS run — the original kit predates this API change and failed here.)
  log "  downloading model (not timed)…"
  curl -sf -X POST "http://localhost:$SPEACHES_PORT/v1/models/Systran/faster-whisper-$model" \
    >/dev/null 2>&1 || log "  WARN: model download call failed — transcription will likely 404"
  # warm-up call; not timed
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
    # sed fallback, not python3 — Unraid has no python3 on the host.
    log "  text head: $(sed -e 's/.*"text"[[:space:]]*:[[:space:]]*"//' -e 's/".*//' "$WORK/out-$model.json" | cut -c1-80)"
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
