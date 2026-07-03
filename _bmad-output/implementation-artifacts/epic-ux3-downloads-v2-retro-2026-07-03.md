# ux3-downloads-v2 Retrospective — Epic 14 v2 Downloads deep page

**Date:** 2026-07-03 · **Facilitator:** Bob (Scrum Master, party-mode) · **Project Lead:** Alexyu
**Epic:** `ux3-downloads-v2` (UX Redesign Phase 3, Epic 4 / delivers Epic 14 v2 core: P3-010 restyle + H-1 SSE + card actions) · **Status:** done

> ⚠️ No time estimates by design — AI-paced delivery.

## 1. Epic Summary

The Downloads deep page migrated to the v2 shell and gained the three capabilities the redesign brief (Part 4 D1) called out: **live progress without polling storms**, **per-download control** (pause/resume/remove), and **batch ops**, plus a dense desktop **Table view**. The whole chain landed behind the `new_shell_enabled` flag with the legacy render byte-unchanged.

### Delivery Metrics

- **8 tracked stories, all done:** design (ux3-4-1) → actions BE (ux3-4-2) → gated SSE broadcaster (ux3-4-2b) → FE (ux3-4-3, split 4-3a restyle / 4-3b actions+SSE+batch) → D7 Table view (ux3-4-4).
- **PRs merged this session:** #110–#116 (5 feature/story + 2 sprint-status chores). Design (#107) and actions BE (#109) landed just prior.
- **Tests:** BE `DownloadProgressBroadcaster` suite `-race`-clean; web suite grew ~2251 → **2289** passing; **7 E2E** in `downloads-v2.spec.ts`.
- **CI:** every PR green on the first pass — **self-heal never triggered, no `-linux` visual-baseline bootstrap needed**.
- **Quality gates all passed:** AC-Drift, Contract-Stamp, A11y pre-flight (jsx-a11y clean), Rule 21 (.pen headers), Rule 7 (BE error codes).

### Headline

A clean strangler migration end-to-end: design → two BE halves → FE list → FE table, zero regressions, one adversarial CR that caught a real cross-story contract ambiguity **before** it shipped to the consumer.

## 2. What Went Well

1. **Strangler / shell-gate held perfectly.** `staticData:{shell:'v2'}` + `useShellVersion()` kept the legacy render byte-for-byte under the flag OFF across every story. No regressions in 2289 web tests.
2. **The gated SSE broadcaster (ux3-4-2b) is the standout design.** One server-side qBittorrent poll fans out to all clients, **gated on `Hub.ClientCount()`** — zero watchers ⇒ zero qBT traffic. The FE then lazy-connects the `EventSource` only while the page is **visible** (§8), so tab-hide disconnects the client and the BE gate drops the server poll to zero. The pattern *removes* idle load rather than relocating it.
3. **Reuse discipline / no fork.** Extracted `DownloadRowActions` so the card and the table row share one action cluster + Radix confirm; one `EventSource`, one poll source; reused `Pagination`/`Button`/`Dialog` atoms and the `libraryStatus` TINT token map so the download status pill and the library badge read as one system.
4. **`[@contract-v1]` stamps did their job.** ux3-4-2b's adversarial CR (H1) surfaced that the `download_progress` payload diverges from `GET /downloads` in three ways (no `parse_status`; bare array vs paginated envelope; full unpaginated list) and wrote them into the contract — so ux3-4-3 correctly implemented merge-not-replace instead of wiping parse badges every ~2s.
5. **Gate-aligned split of a 9-AC story.** Splitting ux3-4-3 into 4-3a (GATE A restyle) / 4-3b (GATE B actions+SSE+batch) kept PRs reviewable; the D7 Table view was further pulled out as ux3-4-4.

## 3. Challenges & Recurring Patterns

### ① Cross-story AC ↔ shipped-capability mismatch (recurring Pattern ②/⑤ from prior retros)

ux3-4-3 **AC5** said "one request for many hashes", but ux3-4-2's slice-accepting methods live at the **qBittorrent Go-client layer** — the **HTTP API is single-hash only** (no batch route). Caught at 4-3b implementation; handled with N parallel single-hash requests (`Promise.allSettled`) + a documented deviation + Discovery Triage ④. Same class as epic-19's "contract text lags shipped behavior": an AC written against an *aspirational* capability that never reached the surface the consumer uses.

### ② Rule 21 .pen-header format is easy to get subtly wrong

Two lint failures on the accepted-form string (DownloadCardV2's first-pass "flow-d…" header; DownloadRowActions' custom "utility — shared…"). The accepted forms are strict; a near-miss fails the build.

### ③ Hook idioms vs lint rules (and a tolerated precedent that's actually wrong)

