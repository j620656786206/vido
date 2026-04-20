# Winston (Architect) Follow-up Prompt — retro-10-AI3 Rule 7 Expansion

**Handoff from:** Amelia (Dev) via `/bmad:bmm:workflows:code-review` in-flow auto-fix
**Date:** 2026-04-20
**Priority:** MEDIUM — architectural hygiene (not blocking any current work)

Copy the block below into a fresh Claude Code session and prefix with `/bmad:bmm:agents:architect`:

---

## Prompt for Winston

Hi Winston. I need your architectural review on a Rule 7 (Error Codes System) expansion that Amelia (Dev) applied in-flow during the `retro-10-AI3-rule7-wire-format-cr-check` code-review today (2026-04-20). Amelia has flagged this for your formal review because the edit touched `project-context.md` — your domain — rather than staying inside `_bmad/` workflow files.

### Context

During adversarial self-CR of retro-10-AI3 (the story that installs the "Rule 7 Wire Format Check" into `code-review/instructions.xml`), the meta-test of the newly installed grep against `apps/api/internal/` revealed that **4 production error-code prefixes were not documented in Rule 7**. All four are actively used in shipped code:

| Prefix | Production Sites | Files |
|--------|-----------------|-------|
| `QB_` | 5 codes | `apps/api/internal/qbittorrent/torrent.go`, `apps/api/internal/qbittorrent/types.go` |
| `METADATA_` | 12 codes (7 unique, 5 duplicated across packages) | `apps/api/internal/retry/metadata_integration.go`, `apps/api/internal/metadata/provider.go` |
| `DOUBAN_` | 5 codes | `apps/api/internal/douban/types.go` |
| `WIKIPEDIA_` | 6 codes | `apps/api/internal/wikipedia/types.go` |

Rule 7 previously listed 9 sources (`TMDB_`, `AI_`, `DB_`, `VALIDATION_`, `SUBTITLE_`, `PLUGIN_`, `SCANNER_`, `SSE_`, `LIBRARY_`). The 4 new prefixes were never formalized — they accrued organically as new subsystems shipped.

### What Amelia applied in-flow (needs your architectural sign-off)

**File 1: `project-context.md`**
- Line 7 — `Last Updated` header bumped to 2026-04-20 with note citing retro-10-AI3
- Lines 279-298 — Rule 7 code-fenced block extended with 4 new prefix rows (example codes drawn from actual production constants)
- Line 300 — New paragraph: `**Authoritative prefix set (13 sources):** ...` with sync-discipline instructions

**File 2: `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml`**
- Lines 111-114 — Inline prefix list in the CR check extended to 13 sources, marker "(13 sources total — keep in sync with project-context.md Rule 7 list)" added
- Lines 138-155 — Auto-fix prefix map expanded with `qbittorrent/**`, `douban/**`, `wikipedia/**`, and `metadata/** or retry/**` entries

### What I need from you

Please make an **architectural judgment** on these four items and tell me which (if any) need revision:

1. **Prefix granularity.** Is it correct that `retry/metadata_integration.go` shares the `METADATA_` prefix with `metadata/provider.go`? There are duplicated code names between the two packages (e.g., `METADATA_TIMEOUT`, `METADATA_RATE_LIMITED`, `METADATA_CIRCUIT_OPEN` appear in both). This is a **Rule 11 Interface Location** smell — one of these should probably be the canonical definition and the other should import it. Out of scope for retro-10-AI3 but worth your eyes.

2. **Scraper cluster vs external-API cluster.** `DOUBAN_` and `WIKIPEDIA_` are both "external data source" prefixes similar in shape to `TMDB_`. Should they have been grouped under a common `EXTERNAL_SOURCE_` or `SCRAPER_` umbrella per Rule 7's `{SOURCE}_{ERROR_TYPE}` convention? Or is the per-source granularity correct (which is what ships today)? I went with per-source to match ship reality, but you may prefer consolidation.

3. **`QB_` naming.** The prefix is unexpectedly shortened (unlike `TMDB_` which is the full abbreviation). Should this be `QBITTORRENT_` to match the package name `qbittorrent`? Renaming would touch `apps/api/internal/qbittorrent/{torrent.go,types.go}` + any test assertions + potentially Swagger annotations. If we ever add Transmission or Deluge downloaders, `QB_` suddenly looks parochial.

