---
stepsCompleted: [1, 2, 3, 4, 5, 6]
workflowStatus: complete
verdict: 'READY FOR PHASE 4 IMPLEMENTATION — re-validated by second pass: 14 findings (3 critical doc defects, 2 new process gaps), none blocking code'
revision: 2  # second pass 2026-07-27 — adversarial re-run; 11/11 findings survived refutation, 3 strengthened (F7 F1 F6), 3 added (F8 F9 F10), 1 candidate gap refuted (NFR-I2 infra already shipped: Dockerfile:47 ffmpeg, docker.yml:80 multi-arch)
workflowType: 'check-implementation-readiness'
assessor: 'John (PM) — adversarial review'
assessmentTarget: 'Subtitle Pipeline (M1) — full re-run now that epics + stories exist'
date: '2026-07-27'
supersedes: '_bmad-output/planning-artifacts/implementation-readiness-report-subtitle-pipeline.md (2026-07-26 — partial; steps 03/05 were N/A because no epics existed)'
documentsAssessed:
  prd: '_bmad-output/planning-artifacts/prd.md'
  architecture: '_bmad-output/planning-artifacts/subtitle-pipeline-architecture.md'
  epics: '_bmad-output/planning-artifacts/epics-subtitle-pipeline.md'
  stories: '_bmad-output/implementation-artifacts/{sub-1-1,sub-1-2,sub-1-7a,sub-1-7b}.md (4 drafted of 9 planned)'
  ux: 'ux-design.pen (flow F v2) + _bmad-output/planning-artifacts/ux-redesign/ + ux-design-specification.md'
---

# Implementation Readiness Assessment Report

**Date:** 2026-07-27
**Project:** vido

---

## Step 1 — Document Discovery

### Inventory

| Type | Whole documents | Sharded |
|---|---|---|
| **PRD** | **`prd.md`** (25.8 KB, 2026-07-27) · `prd-validation-report.md` (3.2 KB) · `prd-v4-migration-changelog.md` (4.7 KB, Mar 23) | `prd/` — 11 files |
| **Architecture** | **`subtitle-pipeline-architecture.md`** (52.6 KB, 2026-07-26) · `adr-series-season-episode-architecture.md` (14.6 KB, Mar 5) | `architecture/` — 26 files (5 pattern docs + 5 ADRs + index) |
| **Epics & Stories** | **`epics-subtitle-pipeline.md`** (25.1 KB, 2026-07-27) · `epics.md` (32.1 KB, Jun 22) · `implementation-readiness-report-epic-9c.md` (12.3 KB, Apr 3) | `epics/` — 29 files (epic-0 … epic-20 + archive) |
| **UX** | `ux-design-specification.md` (222.7 KB, **2026-01-12**) · `multi-library-ux-spec.md` · `ux-design-gap-analysis-v4.md` · `ux3-discover-facet-aggregation-spike.md` | `ux-redesign/` — 7 files · **`ux-design.pen`** (repo root, flow F v2) |

**Missing required documents:** none. All four types present.

### Issue 1 — Whole-vs-sharded duplicates: confirmed **false positive** (ruling upheld)

Generic discovery flags three whole/sharded pairs (`prd.md`↔`prd/`, `subtitle-pipeline-architecture.md`↔`architecture/`, `epics-subtitle-pipeline.md`↔`epics/`). They are **not two formats of one document**: the folders hold the older **v4 project** artifacts (`epics/` contains epic-0 … epic-20, unrelated to the subtitle pipeline) while the root `.md` files are the **subtitle-pipeline** documents. The 2026-07-26 report reached the same conclusion (§1); it is upheld. **No file deletion or renaming required.**

### Issue 2 — `ux-design-specification.md` authority: **RESOLVED (Alexyu, 2026-07-27)**

The 222.7 KB spec predates both the UX Redesign initiative (`ux-redesign/`, June) and `.pen` flow F v2 (July), raising the question of whether it was superseded, still authoritative, or an archive.

**Ruling: still authoritative — and `sub-1-7a` must sync it.**

Scope of that sync, bounded during this step so it cannot balloon:

- A `grep` for `badge` across all 6,373 lines returns **five** hits. Only **`:1086`** — `**StatusBadge:** Parse status, download status, metadata source indicators` (§ Design System Foundation → Customization Strategy) — is a component contract this work invalidates. The others are the "Source: Douban" indicator and a "3 pending" navigation count.
- The document contains **no per-state badge table**, so there is nothing to contradict — only an incomplete enumeration to extend plus a pointer to the authoritative `.pen` spec screen.
- Its subtitle content throughout is **v4-era**: fansub-filename parsing and subtitle-timeline matching, not pipeline status.

**Action taken:** `sub-1-7a` gained **AC #9** + **Task 4** covering exactly this, with an explicit "≤ 2 edited locations, do not copy the table in" bound (two sources of truth for one table is the failure mode).

### Documents adopted for this assessment

| Type | Adopted | Excluded (v4-era / historical) |
|---|---|---|
| PRD | `prd.md` | `prd/`, `prd-v4-migration-changelog.md` |
| Architecture | `subtitle-pipeline-architecture.md` | `architecture/`, `adr-series-season-episode-architecture.md` |
| Epics | `epics-subtitle-pipeline.md` + drafted story files | `epics.md`, `epics/` |
| UX | `ux-design.pen` flow F v2 · `ux-redesign/` · `ux-design-specification.md` (authoritative per Issue 2) | `ux-design-gap-analysis-v4.md` |

---

## Step 2 — PRD Analysis

**Source:** `prd.md` — read in full (321 lines). Frontmatter: `workflowStatus: complete`, `completedDate: 2026-07-23`, 12 workflow steps completed, brownfield / medium complexity / web_app (backend-pipeline character), auth N/A (v4 single-user).

### Functional Requirements

_Phase tags: unmarked = M1 (MVP); [M1.5], [P2] = Growth; [Tier-2] = Expansion._

**A. Subtitle Source Detection & Extraction**

- **FR1:** The system can detect the subtitle tracks in a media file (embedded text, embedded image, external sidecar) and their language.
- **FR2:** The system can extract an embedded text subtitle track without re-encoding the media.
- **FR3:** The system can extract multiple text subtitle tracks in a single pass.
- **FR4:** The system can filter SDH (hearing-impaired) annotations from an extracted subtitle.
- **FR5:** The system can identify a media item as having no usable text source (image-only or none) **and mark it**.

**B. Language Routing & Conversion**

- **FR6:** The system can determine whether a Chinese subtitle is Traditional or Simplified **from its content** (not filename or track tag) — the fix for Bazarr's core zh-TW mislabeling. **Source language** is identified from the track's language tag (per FR1), not from content.
- **FR7:** The system can pass an already-Traditional-Chinese subtitle through unchanged.
- **FR8:** The system can convert a Simplified-Chinese subtitle to Traditional (Taiwan) without AI translation.
- **FR9:** The system can route a subtitle by detected language (Traditional → done, Simplified → convert, English → translate, other → skip).

