# Epic 9c Retrospective — Media Tech Info & NFO Integration

**Date:** 2026-04-06
**Facilitator:** Bob (Scrum Master)
**Epic:** 9c — Media Tech Info, NFO Sidecar, FFprobe Integration
**Status:** Complete (4/4 stories done)

---

## Epic 9c Summary

### Delivery Metrics

- **Stories Completed:** 4/4 (100%)
- **Total Tests Written:** 78+ (19 ShouldOverwrite + 35 NFO + 22 FFprobe + 6 backend + 22 frontend)
- **Files Created/Modified:** 47+
- **Code Review Rounds:** 7 (across all stories)
- **CR Issues Found & Fixed:** 30+ (including 2 Critical, 4 High)
- **Blockers Encountered:** 0
- **Technical Debt Incurred:** 0
- **Production Incidents:** 0

### v4 Features Delivered

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P1-030 | Tech info columns in DB | Done (9c-1) |
| P1-031 | NFO sidecar reader | Done (9c-2) |
| P1-032 | FFprobe tech extraction | Done (9c-3) |
| P1-033 | Metadata source priority | Done (9c-1) |
| P2-030 | Tech badges UI + unmatched filter | Done (9c-4) |

---

## What Went Well

### 1. Pipeline Architecture — Minimal Merge Conflicts
The enrichment pipeline design was excellent. 9c-2 (NFO) inserted at the TOP of `enrichMovie()` while 9c-3 (FFprobe) appended at the BOTTOM. This allowed parallel development with near-zero merge conflicts. The deliberate ordering (NFO → TMDB → FFprobe) with `ShouldOverwrite()` gates made the priority chain clean and testable.

### 2. ShouldOverwrite() Pattern — Robust Data Priority
The metadata source priority chain (manual > nfo > tmdb > douban > wikipedia > ai) with `>=` for same-source idempotency proved robust. 19 unit tests covering all priority combinations. This pattern will be reusable for all future metadata enrichment (Douban ratings in Epic 12, etc.).

### 3. Zero Blockers, Clean Completion
All 4 stories completed without blockers. Dependencies (9c-1 → 9c-2/9c-3 → 9c-4) were well-defined and sequential execution was smooth. No emergency hotfixes or scope changes needed.

### 4. Code Review Depth — Caught Real Issues
The adversarial CR process found 30+ issues across 7 rounds, including critical ones:
- Double DB write in FFprobe enrichment (H1, 9c-3)
- Resolution classification OR→AND logic bug (H1, 9c-2)
- Nil-client guard preventing panic (H2, 9c-2)
- NFO file size limit preventing OOM (H3, 9c-2)
- is_removed filter missing on stats/unmatched queries (9c-4)

Without CR, several of these would have been production bugs.

### 5. UX Verification Passed
Story 9c-4 frontend matched design screenshots (04f, 05d, h7, h8) for badge colors (blue/purple/gold/green), layout positioning, and filter UI. The three-gate verification (Dev → SM → UX) worked as designed.

---

## What Could Be Improved

### 1. Cross-Stack Split Rule Not Enforced
Story 9c-4 had 7 backend tasks + 8 frontend tasks — exceeding the Agreement 5 threshold (>3 each side). It was kept as a single story with an "advisory note" rather than actually splitting. Result: the story was the largest and most complex in the epic, requiring the most CR fixes (22 issues). The split rule exists for a reason.

**Root Cause:** Advisory notes are too easy to ignore. The rule should be enforced, not advised.

### 2. Mock Consistency Cascade
Every story that added a new interface method required updating mocks across multiple test files. In 9c-4 alone, 6+ test files needed GetStats mock stubs. This is a recurring friction pattern.

**Root Cause:** Manual mock maintenance. Go's interface system requires explicit mock updates. No mock generation tool (e.g., mockery) is in use.

### 3. Pre-existing Test Failures Ignored
Stories 9c-1 and 9c-2 both noted "pre-existing test failures in setup_service_test.go (not in scope)." Carrying forward known test failures erodes confidence in the test suite.

**Root Cause:** No policy for fixing pre-existing failures encountered during development.

---

## Previous Retro (Epic 8) Follow-Through

