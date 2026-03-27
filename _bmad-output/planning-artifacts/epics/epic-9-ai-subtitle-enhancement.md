# Epic 9: AI Subtitle Enhancement
**Phase:** Phase 1 — Core Media Pipeline

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
