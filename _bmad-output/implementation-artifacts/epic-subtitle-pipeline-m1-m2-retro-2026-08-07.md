# Retrospective — Subtitle Pipeline M1 + M1.5 + M2 (Combined)

**Date:** 2026-08-07 · **Facilitator:** Bob (SM) · **Participants:** Alexyu (Project Lead), Winston (Architect), Amelia (Dev), Murat (TEA), Sally (UX), John (PM)

**Scope ruling (Alexyu, in-session):** M1's owed retrospective (the 2026-08-05 "retro gates epic done" ruling) is folded into this session together with M2. M1.5 (Epic 2) — whose sprint note already read "retro is the only gate left" — is folded in as well: the three epics are one continuous subtitle-pipeline arc and are closed together by this document.

---

## 1. Epic Summaries

### M1 — epic-subtitle-pipeline-m1 (extraction + LLM translation, 9 stories, 2026-07-27 → 2026-08-05)

sub-1-1 … sub-1-7b (1.5 split per IR-r2). Delivered the D2/D6 contract pair, the routing front-half (probe → extract → detect → route), the item flow (pre-flight / run rows / segment cache / D10 show gate / placer-sole-writer), the D5 flag seam + worker pool + capability gate, and the FE status-badge surface. Every story closed with a same-session adversarial CR carrying real findings. Zero production incidents.

### M1.5 — epic-subtitle-pipeline-m1-5 (Epic 2: in-app key config + untranslated arc, 6 stories, 2026-08-05 → 2026-08-06)

sub-2-1a/b (key resolution + settings page; killed the boot-time key freeze for Claude) and sub-2-2a/b/c/d (the `untranslated` 10th D2 value α/β/γ/2.2d chain: enum + resume marker → badge → design ratification → copy-in-code). Notable: the α bump v1→v2 exercised the full Rule 20 bump/stale-mark machinery in production conditions.

### M2 — epic-subtitle-pipeline-m2 (ASR fallback, 2 stories, 2026-08-06 → 2026-08-07)