| # | Action Item | Status | Evidence |
|---|------------|--------|----------|
| P1 | Retro items → sprint-status tracking | ⏳ Partially | Items were listed in sprint-status but some lacked agent routing |
| P2 | Cross-stack stories split at >3 threshold | ❌ Not enforced | 9c-4 violated (7+8), kept with advisory |
| P3 | Dev-story mandatory task-checkbox verification | ✅ Applied | All 9c stories show task completion tracking |
| D1-D4 | Deployment items (Docker, CI, env, Unraid) | ❌ Pending | Not started — deferred, still in backlog |
| DOC1-3 | Bilingual documentation | ❌ Pending | No user-facing docs in 9c scope |
| TD1 | Fix husky pre-commit hook | ✅ Resolved differently | Rule 12: pre-commit disabled, lint/format moved to CI (2026-04-03) |
| TD2 | Document lazy SSE pattern | ❌ Pending | Not addressed |
| TS1 | TestSprite v4 regeneration | ❌ Pending | Not started |
| E9-1 | Split C-2 into C-2a + C-2b | ✅ Done | Stories 9-2a and 9-2b exist in sprint-status |

**Score: 3/9 completed, 1/9 partially, 5/9 not addressed.**

The recurring theme: items that don't have sprint-status tracking get forgotten. Agreement 4 was the right instinct but needs stricter enforcement.

---

## Key Insights

1. **Pipeline ordering prevents merge conflicts.** Designing the enrichment pipeline with clear insertion points (top/bottom) enabled parallel story development. Apply this pattern to future pipeline extensions.

2. **Adversarial CR is high-ROI.** 30+ issues caught across 4 stories. The double-DB-write and nil-guard issues would have been production bugs. Never skip CR.

3. **Cross-stack split advisory is insufficient.** Must be enforced as a hard rule in create-story workflow, not just a warning note. 9c-4 proved the advisory doesn't work.

4. **ShouldOverwrite() is a reusable pattern.** The metadata source priority chain with `>=` idempotency should be the standard for any future multi-source data enrichment.

5. **Pre-existing test failures compound.** Each ignored failure makes it harder to trust the test suite. Need a "fix or file" policy.

---

## Action Items

### Process Improvements

**AI-1 (HIGH):** Enforce cross-stack split rule in create-story workflow
- The SM agent's create-story workflow should BLOCK (not just warn) when backend tasks > 3 AND frontend tasks > 3
- Owner: SM workflow
- Success criteria: Create-story produces two story files when threshold exceeded

**AI-2 (MEDIUM):** Establish "fix or file" policy for pre-existing test failures
- When a developer encounters a pre-existing test failure during their story, they must either fix it (if quick) or create a tracked backlog item
- Owner: Dev workflow (dev-story)
- Success criteria: No more "pre-existing failure, not in scope" notes without a tracking entry

### Technical Debt

**AI-3 (LOW):** Evaluate mock generation tool (mockery or similar)
- Investigate whether `go generate` + mockery would reduce mock maintenance burden
- Owner: Dev
- Success criteria: Recommendation document with pros/cons

### Documentation

**AI-4 (MEDIUM):** Document lazy SSE pattern in project-context.md (carried from Epic 8 TD2)
- The SSE handler.go lazy connection pattern is critical architecture knowledge
- Owner: Dev (quick-dev)
- Success criteria: SSE section in project-context.md updated with lazy connection details

---

## Next Epic Preview

Epic 9c's direct successor work is **Epic 10 (Homepage TV Wall)** and **Epic 11 (Advanced Search & Filter)**, both in Phase 2 — Discovery & Browse. UX designs for both are already complete (0-UX tasks done 2026-04-05).

**Dependencies on Epic 9c work:**
- Tech badges (9c-4) reusable in homepage cards and search results
- Unmatched filter (9c-4) API pattern reusable for advanced filtering (Epic 11)
- Metadata source priority (9c-1) needed for Epic 12 Douban rating enrichment

**No blockers from 9c for next epics.** All 4 stories are clean and complete.

---

## Readiness Assessment

| Dimension | Status |
|-----------|--------|
| Testing & Quality | ✅ 78+ tests, all passing |
| Code Review | ✅ All 30+ issues resolved |
| UX Verification | ✅ Passed (9c-4 screenshots) |
| Technical Health | ✅ Clean, no debt incurred |
| Deployment | ⚠️ D1-D4 still pending (not 9c-specific) |

**Verdict:** Epic 9c is fully complete and production-ready.

---

## Team Participants

- Bob (Scrum Master) — Facilitator
- Alexyu (Project Lead) — Participant
- Charlie (Senior Dev) — Story analysis
- Alice (Product Owner) — Business context
- Dana (QA Engineer) — Testing insights

---

*Retrospective facilitated by Bob (Scrum Master) on 2026-04-06.*