`useDownloadProgress`'s reconnect self-reference tripped `no-use-before-define`; the ref fix then tripped "no refs during render", requiring the latest-ref-in-effect pattern. Notably `useScanProgress.ts` — the precedent this hook mirrors — carries the same `useRef<…>()`-no-arg + self-reference anti-pattern as *tolerated pre-existing* lint; copying it naively would have shipped an error.

### ④ §8 lazy-SSE tension ("connect when active" vs "never mount-connect")

Genuinely ambiguous for a page whose very purpose is live progress. Resolved: expose `startTracking()` (the §8 trigger), call it from a **visibility-gated** effect (deps `[isVisible,…]`, never bare `[]`), and note that the §8 prohibition targets globally-mounted components — `DownloadsBrowseV2` is route-scoped.

### Operational notes (not defects)

- **gh active-account switching is EXPECTED, not a fault.** Alexyu runs parallel Claude Code sessions across personal (`j620656786206`) and work/TVBS (`alexyu-tvbs`) repos; the active gh account is global, so it flips between sessions. Per-folder git config already routes commits correctly. The right handling — already applied — is `gh auth switch --user j620656786206` before each PR/CI op on this repo. **No action item.**
- **Transient GitHub API outage** blocked one `gh pr create` briefly; a short reachability poll recovered it. Environmental, no action.
- **tsconfig.app.json surfaces ~40 pre-existing type errors** that CI tolerates (Vite strips types at build). The real gate is `nx build/lint/test` + `lint:all`, not `tsc --noEmit`.

## 4. Project Lead's Insight (Alexyu)

The epic went smoothly with no special concerns. The gh account switching is by-design (parallel personal/work sessions, global active account) and does not warrant remediation — the pre-PR switch is the correct handling.

## 5. Action Items

All tracked in `sprint-status.yaml` under "ux3-downloads-v2 Retro Action Items" (Agreement 4 — every item tracked regardless of priority). All **backlog** (the epic shipped clean; these are improvements, not blockers).

| ID | Item | Owner | Priority |
|----|------|-------|----------|
| `retro-ux3-4-ac-capability-check` | Process guard: when a FE story's AC references a BE capability, verify it exists on the **HTTP surface** (not just the Go client) before authoring the AC. Prevents the recurring Pattern ②. | SM | MED |
| `retro-ux3-4-batch-endpoint` | Optional BE: add `POST /downloads/batch/{pause,resume}` + batch `DELETE` if batch volume ever proves the N-request approach insufficient (Triage ④). | DEV | LOW |
| `retro-ux3-4-rule21-ergonomics` | DX: a Rule 21 header cheat-sheet (or an eslint auto-fix suggestion listing the accepted forms + how to look up the .pen node id). | SM | LOW |
| `retro-ux3-4-usescanprogress-ref` | Cleanup: fix `useScanProgress.ts`'s pre-existing `useRef<…>()`-no-arg + self-reference anti-pattern, aligning it with the `useDownloadProgress` fix. | DEV | LOW |

## 6. Readiness Assessment

- **Testing & quality:** ✅ BE `-race`-clean; web 2289 pass; 7 E2E; jsx-a11y clean; all gates green on first CI pass.
- **Deployment:** ✅ All merged to `main`; nothing pending.
- **Technical health:** ✅ Stable. No new tech debt beyond the documented, low-priority items above. Legacy render untouched (byte-identical under the flag).
- **Loose ends:** none — the D7 Table view that was deferred from ux3-4-3b is now shipped (ux3-4-4).

## 7. Next Epic

Per the ratified sequencing, **Epic 14 (this ux3-downloads-v2 chain) was built BEFORE Epic 13 (request-system)** precisely so that **Epic 13's `13-3a` request-status SSE can REUSE ux3-4-2b's gated-broadcaster pattern**. This epic therefore *produced the proven pattern the next epic depends on* — the hardest cross-epic dependency is already satisfied and battle-tested (gated poll, lifecycle, `[@contract-v1]` payload, lazy visibility-gated FE consumption). No epic-plan update required; Epic 13 can proceed when the lead chooses (13-0 design may already be in flight).

## 8. Key Takeaways

1. **Gate on connected clients, not just cadence.** `ClientCount()==0 ⇒ skip` (BE) + visibility-gated `startTracking()` (FE) turns "move polling to the server" into "remove polling when nobody's watching." Reuse this shape for Epic 13's request-status SSE.
2. **Stamp the contract's *shape deltas*, not just its existence.** The value of `[@contract-v1]` was catching the parse_status/envelope/pagination divergence before the consumer merged — make those deltas explicit, not implied.
3. **AC authored against capability, verified against the HTTP surface.** A FE AC that assumes a BE endpoint must be checked against what actually ships on the wire, not the internal client method.
4. **Mirror precedents, but don't inherit their tolerated sins.** `useScanProgress` was the right pattern to copy — except its lint anti-patterns, which the newer hook fixed properly.
