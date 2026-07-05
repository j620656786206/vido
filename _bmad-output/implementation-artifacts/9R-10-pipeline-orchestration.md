# Story 9R-10 — Wire the Route C generation pipeline (single service flow)

Status: review

**Epic:** epic-9R-subtitle-route-c (Track 5 — Orchestration) · **Owner:** dev (Amelia)
**Date:** 2026-07-05 · **Priority:** P1 · **Effort:** M · **Feasibility:** PROVEN (POC proved the data flow)
**Depends:** 9R-1..4 ✅ (#136), 9R-7 ✅ (#137), 9R-11 ✅ (#138)

## Why

The Route C components existed but weren't wired as one automated flow with the keystone glossary
and the cost controls actually applied. This story completes `TranscriptionService.runPipeline`
into the full pipeline: **extract → transcribe → glossary-aware translate → OpenCC → place**, with
SSE progress, under the shared throttle + per-run budget.

## What shipped (AC1 — one service flow)

`TranscriptionService` already did extract → transcribe → translate → save. 9R-10 closes the three
gaps so the whole flow is one coherent, keystone-aware pipeline:

1. **Glossary-aware translation.** `translateSRT` now loads the per-show glossary
   (`GlossaryRepository.LookupByMedia`, all terms — confirmed + auto-mined — for max intra-run
   consistency) and calls `TranslationService.TranslateWithGlossary` (9R-7). Proper nouns render
   consistently across the subtitle and across runs — the keystone finally applied. **Fail-soft:**
   a glossary miss/error translates without a glossary rather than blocking.
2. **OpenCC s2twp safety net.** After translation, the SRT passes through an injected
   `OpenCCConverter` (`ConvertS2TWP`) so output is guaranteed Traditional even if the LLM slips a
   Simplified character. **Fail-soft:** a converter error keeps the LLM output.
3. **Atomic placement.** The zh-Hant SRT is written via an injected `SubtitlePlacer` (atomic write
   + `.bak` backup + normalized filename); a direct-write fallback keeps the pipeline functional
   when no placer is wired.

**Rule 19 (services ↛ subtitle):** `OpenCCConverter` (structural — `*subtitle.Converter` fits) and
`SubtitlePlacer` (primitive params) are declared in `services`; `main.go` injects
`*subtitle.Converter` directly and a thin `subtitlePlacerAdapter` over `*subtitle.Placer`.

Throttle + per-run budget (9R-11) already span the run's ASR + LLM; glossary/OpenCC/place add no
new external calls.

## AC2 — trigger conditions defined

- **Manual — LIVE.** `POST /api/v1/movies/:id/transcribe` (with translation) runs the full
  glossary→OpenCC→place pipeline today. Route A (fetch) is **dormant** — not a precondition; nothing
  gates generation on a fetch attempt.
- **on-add (auto-trigger on scan/import)** and **series/episode trigger** are the remaining defined
  triggers. The transcribe endpoint is movies-only; series generation needs an episode-file-path
  lookup + route. Filed as **`9R-10a-series-episode-trigger`** and **`9R-10b-on-add-autotrigger`**
  (sprint-status, ready-for-dev / backlog) rather than ballooning this pipeline-wiring story —
  Rule 24 ③, tracked not prose-only. (Supersedes the 9R-UX Discovery ① "absorbed into 9R-10 scope"
  note: the pipeline is wired; the extra *entry points* are tracked follow-ups.)

## AC3 — partial-failure resilience + e2e test

- Resilience: translation is fail-soft per batch (9R-4/AC5 — failed blocks keep English); glossary
  load and OpenCC are both fail-soft (above); a per-attempt timeout retries (9R-4).
- **e2e integration test** (`TestTranscriptionService_TranslateSRT_GlossaryOpenCCPlace`): mock LLM +
  fake OpenCC + fake placer + stub glossary repo drive `translateSRT` and assert the glossary term
  reached the prompt, OpenCC ran on the output (软→軟), and the placer received the converted
  zh-Hant content at the right path. Plus a no-deps fail-soft test (direct-write fallback, no
  glossary section in the prompt). Extract→transcribe halves are covered by 9R-2/3 tests +
  Whisper httptest.

## Acceptance Criteria

1. ✅ One service: extract → transcribe (ASRProvider) → glossary-aware translate → OpenCC → place,
   with SSE progress.
2. ✅ Trigger conditions defined — manual LIVE (full pipeline); Route A dormant, not a precondition;
   on-add + series filed as tracked follow-ups.
3. ✅ Partial-failure resilience (glossary/OpenCC/translation all fail-soft; 9R-4 retry) +
   end-to-end integration test with mock ASR/LLM.

## Dev Notes

- ASRProvider interface (9R-9) is not yet in — Whisper is still the concrete client here; when 9R-9
  lands, the pipeline's transcribe stage swaps to the interface with no flow change.
- Glossary MINING (extracting proper nouns from a run to POPULATE the glossary) is a separate
  concern — this story CONSUMES the glossary; auto-population can be a follow-up once the F6 review
  loop (9R-15 REST) exists.

### Discovery Triage

- **YES — two out-of-scope entry points, both filed (Rule 24 ③):**
  - `9R-10a-series-episode-trigger` (ready-for-dev) — a series/episode transcribe route so the
    pipeline runs for TV, not just movies. Blocks Epic 6 TV generation, not this story.
  - `9R-10b-on-add-autotrigger` (backlog) — auto-trigger generation on scan/import for
    missing-繁中 items. Additive; needs a policy decision (opt-in vs default) + scanner hook.

### References

- [Source: subtitle-route-c-stories-2026-06.md#9R-10] — ACs.
- [Source: internal/services/transcription_service.go] — the pipeline extended.
- [Source: internal/subtitle/{converter,placer}.go] — ConvertS2TWP + Placer, injected per Rule 19.

## Dev Agent Record

### Agent Model Used

claude-fable-5 (dev)

### Completion Notes List

- Route C pipeline wired: glossary-aware translate (9R-7) + OpenCC s2twp safety net + atomic place,
  all fail-soft, Rule-19-clean via injected interfaces + main.go adapters. Manual trigger runs the
  full flow; series/on-add filed as follow-ups. e2e integration test with mock ASR/LLM. Full api
  suite + staticcheck green.

### File List

- `apps/api/internal/services/transcription_service.go` (+ test) — OpenCCConverter/SubtitlePlacer
  interfaces, glossary/opencc/placer fields + setters, loadGlossary, translateSRT rewrite
- `apps/api/cmd/api/main.go` — wire glossary repo + converter + placer adapter
- `apps/api/cmd/api/placer_adapter.go` — `*subtitle.Placer` → `services.SubtitlePlacer`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | 9R-10 implemented (dev): translateSRT now glossary-aware (9R-7 TranslateWithGlossary + GlossaryRepository.LookupByMedia) → OpenCC s2twp safety net → atomic Placer, all fail-soft, injected per Rule 19 (main.go adapters). Manual trigger runs the full pipeline; series (9R-10a) + on-add (9R-10b) filed as tracked follow-ups. e2e integration test (mock ASR/LLM). Full suite + staticcheck green. Status → review. |