**C. AI Translation**

- **FR10:** The system can translate an English subtitle to Traditional Chinese via a translation provider.
- **FR11:** The system can translate a subtitle while preserving its cue numbering and timestamps exactly.
- **FR12:** The user can trigger subtitle translation for a media item on demand.
- **FR13:** The system can translate subtitles automatically when new media is added.
- **FR14:** The user can see an estimated cost before translating a batch/season. **[P2]**

**D. Translation Quality Assurance**

- **FR15:** The system can guarantee Traditional script by applying a deterministic conversion pass to the translation output.
- **FR16:** The system can detect and retry only the individual cues that come back empty, untranslated, or in the wrong language.
- **FR17:** The system can verify that translated cue timestamps match the source.

**E. Glossary & Cross-Episode Consistency [P2]**

- **FR18:** Harvest proper-noun/term mappings from a translation into the show's glossary as unconfirmed suggestions. **[P2]**
- **FR19:** The user can review, confirm, edit, or delete glossary terms. **[P2]**
- **FR20:** Apply a show's glossary as translation context for any episode, regardless of episode order. **[P2]**

**F. Provider & Key Management**

- **FR21:** The user can supply their own API keys for translation and (optional) ASR providers.
- **FR22:** Translation with a locally-hosted provider requiring no external key. **[Tier-2]**
- **FR23:** Gracefully disable translation with a clear message when no provider/key is configured (no silent failure).
- **FR24:** The user can select which translation/ASR provider and model is active. **[Tier-2]**
- **FR25:** The user can configure and edit provider keys from within the app after initial setup. **[M1.5]**

**G. Metadata Context & Match Correction**

- **FR26:** Supply media metadata (title, genre, overview, cast, country) as context to the translation provider.
- **FR27:** The user can correct an incorrect TMDb match by searching for and selecting the correct entry.
- **FR28:** Re-translate a subtitle using the corrected metadata after a match is fixed. **[P2]**

**H. Source Fallback [P2]**

- **FR29:** Generate subtitles from audio via ASR when no text source exists. **[P2]**
- **FR30:** Prioritize sources as extract > ASR > online-search. **[P2]**
- **FR31:** Route ASR to a cloud API, an external worker, or a local model based on available compute. **[P2]**

**I. Pipeline Operation & Status**

- **FR32:** Place the produced Traditional-Chinese subtitle beside the media file for player auto-load.
- **FR33:** The user can see real-time progress status for subtitle processing.
- **FR34:** Run batch subtitle processing over a scope (missing-only / season / library). **[P2]**

**Total FRs: 34.** M1 scope = **22** (FR1–13, 15–17, 21, 23, 26, 27, 32, 33) · M1.5 = 1 (FR25) · P2 = 9 (FR14, 18–20, 28, 29–31, 34) · Tier-2 = 2 (FR22, FR24).

### Non-Functional Requirements

**Performance**

- **NFR-P1:** On DS920+ (J4125 / 4 GB), processing one item (extract + translate an embedded English sub) must not degrade other NAS services — extraction is I/O-bound (seconds); cloud translation is HTTP-only.
- **NFR-P2:** In M1 the NAS performs no heavy local AI compute (no local ASR); heavy compute is offloaded or gated by compute-aware defaults.
- **NFR-P3:** Batch/queue concurrency is bounded per hardware tier (ARM / low-RAM → 1).

**Security & Privacy**

- **NFR-S1:** Provider API keys stored encrypted (secrets service), never written to logs (slog sanitize).
- **NFR-S2:** No subtitle dialogue leaves the NAS unless a cloud provider key is configured (capability-gated); a fully-local path exists (Tier-2).

**Reliability**

- **NFR-R1:** Every external-dependency failure is fail-soft — degrade per item to a clear state, never a page/app failure.
- **NFR-R2:** Idempotent — re-processing does not duplicate or corrupt output; an existing acceptable `.zh-Hant.srt` is not overwritten without intent.
- **NFR-R3:** Granular recovery — per-cue retry; batch preserves completed results on cancel and is resumable.

**Integration & Compatibility**

- **NFR-I1:** External providers integrated per Rule 27's Five Pillars (rate-limit, cache, degrade, error-codes, keys).
- **NFR-I2:** Ships as a single multi-arch (amd64 + arm64) Docker image for Synology Container Manager / QNAP Container Station.
- **NFR-I3:** Output uses the `.zh-Hant.srt` / `.zh-Hans.srt` sidecar convention for Plex / Jellyfin / Video Station auto-detection.

**Total NFRs: 11.** _(Scalability folded into Performance — single-user, single NAS. Accessibility deliberately skipped — new UI inherits the design-system a11y baseline.)_

### Additional Requirements & Constraints

**Success criteria that must become measurable acceptance criteria:**

- **Technical:** M1 runs on DS920+ (J4125 / 4 GB) using extraction + cloud translation only. Soft-subtitle output, zero re-encode, `.zh-Hant.srt` per project-context §9b (**not** `.zh-TW.srt`).
- **M1 acceptance (spec §8):** Docker deploy on DS920+ → scan → extract an English embedded sub → Claude translate → `.zh-Hant.srt` appears beside the video → Video Station lists the Traditional-Chinese track.
- **⚠️ Quantification gap (carried from the 2026-07-26 report, gaps G2/G4, still open):** "time-to-`.zh-Hant.srt` on the order of tens of seconds" and "accepted without hand-editing in ≥ a target %" are **deliberately unquantified pending the M1 pilot**. They are non-blocking for build but **must become measurable acceptance criteria at story time**. No drafted story currently carries them.

**Constraints the PRD imposes on implementation:**

- Line segmentation: the extract path (M1) **preserves** the source's professional segmentation — translate in place, do **not** re-break. The CJK ~14–16 full-width / max-2-line norm applies **only** to the ASR path (M2). The PRD explicitly records this as a correction of an earlier over-application.
- Translation quality depends on a correct TMDb match; correction uses the shipped `ManualSearchDialog` / `POST /api/v1/metadata/manual-search` (Story 3.7). Re-translate after a late correction is **[P2]** (FR28).
- Two-stage guarantee: **LLM = semantics, OpenCC = orthography**; per-cue quality gate (Simplified leak / empty / echo / cue-count) → retry affected cues only; assert output timestamps equal source cue-by-cue.
- Bible compliance called out in the PRD itself: Rule 1 (`apps/api`), Rule 2 (slog), Rule 4 (layered), Rule 7 (`SUBTITLE_` codes + `code-review/instructions.xml` sync), §8 (SSE consumers stay lazy — no connect-on-mount).

### PRD Completeness Assessment

**Complete and internally consistent.** 34 FRs across 9 capability areas, all phase-tagged; 11 NFRs across 4 categories with the omissions justified; 5 user journeys (J1–J5) mapping to the FR groups; MVP/M1.5/P2/Tier-2 scoping with explicit risk analysis and de-scoping rationale.

The two contradictions the 2026-07-26 architecture review flagged are **confirmed fixed** in this revision:

- **FR6** now scopes content-based detection to the Traditional-vs-Simplified *variant*, with source language from the track tag — removing the FR6-vs-P0 conflict.
- **FR28** now carries `[P2]`, matching the Post-MVP prose and the glossary linkage.

**One traceability observation carried into Step 3:** FR5's wording is *"identify … and **mark it**"*. The mark has two halves — a system-side state (the DB column) and a user-visible state (the badge). Only the first half was covered by the epic breakdown as originally written; whether the second half is now covered is a Step 3 question.

**Assessment: PRD is READY.** No PRD edits required by this review.

---

## Step 3 — Epic Coverage Validation

**Source:** `epics-subtitle-pipeline.md` (read in full, incl. the 2026-07-27 additions of Story 1.7a/1.7b) + the 4 drafted story files.

### Coverage claimed by the epics document

The document carries an explicit **FR Coverage Map** closing with: _"All 34 FRs accounted for: 22 → Epic 1, 1 → Epic 2, 11 → deferred (P2/Tier-2)."_ That arithmetic is **correct** and every PRD FR appears exactly once. No FR exists in the epics that is absent from the PRD.

### Story-level coverage matrix (M1 — what the epic map does *not* provide)

The epic map resolves FR → **epic**. Now that stories exist, the meaningful question is FR → **story**. Resolved by reading the epic's story ACs and the 4 drafted story files:

| FR | Requirement (abbrev.) | Story | Status |
|---|---|---|---|
| FR1 | Detect subtitle tracks + language | sub-1-4 | ✓ |
| FR2 | Extract embedded text track, no re-encode | sub-1-4 | ✓ |
| FR3 | Multi-track single-pass extraction | sub-1-4 | ✓ |
| FR4 | SDH filter (pre-translation) | sub-1-4 | ✓ |
| FR5 | No usable text source **and mark it** | sub-1-2 (enum) · sub-1-4 (writes) · **sub-1-7b (user-visible mark)** | ⚠️ **map stale** |
| FR6 | Content-based CJK variant | sub-1-4 | ✓ |
| FR7 | Traditional pass-through | sub-1-4 | ✓ |
| FR8 | Simplified → OpenCC s2twp | sub-1-4 | ✓ |
| FR9 | Language routing (… other → skip) | sub-1-4 · **sub-1-7b (`skipped` user-visible)** | ⚠️ **map stale** |
| FR10 | English → 繁中 via provider | sub-1-5 (client from sub-1-1) | ✓ |
| FR11 | Preserve cue numbering + timestamps | sub-1-5 | ✓ |
| FR12 | Manual trigger | sub-1-6 | ✓ |
| FR13 | Auto on media-add | sub-1-6 | ✓ |
| FR15 | Deterministic Traditional guarantee | sub-1-5 | ✓ |
| FR16 | Per-cue retry | sub-1-5 | ✓ |
| FR17 | Timestamp equality assertion | sub-1-5 | ✓ |
| FR21 | BYO key (env-var in M1) | **no story — already shipped** (`CLAUDE_API_KEY` in `config.go`) | ⚠️ see Finding 3 |
| FR23 | Capability gate, no silent failure | sub-1-6 | ✓ |
| FR26 | Metadata as translation context | sub-1-5 | ✓ |
| FR27 | Correct an incorrect TMDb match | **no story — already shipped** (Story 3.7 `ManualSearchDialog`) | ⚠️ see Finding 3 |
| FR32 | Place sidecar beside media | sub-1-5 | ✓ |
| FR33 | Real-time progress | sub-1-3 (stage enum) · sub-1-6 (SSE broadcast) | ✓ |
| FR25 | In-app key config | sub-2-1 (Epic 2) | ✓ |

### Coverage Statistics

- **Total PRD FRs:** 34
- **FRs covered in epics:** 34 (22 → Epic 1 · 1 → Epic 2 · 11 → explicitly deferred with a named phase)
- **Coverage:** **100%** — no orphaned FR, no epic-only FR
- **M1 FRs with an implementing story:** 20 of 22. The other 2 (FR21, FR27) are satisfied by already-shipped code.

### Missing Requirements

**None.** No FR lacks an implementation path. The findings below are **traceability-document defects**, not coverage gaps.

### Findings

#### 🔴 F1 — The FR Coverage Map is stale for FR5 and FR9 (and omits Stories 1.7a/1.7b entirely)

The map rows read:

- `FR5 | Epic 1 | No-text-source state (new enum value)`
- `FR9 | Epic 1 | Language routing`

FR5's PRD text is _"identify a media item as having no usable text source **and mark it**"_. That mark has two halves: a **system-side** state (the `subtitle_status` column, sub-1-2) and a **user-visible** state (the badge, sub-1-7b). The map documents only the first. Same for FR9's `skipped`.

Stories **1.7a** and **1.7b** were added to Epic 1 on 2026-07-27 and appear in the story list, but **no row in the FR Coverage Map mentions them**. The map is the document's evidence that all 34 FRs have a home; when it under-reports, the traceability claim is unverifiable.

**Impact:** medium. Nothing is un-built; the audit trail is wrong. A future reader concludes the badge work is unrequirmented scope and is entitled to cut it.
**Recommendation:** extend the FR5 and FR9 rows to name sub-1-7b as the user-visible half. **No new FR is required** — FR5's "mark it" already carries it.

#### 🟡 F2 — Story-count estimate drifted (5–6 → 8) and is not reflected in the PRD

`prd.md` § Resource Requirements states _"M1 ≈ 5–6 stories."_ Epic 1 now contains **8** (1.1–1.6 + 1.7a + 1.7b).

**Impact:** low — an estimate, not a contract. But it is the only place the PRD sizes M1, and a 33–60% overrun on the sole size signal is worth recording rather than silently absorbing.
**Recommendation:** update the parenthetical to "M1 ≈ 8 stories (6 backend + 1 UX spec + 1 frontend)". Alternatively leave it and note the drift in the retro. **Requires an Alexyu ruling — see Step 6.**

#### 🟡 F3 — FR21 and FR27 are "covered" by already-shipped code, with no story to point at

The map credits both to Epic 1 with the notes _"BYO key (env-var in M1)"_ and _"TMDb match correction (existing flow)"_. Both are accurate — `CLAUDE_API_KEY` already exists in `apps/api/internal/config/config.go`, and `ManualSearchDialog` / `POST /api/v1/metadata/manual-search` shipped in Story 3.7. But a reader scanning for "which story delivers FR21?" finds nothing.

**Impact:** low, but it is a **planning trap in the other direction** — someone may schedule work that is already done.
**Recommendation:** annotate both rows explicitly as **"already shipped — zero new work in M1"** with the pointer (`config.go` / Story 3.7).

#### 🟡 F4 — The G2/G4 measurable thresholds have no owning story

Carried forward from the 2026-07-26 report, still open. NFR-P1's degradation threshold and the "usable without hand-editing ≥ target %" trust bar are deliberately deferred to the M1 pilot — but the earlier report's own instruction was that they _"must become measurable acceptance criteria at story time"_. Story time has arrived: **8 stories exist and none carries either threshold.**

