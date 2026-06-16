# Epic 9: AI Subtitle Enhancement
**Phase:** Phase 1 — Core Media Pipeline

> **🔶 2026-06-16 REVISION — promoted from optional/P3 to the 繁中 CORE path.**
> See `../architecture/adr-subtitle-route-c-generation.md` + `../subtitle-v4-replan-and-feasibility-audit-2026-06.md`.
> With Route A fetch confirmed non-viable for 繁中 (Epic 8 revision), **Route C generation (transcribe → LLM-translate) is now the sole 繁中 path, not an optional enhancement.** A live POC validated the full pipeline end-to-end on a real 4K episode (ffmpeg → Whisper → Claude → OpenCC → write), producing natural 繁中. New scope added below: **(a)** per-show **glossary keystone** + `TranslationRequest{Fields,Glossary}` generalization + metadata-aware translation; **(b)** **`ASRProvider`** pluggable engines (cloud / self-hosted Speaches·WhisperLive-OpenVINO); **(c)** **VAD hallucination filter**; **(d)** 4 POC-surfaced production bug fixes (model 404, Whisper language pin, chunking 413, retry); **(e)** **metadata localization (Section E)** reusing the same infra. C-1 (terminology correction) stays. Success criteria below superseded by the re-plan's feasibility-gated backlog.

Optional AI-powered subtitle enhancements for users who want higher quality Traditional Chinese subtitles. Terminology correction uses Claude API for word-level 簡→繁 fixes that OpenCC misses (e.g., proper nouns, technical terms, regional vocabulary). MKV English audio track translation uses Whisper for transcription followed by AI translation, enabling subtitle generation for content with no existing Chinese subtitles.

**v4 Feature IDs covered:** P1-020, P1-021

**Dependencies on Completed Work:**
- Epic 3: AI Provider Abstraction Layer (reusable for Claude API calls)
- Epic 8: Subtitle engine (provides subtitle files to enhance)

**Stories (to be created):**
- C-1: AI terminology correction — Claude API integration for word-level 簡→繁 fixes beyond OpenCC's dictionary (P2 priority)
- C-2a: Whisper audio transcription — Extract and transcribe English audio tracks from MKV files using Whisper, output timestamped SRT (P3 priority)
- C-2b: AI subtitle translation — Translate Whisper-generated English SRT to Traditional Chinese using Claude API, with context-aware sentence merging and terminology consistency (P3 priority, depends on C-2a)

**Split Rationale (Epic 8 Retro, Agreement 5):**
Original C-2 combined two distinct technology stacks (Whisper + AI translation) into one story. Per the cross-stack splitting rule, separating them ensures:
1. C-2a can be independently validated (transcription quality) before translation begins
2. Whisper integration (new tech stack) can be spiked/researched without blocking AI translation design
3. Each story stays within the >3-task-per-side limit

**Success Criteria:**
- >95% terminology correction accuracy on test corpus of known 簡繁 edge cases
- Whisper transcription word error rate <15% on clear English dialogue tracks
- AI-generated subtitles achieve >80% comprehensibility rating in user testing