Originally three workstreams (ASR fallback + provider completion + search-priority reorder). **spike-2026-08-06-pipeline-ordering-evidence (PR #207) collapsed it to one leg before any implementation spend**: measured end-to-end usable rate of the online-search layer against the real fall-through population was 1/14 (the sole "hit" being the wrong episode), and alass could not serve as an automatic acceptance gate (20% failure; cross-show negatives overlap the correct-but-offset range) → search layer CANCELLED for the automatic path. sub-3-1 (movie leg, PR #208) + sub-3-2 (episode leg, PR #209, same-day discovery-promotion of `backlog-episode-asr-fallback`). D2 `[@contract-v2→v3]` (no_text_source terminality), SSE media_id `[@contract-v1→v2]` (CR catch). 2/2 stories done, CRs 0H/2M/3L and 0H/1M/2L, all addressed in-session.

---

## 2. What Went Well

1. **Spike-before-epic (Alexyu's headline pick).** The A/B route question was settled with measurements — hit rates, end-to-end usable rate, alass gate failure modes — *before* choosing a path. A whole planned workstream (search layer) was cancelled on evidence instead of being built and discovered useless. The spike doc doubles as the permanent audit trail every planning doc now cites.
2. **Rule 24 lane-③ at full speed.** sub-3-1 discovered the episode gap at implementation, filed the backlog entry with breadcrumb comments at both guard sites, and sub-3-2 was promoted, implemented, CR'd and merged the same day. Discovery → tracked entry → story → shipped, no prose-only time bombs.
3. **Contract machinery caught real drift twice.** The D2 v2→v3 bump ran the full producer-side stale-mark sweep (all acked consumers frozen), and sub-3-2's CR caught the `transcription_*` SSE `media_id` stamp silently widening from movie-id to media-id — exactly the class of wire-contract rot Rule 20 exists to stop.
4. **Additive-seam patterns kept regression surface near zero.** Optional port (`WithSpeechTranscriber`), additive option (`WithMediaType`, movie default), narrow episode interfaces: existing callers byte-unchanged, eight test call sites updated mechanically with assertions untouched, both PRs green on first pass (modulo infra flakes).
5. **CR discipline held across all 17 stories** of the arc — every story same-session adversarial CR, every M finding fixed before done, three of them (`sidecarWrittenSince`, SSE stamp, EpisodeList 10th value in sub-2-2b) being genuine correctness/contract catches.

## 3. Challenges

1. **Docker PR builds were never warm (diagnosed in-session, see AI-1).** Every PR build logs `failed to authorize: failed to fetch anonymous token` importing `ghcr.io/…:buildcache` — the GHCR login step is gated off for `pull_request` events, so every PR build runs fully cold on both arches. Cold builds usually squeeze in at 11–16 min; two slow-runner draws (#207 ×1, #209 ×2) blew the 30-min cap. The old quirk memory ("go.mod dep changes") was a wrong trigger model and has been corrected.
2. **gh active account flip-flops.** Externally switched to `alexyu-tvbs` twice mid-pipeline, producing misleading errors ("must be a collaborator" on PR create, 403 on rerun). Known parallel-session behavior (ux3-downloads-v2 retro called it expected), but the ship pipeline should defend at every mutation, not once at start (AI-2).
3. **Seam-reach authoring blind spot.** sub-3-1 was authored against the `RunTranscription` seam's *signature* without verifying its *data-layer reach* (movie-bound writer/reader) — discovered at implementation, forcing a mid-story scope narrowing, a CR AC amendment, and a follow-up story. Cheap to prevent at authoring time (AI-3).
4. **FE flake tax.** InstantSearchBar debounce flaked on CI (#208) and one local full-suite run flaked once — both cleared by burn-in as unrelated, but each cost a rerun cycle (AI-4).

## 4. Previous-Retro Follow-Through

Closest predecessor with action items: ux3-downloads-v2 retro (2026-07-03). Its gh-account observation ("expected, not an action item") is **revised** by this retro — two mid-pipeline breaks later, it graduates to AI-2. Epic-19/Epic-9-lineage process rules (Rule 20/24, regression gate, a11y gate) demonstrably held throughout the arc — the AC-drift/contract-stamp/checkbox-audit bindings appear in every story's Dev Agent Record.

## 5. Readiness Assessment

- **Testing & Quality:** ✅ All suites green (api + web 228/2547); every story's full-regression gate recorded.
- **Deployment:** ⚠️ **Shipped but not enabled.** Production NAS still runs `legacy` mode with no ASR key — the 68.3% recovery promise is not yet delivering user value. Alexyu ruled in-session: enable it. Steps + ASR-key acquisition guide handed off in-session (see §7 critical path).
- **Stakeholder acceptance:** N/A (single-user product; Alexyu is the stakeholder and participated).
- **Technical health:** Sound. Known carried debt is all tracked: `backlog-assrt-search-response-parsing` (manual-dialog data source), `backlog-compute-aware-asr-default`, `backlog-asr-runtime-key-resolution` (restart requirement — directly relevant to the enablement flow), episode glossary limitation (recorded in sub-3-2 AC #8).
- **Blockers:** None.

## 6. Action Items (all tracked in sprint-status.yaml — Agreement 4)

| ID | Priority | Owner lane | Summary |
| --- | --- | --- | --- |
| retro-m2-AI1-docker-pr-cache-auth | HIGH | QD | Remove the `if:` gate on the GHCR login step so PR builds authenticate and `cache-from` works — root cause diagnosed in-session; expected PR Docker time drops to minutes. Quirk memory corrected. |
| retro-m2-AI2-ship-gh-account-guard | MED | SM/QD | Ship skill: defensive `gh auth switch` before EVERY gh mutation, not once at pipeline start. |
| retro-m2-AI3-create-story-seam-reach-check | MED | SM | create-story template: stories riding an existing seam must record the seam's data-layer reach (which repos its writer/reader touch), not just its signature. |
| retro-m2-AI4-instantsearchbar-flake | LOW | TEA | Deflake the debounce spec (fake timers / retry-within-window) or record the exemption. |

## 7. Next Steps / Critical Path

1. **Enable the pipeline on the NAS** (Alexyu, guide handed off in-session): set `VIDO_SUBTITLE_PIPELINE_MODE=pipeline` + ASR key (+ verify Claude key), restart container, verify boot log, trigger a scan, watch the first `no_text_source` recoveries.
2. **retro-m2-AI1** before the next code PR if possible — it removes the recurring Docker rerun tax.
3. Next epic: **M3 (產品化 + Tier-2 本地)** per spec §8 — not yet planned in sprint-status. Prep inputs when planning starts: `backlog-compute-aware-asr-default` (hardware-representative benchmark or promote spec §6 compute-aware selection), `backlog-asr-runtime-key-resolution` (restart-free key rotation pairs naturally with 批次 UX), Assrt manual-dialog story.
4. **No significant-discovery epic-update alert**: M2's own spike already performed the plan correction this section exists to catch; M3's sketch in spec §8 remains valid.

## 8. Epic Closure

Per the Alexyu retro-gates-done ruling (2026-08-05): with this retrospective run, `epic-subtitle-pipeline-m1`, `epic-subtitle-pipeline-m1-5`, and `epic-subtitle-pipeline-m2` all transition to **done**; all three `*-retrospective` keys → **done** (M1's and M1.5's satisfied by this combined session). The M1 epic-line's "do not flip to done" guard note is retired by this document.