**Impact:** medium. M1's stated purpose is to validate exactly this assumption. Shipping M1 without a pre-agreed pass mark means the pilot produces data no one has committed to interpret.
**Recommendation:** the natural home is **sub-1-6** (the story that makes the pipeline actually run end-to-end) or a dedicated pilot-instrumentation story. **Requires an Alexyu ruling — see Step 6.**

#### ℹ️ F5 — Only 4 of 9 story files are drafted (constrains Step 5)

Drafted: `sub-1-1`, `sub-1-2`, `sub-1-7a`, `sub-1-7b`. Not yet drafted: `sub-1-3`, `sub-1-4`, `sub-1-5`, `sub-1-6`, `sub-2-1` — these exist only as epic-level ACs.

**Impact:** Step 5 (story quality) can assess **4 files in depth** and the remaining 5 **only at epic-AC level**. That is a genuine limitation of this assessment, not a defect in the plan.

---

## Step 4 — UX Alignment

### UX Document Status: **FOUND**

| Artifact | Role |
|---|---|
| **`ux-design.pen`** flow F v2 (repo root) | The M1 UX source of truth — generation-centric subtitle management (13 desktop + 4 mobile screens). Encrypted; Pencil MCP access only. |
| `_bmad-output/screenshots/flow-f-subtitle-v2/` | Rendered evidence (18 PNGs) that implementation verifies against. |
| `ux-redesign/` (7 files) | Design language v2 + nav IA ADR + pen review — the standing UX-redesign context. |
| `ux-design-specification.md` (222.7 KB) | **Authoritative** per Step 1 Issue 2 (Alexyu ruling 2026-07-27). |
| **`flow-j-specs/`** | Cross-cutting design-decision spec screens (`j1-d.png` today; `j2-d.png` added by sub-1-7a). |

UX is not merely implied — it is documented, rendered, and version-controlled. **No missing-UX warning applies.**

### UX ↔ PRD Alignment

**Aligned:** flow F v2 maps cleanly onto the M1 FRs — F2-D-v2 `btn-generate` is the FR12 manual trigger; F5-D-v2 is the FR23 capability gate; F3-D-v2 is the FR33 progress surface. PRD journeys J1/J3/J5 each have a screen.

**Three misalignments — all already routed to `sub-1-7a` AC #7:**

| # | Misalignment | PRD position | Fix |
|---|---|---|---|
| 1 | **F2-D-v2 copy says 轉錄 (transcription)** | Transcription is **ASR = FR29, [P2]**. M1 extracts an existing embedded track (FR2). The screen promises a capability M1 does not ship. | sub-1-7a AC #7 · 轉錄 → 抽取內嵌字幕 |
| 2 | **F5-D-v2 前往設定 links to a settings page that does not exist in M1** | FR25 is **[M1.5]**. The architecture records this explicitly: _"J3's dead 前往設定 loop remains broken through M1 as accepted debt."_ Accepted debt is fine; an unlabelled dead link is not. | sub-1-7a AC #7 · state M1 behaviour |
| 3 | **F5-D-v2 frames FFmpeg as a user setting** | **NFR-I2** ships ffmpeg **bundled in the multi-arch image** — it is a deployment property, never user-configurable. This is a UX↔**architecture** conflict, not just copy. | sub-1-7a AC #7 · reframe as deployment |

**One structural gap — closing:** PRD **FR5** and **FR9** require a *user-visible* mark (`no_text_source` / `skipped`), but flow F v2 contains **no badge screen** — badges live in flows A/B (library/detail), which the subtitle-pipeline UX work never covered. This is precisely why `sub-1-7a` creates a standalone `flow-j-specs` spec screen rather than editing a flow-F screen. **Gap identified and already assigned.**

### UX ↔ Architecture Alignment

**Aligned:** D6 (SSE stage enum extension) supports F3-D-v2's progress surface. D2's `subtitle_status` extension supplies the state the badge renders. Performance is not a UX risk here — the pipeline is I/O + HTTP bound and the UI is existing-component reuse (no new rendering load).

**🔴 F6 — The architecture's Delta tree contains zero `apps/web/**` entries.**

`subtitle-pipeline-architecture.md` § Project Structure & Boundaries lists ~28 files across `apps/api/**`, `docs/`, `project-context.md`, and `code-review/instructions.xml` — and closes with _"~16 new · ~12 modified · ~14 explicitly untouched."_ That was accurate when M1 was backend-only. It no longer is: sub-1-7b modifies `apps/web/src/utils/libraryStatus.ts`, `apps/web/src/components/media/EpisodeList.tsx`, and their two co-located specs; sub-1-7a modifies `ux-design.pen`, `scripts/export-pen-screenshots.py`, and `ux-design-specification.md`.