4. **Authoritative-list sync discipline.** I added a new paragraph after Rule 7 stating the sync rule: when Rule 7 changes, also update `instructions.xml`. Is this location correct? Or should the rule live in a dedicated `architecture/adr-error-code-prefix-registry.md` ADR with a link from Rule 7? (This is effectively asking: is `project-context.md` the right surface for "prefix governance" meta-rules, or do those belong in `architecture/`?)

### Deliverable

Please produce **one of**:

- **Approval** — confirm the 4 prefixes are correctly added, the sync paragraph belongs where it sits, and close the loop. A short comment in this file's changelog suffices; no further edits needed.
- **Revisions** — specify which prefix names, groupings, or ADR locations to change. If a rename affects Go source code (e.g., `QB_` → `QBITTORRENT_`), create a follow-up story under `_bmad-output/implementation-artifacts/` with clear scope and target files. Don't edit the Go code directly; that's a Dev story.
- **ADR creation** — if you decide the prefix governance deserves a dedicated architecture decision record, draft `_bmad-output/planning-artifacts/architecture/adr-error-code-prefix-registry.md` and link it from Rule 7 with a one-line pointer.

Supporting references:
- CR findings summary: `_bmad-output/implementation-artifacts/retro-10-AI3-rule7-wire-format-cr-check.md` → Completion Notes → H1 entry
- Sprint status entry: `_bmad-output/implementation-artifacts/sprint-status.yaml` line 444 (`retro-10-AI3-rule7-wire-format-cr-check: done`)
- Precedent (Rule 7 origin): `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md` Challenges § Pattern #3

Take as long as you need. No timeboxing — this is hygiene, not delivery blocker.

— Amelia

---

## Architect Resolution (Winston) — 2026-04-20

**Outcome:** Hybrid — 2 approvals + 2 backlog follow-ups + no ADR.

| Item | Verdict | Action |
|------|---------|--------|
| 1. METADATA_ Rule 11 smell | **Confirmed — worse than described** | Filed `followup-metadata-prefix-dedup.md` (MEDIUM). `retry/metadata_integration.go` not only mirrors 5 codes but introduces 4 retry-only wire codes (`METADATA_GATEWAY_ERROR`, `METADATA_NETWORK_ERROR`, `METADATA_NOT_FOUND`, `METADATA_UNKNOWN_ERROR`) that silently expand the wire contract beyond `provider.go`. Canonicalize provider.go; retry imports. |
| 2. DOUBAN_/WIKIPEDIA_ per-source | **Approved as shipped** | No change. Per-source preserves debug/routing signal; matches TMDB_ precedent. Rule 7's SOURCE = data source, not category. |
| 3. `QB_` naming | **Revision recommended** | Filed `followup-qbittorrent-prefix-rename.md` (LOW). `QB_` is the only prefix breaking the `SOURCE = uppercase(package name)` convention used by all 12 other prefixes. Rename to `QBITTORRENT_` (~16 call-site files). |
| 4. Sync-discipline location | **Approved as placed** | No ADR. `project-context.md` is correct — Agreement 3 bible loads in every agent context; ADRs are for decisions-with-alternatives, not process rules. At 13 prefixes, inline is readable. Revisit if registry grows to 20+. |

**Files created:**
- `_bmad-output/implementation-artifacts/followup-metadata-prefix-dedup.md`
- `_bmad-output/implementation-artifacts/followup-qbittorrent-prefix-rename.md`

**Files edited:**
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — two new `backlog` entries after AI5
- `_bmad-output/implementation-artifacts/retro-10-AI3-rule7-wire-format-cr-check.md` — Change Log appended with architect sign-off line

**Not changed (intentional):**
- `project-context.md` — Rule 7 as-shipped is ratified (all 4 prefixes correct, sync paragraph in right place)
- `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` — as-shipped is ratified
- No `architecture/adr-error-code-prefix-registry.md` created — not ADR-worthy at current scale

Close the loop: retro-10-AI3 status stays `done`; the two follow-ups are fresh backlog items Bob (SM) will schedule when convenient.

— Winston
