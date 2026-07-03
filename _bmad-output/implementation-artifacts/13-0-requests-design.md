# Story 13-0 — Requests design (`.pen` flow-l-requests-v2)

**Epic:** Epic 13 — Request System · **Status:** review (design complete — Sally review PASS 2026-07-04; PR pending merge)
**Owner:** ux-designer (Pencil MCP) · **Type:** design · **FRs:** P3-001 / P3-002 / P3-003 (the UI-bearing FRs) · **GATE A** for all Epic-13 frontend stories (13-1b / 13-2b / 13-3b)

> **⚠️ Flow-name correction:** the epic file said `flow-g-requests` — but **`flow-g` is already `flow-g-ai-subtitle`**. This design uses **`flow-l-requests-v2`** (next free letter L + v2-era suffix, matching `flow-k-activity-v2`). The `epic-13-request-system.md` reference is corrected to match.

## Story

As the design system,
I want the **Request System** UI drawn to v2 — the one-click 想要 button, the partial-request season/episode tree, and the request status list (per-flow recipe step 1),
so that dev builds Epic 13's frontend against a spec — lighting up the **Discover reserved Requests entry** (ux3-3-1 PH3-R2), defining the season/episode interaction that pins the 13-2 backend API shape, and rendering the request status pipeline (pending → searching → downloading → completed → failed) against the v2 design language.

## Context — net-new surfaces; the Requests landing lives in Discover

Epic 13 is the Overseerr/Jellyseerr replacement. Its UI has **three surfaces**, and the nav-IA ADR settles where the landing lives:

- **Request creation** — a one-click **想要** button on **探索 (Discover) result cards** and the **detail page** (movie/TV).
- **Requests landing / status list** — **lands in `discover.tsx`** (ADR `01-nav-ia-decision-adr.md:630`: "Epic 13 (Requests) → lands in `discover.tsx`, D3 boundary: discovery side"). It is reached by **lighting up the inert `想要清單·即將推出` entry** that **ux3-3-1 already reserved** in the Discover toolbar (the reserved-slot pattern, like Home's continue-watching). **It is NOT a new top-level destination.**
- **Partial request** — a **season/episode tree** on the TV request flow (from the detail page) — **the highest design-risk surface** (Winston, party-mode): the tree's granularity defines the 13-2a backend API.

This is a **net-new draw** (no v1 request flow exists), in a **new folder `flow-l-requests-v2`**. The request-button affordances are added in-context on the existing 探索 card (`flow-i-discover-v2`) and detail (`flow-b-detail-v2`) surfaces — referenced, not forked.

## Design scope — what to draw (in `ux-design.pen`, new folder `flow-l-requests-v2`)

Draw to v2 via Pencil MCP, reusing the shipped v2 shell + atoms (`01-design-language-v2.md` §5.1), token-only color, Noto Sans TC for all CJK, **JetBrains Mono for all numerics** (status counts, progress %, season/episode numbers), 44px touch floor, Base UI primitives for dialog/tree.

Frames to land (folder-scoped codes):

- **`L1-D-v2`** — **Requests landing / status list** (the Discover-hosted 想要清單): rows of requested titles, each with a **status token** (pending / searching / downloading / completed / failed — the `requests.status` enum) + Mono progress where `downloading`. Reached from the now-live Discover toolbar entry. This is the G-3 surface.
- **`L2-D-v2`** — **request-creation affordance**: the **想要** button in its three states on a 探索 card AND the detail page — `可請求` (not requested) / `已請求·處理中` (pending) / `已入庫` (already in library, no action). Never a broken/duplicate request.
- **`L3-D-v2`** — **partial request season/episode tree** (G-2, highest risk): whole-series vs specific-seasons vs individual-episodes granularity; reflect **already-owned / already-requested** episodes; a confirm action. This frame **defines the 13-2a API shape** — draw it precisely.
- **`L4-M-v2`** — **mobile**: request button + requests list condensed; 44px targets; reachable on mobile.
- **`L8-D-v2`** — **request-submitted feedback** (toast / inline confirmation after 想要 is pressed).

Four-state standard (N4 — `01-design-language-v2.md` §7; draw ALL or it doesn't ship):

- **`L5-D-v2`** — **loading skeleton** (list-shaped; reduced-motion respected).
- **`L6-D-v2`** — **empty: no requests** (distinct from a failure) — a calm `尚無請求` + a quiet `前往探索` affordance, never a bare blank.
- **`L7-D-v2`** — **per-section fail-soft**: the fulfilment/status source (*arr / request pipeline) unavailable → inline `無法載入請求狀態` + `重試`; the shell + nav still render; page never hard-fails.

A reusable `Component/RequestRow-v2` (title + status token + progress + action) should be added — it is the repeated unit of L1.

## Key design decisions (resolved — flag the capability honor)

1. **Requests landing in Discover, not a new destination (ADR).** Light up the **ux3-3-1 reserved `想要清單` entry** (currently inert `想要清單·即將推出`) → live; render the list as a Discover-hosted view. Do **not** add a 6th sidebar destination. *(Optional, deferred: an Activity summary-row mirroring the downloads pattern — note only, not drawn this story.)*
2. **Status vocabulary = the `requests.status` enum**, mapped through **DL-v2 §2.5 status→token**, aligned with the N1 lifecycle moat (`想要 → 下載中 x% → 整理中 → 已入庫`). Reuse the SAME token mapping as the poster badge (ux3-0-2) and downloads (ux3-4-1) — no bespoke request palette.
3. **⚠️ Capability-honor (Rule 24) — draw target state, FE consumption is GATE-B.** The status pipeline (13-3a) + *arr fulfilment (13-4) are **net-new backend, not built**. Draw all five statuses as the **target**, but the design records that the FE (13-3b) is **GATE-B on the BE merge**. Draw **only** the five enum statuses — invent no extra states. The request button's `pending` state likewise depends on 13-1a's create endpoint.
4. **Partial-request tree defines the API (G-2).** Because 13-2a's `requests.seasons/episodes` shape follows this interaction, draw the granularity unambiguously (whole / season-set / episode-set) and the already-owned/requested reflection. This is the one frame to over-invest in.
5. **Request button = three honest states**, never a duplicate-request or a broken affordance; already-in-library shows `已入庫` (no action).
6. **Shell/atom reuse, no fork.** v2 atoms per DL-v2 §5.1; Noto Sans TC labels, status tokens, 44px, `focus-ring`; the 探索/detail entry points reuse the existing card/detail components.

## Acceptance Criteria

1. **Given** the v2 Design Language, **when** Requests is drawn, **then** all frames land in a **new `flow-l-requests-v2`** folder (NOT `flow-g`, which is ai-subtitle); the existing `flow-i-discover-v2` / `flow-b-detail-v2` are touched **only** for the in-context request-button affordance (no fork, no unrelated mutation).
2. **Given** Epic 13's UI scope, **then** the design covers: the **想要 button** (3 states, card + detail), the **partial-request season/episode tree** (G-2), the **requests landing/status list** (5-status rows, G-3), and **request-submitted feedback**.
3. **Given** N4, **then** all four states are drawn: default (`L1`), loading skeleton (`L5`), **empty-no-requests distinct-from-failure** (`L6`), and **status-source fail-soft** (`L7`).
4. **Given** the nav-IA ADR, **then** the Requests landing **lives in Discover** (lights up the ux3-3-1 reserved `想要清單` entry) and is **NOT** a new top-level destination.
5. **Given** Rule-24 capability-honor, **then** the design draws **only** the five `requests.status` enum values (pending/searching/downloading/completed/failed) via DL-v2 §2.5 tokens, records that FE consumption is **GATE-B on the 13-3/13-4 backend**, and invents no statuses.
6. **Given** mobile IA, **then** a mobile frame (`L4-M-v2`) covers the request button + list with 44px targets.
7. **Given** v2 enforcement, **then** color is token-only (no hex), all CJK is Noto Sans TC (TY-1), all numerics are JetBrains Mono, colored body text uses `*-text` AA variants (TC-2), `text-disabled` carries no load-bearing text (TC-1).
8. **Given** the UX screenshots workflow (CLAUDE.md), **then** `scripts/export-pen-screenshots.py` `SCREENS` dict is extended with every new `flow-l-requests-v2` node ID → code, screenshots are regenerated, and **only genuinely-changed PNGs** are committed alongside the `.pen`.

## Tasks / Subtasks (designer)

- [x] (AC #1) Pencil MCP: `get_editor_state(include_schema:true)`; confirm v2 shell + atoms; create `flow-l-requests-v2` frames.
- [x] (AC #2, #4) Draw `L1-D-v2` (requests list, 5-status rows) + light up the ux3-3-1 reserved `想要清單` entry; `L2-D-v2` (想要 button, 3 states, card + detail); `L8-D-v2` (submitted feedback). Spec `Component/RequestRow-v2`.
- [x] (AC #2, #5) Draw `L3-D-v2` partial-request season/episode tree (granularity + already-owned/requested reflection — defines 13-2a API).
- [x] (AC #3) Draw `L5` skeleton, `L6` empty-no-requests, `L7` status-source fail-soft.
- [x] (AC #6) Draw `L4-M-v2` mobile (button + list, 44px).
- [x] (AC #7) Token-lint pass: no literals, Noto Sans TC CJK, Mono numerics, AA color rules.
- [x] (AC #8) Update `SCREENS` dict; `python3 scripts/export-pen-screenshots.py`; commit `.pen` + only-changed PNGs (`feat(13-0): Requests v2 design (.pen flow-l-requests-v2)`).

## Dev Notes

- **Design Language v2:** `_bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md` — tokens §2 (status→token §2.5 — reuse for request status), type §3 (Mono numerics), atoms §5.1, four-state §7, a11y §8.
- **Nav/IA ADR:** `…/01-nav-ia-decision-adr.md:630` — Requests lands in `discover.tsx` (D3 discovery side); not a new destination.
- **Reserved-entry precedent (light this up):** `ux3-3-1-discover-design.md` §"Epic 13 Requests reservation (PH3-R2)" — the inert `想要清單·即將推出` toolbar entry; `ux3-1-3-continue-watching-slot.md` — the inert→live reserved-slot pattern.
- **Epic source:** `epics/epic-13-request-system.md` — the artery (13-0 design → 13-1 → 13-4 → 13-3 → 13-2 → 13-5), the `requests.status` enum, and the capability findings (built-in path can't fulfil; *arr is the engine).
- **Entry-point surfaces (in-context):** `flow-i-discover-v2` (探索 cards) + `flow-b-detail-v2` (detail page) — add the 想要 button affordance, do not fork.
- **Memory:** _design-must-conform-to-current-design-system_; _pencil-mcp-edits-need-manual-save_ (Cmd+S before screenshot regen).

### Project Structure Notes

- Design-only: edits `ux-design.pen` + `_bmad-output/screenshots/flow-l-requests-v2/` + `scripts/export-pen-screenshots.py` (`SCREENS`). No app code; no cross-stack split (design story).

### Time-dependent visual coverage

- N/A — design-only story; adds/modifies no `apps/web/src/components/**/*.{ts,tsx}`. (Rule 23 applies to the downstream FE stories — request "requested 3h ago" relative times, if any, are evaluated there.)

### Discovery Triage

- **Carried from the Epic 13 breakdown:**
  - **② GATE-B (already filed)** — the status pipeline (13-3a) + *arr fulfilment (13-4) are net-new backend; the FE stories (13-3b etc.) are blocked on them, NOT this design. This design draws the target state.
  - **③** — optional Activity summary-row for requests (mirroring downloads): noted, deferred; file a `backlog` note if wanted later (not drawn this story).
  - **Doc-fix** — `epics/epic-13-request-system.md` + `sprint-status.yaml` `flow-g-requests` → `flow-l-requests-v2` (g collides with ai-subtitle); corrected alongside this story.
- Reference: `project-context.md` Rule 24.

### References

- [Source: _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md#§2-§8]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md#L630 (Requests in discover.tsx)]
- [Source: _bmad-output/implementation-artifacts/ux3-3-1-discover-design.md#Epic-13-Requests-reservation]
- [Source: _bmad-output/planning-artifacts/epics/epic-13-request-system.md]

## Change Log

| Date       | Change                                                                                                                              |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------- |
| 2026-07-01 | Story created (SM create-story). Design-only, GATE-A for Epic-13 FE. Corrected flow name `flow-g`→`flow-l-requests-v2` (g = ai-subtitle). Requests landing in Discover (ADR), lights up ux3-3-1 reserved entry. Capability-honor: draws the 5-status target; FE consumption GATE-B on 13-3/13-4 BE. Status → ready-for-dev (design not started). |
| 2026-07-04 | Design drawn (Pencil Inline Agent, prompt by Sally) + Sally MCP review **PASS**. L1–L8 all land; `Component/RequestRow-v2` created + registered in Component Library. Review fixes: 44px 想要清單 entry ×4, `107 分` Mono/Noto split ×3 (→ new **Rule TY-3** in DL-v2 §3.1), GATE-B capability-honor note on canvas. **Decision: `searching`→`warning-tint`/「搜尋中」** (transient-work family) — added to DL-v2 §2.5 + new §8 pipeline section in the `.pen` DL-v2 reference frame. Checkbox `Indeterminate`/`DisabledChecked`/`DisabledEmpty` componentized (L3 swapped to instances). **Scope decision:** `flow-i`/`flow-b` deliberately NOT mutated — flow-b v2 frames depict in-library titles (想要 would contradict `已入庫`), flow-i has no hover frame; L2's context excerpts carry the affordance spec (AC #1 "touched only for" = upper bound). flow-i reserved entries verified untouched/inert. CLAUDE.md flow list A–L. Found (deferred to chore): `.pen` `text-muted` still `#808080` vs §2.2 `#A0AABE`; `DownloadCard-v2` unregistered in library. Status → review. |