**Impact:** medium. The Delta tree is the architecture's answer to "what does this feature touch". A dev planning frontend work finds no architectural home for it, and the file counts are now wrong.
**Recommendation:** add a frontend delta section + recount. **Do not** move the badge label/tint ruling into the architecture — that is a UX decision whose home is the `.pen` spec screen; the architecture should cross-reference it. (The architecture's own § Scope of this section already forbids restating bound decisions, precisely to avoid a second source of truth.)

### 🔴 F7 — The unified Spec is stale in three places, and it is the PRD's declared primary source

`prd.md` frontmatter lists `vido-subtitle-pipeline-spec.md` first under `inputDocuments`, and the PRD header names it _"Primary source (unified tech spec)"_. The architecture's § Open Inconsistencies adjudicated three Spec-vs-PRD conflicts and ruled **"PRD wins"** in all three — **but the Spec itself was never corrected.** Anyone tracing a requirement upstream to the "primary source" reads the losing side.

| # | Spec says | PRD/architecture rules | Spec locations |
|---|---|---|---|
| 1 | Output extension **`.zh-TW.srt`** | **`.zh-Hant.srt`** — PRD § Technical Success writes it as _"`.zh-Hant.srt` / `.zh-Hans.srt` per project-context §9b (**NOT** `.zh-TW.srt`)"_; NFR-I3 repeats it | `:16`, `:61`, `:148` — **three occurrences** |
| 2 | § 5 金鑰設定 UI marked **必做 (must-do)**, incl. fixing the `ManageSubtitleDialogV2` 前往設定 dead loop | FR25 is **[M1.5]** — M1 uses an env-var key and the dead loop is accepted debt through M1 | § 5 heading + body |
| 3 | Line-segmentation ≤N-char rule applied broadly | PRD restricts it to the **ASR path (M2)**; the extract path preserves professional segmentation — the PRD flags this explicitly as a correction of an earlier over-application | § 7 |

**Impact: medium-high, and rising.** Inconsistency 1 is the dangerous one — it is a *concrete literal* (`.zh-TW.srt`) that appears **three times** and that a dev could copy into `placer.go`. Writing the wrong extension breaks NFR-I3 (Plex / Jellyfin / Video Station auto-detection) — the failure is silent at build time and only visible when a player does not list the track.

**Recommendation:** correct the Spec, or mark it superseded. Two options, needing an Alexyu ruling — see Step 6:

- **(a) Correct in place** — fix all three, add a note that the PRD + architecture supersede on conflict. Keeps the Spec usable as the unified overview it was written to be.
- **(b) Mark superseded** — add a header banner: _"Superseded by `prd.md` + `subtitle-pipeline-architecture.md` for anything they cover; retained for §2 hardware analysis and §6/§6.5 P2 design."_ Cheaper, and honest about the Spec having done its job.

Either way, **do not leave a literal `.zh-TW.srt` in the declared primary source.**

### Warnings

- ⚠️ **No missing-UX warning** — documentation exists at every level.
- ⚠️ **F6** — architecture does not account for the frontend surface it now has.
- ⚠️ **F7** — the declared primary source contradicts the PRD in three places, one of them a copy-pasteable filename literal.
- ℹ️ The three flow-F copy misalignments are **already assigned** to sub-1-7a AC #7 and are not open risks.

---

## Step 5 — Epic Quality Review

**Standard applied:** `create-epics-and-stories` best practices — user value, epic independence, no forward dependencies, story sizing, AC quality, database-creation timing.
**Coverage limitation (F5):** 4 of 9 story files assessed in depth (`sub-1-1`, `sub-1-2`, `sub-1-7a`, `sub-1-7b`); `sub-1-3` … `sub-1-6` and `sub-2-1` assessed at epic-AC level only.

### Epic Structure

| Check | Epic 1 | Epic 2 |
|---|---|---|
| User-centric title | ✓ "Automatic Traditional-Chinese subtitles for English media" | ✓ "In-app provider-key configuration" |
| Describes a user outcome | ✓ a correctly-timed `.zh-Hant.srt` appears beside the file | ✓ configure keys without env-vars |
| Benefits users **alone** | ✓ explicitly: _"Standalone and shippable using an env-var API key; does not require Epic 2"_ | ✓ builds on Epic 1 only |
| Independence direction | ✓ stands alone | ✓ depends on Epic 1 (allowed); **Epic 1 does not depend on Epic 2** |

**No epic-independence violation.** No epic requires a later epic. ✓

### Dependency Analysis — all backward, zero forward ✓

| Story | Depends on | Direction |
|---|---|---|
| 1.1 | — (independent) | ✓ |
| 1.2 | — (independent; **zero file overlap with 1.1**, verified at story-authoring) | ✓ |
| 1.3 | — (self-contained: codes + stage enum + docs) | ✓ |
| 1.4 | 1.2 (enum) | ✓ backward |
| 1.5 | 1.1 (SDK client) · 1.2 (provenance) · 1.4 (extracted subtitle) | ✓ backward |
| 1.6 | 1.5 | ✓ backward |
| 1.7a | 1.2 (contract only) | ✓ backward |
| 1.7b | 1.7a · 1.2 | ✓ backward |
| 2.1 | Epic 1 · 1.7a (AC #7 unblock) | ✓ backward |

**No story references future work. No circular dependencies.** This is clean and is the single most important structural check — it passes without qualification.

### 🔴 Critical Violations

**None.** No technical epic, no forward dependency, no epic-sized unfinishable story.

### 🟠 Major Issues

#### M1 — `sub-1-5` is under-split and is simultaneously the highest-risk story

Its epic ACs carry **five** distinct concerns: (i) the per-cue quality gate + retry — which is **NAIL 3**, test-first; (ii) timestamp integrity (numbered-text-only prompt, Go-side re-stitch, cue-by-cue equality assertion); (iii) OpenCC final pass **plus** `placer.go` delivery with the overwrite predicate; (iv) the versioned cache key with the 4096-token disable-and-record rule; (v) D10 per-show first-request serialization.

That is an orchestrator, a quality subsystem, a caching subsystem, a concurrency gate, and a delivery path in one story — carrying the epic's own 🔴 top risk marker.

**This repo already set the precedent for exactly this shape:** Story 13-4 was _"BACKEND-ONLY (no cross-stack split) but LARGE → size-split"_ into 13-4a / 13-4b. The Epic-8-retro story-splitting rule keys on cross-stack task counts and therefore does **not** trigger here — which is precisely why it needs a human call rather than a rule.

**Impact:** high. If 1.5 stalls, M1 stalls — 1.6 depends on it, and 1.1–1.4 deliver nothing user-visible on their own (see M2). A story this size also makes the test-first obligation on NAIL 3 easy to erode under schedule pressure.
**Recommendation:** size-split into **1.5a** (orchestrator + translate + quality gate + retry — NAIL 3 lives here) and **1.5b** (cache key + per-show serialization + placer delivery), following the 13-4a/13-4b precedent. **Requires an Alexyu ruling — see Step 6.**

#### M2 — Three stories are enablers with no independently demonstrable user value

`sub-1-1` (_"As a maintainer"_), `sub-1-2` (_"As the pipeline"_), and `sub-1-3` (_"As a developer and a frontend consumer"_) have no end-user beneficiary. Strict vertical-slice orthodoxy would reject all three.

**The counter-argument is real and should be recorded, not dismissed:** the architecture sequenced them deliberately (§ Decision Impact Analysis, steps 1–4 of 7), 1.1 is a genuine **brownfield migration story** — the category the standard explicitly expects in brownfield projects — and Epic 1 as a whole *is* the vertical slice.

**The residual risk is what matters:** if M1 is cut short after 1.3, **three stories of work ship zero user value**. Nothing is demonstrable until 1.4 extracts a subtitle and 1.5 translates it. That is a real product-risk position, and it should be an accepted one rather than an unnoticed one.
**Recommendation:** accept with eyes open. Optionally reorder so a thin end-to-end path lands earlier — but note the architecture's dependency chain makes that expensive, and the deliberate sequencing is itself a documented decision.

### 🟡 Minor Concerns

- **m1 — Table created 3 stories before its first writer.** `sub-1-2` creates `subtitle_runs`; the first code that writes a row is `sub-1-5`. The standard says _"tables created only when first needed."_ **Justified** by Rule 20: D2's `subtitle_status` enum and D6's SSE stages are both frontend wire contracts and the architecture explicitly wants them _"stamped together so the frontend absorbs one coordinated change, not two."_ Recording it so it reads as a decision, not an oversight.
- **m2 — `sub-1-1` ships an interface with no consumer for four stories.** AC #5's `CachingCompleter` is `[@contract-v1]`-stamped with `sub-1-5` as its only consumer. Speculative generality by the letter, but the epic AC demands the capability and the story explicitly defers all *policy* to 1.5. Acceptable.
- **m3 — Epic 1 now carries work whose sole beneficiary is Epic 2.** `sub-1-7a` AC #7 closes the three `.pen` copy revisions blocking Epic 2. The dependency direction stays legal (Epic 2 → Epic 1), but it mildly dilutes Epic 1's user-value focus. **Deliberate, product-owner-ruled (2026-07-27), with recorded rationale** (one `.pen` edit, one regeneration, one commit vs. paying the non-deterministic re-render risk twice). Accepted.
- **m4 — `sub-1-7a` AC numbering is out of narrative order** (#1–#5, #7, #6, #9, #8) after two scope expansions. Every AC is uniquely labelled and all cross-references (sprint-status, epics doc, task headers) are consistent, so this is cosmetic. Renumbering now would break external references — leave it.
- **m5 — `sub-1-2` ACs do not cover migration failure/rollback** beyond `Down()`. The 027 test precedent covers CHECK rejection and defaults but not partial-apply recovery. Low risk (the runner wraps `Up` in a transaction), noted for completeness.

### Best-Practices Compliance Checklist

| Check | Epic 1 | Epic 2 |
|---|---|---|
| Delivers user value | ✓ | ✓ |
| Functions independently | ✓ | ✓ (needs Epic 1 — legal) |
| Stories appropriately sized | ⚠️ **M1 — 1.5 under-split** | ✓ |
| No forward dependencies | ✓ | ✓ |
| Tables created when needed | ⚠️ **m1 — justified deviation** | n/a |
| Clear acceptance criteria | ✓ **well above standard** on the 4 drafted files — hit-count assertions, exact paths, per-test disposition tables, ordering-regression tests. These are verifiable claims, not statements of intent. | ✓ (epic-level) |
| Traceability to FRs | ⚠️ **F1 — map stale for FR5/FR9; 1.7a/1.7b absent** | ✓ |

### Brownfield Compliance

✓ Correct. No greenfield scaffolding story — explicitly and correctly justified (_"No scaffolding story — brownfield. First story is the `claude.go` SDK migration, not project init."_). Integration points are named and bounded: the strangler wrapper + feature-flag seam at exactly `batch.go:244`, `engine.go` demoted to search fallback, `placer.go` as sole writer. This is textbook brownfield sequencing.

---

## Summary and Recommendations

### Overall Readiness Status: ✅ **READY FOR PHASE 4 IMPLEMENTATION**

The 2026-07-26 assessment returned _"READY FOR EPIC BREAKDOWN — NOT YET READY FOR PHASE 4"_ for one structural reason: no epics or stories existed, so steps 03 and 05 were N/A. **That gate is now cleared.** Epics exist, 9 stories are planned across 2 epics, 4 are drafted to a high standard, FR coverage is 100%, and the dependency graph is clean in the direction that matters.

**Nothing in this assessment blocks implementation starting today.** Every finding is a documentation defect, a sizing judgement, or a deferred decision — not a missing capability or a broken dependency.

### The one finding I would not ship without fixing

**F7 — the unified Spec contains a copy-pasteable wrong filename.** `vido-subtitle-pipeline-spec.md` says **`.zh-TW.srt`** in three places (`:16`, `:61`, `:148`) while the PRD says `.zh-Hant.srt` and calls out `(NOT .zh-TW.srt)` explicitly. That Spec is the PRD's **declared primary source**. A developer tracing upstream reads the losing side of a settled ruling, and writing the wrong extension silently breaks NFR-I3 — Plex / Jellyfin / Video Station stop auto-detecting the track, and nothing fails at build time.

Everything else can be scheduled. This one is a live trap.

### All findings by severity

| ID | Severity | Finding | State |
|---|---|---|---|
| **F7** | 🔴 | Unified Spec stale in 3 places (`.zh-TW.srt` ×3 · §5 key UI marked 必做 vs FR25 [M1.5] · §7 line-length over-applied) — and it is the PRD's declared primary source | **needs ruling** |
| **F1** | 🔴 | FR Coverage Map stale: FR5/FR9's user-visible half unrecorded; Stories 1.7a/1.7b absent from the map entirely | **needs fix** |
| **F6** | 🔴 | Architecture Delta tree has zero `apps/web/**` entries; file counts wrong now that 1.7a/1.7b exist | **needs fix** |
| **M1** | 🟠 | `sub-1-5` under-split — orchestrator + quality gate (NAIL 3) + cache key + concurrency gate + delivery in one 🔴-risk story | **needs ruling** |
| **M2** | 🟠 | 1.1/1.2/1.3 are enablers with no independently demonstrable user value — nothing ships if M1 is cut short after 1.3 | **accept with eyes open** |
| **F4** | 🟡 | G2/G4 measurable thresholds (NFR-P1 degradation bar, trust %) have no owning story, though "story time" has arrived | **needs ruling** |
| **F2** | 🟡 | PRD's "M1 ≈ 5–6 stories" is now 8 | **needs ruling** |
| **F3** | 🟡 | FR21 / FR27 credited to Epic 1 but delivered by already-shipped code, with no pointer | **needs fix** |
| **m1–m5** | 🟡 | Table created 3 stories early (justified) · unused interface for 4 stories (justified) · Epic 1 carries Epic 2's unblock (ruled) · AC numbering out of order (cosmetic) · no migration-rollback AC | **accepted / noted** |
| **F5** | ℹ️ | Only 4 of 9 story files drafted — constrains this assessment, not the plan | informational |

**Also resolved during this assessment:** the `ux-design-specification.md` authority question (Alexyu ruled: still authoritative → `sub-1-7a` AC #9 + Task 4 added, bounded to ≤ 2 edits). The three flow-F copy misalignments were already assigned to `sub-1-7a` AC #7 before this review and are not open risks.

### Recommended Next Steps

1. **Fix the Spec (F7) before any dev picks up `sub-1-5`.** Either correct all three inconsistencies in place, or add a superseded-by banner. `.zh-TW.srt` must not survive in the declared primary source.
2. **Fix the three documentation defects in one pass** — F1 (FR Coverage Map rows for FR5/FR9 + 1.7a/1.7b), F6 (architecture frontend delta + recount + `subtitle_run` → `subtitle_runs`), F3 (annotate FR21/FR27 as already-shipped). ~30 minutes of editing; all are audit-trail integrity, none blocks code.
3. **Rule on `sub-1-5`'s split (M1)** before it is drafted — splitting after a story file exists costs more than splitting the epic AC now.
4. **Rule on where the G2/G4 thresholds live (F4)** — `sub-1-6` or a dedicated pilot-instrumentation story. M1's entire purpose is validating this assumption; shipping it without a pre-agreed pass mark wastes the pilot.
5. **Proceed with `sub-1-1` and `sub-1-2` in parallel now.** They are independent, zero file overlap, and both are `ready-for-dev`. None of the findings above touches them.

### Cross-document action register

| Document | Change | Driver |
|---|---|---|
| `vido-subtitle-pipeline-spec.md` | `.zh-TW.srt` → `.zh-Hant.srt` ×3 · §5 mark [M1.5] · §7 scope line-length to ASR — **or** a superseded-by banner | F7 |
| `epics-subtitle-pipeline.md` | FR Coverage Map: FR5/FR9 rows + 1.7a/1.7b · overview's "20 existing Claude tests" → **26** | F1 · Step 5 evidence |
| `subtitle-pipeline-architecture.md` | Delta tree frontend section + recount · `subtitle_run` → `subtitle_runs` (Rule 6) · cross-ref the `.pen` badge spec (do **not** restate it) | F6 |
| `prd.md` | "M1 ≈ 5–6 stories" → 8 _(optional — F2)_ | F2 |
| `ux-design-specification.md` | `StatusBadge` enumeration + pointer to `flow-j-specs/j2-d.png` | **already assigned** — `sub-1-7a` AC #9 |
| `ux-design.pen` | 3 Epic 2 copy revisions + the new `j2-d` spec screen | **already assigned** — `sub-1-7a` AC #7 / AC #1 |

### One number worth correcting immediately

`epics-subtitle-pipeline.md` § Overview lists NAIL 1 as _"All **20** existing Claude tests stay green."_ The verified count is **26** — 24 in `claude_test.go` plus **2 in `retry_test.go`** (`TestClaudeProvider_RetriesTransientThenSucceeds`, `TestClaudeProvider_NoRetryOnPermanent4xx`). Those two are the guard for **NAIL 2** (the D8 double-retry ban). A developer reading the epic rather than the story could declare NAIL 1 satisfied at 24 green while the double-retry guard was never run.

`sub-1-1` carries the correct list and disposition table, so implementation is protected — but the epic is the document humans read, and it currently understates its own guard.

### Final Note

This assessment identified **11 findings across 5 categories** (3 critical, 2 major, 4 minor, 1 informational, plus 5 accepted sub-items) and **resolved 1 open question** from the prior report. Three criticals are documentation-integrity defects, not capability gaps.

**Verdict: implementation may begin now.** Fix F7 before `sub-1-5` is drafted; fix F1/F6/F3 in a single documentation pass; rule on M1 and F4 at your convenience. `sub-1-1` and `sub-1-2` are unblocked and independent — start them today.

---

**Assessed:** 2026-07-27 · **Assessor:** John (PM), adversarial review
**Standard:** `check-implementation-readiness` — full 6-step run (the 2026-07-26 run was partial; steps 03/05 were N/A)
**Supersedes:** `implementation-readiness-report-subtitle-pipeline.md` (2026-07-26)

---

## Second Pass — Adversarial Re-validation (same day, re-run at higher effort)

**Method:** attempt to **refute** each first-pass finding with direct evidence (not re-read my own conclusions), then sweep for what the first pass missed. Every claim below carries a file:line.

### Refutation results — all 11 findings survive; three are strengthened

| ID | Second-pass result |
|---|---|
| **F7** | **SURVIVES, STRENGTHENED — now 4 stale items across 6 locations.** First pass cited §7's line-length over-application from the architecture's ruling without direct verification (a grep for `字元/兩行` missed it). Direct read confirms it at **`:138`** — _"斷句:每行 ≤ N 字、雙行規則、SDH 過濾"_ stated **unqualified** (the PRD restricts it to the ASR path). **New 4th item:** the **§8 M1 milestone checklist itself** lists _"金鑰設定 UI(§5)+ 修死迴圈"_ as an M1 deliverable (**`:149`**) — so the Spec's M1 definition contradicts FR25 [M1.5] in **two** places (§5 heading + §8 checklist), not one. Full stale set: `.zh-TW.srt` ×3 (`:16`, `:61`, `:148`) · §5 必做 (`:87-93`) · §8 M1 checklist (`:149`) · §7 line-length (`:138`). |
| **F1** | **SURVIVES, STRENGTHENED.** The "20 Claude tests" figure appears **three** times in the epics doc, not two: § Overview ceremony (`:21`), § Additional Requirements (`:105`), and — the one the first pass missed — **inside Story 1.1's own AC text (`:192`)**: _"all 20 existing `claude_test.go` tests pass"_. That third occurrence is doubly wrong: the count (26, not 20) **and the file scope** — naming `claude_test.go` alone excludes the 2 `retry_test.go` tests that guard NAIL 2. FR map rows FR5 (`:133`) / FR9 (`:137`) confirmed unchanged; no 1.7a/1.7b row exists. |
| **F6** | **SURVIVES, STRENGTHENED — the Delta tree is stale in a third direction.** Architecture validation **V3** (`:551`) states _"Adds `scanner_service.go` ✏️ to the delta tree"_ — but the Delta tree's `services/` block (`:445-448`) still lists only `ffprobe_service.go 🔒 / translation_service.go ✏️ / terminology_service.go 🔒`. **The document claims an amendment it never made.** Also noted: § Validation opens with _"the 23 M1 FRs"_ (`:532`) — the PRD count is 22. Fold both into the F6 fix pass. |
| F2, F3, F4, F5, M1, M2, m1–m5 | Survive unchanged. **F3's already-shipped claims now positively verified:** `CLAUDE_API_KEY` at `config.go:122` ✓ · `ManualSearchDialog.tsx` exists at `apps/web/src/components/manual-search/` ✓. |

### Refuted — one potential gap is NOT a gap (good news)

**"NFR-I2 has no owning story" would have been a false finding.** Direct evidence: `apps/api/Dockerfile:47` already runs `apk add --no-cache ca-certificates tzdata ffmpeg` (Alpine's ffmpeg package bundles ffprobe), and `.github/workflows/docker.yml:80` already builds `platforms: linux/amd64,linux/arm64`. **The multi-arch-image-with-ffmpeg requirement is shipped infrastructure, not missing work.** The 2026-06 audit's silent-degradation risk is already closed at the image level. Only the *documentation* half remains — see F9.

### New findings (missed by the first pass)

#### 🟠 F8 — The TestSprite TC↔epic mapping was never declared (standing process-rule violation)

`sprint-status.yaml` carries a **PROCESS RULE (Alexyu, 2026-07-22)**: _"an epic whose scope maps to TestSprite journeys may NOT close until its mapped TCs pass on the local seeded env. **TC↔epic mapping is declared in the epic's entry when stories are authored (sm create-story)**."_

Stories were authored 2026-07-27. The `epic-subtitle-pipeline-m1` entry declares **no TC mapping**. And the scope unambiguously maps:

- **TC071–TC078** (subtitle search dialog journeys) exist in the v4 plan today and sit directly on surfaces this epic changes — **TC073** asserts the empty-state copy _"尚無結果 — 線上來源成功率低，建議改用生成字幕"_, which is exactly the F2-D-v2 dialog area whose copy `sub-1-7a` AC #7 revises. The epic will **break or invalidate** TC073-class assertions as it lands.
- PR #174 (`chore(testsprite-r3)`) seeded **round-3 candidates + a subtitle-pipeline spec** — the successor TCs for this epic's journeys are already queued material.

**Impact:** medium — the epic's **close-gate is undefined**. Without a declared mapping, "may NOT close until mapped TCs pass" is unenforceable, and the rule was created (2026-07-22) precisely to prevent that.
**Who fixes it:** the SM (the rule names `sm create-story` as the owner — this is Bob's miss, caught here). One line on the epic entry: declare TC071–078 as affected-and-must-pass (with TC073 flagged as will-need-reauthoring after the F2 copy revision), plus a pointer to the round-3 subtitle-pipeline candidates for post-M1 coverage.

#### 🟡 F9 — The Delta tree's deployment-doc updates have no owning story

Architecture § Development-workflow integration (`:527`) and the Delta tree both require `docs/deployment.md` **+ its zh-TW counterpart** (Rule 17) to document ffmpeg bundling, multi-arch, and NFR-S3. No story owns this: 1.3's bilingual-docs scope is `sse-event-types` only; 1.6 and 2.1 don't mention deployment docs.
**Recommendation:** assign the ffmpeg/multi-arch half to **sub-1-6** (it ships the end-to-end pipeline that makes the docs true) and the NFR-S3 half to **sub-2-1** (it ships the key page NFR-S3 governs). Two one-line AC additions when those stories are drafted.

#### 🟡 F10 — The epics frontmatter scope statement is stale

`epics-subtitle-pipeline.md` frontmatter: _"scope: '… M1.5/P2/Tier-2 requirements are inventoried for traceability but **epics are M1-only**.'"_ The document has contained **Epic 2 (M1.5) with Story 2.1** since creation, and Story 1.7a now explicitly carries Epic 2's unblock. The scope line misdescribes the document it heads.
**Recommendation:** reword to "stories are drafted M1-first; Epic 2 (M1.5) is defined and unblocks via Story 1.7a." Fold into the F1 fix pass (same file).

#### ℹ️ Also noted (fold into existing fix passes)

- Architecture § Open cross-document actions (`:605-609`) still lists the FR6 amendment and FR28 tag as **open** — both were closed in the PRD on 2026-07-26 (verified in Step 2). Annotate as done when fixing F6, and mark action 3 (`.pen` revisions) as "→ sub-1-7a AC #7".

### Revised findings summary (supersedes the first-pass table)

| ID | Severity | Finding | State |
|---|---|---|---|
| **F7** | 🔴 | Spec stale — **4 items / 6 locations** (`.zh-TW.srt` ×3 · §5 必做 · **§8 M1 checklist** · §7 line-length) in the PRD's declared primary source | **needs ruling** (fix in place vs supersede banner) |
| **F1** | 🔴 | Epics doc traceability stale — FR5/FR9 map rows, no 1.7a/1.7b row, "20 tests" **×3** incl. Story 1.1's own AC (count + file scope both wrong) | needs fix |
| **F6** | 🔴 | Architecture Delta tree stale ×3 — no `apps/web/**` · V3's claimed `scanner_service.go` amendment never applied · counts wrong ("23 M1 FRs", "~16/~12/~14") + `subtitle_run`→`subtitle_runs` | needs fix |
| **F8** | 🟠 **new** | TestSprite TC↔epic mapping never declared — epic close-gate undefined; TC071–078 affected, TC073 will need reauthoring | needs fix (SM, one line) |
| **M1** | 🟠 | `sub-1-5` under-split (five concerns, 🔴 top-risk story; 13-4a/b precedent) | **needs ruling** |
| **M2** | 🟠 | 1.1–1.3 are enablers; nothing demonstrable if M1 stops early | accept with eyes open |
| **F4** | 🟡 | G2/G4 thresholds own no story | **needs ruling** |
| **F9** | 🟡 **new** | Deployment-doc updates (bilingual, Rule 17) unowned — assign to 1.6 + 2.1 | needs fix (2 AC lines) |
| **F2** | 🟡 | PRD "M1 ≈ 5–6 stories" → 8 | needs ruling |
| **F3** | 🟡 | FR21/FR27 = already-shipped code, unannotated (both now positively verified) | needs fix |
| **F10** | 🟡 **new** | Epics frontmatter "epics are M1-only" misdescribes the document | needs fix |
| m1–m5 · F5 | 🟡/ℹ️ | unchanged | accepted / noted |

**Verdict unchanged: ✅ READY FOR PHASE 4.** The re-run added 3 findings and strengthened 3 — every one is documentation/process integrity. `sub-1-1` and `sub-1-2` remain unblocked; nothing new touches them. The refuted NFR-I2 gap means the Docker/ffmpeg story some plans would have scheduled is **already-shipped work — do not schedule it**.

---

**Second pass:** 2026-07-27 · same assessor, re-run on request after model upgrade · first-pass findings: 11/11 survived refutation, 3 strengthened, 3 added, 1 candidate gap refuted

---

## Resolutions Applied (2026-07-27 — Alexyu rulings, post-assessment)

| # | Ruling | Applied |
| --- | --- | --- |
| **F7** | **(a) fix in place** | Spec corrected: `.zh-Hant.srt` ×3 · §4-table/§5/§8 金鑰 UI → **M1.5** (FR25) · §7 line-length scoped to the ASR path · supersession note added under the header ("PRD + architecture win on conflict") |
| **M1** | **split** | Story 1.5 → **1.5a** (translate core + quality gate — NAIL 3, test-first) + **1.5b** (cache key + D10 serialization + delivery + P9 provenance + NFR-R3 resume). Epics doc + sprint-status updated; sub-1-1/sub-1-2 consumer notes updated. M1 = 9 stories |
| **F4** | **→ sub-1-6** | Story 1.6 gained the G2/G4 measurable-bars AC: NFR-P1 resource bound · concrete time-to-`.zh-Hant.srt` number · trust-% bar (X set with Alexyu at drafting) |
| F1/F3/F10 | fix pack | Epics doc: FR5/FR9 rows now carry the user-visible half (1.7a→1.7b) · FR21/FR27 annotated **already shipped** (`config.go:122` / Story 3.7) · "20 tests" → **26** ×3 incl. Story 1.1's own AC (now naming `retry_test.go`) · migration enum list +`probing` · `subtitle_run`→`subtitle_runs` (Rule 6) · frontmatter scope reworded |
| F6 | fix pack | Architecture: frontend delta block added (4 web files + `.pen` + export script + ux-spec) with a pointer to the j2 spec screen (not a restatement) · `scanner_service.go` (the V3 amendment) finally applied · migration filename synced to sub-1-2 · counts recounted (~20 modified) · "23 M1 FRs"→22 · claude_test rows → 24+2=26 · both cross-document action lists annotated (FR6/FR28 ✅ 07-26; `.pen` → sub-1-7a AC #7) |
| **F8** | fix pack | TC↔epic mapping **declared** on `epic-subtitle-pipeline-m1`: TC071–078 must pass on the local seeded env before close · TC073 reauthor after the F2 copy revision · round-3 candidates = successor coverage |
| F9 | fix pack (same surface as F4) | Deployment-doc ownership assigned: ffmpeg/multi-arch half → Story 1.6 AC · NFR-S3 half → Story 2.1 AC (infra itself verified already shipped) |
| F2 | **left open** | PRD "M1 ≈ 5–6 stories" is now 9 — optional one-line edit; take it or record at the retro |

All rulings executed same-day. Remaining open items: **F2** (optional) and the two accepted-risk positions (M2 enabler run, m1–m5).
