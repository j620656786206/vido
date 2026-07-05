# Spike 9R-S2 — NAS Whisper Benchmark + OpenVINO Eval

Status: in-progress (kit + research done — awaiting the NAS run)

**Epic:** epic-9R-subtitle-route-c (Spike S2) · **Gates:** ADR D2 default (cloud vs local Whisper), 9R-9 local-engine option
**Owner:** dev (Amelia) · **Date:** 2026-07-05 · **Effort:** S

## Question (from `subtitle-route-c-stories-2026-06.md` §Spikes/S2)

> faster-whisper `base`/`small` throughput on the real target NAS CPU; does
> WhisperLive-OpenVINO usefully use the Intel iGPU?
> **Pass:** measured min/episode for each; a go/no-go on local default.

## Why this cannot be answered from the dev machine

Throughput is a function of the NAS's exact CPU (and iGPU generation). Any number measured on
the Apple-Silicon dev machine is meaningless for the D2 decision. The spike therefore ships a
**one-command benchmark kit** to run on the NAS; the verdict lands when its output is pasted
back here.

## Engine research (2026-07-05)

| Engine | Deployment | API | iGPU | Notes |
|---|---|---|---|---|
| **Speaches** (faster-whisper server, MIT) | `ghcr.io/speaches-ai/speaches:latest-cpu` | OpenAI-compatible `POST /v1/audio/transcriptions` | ❌ CPU (int8 recommended: `WHISPER__COMPUTE_TYPE=int8`) | Drop-in for vido's existing `WithWhisperBaseURL` — one-line base-URL swap (9R-9) |
| **WhisperLive** (Collabora, MIT) | `ghcr.io/collabora/whisperlive-openvino`, `docker run --device=/dev/dri -p 9090:9090` | WebSocket (python client `whisper_live.client`) | ✅ OpenVINO backend targets Intel CPU/iGPU/dGPU; Docker image auto-enables GPU, no host OpenVINO install | NOT OpenAI-shaped — adopting it as a vido engine would need either its OpenAI façade (hwdsl2/docker-whisper-live) or a WS client; for the SPIKE it answers only "does the iGPU help" |
| (alt) whisper-pro-asr | `ventura8/whisper-pro-asr` | OpenAI-compatible | ✅ Intel NPU/Arc/iGPU + CUDA | Fallback candidate if WhisperLive image misbehaves on the NAS |

Key facts feeding D2:
- faster-whisper is ~4× realtime-class on modest CPUs for `base`-tier models (published
  guidance), but NAS-class low-power CPUs (N-series/J-series) are far below desktop numbers —
  hence measure, don't guess.
- OpenVINO is the only credible path to the NAS's Intel iGPU; faster-whisper (CTranslate2)
  does NOT use iGPUs.

## The kit — `scripts/whisper-nas-benchmark.sh` (committed)

Run **on the NAS** (Docker required), pointing at a REAL episode file:

```bash
./scripts/whisper-nas-benchmark.sh "/volume1/media/tv/…/episode.mkv"   # [slice_minutes], default 10
```

It prints + saves `whisper-bench-results.txt`:
1. CPU model / threads / RAM / `/dev/dri` presence (auto-detects iGPU availability)
2. Extracts a 10-min 16kHz mono WAV slice (host ffmpeg, else dockerized linuxserver/ffmpeg)
3. **Speaches CPU**: `faster-whisper-base` then `-small` (int8), timed via the OpenAI endpoint
   (model download happens in an untimed warm-up call) → **xRT + extrapolated min/45-min-episode**
4. **WhisperLive-OpenVINO** (only if `/dev/dri` exists): `small` via the python client
   (dockerized; wall time includes a pip-install overhead — treat as an upper bound; rerun
   twice and take the second number for a cleaner figure)

## Results (PASTE FROM THE NAS RUN)

```
(pending — output of whisper-bench-results.txt)
```

| Engine / model | xRT | min per 45-min episode | Notes |
|---|---|---|---|
| Speaches CPU / base (int8) | _(pending)_ | _(pending)_ | |
| Speaches CPU / small (int8) | _(pending)_ | _(pending)_ | |
| WhisperLive OpenVINO / small | _(pending)_ | _(pending)_ | skipped if no /dev/dri |

## Go/no-go framework (decide once numbers land)

- **Local-default GO** if `small` ≲ 2× realtime (xRT ≥ 0.5 → a 45-min episode ≤ ~90 min) AND
  the NAS stays usable during the run — matches D2's "user accepts slowness" opt-in framing
  being safe to even DEFAULT for overnight batch.
- **Local opt-in only (cloud default)** if `small` is 3-10× slower than realtime — overnight
  batch viable, interactive generation is not; D2 stays cloud-default (current ADR position).
- **Local NO-GO** if > ~10× slower than realtime or OOM — cloud-only; 9R-9 keeps the base-URL
  pluggability anyway (someone with a beefier box benefits).
- iGPU verdict: OpenVINO must beat Speaches-CPU by ≥ ~1.5× to justify the extra moving part.

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | Kit + engine research shipped (Amelia); dev-machine numbers ruled meaningless for D2; awaiting the NAS run of `scripts/whisper-nas-benchmark.sh` to close. |
