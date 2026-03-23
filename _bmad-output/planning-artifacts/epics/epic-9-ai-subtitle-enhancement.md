# Epic 9: AI Subtitle Enhancement
**Phase:** Phase 1 — Core Media Pipeline

Optional AI-powered subtitle enhancements for users who want higher quality Traditional Chinese subtitles. Terminology correction uses Claude API for word-level 簡→繁 fixes that OpenCC misses (e.g., proper nouns, technical terms, regional vocabulary). MKV English audio track translation uses Whisper for transcription followed by AI translation, enabling subtitle generation for content with no existing Chinese subtitles.

**v4 Feature IDs covered:** P1-020, P1-021

**Dependencies on Completed Work:**
- Epic 3: AI Provider Abstraction Layer (reusable for Claude API calls)
- Epic 8: Subtitle engine (provides subtitle files to enhance)

**Stories (to be created):**
- C-1: AI terminology correction — Claude API integration for word-level 簡→繁 fixes beyond OpenCC's dictionary (P2 priority)
- C-2: MKV audio track translation — Whisper transcription + AI translation pipeline for English audio (P3 priority)

**Success Criteria:**
- >95% terminology correction accuracy on test corpus of known 簡繁 edge cases
- AI-generated subtitles achieve >80% comprehensibility rating in user testing
