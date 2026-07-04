# Vido Project Context - AI Agent Quick Reference

> **Purpose:** Mandatory reading for all AI agents before implementing ANY code. This document ensures consistency across all implementations.

**Full Documentation:** See `_bmad-output/planning-artifacts/architecture/index.md` for complete architectural decisions and patterns (sharded into ~20 focused files).

**Last Updated:** 2026-07-04 (story 13-4a — Epic 13 `*arr` DVR plugin part 1 (G-4/P3-004, artery #2): greenfield `internal/plugins/` — `DVRPlugin`/`PluginConfig`/`QueueItem`/`AddOptions` ([@contract-v1] AC #1 — consumers 13-4b/13-3a/13-2a) + typed `PluginError`; Radarr API-v3 client (`X-Api-Key`, single reused 10s http.Client, 10 req/s burst-10 limiter with `Wait(ctx)` first line per Rule 27 ①, paginated queue normalized to `QueueItem` (movieId→ExternalID, downloadId→DownloadID for the 13-3a qBT join), quality-profile/root-folder extras behind new `plugins.ProfileLister` — NOT on the stamped DVRPlugin interface); `plugins.Manager` (fingerprint-cached clients `URL|APIKey` per Rule 14, config via settings table + secretsService following the qBittorrent precedent — ZERO migrations by ruling, self-contained 60s health scheduler with immediate startup sweep using the retry/scheduler.go lifecycle, healthy/unhealthy/unconfigured transitions written to `connection_history`; `ServiceNameRadarr`/`ServiceNameSonarr` constants + `ValidServiceNames` extended so `/health/services/:service/history` works; the hardcoded 5-service `ServicesHealth` left UNTOUCHED per ruling — plugin health surfaces via the settings GET); settings triad `GET/PUT /settings/{plugin}` + `POST /settings/{plugin}/test` + quality-profiles/root-folders passthrough, parameterized per plugin so 13-4b sonarr adds one registration string ([@contract-v1] AC #4 — consumer 13-6), with the NEW server-side test-before-save guard (PUT runs TestConnection INSIDE SaveConfig and refuses persistence 409 `DVR_TEST_FAILED`; disabled-save deliberately skips the probe — turning a plugin off must not require a reachable server; empty body `api_key` keeps the stored key); movie fulfilment-on-create ([@contract-v1] AC #6 — consumers 13-3a/13-5): optional nil-safe `FulfilmentService` dep on `RequestService` (13-1a constructor untouched) — pending→searching + `external_id` + `fulfilment_source='arr'` on synchronous AddMovie(SearchNow) success with the 201 carrying the transition (a value inside the 13-1a 5-value enum — no bump; ack recorded), EVERY failure path stays pending with a zh-TW `error_message` + slog per Rule 13 (graceful 201, fulfilment is best-effort), tv rows stay pending until 13-4b (Sonarr), stranded-pending retry EXPLICITLY handed to 13-3a's reconcile loop; NEW repo method `RequestRepository.UpdateFulfilment` (success transition + degradation annotation, zero-rows-affected maps to `ErrRequestNotFound`). Rule 7 prefix set extended 14→15: `DVR_` (7 codes: `NOT_CONFIGURED/CONNECTION_FAILED/AUTH_FAILED/TIMEOUT/ADD_FAILED/TEST_FAILED/NOT_SUPPORTED`) + first live use of reserved `PLUGIN_INIT_FAILED`/`PLUGIN_HEALTH_CHECK_FAILED` in the manager — code-review/instructions.xml Step 3 prefix list synced 2026-07-04 (reconciles 13-1a's "13-4a adds `DVR_` as the 15th" note). acks 13-1a [@contract-v1] AC #2/#3 — request resource shape untouched. sprint-status.yaml 13-4a-arr-dvr-plugin → review.). Prior: 2026-07-04 (story 13-1a — Epic 13 (Request System) backend foundation: migration 027 `requests` table (5-status CHECK enum pending|searching|downloading|completed|failed = the single source of truth the pipeline (13-3a) and FE render against; partial unique index on (`tmdb_id`, `media_type`) WHERE status active = DB-level duplicate-request guard; `media_type` CHECK 'movie'|'tv' — TMDB/FE vocabulary, deliberately NOT the media-libraries 'series') + `POST/GET /api/v1/requests` + RequestRepository/RequestService/RequestHandler (filter-presets template chain; server-side zh-TW title resolve via the existing Epic-2 TMDbService — Rule 27 by reuse, zero new client/limiter/key; owned-guard via `FindOwnedTMDbIDs` bulk, no error-string matching). Rule 7 prefix set extended 13→14: `REQUEST_` (`REQUEST_DUPLICATE`, `REQUEST_ALREADY_IN_LIBRARY`, both 409) — code-review/instructions.xml Step 3 prefix list + sync date updated 2026-07-04 (13-4a will add `DVR_` as the 15th and reconciles the count at its merge). [@contract-v1] stamped on story AC #2/#3 (request resource JSON shape, create/list) — consumers 13-1b/13-2a/13-3a/13-3b. Capability-honor: rows are born `pending`, NO fulfilment (13-4), NO transitions/SSE (13-3a), seasons/episodes columns reserved NULL (13-2a). sprint-status.yaml 13-1a-one-click-request → review.). Prior: 2026-06-14 (story-20-3 — epic-20 (Test Quality Hardening) process-hardening: extended **Rule 15** DB Column Sync — the repo SELECT column list AND row scan must sync when a column is added/used (a read path drifting from the schema silently returns the zero value), and real DB reads need an integration test not just a mocked-repo unit test; reinforced **Rule 24** with a superseded-mechanism corollary — a migration introducing a REPLACEMENT (new table/column/service) must triage the old mechanism's retirement + re-pointing every reader AT THAT MOMENT, never dual-living. Origin: bugfix-20-1 — the `seasons` table (mig 015) superseded the `series.seasons` JSON column (mig 006) but GetSeasons was never re-pointed and seriesSelectColumns never SELECT-ed the column, so the season accordion was always empty; caught by the Phase-2 first real-data test (the unit test mocked the repo, hiding it). NO CR-workflow sync — Rule 15/24 are dev-discipline rules, not grep-checked in code-review/instructions.xml (unlike Rules 7/20/25). Pure docs — 0 Go, 0 FE, 0 ESLint, 0 baseline. sprint-status.yaml story-20-3-process-hardening → done; bugfix-20-1-season-summary-read → done (#72)). Prior: 2026-06-10 (retro-11-AI3 — epic-11 retro action item (MED, prep): ARCH Winston authored **Rule 27 (External Integration Standard — the Five Pillars)** in this doc (inserted after Rule 26, before "Known dev-mode artifacts") + the backing ADR `_bmad-output/planning-artifacts/architecture/adr-external-api-integration-standard.md` (Status ACCEPTED). Aligns Epic 12 F-3..F-6 (recommendations / watch providers / trailers / Douban) on ONE external-integration shape before F-3 begins, so the four stories converge instead of re-inventing rate-limit / caching / graceful-degradation / key-mgmt four times. The Five Pillars every third-party network path MUST provide: ① rate limit (one `*rate.Limiter` per upstream, built once + reused per Rule 14, `Wait(ctx)` first line, limit = published ceiling — e.g. tmdb 40/10s) · ② cache (tiered AD #4, key `{source}:{type}:{id}:{version}`, checked BEFORE the limiter so a hit never waits — keeps detail page < 1.5s warm) · ③ degrade (external data is enrichment not core content → fail-soft per-section, never fail-page; bounded exp-backoff 1s→2s→4s→8s; stale-on-error; scrapers reuse the existing client's robots.txt / UA-rotation / `enabled` switch) · ④ error codes (Rule 7 `{SOURCE}_{ERROR_TYPE}`, REUSE existing prefixes — F-3/F-4 `TMDB_*`, F-6 `DOUBAN_*`, NO new prefix for Epic 12) · ⑤ keys (via `ClientConfig` from settings/env, log only through slog `sanitizeAttr`, NO new secret). KEY RULINGS: reuse-over-reinvent — three of four integrations ride the already-shared `internal/tmdb` client (F-3/F-4 add endpoint wrappers; F-5's `GetMovieVideos`/`GetTVShowVideos` already exist), F-6 routes through existing `internal/douban`; **F-5 YouTube = client-side `youtube-nocookie.com/embed/{key}` iframe using the key TMDB already returns — NO backend YouTube call, NO YouTube Data API key, NO quota** (ADR Decision 4); NO shared `internal/externalapi` package now — codify the convention only, defer extraction until a third client re-hand-rolls the triplet (YAGNI, ADR Decision 3, mirrors 11-1's dead-code-deletion culture); F-4 re-adds the `GetWatchProviders` that 11-1 removed as dead code now that it has a real consumer. Does NOT block F-1/F-2 (read existing data). Pure docs + ADR — 0 Go, 0 FE, 0 ESLint, 0 baseline; no CR-workflow sync (self-contained convention, no per-story contract). sprint-status.yaml `retro-11-AI3-epic12-external-api-arch` → done.). Prior: 2026-06-09 (retro-11-AI2 — epic-11 retro action item (MED): SM Bob + DEV authored **Rule 26 (TanStack Router Search-Param Coercion)** in this doc (inserted after Rule 25, before "Known dev-mode artifacts") — TanStack Router's default search parser JSON-parses each query value, so a lone-numeric param (`?genre=16`, `?platform=8`) arrives as a `number` and a `typeof x === 'string'` `validateSearch` guard is false → the param is SILENTLY DROPPED on single-value deep links (a multi-value `?genre=16,28` stays a string and slips through, so it evades casual manual testing). Fix: coerce number → string up front via the canonical `apps/web/src/routes/discover.tsx` `toCsvString` helper (or defensive `String()`); genuinely-numeric params use a `toOptionalNumber`-style coercion, and a string-enum guard (`subtitleStatus`) is only safe when the value can NEVER be all-digits. Precedents: story 11-2 (`?genre=16`/`?platform=8` deep links dropped → `toCsvString()` + `String()`, E2E-guarded, CR caught it HIGH) + story 8-11 (`subtitleStatus`, same class) = two strikes across two epics → codified per Epic 11 Retro Insight 3 / Rule 22. Scoped to a doc note (no ESLint rule — false-positive-prone on string enums; no CR-workflow sync — Rule 26 is a self-contained frontend convention, not a cross-story contract). Pure docs — 0 Go, 0 FE, 0 ESLint, 0 baseline. sprint-status.yaml `retro-11-AI2-tanstack-numeric-param-note` → done.). Prior: 2026-06-02 (retro-19-P4 — epic-19 retro action item (LOW): SM Bob + CR authored **Rule 25 (Mega-line Rebase Conflict Resolution)** in this doc (inserted after Rule 24, before "Known dev-mode artifacts") governing how to resolve a git rebase/merge conflict on THIS "Last Updated" mega-line. THE BAN: never resolve by whole-side takeover (`git checkout --ours`/`--theirs`, editor "Accept Current"/"Accept Incoming", or deleting either `<<<<<<<`/`=======`/`>>>>>>>` half wholesale) — every branch PREPENDS its own entry, so taking one side silently DROPS the other side's entry. THE RULE: the resolution MUST be a UNION of both sides' entries, newest-first — the branch rebased on top keeps its entry as the `Last Updated:` lead, the already-landed branch's former lead is DEMOTED to a `Prior:` entry, and the shared older `Prior:`/`Earlier:` tail is kept EXACTLY ONCE. VERIFICATION: post-merge entry count MUST be ≥ max(both sides) and a grep confirms BOTH conflicting story/retro IDs survive; re-run `pnpm exec prettier --check project-context.md` (entries stay English-only — a CJK char makes prettier reflow the whole mega-line and masks a dropped entry). Origin: 19-8's "Last Updated" entry was silently dropped during a 2026-05-28 rebase-conflict resolution (whole-side takeover), caught only by a later adversarial CR (H1). CR SYNC: `code-review/instructions.xml` Step 3 gained a MANDATORY "RULE 25 MEGA-LINE MERGE CHECK" action (scope-filtered on a mega-line edit behind a rebase/merge; HIGH finding if any entry ID present at the merge base is missing post-merge; `{{megaline_check_result}}` binding surfaced in the Step 4 findings summary). Pure process rule + CR-workflow sync — 0 Go, 0 FE, 0 ESLint, 0 baseline. sprint-status.yaml `retro-19-P4-pen-context-megaline-rebase-guard` set to done.). Prior: 2026-06-02 (retro-19-P3 — epic-19 retro action item: QD CI gate-integrity audit. Audited ALL GitHub Actions paths / dorny filters (only `visual-regression.yml` + `bisect-regression.yml` use them; `test.yml` / `docker.yml` / `testsprite-monthly.yml` / `testsprite-quota-warning.yml` have none). Dead-path scan = 0 hits: every path listed in both filters verified present on disk. Missing-real-surface fix: added 3 missed rendering surfaces to `visual-regression.yml` `visual:` filter — `pnpm-lock.yaml` (the known gap; filter watched `package.json` but a transitive / lockfile-only bump silently skipped the visual PR gate; bugfix-19-4b-1-followup CR M1, carry-forward to P3; `bisect-regression.yml` already had it), `apps/web/postcss.config.js` (the @tailwindcss/postcss + autoprefixer transform pipeline that drives the already-watched `tailwind.config.js`), `apps/web/vite.config.mts` (the `nx serve web` dev-server config the visual diff renders against). All 3 are pure widening (gate fires on more PRs, never fewer) so they follow the always-trigger-refactor precedent of widening-not-narrowing needing no Rule 20 contract bump (19-5 AC #1 stays at contract-v2; downstream-ack grep for 19-5 AC #1 returns 0 consumers, so even the strict path records no downstream consumers). Skip-notice echo block synced. `bisect-regression.yml` filter left intentionally narrow per its documented parse-subsystem-only scope. Rule 20 grep-helper backtick-tolerance (this doc): the two confirmed-against helpers now place an optional-backtick token before the bracket so a token-wrapped ack written with inline-code backticks is no longer invisible to the bump-side stale-mark grep — the exact silent-strand class the bump obligation guards; new backtick-tolerance note added; bare-token helpers unchanged (already tolerant). Origin: bugfix-19-4b-1-followup CR H1 (backtick-wrapped ack defeated grep). CR SYNC: `code-review/instructions.xml` Step 3's embedded copy of the same confirmed-against grep got the identical optional-backtick fix (mirrors retro-19-P2's CR-sync pattern). Pure CI-filter + process + docs — 0 Go, 0 FE, 0 ESLint, 0 baseline regeneration. sprint-status.yaml `retro-19-P3-ci-gate-integrity-audit` set to done.). Prior: 2026-06-02 (retro-19-P2 — epic-19 retro action item: ARCH Winston extended **Rule 20 (AC Contract Versioning)** with a 🔁 **bump → downstream stale-mark** sub-section (PRODUCER-side obligation). The existing ack rule is consumer-side and snapshots at the downstream story's authoring time — it cannot protect a downstream DRAFT orphaned when its upstream bumps `[@contract-vN]` → `[@contract-v(N+1)]` after the draft was written. New obligation: the bump-author MUST, in the SAME change, (1) grep every downstream consumer that acked the bumped AC (`confirmed against [@contract-vN]` + upstream story-id + AC#), (2) for each NOT-done consumer (backlog/ready-for-dev/in-progress/review) stale-mark it with `⚠️ STALE [@contract-vN→v(N+1)]: …` in BOTH its Dev Notes AND its sprint-status.yaml entry; done consumers are FROZEN (forward-only), (3) record `no downstream consumers` in the Change Log row if the grep is empty. New ❌ anti-pattern (Change-Log row alone is insufficient) + a 4th 🔎 grep helper + a retro-19-P2 Precedent block added. CR SYNC: code-review/instructions.xml Step 3 gained a MANDATORY "RULE 20 CONTRACT BUMP STALE-MARK CHECK" action (mirrors the Rule 7 wire-format check — scope-filtered on a `[@contract-vN→v(N+1)]` bump token, HIGH finding for any not-done consumer missing a stale-mark, `{{contract_bump_check_result}}` binding surfaced in the Step 4 findings summary). Origin: 19-6 `[@contract-v1→v2]` bump silently stranded 19-7's already-drafted story (acked 19-6 v1 AC #2/#4/#5/#6) → surfaced only at 19-7 impl → full pre-impl re-architecture via correct-course (sprint-change-proposal-2026-05-20.md). Pure process rule + CR-workflow sync — 0 Go, 0 FE, 0 ESLint, 0 baseline. sprint-status.yaml `retro-19-P2-contract-bump-downstream-notify` → done.). Prior: 2026-06-02 (retro-19-P1 — epic-19 retro action item: ARCH Winston authored **Rule 24 (Discovery Triage)** in this doc, sibling to Rule 21/22/23 (inserted after Rule 23, before "Known dev-mode artifacts"). Rule 24 forces EXACTLY-ONE-OF-THREE classification at the MOMENT a story discovers out-of-scope work — ① expand-scope-in-place (absorb into current story, MUST add an AC/sub-task so the absorbed work is itself tracked) / ② spawn-blocking-story (new `sprint-status.yaml` entry, current story marked `blocked`, sequenced ahead) / ③ backlog-with-carry-forward-link (file `backlog`/`bugfix-N` entry at discovery time with bidirectional story↔entry link). THE BAN: any out-of-scope finding appearing in narrative (Dev Notes / Completion Notes / PR description / retro / TODO comment) MUST have a matching `sprint-status.yaml` entry or be absorbed under ① — prose-only mentions are banned as deferred-discovery time-bombs. Recording requirement: a story triaging any discovery enumerates each one + its lane + tracked-entry-ID/added-AC in Completion Notes. Enforcement Phase 1 (manual): SM `/create-story` template gains a "Discovery Triage" field — **SM Bob's paired retro-19-P1 deliverable is STILL PENDING** (template edit not yet done; only the project-context.md rule authoring half is complete). Origin: Alexyu pain point B — find→fix gap spanned a whole story; epic-19's 3 bugfixes all traced to un-triaged originating-story findings (canonical chain 19-4b → 19-7/19-8 → 19-9 → bugfix-19-9). Rule 24 generalizes retro-8-P1 from end-of-epic retros to mid-story discovery. No ESLint/code surface — pure process rule + docs; 0 Go, 0 FE, 0 baseline regeneration. sprint-status.yaml `retro-19-P1-discovery-triage-rule-24` updated: ARCH half done, SM half open.). Prior: 2026-05-28 (story 19-9 — epic-19 post-capstone TEMPORAL-dimension hardening: introduced **Rule 23 (Time-Dependent Component Fixture Stability)** in this doc — full spec text after Rule 22 (4 trigger criteria + 3 accepted marker forms `Clock-mocked` / `Clock-injected` / `Time-bomb-exempt` + ≥2-state coverage requirement + component-UI-only scope + party-mode 2026-05-26 + 19-8 PR #8 precedent citation; Rule 22 tooling block extended with one cross-reference sentence linking to Rule 23 as the temporal counterpart). New ESLint rule `local/time-dependent-fixture-stability` (`apps/web/src/eslint-rules/time-dependent-fixture-stability.js`) wired in `eslint.config.mjs` scoped to `apps/web/src/components/**/*.{ts,tsx}` (excluding `*.spec.*` + `index.ts` barrels — same scoping as 19-3 Rule 21 rule); AST visitor for `Date.now` MemberExpression + `new Date()` NewExpression; spec file ≥12 tests covering all 3 marker forms + position invalid cases + scoping. New Playwright clock-mock helper at `tests/visual/clock-mock.ts` exports `withFixedClock(page, iso)` wrapping `page.clock.install({ time, shouldAdvance: false })` (Playwright 1.57.0 supports GA API, no fallback needed); `tests/visual/components.visual.spec.ts` extended with optional `clockTime` fixture-row field (backward-compatible — pre-existing 122 fixtures unaffected). Audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md` — durable record of all time-bomb candidates with per-file disposition (migrated / clock-injected / time-bomb-exempt / pre-existing-safe-via-fixed-date / out-of-scope-non-rendering); future re-scans grep against this baseline. **Critical fixture migration: `library-recently-added` split into `recent` (`clockTime: '2026-05-15T00:00:00Z'`, `isWithin7Days=true`, green 新增 badge visible) + `stale` (`clockTime: '2026-05-30T00:00:00Z'`, false, no badge) — dual-state baselines `-darwin` + `-linux` (latter via 19-4b CI bootstrap per 19-5 workflow); old `default` baseline removed.** Story 19-8 PR #8 `Visual Regression / PR` check failure on `components/library-recently-added/default` baseline was the originating evidence — once 19-9 merges to main, 19-8 owner rebases on top → check goes green → PR #8 merges → epic-19 capstone fully sealed (AC #10 of this story; rebase command in 19-9 Dev Agent Record). SM `/create-story` template extended with new Dev Notes sub-section "Time-dependent visual coverage" (forward-only; future stories MUST list ≥2 fixture state baseline paths when their component reads the wall clock — Sally's party-mode 2026-05-26 request: turn evaluator discretion into template-enforced field). Backlog entry `preexisting-fail-library-recently-added-visual` removed from `sprint-status.yaml` at close (the very thing this story fixed). 19-9 stamps `[@contract-v1]` on AC #1–#5 (Rule 23 spec / ESLint rule API / audit-doc shape / `withFixedClock` helper signature + `clockTime` field / dual-state baseline convention). Consumes upstream per Rule 20 forward-only retrofit: confirmed against 19-3 `[@contract-v3]` (ESLint plugin pattern + accepted-marker grammar style), 19-4 `[@contract-v1]` AC #1 (`visual` Playwright project), 19-4 `[@contract-v1]` AC #5 (baseline path convention), 19-4b `[@contract-v1]` AC #5 (`-darwin`/`-linux` platform-suffix). No upstream contract bumps. Three orthogonal Rules now cover all observed visual-baseline drift classes: **Rule 21 spatial · Rule 22 cadence · Rule 23 temporal**. Branched off `origin/main` (NOT off feat/19-8) per AC #10 canonical flow — 19-9 merges first, 19-8 rebases on top.). Prior: 2026-05-26 (story 19-8 — epic-19 CAPSTONE: comprehensive design-implementation drift sweep over ALL 131 `apps/web/src/components/**` files (full-sweep, NOT a ≥5 Rule-22 sample — explicit override; future retros MUST NOT cite 19-8 as the ≥5-sample precedent). Outcome by classification: **0 material · 2 minor · 97 exact · 25 N/A-utility · 7 N/A-design-gap = 131**. The 2 minor (HeroBanner image-fallback, TrailerModal autofocus) share no theme and are < the 3-shared-theme bundling threshold → log-only, 0 polish-bundle story. The 97 exact = 12 Cat-A canonically-mapped + 85 Cat-C screen-sections. The 7 N/A-design-gap = 6 `setup/*` wizard steps + `learning/LearnPatternPrompt` (features postdate the `.pen` design — no Screen Frame exists). Top-line conclusion: **drift is NON-EXISTENT — design and implementation are aligned**; the `bugfix-10-4` `PosterCardHover` drift that motivated epic-19 was an ISOLATED incident, NOT systemic; the "systemic drift" hypothesis is empirically disproven. Deliverables: durable audit doc `_bmad-output/audit/drift-sweep-2026-05.md` (131-row findings table + 94-row screen-section mapping resolution + audit-trail markers); **0** `bugfix-N` stories spawned (0 material drift); **94** Cat-C `<screen-section — pending epic-19-8 mapping>` placeholders upgraded to the soft `// Design ref:` form (87 → a `.pen` Screen Frame, 7 → the design-coverage-gap variant `// Design ref: ux-design.pen — no current screen frame; {reason}`). ESLint rule `local/implements-pen-node-id` gained a 5th accepted form (the `// Design ref:` Screen form + the design-coverage-gap variant) — a 19-3 [@contract-v3] grammar WIDENING (rule spec 22→27 tests); every header accepted before still passes. Rule 21 `<screen-section …>` placeholder closed past-tense (AC #10 option (i) — definition retained for future re-backfills); Rule 22 full-sweep sample-policy OVERRIDE + audit-doc precedent recorded in the tooling block. 19-8 AC #2 + AC #5 re-stamped [@contract-v1→v2] (CR 2026-05-29) to fold the `N/A-design-gap` classification + the 4th `// Design ref: — no current screen frame` marker form into the contract taxonomy — the implementation shipped them; the contract now matches so future Rule 22 drift-rate comparisons stay valid. Pure FE header-line edits + docs — 0 Go, 0 frontend logic, 0 baseline regeneration, `.pen` read-only via Pencil MCP. Originating `Visual Regression / PR` red on `components/library-recently-added/default` was a pre-existing time-bomb (RecentlyAdded.tsx got a comment-only edit; comments don't render) deferred to and RESOLVED by 19-9 (dual-state clock-mocked `recent`/`stale` baselines; the failing `default` baseline removed). Rebased onto post-PR#9 main; PR #8 merged `af645c6` — closes epic-19.). Prior: 2026-05-20 (story 19-7 — epic-19 journey-half month-end watchdog: `.github/workflows/testsprite-quota-warning.yml` (`TestSprite Quota Warning` workflow, single job `TestSprite Quota Warning / Check`) fires `0 3 28 * *` UTC + `workflow_dispatch`, reads `_bmad-output/audit/testsprite-queue.yaml`'s `last_run` block and opens a deduped GitHub Issue — label `testsprite-quota-warning` is the dedup key (at most one open Issue; 4-branch open/comment/no-op/auto-close tree) — when 19-6's last monthly run is UNHEALTHY: `last_run.status` ∈ {api-failure, test-failures-only}, or `last_run` null / >35-day stale, or queue `schema_version != 2`. The 28th-of-month cron leaves ~2–3 days before month-end Free-plan credit expiry for a manual `workflow_dispatch` catch-up of 19-6. Re-scoped via correct-course 2026-05-20 (`sprint-change-proposal-2026-05-20.md`): the original design's live `testsprite_check_account_info` credit-threshold check was unimplementable — the bare `@testsprite/testsprite-mcp` CLI exposes no such subcommand (19-6 Phase 2 finding; only MCP-server mode does) and 19-6 [@contract-v2] keeps `last_run.credits_*` permanently null — so 19-7 was re-framed from a live-credit threshold to a queue-file `last_run` health watchdog. Pure repo-file reader (`actions/checkout` + `yq` + `actions/github-script@v7`); needs NO TestSprite secret — the live-account-info dependency was removed, only the auto-issued `GITHUB_TOKEN` with job-scoped `{ contents: read, issues: write }` is used. Runner pinned `ubuntu-24.04`; concurrency `group: testsprite-quota-warning` `cancel-in-progress: false` (distinct from 19-6's `testsprite-monthly` group). Consumes ONLY 19-6 [@contract-v2] AC #2 (queue `schema_version: 2` + `last_run` block shape); the live-credit + secret dependency on 19-6 AC #4/#5/#6 was removed. Live remaining-credit alerting deferred to a potential future 19-7a (MCP-stdio dance). 19-7 stamps `[@contract-v1]` on AC #1–#3. Prior: 2026-05-19 (story 19-6 — epic-19 journey-half drift-prevention CI wired up: monthly cron consumer at `.github/workflows/testsprite-monthly.yml` (`TestSprite Monthly` workflow, single job `TestSprite Monthly / Consume`) fires `0 3 1 * *` UTC + `workflow_dispatch`, reads `_bmad-output/audit/testsprite-queue.yaml` (`schema_version: 2` — Rule 20 [@contract-v2] surface after Phase 2 + CR re-stamps; downstream 19-7 quota-warning depends on this shape) as the single source of truth for what runs when, slices testIds upfront from the queue head — `N = MIN(queue_len, PER_RUN_CAP / per_case_estimate) = MIN(queue_len, 24)` — to enforce dual budget ceiling: `consumption_cap_pct: 80` (120/150 credits per run = the pre-run cap) **AND** `reserved_credits: 30` (floor, naturally reserved by the upfront slice so the manual ad-hoc lane never starves). v2 swapped the originally-spec'd LIVE per-case `testsprite_check_account_info` check for pre-run slicing after Phase 2 discovered the MCP-tool isn't a CLI subcommand (only the MCP server mode exposes it); the dual-ceiling end-state is mathematically equivalent. Tradeoff: a human consuming credits mid-cron-run doesn't trigger a mid-run abort; race acceptable for monthly cadence. Runner pinned `ubuntu-24.04` (mirrors 19-5 AC #5 — TestSprite `TC*.py` cases may depend on headless-browser font rendering, image bumps are deliberate PRs). Queue rotation: each run moves consumed entries from `queue:` to `history:` (capped at 200, FIFO prune); when `queue:` empties, oldest from `history:` cycle back — continuous ~3-month per-case coverage across 50 v4-catalog test cases (story originally anticipated 62 then DEV closeout seeded the actual-at-impl 30, then Phase 2 reseeded to the canonical 50 v4 plan IDs from `testsprite_frontend_test_plan.json` dated 2026-03-24: TC009–014, TC035–038, TC039–078 with permanent gaps at TC024–026 / TC030–032). Phase 2 swap rationale: the original 30 contained 22 v3-orphans whose features are gone from the v4 build — running them produced false positives. Commit-back format `chore(testsprite): monthly run {YYYY-MM} — {N} cases consumed` direct-push to main under `github-actions[bot]` identity with job-scoped `contents: write` permissions (NO PR, NO PR-review overhead; queue is mechanical audit metadata, owner reverts if catastrophic). v2 dropped the `, {credits} credits` heading suffix + the `Credits: {start} → {end} (Δ {consumed})` body line — those depended on LIVE per-case `testsprite_check_account_info` queries the bare CLI can't make. Concurrency `group: testsprite-monthly` `cancel-in-progress: false` (opposite of 19-5 PR job — mid-run cancel would corrupt queue with partial-commit-no-rollback). Failure bifurcation (AC #6): workflow exit ≠ 0 ONLY for "human must intervene" — (i) `TESTSPRITE_API_KEY` auth fail, (ii) `TESTSPRITE_TARGET_URL` unreachable/unset, (iii) `git push` rejected after 3-retry rebase loop (typically branch-protection misconfig); single-case `fail` = data (recorded in `history.last_status: fail`, exit 0), so Actions email reserved for human-intervene class only. Repo Secret `TESTSPRITE_API_KEY` (encrypted) + repo Variable `TESTSPRITE_TARGET_URL` (plaintext — HTTP URL not sensitive, owner flips between NAS-tunnel / cloud-staging / runner-local docker-compose without code change) — both owner-wired post-merge per AC #9 Completion Notes follow-up (mirrors 19-5 AC #2's branch-protection-rule operational pattern). Local TestSprite API key in `testsprite_tests/tmp/config.json` is double-gitignored (`.gitignore:8` + `:74`); verified at story-create + story-implementation. `@testsprite/testsprite-mcp@0.0.37` pinned via `npx` not `package.json` devDep (invoked only by this one workflow, no lockfile value). [@contract-v1] stamped on AC #1, #5 (workflow file location + trigger model; secret/variable names) + [@contract-v2] stamped on AC #2, #3, #4 (queue schema bumped to `schema_version: 2` after CR re-stamp; budget rule swapped to pre-run testIds slicing in Phase 2; commit-back format dropped `{credits}`/`Credits:` after CR re-stamp) — no upstream version-stamped consumer (TestSprite `TC*.py` files are docs not contract-stamped ACs). 19-7 quota-warning workflow (ready-for-dev) will consume 19-6's [@contract-v2] AC #2 `schema_version: 2` + [@contract-v2] AC #4 commit-message format. Prior: 2026-05-18 (story 19-5 — Rule 22 tooling wired into PR-blocking CI: `.github/workflows/visual-regression.yml` (`Visual Regression` workflow) runs `pnpm run test:visual` on every PR touching design-rendering surfaces (`apps/web/src/{components,routes,styles}/**`, `tailwind.config.js`, `index.css`, `main.tsx`, `tests/visual/**`, `playwright.config.ts`, `package.json`, the workflow itself) and on every push to main/develop. PR job (`Visual Regression / PR`) fails on any pixel diff against committed baselines — combined with a branch-protection rule the owner enables out-of-band, blocks merge. Main job (`Visual Regression / Main`) self-bootstraps the `-linux` baseline set on its first execution via a `requires-manual-review` PR (Sally / UX re-engagement gate per 19-4b Task 5 ruling) and runs verify-only thereafter. Runner pinned to `ubuntu-24.04` (NOT `-latest`) — Linux font rendering is the deterministic-baseline lever; image bumps are deliberate-rebless PRs. Diff artifacts (`actual.png`, `diff.png`, traces) uploaded with 14-day retention as `visual-regression-diffs-{pr|main}-${{ github.run_id }}`. Concurrency split: PR group `visual-regression-${{ github.workflow }}-${{ github.ref }}` with `cancel-in-progress: true` (force-push cancels prior run); main group `visual-regression-main` with `cancel-in-progress: false` (serializes). Decision record: Nx `e2e:visual` wrapper REJECTED in favour of direct `pnpm run test:visual` (no Nx-cache lever for pixel work + `paths:` filter does trigger-level skipping already). [@contract-v1] stamped on AC #1–#5 (workflow file existence + trigger model; PR-blocking semantics; npm-script invocation; Linux bootstrap flow; runner-image pinning) — consumes 19-4 [@contract-v1] AC #1/#2/#5 unchanged). Prior: 2026-05-12 (story 19-4 — Rule 22 tooling LIVE: Playwright `visual` project + `pnpm run test:visual` + dev-only component gallery route `apps/web/src/routes/test/gallery.tsx` (`/test/gallery`) produce committed per-component default/hover/focus baselines under `tests/visual/components.visual.spec.ts-snapshots/`, each tagged with its `data-pen-node`; this story = harness + ~25 reference components, the rest tracked in `19-4b-visual-baseline-bulk-fill`; audit doc `_bmad-output/audit/visual-baseline-19-4.md`; 19-5 wires it into PR CI). Prior: 2026-05-12 (story 19-3 — Rule 21 Phase-2 enforcement LIVE: custom ESLint rule `local/implements-pen-node-id` (`apps/web/src/eslint-rules/implements-pen-node-id.js`, wired in `eslint.config.mjs`) errors on missing/malformed `// Implements:` headers under `apps/web/src/components/**/*.{ts,tsx}`; all 131 component-dir files backfilled (12 mapped to real `.pen` Reusable Components, 25 `<utility — no .pen counterpart>`, 94 `<screen-section — pending epic-19-8 mapping>` — a NEW 4th accepted form added per Sally+Amelia+Bob Party Mode 2026-05-12 for components that render a section of a designed screen frame; canonical screen-frame mapping tracked by epic-19-8; audit doc `_bmad-output/audit/drift-19-3-2026-05.md`). Rule 21's accepted-marker grammar list updated to include the multi-component `+` form and the `<screen-section …>` placeholder. Prior: 2026-05-08 (Rule 21 + Rule 22 added — Party Mode consensus on design-implementation drift prevention; Rule 21 requires every component file under `apps/web/src/components/` to header-reference its `.pen` node ID via `// Implements: Component/{Name} ({pen-node-id})`, Rule 22 mandates each epic retro samples ≥5 components for design-drift audit with exact/minor/material classification. Origin: bugfix-10-4 root cause discovery — `HoverPreviewCard.tsx` was independently invented and diverged from existing `.pen` `Component/PosterCardHover` (node `MQbvp`); drift hypothesized as systemic. `epic-19` (Design-Implementation Drift Audit) cross-cutting initiative — 8 stories (19-1…19-8) — tracks Phase-2 enforcement infrastructure: ESLint rule, Playwright visual baselines, GitHub Actions for visual regression + monthly TestSprite quota consumption + month-end quota warning. Sally + Bob + Winston + Amelia + Murat consensus). Prior: 2026-04-24 (Rule 7 prefix rename `QB_` → `QBITTORRENT_` via followup-qbittorrent-prefix-rename; restores `SOURCE = uppercase(package)` uniformity across all 13 registered prefixes — was the only outlier per Winston 2026-04-20 retro-10-AI3 Item 3 ruling). Prior: 2026-04-22 (Rule 20 AC Contract Versioning — retro-10-AI5; introduces `[@contract-vN]` prefix + bump/ack protocol + forward-only retrofit, Pattern #2 from Epic 10 retro, spike doc committed as 4a598e5; CR follow-up 2026-04-22 hoisted grep helpers into Rule 20 body, unified Change Log format to `{what changed, what breaks downstream}`, documented v0 fallback). Prior: 2026-04-22 (Rule 15 HTTP Route ↔ Client Method Sync extension — retro-10-AI4; adds 4th sub-section guarding "client method exists ≠ HTTP route registered", Story 10-2 precedent). Earlier: 2026-04-20 (Rule 7 expansion — added `QB_`, `METADATA_`, `DOUBAN_`, `WIKIPEDIA_` prefixes already in production use; surfaced by retro-10-AI3 CR grep on 2026-04-20)
**Architecture Status:** ✅ Validated and Ready for Implementation (5,463 lines, 8 steps completed)

---

## 🚨 CRITICAL: Current Project State

### Dual Backend Architecture Problem

**The project currently has TWO separate Go backends with divided features:**

1. **Root Backend** (`/cmd` + `/internal`)
   - ✅ Has: Swagger, zerolog logging, TMDb client, advanced middleware
   - ❌ Missing: NO database, NO data persistence

2. **Apps Backend** (`/apps/api`)
   - ✅ Has: SQLite database, migrations, repository pattern
   - ❌ Missing: NO Swagger, NO structured logging, NO TMDb integration

### ⚠️ ALL NEW CODE MUST GO TO: `/apps/api`

**Consolidation Plan (5 Phases):**

**Phase 1: Backend Consolidation** (⭐ CURRENT PRIORITY)

- **Step 1.1:** Migrate TMDb client: `/internal/tmdb/` → `/apps/api/internal/tmdb/` (update to use slog)
- **Step 1.2:** Migrate Swagger: `/cmd/api/main.go` → `/apps/api/main.go` + `/apps/api/docs/`
- **Step 1.3:** Migrate middleware: `/internal/middleware/` → `/apps/api/internal/middleware/`

**Phase 2-5:** Implement architectural decisions, frontend alignment, core features, and testing.
See `_bmad-output/planning-artifacts/architecture/consolidation-refactoring-plan.md` for complete 5-phase roadmap.

**Root backend** (`/cmd`, `/internal`) will be archived to `/archive/` after Phase 1 completion.
**DO NOT add code to `/cmd` or root `/internal`** - these are deprecated.

---

## 🎯 Core Architectural Decisions (MANDATORY)

### 1. CSS Framework: Tailwind CSS v3.x

- **Use:** Utility-first classes for all styling
- **Config:** `/apps/web/tailwind.config.js`
- **Why:** Bundle size optimization, design system consistency

### 2. Testing Infrastructure

- **Backend:** Go testing + testify (coverage >80%)
- **Frontend:** Vitest + React Testing Library (coverage >70%)
- **E2E Feature-level:** Playwright (328 tests, runs in CI/nightly)
- **E2E Journey-level:** TestSprite (journey tests against deployed NAS at `http://192.168.50.52:8088`, manual trigger after deploy). 62 test cases across 6 P0 journeys, production server mode. Plan v4-regenerated 2026-03-27 for Epic 7+8. Test plan: `testsprite_tests/`. Baseline strategy: regenerate on deploy, mark `intentional-change` for bugfix breaks.
- **Pattern:** Co-located tests (`*_test.go`, `*.spec.tsx`)

### 3. Authentication — REMOVED (v4)

> **v4 Decision:** Vido v4 is single-user with no authentication required. Multi-user support is deferred to v5.0. All auth-related code, middleware, and configuration have been removed from scope.

### 4. Caching: Tiered (Memory + SQLite)

- **Tier 1:** In-memory (bigcache/ristretto) for hot data
- **Tier 2:** SQLite `cache_entries` table for persistent cache
- **TTL:** TMDb 24h, AI parsing 30d, images permanent

### 5. Background Tasks: Worker Pool

- **Implementation:** Goroutines + channels (NO external queue)
- **Workers:** 3-5 goroutines
- **Retry:** Exponential backoff (1s → 2s → 4s → 8s)

### 6. Error Handling: slog + Unified AppError

- **Logging:** Go `log/slog` (NOT zerolog, NOT fmt.Println)
- **Errors:** Custom `AppError` type with error codes
- **Format:** Structured JSON logs with sensitive data filtering

### 7. Plugin Architecture: Go Interfaces

**Decision:** Embedded plugin system using Go interfaces for external service integration.

**Interfaces:**

- `MediaServerPlugin` — Plex, Jellyfin (SyncLibrary, GetWatchHistory)
- `DownloaderPlugin` — qBittorrent, NZBGet (AddDownload, GetStatus, Pause, Remove)
- `DVRPlugin` — Sonarr, Radarr (AddMovie, AddSeries, GetQueue)
- Common: `Name()`, `TestConnection(config PluginConfig) error`

**Plugin Manager:** Registration at startup, per-plugin config in SQLite, health check scheduler.
**Location:** `/apps/api/internal/plugins/`

**Rules:**

- All plugin configs must pass `TestConnection()` before being saved
- Plugins must implement graceful degradation (feature disabled when plugin unavailable)
- Plugin health checks run at configurable intervals (default 60s)

### 8. Real-Time Events: SSE Hub

**Decision:** Server-Sent Events for real-time progress updates, replacing polling for downloads/scans/subtitles.

**Architecture:** Single Hub goroutine, fan-out to client channels via `http.Flusher`.
**Broadcast Event Types:** `scan_progress`, `scan_complete`, `scan_cancelled`, `subtitle_progress`, `subtitle_batch_progress`, `notification`
**Control Event Types:** `connected` (handshake), `ping` (keepalive)
**Location:** `/apps/api/internal/sse/`

**Rules:**

- SSE endpoint: `GET /api/v1/events`
- Buffered channels per client (capacity 100), drop on overflow via non-blocking send
- Hub internal channels: broadcast (256), register/unregister (64 each)
- Wire format: `event: {type}\ndata: {json}\n\n` — note: `{json}` is the full `Event` struct (`id`, `type`, `data`), so `type` appears both in the SSE event line and inside the JSON payload
- Reconnection (`Last-Event-ID`) not yet supported; `Event.ID` field exists but is not emitted as SSE `id:` line

**Lazy Connection Pattern** (`handler.go`):

1. Client HTTP request arrives at `GET /api/v1/events`
2. SSE headers are set (`text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no`)
3. Client registers with Hub **after** HTTP handshake completes — lazy registration
4. Hub assigns UUID client ID, creates buffered channel (capacity 100)
5. Initial `connected` event sent with `clientId` to confirm handshake
6. Event streaming begins via `c.Stream()` loop
7. **Keepalive:** 30-second `ping` events (with timestamp payload) prevent proxy/client timeouts
8. On client disconnect, deferred `Unregister()` enqueues removal; Hub's `Run()` goroutine then closes the channel and deletes the client

**Non-blocking Broadcast** (`hub.go`):

- `Broadcast()` sends to Hub's broadcast channel (capacity 256) via `select...default` — drops event with warning log if full
- `Run()` goroutine fans out each broadcast to all registered clients via `select...default` — drops per-client if that client's channel is full
- `Close()` uses `atomic.Bool` for once-only shutdown, signals via `done` channel, closes all client channels

**Frontend Lazy SSE Connection Pattern** (CRITICAL — Epic 7 retro lesson):

Any persistent connection (SSE, WebSocket) in a globally-mounted or root-level component **MUST** be lazy-initialized — never connect on mount. Eager SSE connections break Playwright E2E tests because `networkidle` waits for 0 open connections, which is impossible with a persistent SSE stream.

**Pattern:** Expose a `startTracking()` / `connect()` trigger; only open `EventSource` when the feature is actually needed.

**Existing implementations:**

- `useScanProgress.ts` — SSE connects via `startTracking()`, called only when a scan is triggered. No connection on mount.
- `useParseProgress.ts` — SSE connects only when `taskId` is non-null (conditional `useEffect`).

**Rules for new SSE consumers:**

1. NEVER call `new EventSource()` in `useEffect` with `[]` deps (mount-time)
2. Use a gating condition (user action, non-null ID, active status) before connecting
3. Always clean up `EventSource.close()` in `useEffect` return
4. Reconnect with backoff on error — do NOT fall back to polling (SSE reconnect is sufficient)
5. Guard all dispatches with `mountedRef.current` to prevent updates after unmount

### 9a. Media Library Management (ADR 2026-03-29)

**Decision:** Multi-library system with per-folder content type assignment (Route 2 — Progressive Enhancement).

**Data Model:**

- `media_libraries` table: id, name, content_type (movie|series), auto_detect (Phase 2 reserve), sort_order
- `media_library_paths` table: id, library_id (FK), path (UNIQUE), status, last_checked_at
- `movies`/`series` tables: +library_id (FK), +detected_type (Phase 2), +override_type (Phase 2)
- Migration: #020

**API Endpoints:**

- `GET/POST /api/v1/libraries` — list/create libraries
- `PUT/DELETE /api/v1/libraries/:id` — update/delete library
- `POST/DELETE /api/v1/libraries/:id/paths` — add/remove paths
- `POST /api/v1/libraries/:id/paths/refresh` — refresh path statuses

**Service Changes:**

- `MediaService`: reads from `MediaLibraryRepository` (DB), fallback to `VIDO_MEDIA_DIRS` env var
- `ScannerService`: iterates libraries (not raw paths), assigns `library_id` + uses `content_type` for movie/series classification
- `SetupService`: creates library records instead of storing single `media_folder_path`

**Deprecation:**

- `settings.media_folder_path` → replaced by `media_libraries`
- `VIDO_MEDIA_DIRS` → demoted to fallback (log deprecation warning)

**ADR:** `architecture/adr-multi-library-media-management.md`

### 9b. Subtitle Engine Pipeline

**Decision:** Multi-source subtitle search with content-based language detection and OpenCC conversion.

**Pipeline:** search → score → download → post-process (OpenCC 簡繁轉換) → place
**Sources:** Assrt API, Zimuku scraper, OpenSubtitles API
**Scoring:** Language match 40% + Resolution match 20% + Source trust 20% + Group reputation 10% + Downloads 10%
**Location:** `/apps/api/internal/subtitle/`

**Rules:**

- Language detection MUST analyze subtitle file content (not filename) — this fixes Bazarr's core zh-TW bug
- OpenCC conversion direction: s2twp (Simplified → Traditional with Taiwan phrases)
- CN content policy: Skip conversion when `production_countries` contains `CN` (mainland content keeps simplified subtitles — dialogue expressions match audio)
- Conversion is user-overridable: per-search toggle in subtitle dialog, global preference in settings
- Edge cases: Co-productions (multiple countries) default to convert (conservative); already-traditional subtitles pass through unchanged (idempotent)
- Subtitle files use `.zh-Hant.srt` or `.zh-Hans.srt` extension based on final language for Plex/Jellyfin compatibility

---

## 📋 MANDATORY Rules (ALL Agents MUST Follow)

### Rule 1: Single Backend Location

```
✅ ALL backend code → /apps/api
❌ NEVER add code to /cmd or root /internal (deprecated)
```

### Rule 2: Logging with slog ONLY

```go
// ✅ CORRECT
slog.Info("Fetching movie", "movie_id", id)
slog.Error("Failed to parse", "error", err, "filename", filename)

// ❌ WRONG
log.Println("Fetching movie")
fmt.Println("Error:", err)
```

### Rule 3: API Response Format

```json
// ✅ Success
{
  "success": true,
  "data": { ... }
}

// ✅ Error
{
  "success": false,
  "error": {
    "code": "TMDB_TIMEOUT",
    "message": "無法連線到 TMDb API，請稍後再試",
    "suggestion": "檢查網路連線或稍後重試。"
  }
}
```

### Rule 4: Layered Architecture

```
✅ Handler → Service → Repository → Database
❌ Handler → Repository (FORBIDDEN - skip service layer)
```

### Rule 5: TanStack Query for Server State

```typescript
// ✅ CORRECT - Use TanStack Query for API data
const { data: movie } = useQuery({
  queryKey: ['movies', 'detail', movieId],
  queryFn: () => movieService.getMovie(movieId),
});

// ❌ WRONG - Never use Zustand for server data
const movie = useMovieStore((state) => state.movie);
```

### Rule 6: Naming Conventions

```
Database:   snake_case plural (movies, media_files)
API Paths:  /api/v1/{resource} (plural: /api/v1/movies)
Go Files:   snake_case.go (movie_handler.go)
Go Structs: PascalCase (Movie, TMDbClient)
TS Files:   PascalCase.tsx (MovieCard.tsx)
TS Types:   PascalCase (Movie, ApiResponse<T>)
JSON Fields: snake_case (release_date, tmdb_id)
```

### Rule 7: Error Codes System

```
Format: {SOURCE}_{ERROR_TYPE}

TMDB_TIMEOUT, TMDB_NOT_FOUND, TMDB_RATE_LIMIT, TMDB_INVALID_YEAR_RANGE
AI_TIMEOUT, AI_QUOTA_EXCEEDED
DB_NOT_FOUND, DB_QUERY_FAILED
VALIDATION_REQUIRED_FIELD, VALIDATION_INVALID_FORMAT
SUBTITLE_NOT_FOUND, SUBTITLE_DOWNLOAD_FAILED, SUBTITLE_CONVERT_FAILED
PLUGIN_INIT_FAILED, PLUGIN_HEALTH_CHECK_FAILED, PLUGIN_NOT_CONFIGURED
SCANNER_PERMISSION_DENIED, SCANNER_PARSE_FAILED
SSE_CONNECTION_FAILED
LIBRARY_NOT_FOUND, LIBRARY_DUPLICATE_PATH, LIBRARY_PATH_NOT_ACCESSIBLE
LIBRARY_PATH_NOT_DIRECTORY, LIBRARY_DELETE_HAS_MEDIA
QBITTORRENT_TORRENT_NOT_FOUND, QBITTORRENT_CONNECTION_FAILED, QBITTORRENT_AUTH_FAILED, QBITTORRENT_TIMEOUT, QBITTORRENT_NOT_CONFIGURED
METADATA_TIMEOUT, METADATA_RATE_LIMITED, METADATA_UNAVAILABLE, METADATA_NO_RESULTS, METADATA_CIRCUIT_OPEN, METADATA_INVALID_REQUEST, METADATA_ALL_FAILED, METADATA_GATEWAY_ERROR, METADATA_NETWORK_ERROR, METADATA_NOT_FOUND, METADATA_UNKNOWN_ERROR
DOUBAN_BLOCKED, DOUBAN_NOT_FOUND, DOUBAN_PARSE_ERROR, DOUBAN_RATE_LIMITED, DOUBAN_TIMEOUT
WIKIPEDIA_NOT_FOUND, WIKIPEDIA_NO_INFOBOX, WIKIPEDIA_PARSE_ERROR, WIKIPEDIA_RATE_LIMITED, WIKIPEDIA_TIMEOUT, WIKIPEDIA_API_ERROR
REQUEST_DUPLICATE, REQUEST_ALREADY_IN_LIBRARY
DVR_NOT_CONFIGURED, DVR_CONNECTION_FAILED, DVR_AUTH_FAILED, DVR_TIMEOUT, DVR_ADD_FAILED, DVR_TEST_FAILED, DVR_NOT_SUPPORTED
```

**Authoritative prefix set (15 sources):** `TMDB_`, `AI_`, `DB_`, `VALIDATION_`, `SUBTITLE_`, `PLUGIN_`, `SCANNER_`, `SSE_`, `LIBRARY_`, `QBITTORRENT_`, `METADATA_`, `DOUBAN_`, `WIKIPEDIA_`, `REQUEST_`, `DVR_`. When adding a new subsystem with its own error codes, extend this list AND sync `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` Step 3 "Rule 7 Wire Format Check" (both the HTML comment sync date and the inline prefix list).

### Rule 8: Date/Time Format

```
API:      ISO 8601 with timezone → "2024-01-15T14:30:00Z"
Database: TIMESTAMP (created_at, updated_at)
Go:       time.Time (auto-marshals to ISO 8601)
Display:  toLocaleDateString('zh-TW') → "2024年1月15日"
```

### Rule 9: Test Co-location

```
✅ Backend: movie_handler.go → movie_handler_test.go (same dir)
✅ Frontend: MovieCard.tsx → MovieCard.spec.tsx (same dir)
❌ NO separate tests/ directory
```

### Rule 10: API Versioning

```
✅ /api/v1/movies
✅ /api/v1/events
❌ /movies (missing version)
❌ /api/movie (singular)
```

### Rule 11: Interface Location

```
✅ Define interfaces in services package (e.g., services.MovieServiceInterface)
✅ Handlers import and use interfaces from services package
✅ Repository interfaces in repository package (e.g., repository.MovieRepositoryInterface)
❌ Never duplicate interface definitions across packages
❌ Never define service interfaces in handlers package
```

### Rule 12: Code Quality Checks (CI-based)

```
⚠️  Pre-commit hook DISABLED (2026-04-03) — Zed editor's background
    `git status` races with lint-staged's git stash, causing persistent
    index.lock conflicts. Attempted fixes: 87c85dd, c560311 — neither resolved.
✅ Lint and format checks run in CI instead
✅ Run `pnpm lint:all` locally before pushing (mirrors CI exactly)
❌ Do NOT re-enable the pre-commit hook until the Zed lock race is resolved
```

**`pnpm lint:all`** (defined in root `package.json`) runs these four checks **sequentially** — each step must pass before the next runs, matching CI's `lint` job order exactly:

1. `go vet ./...` — from `apps/api/` (via `nx run api:lint`)
2. `staticcheck ./...` — from `apps/api/`, pinned to `@2026.1` via a versioned binary at `$GOPATH/bin/staticcheck-2026.1` (auto-installs on first run if the versioned binary is missing; pre-existing unversioned `staticcheck` binaries from other projects are NOT used, preventing silent version drift)
3. `eslint .` — from repo root (via `pnpm run lint`; covers `apps/web/`, `libs/shared-types/`, and `tests/` — same scope as CI)
4. `prettier --check .` — from repo root (via `pnpm run format:check`)

If any step fails, fix it locally — do not push. For formatting, `pnpm exec prettier --write <files>` fixes in place. The four tools mirror CI's `lint` job exactly (`.github/workflows/test.yml`), so `pnpm lint:all` green ⇒ CI lint green.

If `go install` fails (e.g., no network), pre-install staticcheck manually:

```bash
# Installs to versioned path used by lint:all
STATICCHECK_TMP=$(mktemp -d) && GOBIN="$STATICCHECK_TMP" \
  go install honnef.co/go/tools/cmd/staticcheck@2026.1 && \
  mv "$STATICCHECK_TMP/staticcheck" "$(go env GOPATH)/bin/staticcheck-2026.1" && \
  rmdir "$STATICCHECK_TMP"
```

### Rule 13: Error Handling Completeness

```go
// ✅ CORRECT — propagate ALL errors
result, err := s.repo.UpdateStatus(ctx, id, status)
if err != nil {
    return fmt.Errorf("update status: %w", err)
}

// ✅ CORRECT — log then return error
if err := s.repo.Save(ctx, item); err != nil {
    slog.Error("Failed to save item", "error", err, "id", item.ID)
    return err
}

// ❌ WRONG — swallowed error (silent failure)
result, err := s.repo.UpdateStatus(ctx, id, status)
if err != nil {
    slog.Error("update failed", "error", err)
    // BUG: no return! Continues with stale result
}

// ❌ WRONG — error ignored entirely
s.repo.UpdateStatus(ctx, id, status)
```

```
Every error return MUST be either:
  1. Propagated to caller (return err / return fmt.Errorf("context: %w", err))
  2. Explicitly logged AND execution halted (return after log)
  3. Intentionally discarded with comment explaining why (rare, needs justification)
Never log an error and continue executing as if it succeeded.
```

### Rule 14: Resource Lifecycle Management

```
Bounded Maps:
  ✅ In-memory maps/caches MUST have an upper bound or eviction policy
  ✅ Use sync.Map with periodic cleanup or fixed-size LRU
  ❌ Unbounded map[string]T that grows forever in long-running processes

Graceful Shutdown:
  ✅ Background goroutines MUST accept context.Context and honor cancellation
  ✅ Use errgroup or WaitGroup to ensure clean shutdown
  ❌ Goroutines that ignore context and run until process kill

Client Caching:
  ✅ Expensive clients (HTTP, DB, API) MUST be created once and reused
  ✅ Cache with config fingerprint — recreate only when config changes
  ❌ Creating new client instances per request or per poll cycle
```

### Rule 15: Pre-commit Self-verification

```
Before marking a story task complete, verify:

main.go Wiring:
  ✅ New handlers/services registered in main.go dependency injection
  ✅ New routes added to router setup
  ❌ Implementing handler but forgetting to wire it up

DB Column Sync:
  ✅ New model fields have corresponding migration ALTER/CREATE
  ✅ Repository INSERT/UPDATE SQL includes ALL model fields
  ✅ Repository SELECT column list AND row scan include the new column too —
     a read path drifting from the schema silently returns the zero value
  ❌ Adding model field but missing it in repository SQL or migration
  ❌ Using/persisting a column whose repo SELECT/scan never loads it back
  📌 Precedent (bugfix-20-1): `series.seasons` (JSON col, mig 006) was never in
     seriesSelectColumns/scanSeries, so GetSeasons always returned [] and the
     season accordion was empty for every series — undetected because the unit
     test mocked the repo's FindByID to return a pre-populated value. Sync the
     SELECT/scan when a column is added, and cover real DB reads with an
     integration test (not just a mocked-repo unit test).

Swagger:
  ✅ New/changed endpoints have updated Swaggo annotations
  ✅ Run swag init if annotations changed
  ❌ Changing API contract without updating docs

HTTP Route ↔ Client Method Sync:
  ✅ If a task description says "endpoint already exists in client" or
     "method already registered", grep apps/api/cmd/api/main.go for the
     corresponding {handler}.RegisterRoutes(apiV1) call AND verify the
     exact HTTP method + path in the handler file.
  ✅ Client method existing ≠ HTTP route registered. Assume nothing.
  ✅ If route is missing, expand story scope (new task + AC) before
     continuing. Do not silently add it.
  ❌ Trusting a client method's existence as proof the server route is wired.
  📌 Precedent (Epic 10 Retro AI-4, Story 10-2 Task 3.3): the Go client
     method tmdb.GetMovieVideos in apps/api/internal/tmdb/client.go existed,
     but the internal backend route GET /api/v1/tmdb/movies/:id/videos →
     tmdbHandler.GetMovieVideos (apps/api/internal/handlers/tmdb_handler.go:440)
     was never wired — DEV had to add it mid-story, silently expanding scope.
```

### Rule 16: Test Assertion Quality

```typescript
// ✅ CORRECT — specific DOM assertion
expect(screen.getByText('Movie Title')).toBeInTheDocument();

// ✅ CORRECT — use toBeAttached for CSS hover/transition elements
expect(overlay).toBeAttached();

// ✅ CORRECT — specific value assertion
expect(result).toEqual({ id: '1', title: 'Test' });

// ❌ WRONG — toBeTruthy for DOM presence (too vague)
expect(screen.getByText('Movie Title')).toBeTruthy();

// ❌ WRONG — toBeVisible for CSS hover-dependent elements (flaky)
expect(overlay).toBeVisible();

// ❌ WRONG — generic boolean for structured data
expect(!!result).toBe(true);
```

```
Use the MOST SPECIFIC assertion matcher available:
  - DOM presence: toBeInTheDocument() (not toBeTruthy)
  - CSS hover/transition elements: toBeAttached() (not toBeVisible)
  - Text content: toHaveTextContent() (not check innerHTML)
  - Equality: toEqual/toStrictEqual (not toBe for objects)
  - Errors: toThrow/toReject (not try-catch with toBeTruthy)
```

### Rule 17: Bilingual Documentation

```
All user-facing documentation MUST be bilingual (EN + zh-TW):

File Naming:
  ✅ doc-name.md (English, primary)
  ✅ doc-name.zh-TW.md (Traditional Chinese)
  ❌ doc-name.zh.md (wrong language tag)
  ❌ Chinese-only doc without English version

Scope:
  ✅ docs/ folder: installation guides, API references, event docs
  ✅ README.md + README.zh-TW.md (when user-facing)
  ❌ Internal docs (_bmad-output/, architecture/) — English only
  ❌ Code comments — English only

Translation Rules:
  ✅ Code blocks, URLs, file paths remain in English
  ✅ Technical terms keep English with optional Chinese annotation
  ✅ Tables preserve same structure in both languages

Reference: Epic 8 Agreement 6
```

### Rule 18: API Boundary Case Transformation

```
All frontend services MUST transform data at the API boundary:

Response (backend → frontend):
  ✅ snakeToCamel(data.data) on every API response
  Already enforced via shared fetchApi in libraryService.ts

Request (frontend → backend):
  ✅ JSON.stringify(camelToSnake(params)) on every POST/PUT body
  ❌ JSON.stringify(params) — sends camelCase keys, backend rejects or ignores

Implementation:
  import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

  // Response: always transform
  return snakeToCamel<T>(data.data);

  // Request: always transform body
  body: JSON.stringify(camelToSnake(params))

Reference: Bugfix sprint 2026-03-28 audit — 4 services found missing camelToSnake
```

### Rule 19: Package Dependency Boundaries

```
Go internal package import direction (apps/api/internal/):

Allowed (single-direction layering, extends Rule 4):
  Handler  → Service    → Repository → Database
  Handler  → Subtitle   → Service              (subtitle uses services.TerminologyCorrectionServiceInterface)
  *        → ai, models, sse, retry, cache  (leaf packages — see list below)

  NOTE: Handler → Repository is FORBIDDEN by Rule 4. Rule 19 does not
  introduce an exception. Go through a service.

FORBIDDEN:
  Service ↛ Subtitle    (would cycle: subtitle already imports services)
  Service ↛ Handler     (Rule 4 — never reach back up the request stack)
  Repository ↛ Service  (Rule 4)
  Repository ↛ Subtitle (Rule 4 — repository sits below services)

Known Cycle Points (verified 2026-04-13):
  - subtitle/engine.go:61  → services.TerminologyCorrectionServiceInterface (field)
  - subtitle/engine.go:90  → services.TerminologyCorrectionServiceInterface (setter)
  Therefore: NO file under internal/services/ may import
  "github.com/vido/api/internal/subtitle" — `go build` will reject with
  "import cycle not allowed".

Leaf packages (zero internal deps — always safe to import from anywhere):
  ai, models, sse, retry, cache

Verified 2026-04-13 via `go list -deps ./internal/<pkg>`. The list is
enforced by boundaries_test.go::TestLeafPackagesHaveNoInternalDeps so it
cannot silently rot. Notable non-leaves (do NOT add to this list without
re-verifying):
  - secrets  → depends on internal/crypto
  - logger   → depends on internal/{models, retry, repository}
  - errors   → not present (no such package today)

Workaround Pattern: Mirror Types
  When a service needs subtitle-package logic (parse SRT, format blocks, etc.):

  Step 1: Mirror the minimal type in services/ — only the fields you need.
          Do NOT re-export or alias from subtitle. Keep it a separate type.
  Step 2: Inline the minimum logic. Match the source's validation rules
          (same regex, same error handling) so behavior stays identical.
  Step 3: Add a one-line comment citing this rule:
            // services ↛ subtitle — see project-context.md Rule 19.
  Step 4: Keep the two implementations in sync via code review.
          When subtitle.SubtitleBlock fields change, update the mirror.
          When subtitle.ParseSRT validation changes, update the inline parser.

Reference Implementation (already in production as of Epic 9):
  - apps/api/internal/services/translation_service.go:30-39
      → TranslationBlock mirrors subtitle.SubtitleBlock
  - apps/api/internal/services/transcription_service.go:362-369
      → ParseSRTToTranslationBlocks inlines subtitle.ParseSRT validation
        (exported only so the external-test-package parity check can
         call it cross-package — see srt_parity_test.go)

Enforcement (stdlib-only):
  boundaries_test.go (apps/api/internal/, package internal):
  - TestServicesMustNotImportSubtitle   — primary cycle gate
  - TestScanImports_DetectsViolation    — sanity that actually exercises the
                                          scanImports helper (tempdir with a
                                          violating file + an external test
                                          file that must be skipped)
  - TestForbiddenImportEdges            — services↛handlers, repository↛{services,subtitle}
  - TestLeafPackagesHaveNoInternalDeps  — keeps the leaf list above honest

  srt_parity_test.go (apps/api/internal/services/, package services_test):
  - TestParseSRT_ParityWithSubtitle     — Mirror-Types drift detector;
                                          lives in an external test package so
                                          it can import both services and
                                          subtitle without creating a cycle

Reference: Epic 9 retro AI-5 (insight #3) — surfaced during 9-2b implementation.
```

### Rule 20: AC Contract Versioning

```
AC Contract Versioning:
  ✅ Cross-story-referenced ACs MAY carry `[@contract-v1]` prefix.
     Format: `AC #N [@contract-v1]: Given/When/Then...`
  ✅ When changing a stamped AC's contract shape/semantics, bump
     `[@contract-vN]` → `[@contract-v(N+1)]` AND add Change Log entry:
     `| {Date} | [@contract-vN→v(N+1)] AC #N: {what changed, what breaks downstream} |`
     Two-stage verify: (a) row present, (b) row body has ≥2 non-empty
     sub-tokens after `AC #N:` (both "what changed" AND "what breaks
     downstream" populated). Degenerate entries like `AC #N: tweak` pass
     (a) but fail (b) — flag as MEDIUM CR finding.
  ✅ Downstream stories referencing a stamped AC MUST record in Dev Notes:
     `confirmed against [@contract-vN] (Story X-Y AC #N)`
  ✅ Historical unstamped ACs are implicitly `v0` (frozen); stamp only
     when newly referenced by a forward story (forward-only retrofit).
     If a downstream→upstream grep returns 0 hits (upstream is pre-Rule-20),
     treat the upstream as implicit v0 and skip the ack requirement.
  🔁 Bump → downstream stale-mark (PRODUCER-side obligation, added retro-19-P2):
     The ack rule above is CONSUMER-side and snapshots at the downstream
     story's authoring time — it does NOT protect a downstream DRAFT written
     against vN and then orphaned when upstream bumps to v(N+1) before the draft
     ships. To close that gap, WHEN you bump `[@contract-vN]` → `[@contract-v(N+1)]`
     on an upstream AC, in the SAME change you MUST:
       1. GREP for every downstream story that acked the bumped contract — the
          ack records {upstream story} + {AC #}, so both are greppable:
            grep -rnE 'confirmed against `?\[@contract-v[0-9]+\]' \
              _bmad-output/implementation-artifacts/ | grep -E 'Story <X>-<Y>.*AC #<N>'
          (the `` `? `` tolerates a backtick-wrapped token — `confirmed against `[@contract-v1]`
          — see the 🔎 backtick-tolerance note below; loosen further to the bare
          upstream story-id or AC-number if ack wording varies.)
       2. For each hit, look up that downstream story's `sprint-status.yaml` state:
          - NOT done (backlog / ready-for-dev / in-progress / review = a "draft"
            that has NOT yet shipped against the old contract) → STALE-MARK it.
            Add to the downstream story's Dev Notes AND its sprint-status entry:
              ⚠️ STALE [@contract-vN→v(N+1)]: upstream {X}-{Y} AC #{N} bumped
              {date}; re-confirm against v(N+1) before dev — see {X}-{Y} Change Log.
            The downstream owner re-acks (or files a correct-course) before the
            story leaves draft. The sprint-status line keeps it from being picked
            up blind.
          - done → FROZEN; do NOT retro-stale-mark. It shipped against the contract
            live at the time (forward-only, same philosophy as the v0 retrofit —
            history is immutable).
       3. If the grep returns 0 downstream hits, write `no downstream consumers`
          into the bump's Change Log row so the next reviewer sees the scan ran.
  ❌ Bumping a stamp without a Change Log entry, OR without running the
     downstream grep + stale-marking every not-done consumer. A Change-Log row
     alone is INSUFFICIENT — it protects the upstream's own history, not the
     orphaned downstream draft (the exact 19-6 v1→v2 / 19-7 failure).
  🔎 Grep helpers (shared by DEV and CR workflows):
       # List every stamped AC in a single story file
       grep -nE '\[@contract-v[0-9]+\]' <story_file>
       # List all stamped ACs across implementation artifacts
       grep -rnE '\[@contract-v[0-9]+\]' _bmad-output/implementation-artifacts/
       # Find downstream stories acknowledging upstream contracts (backtick-tolerant)
       grep -rnE 'confirmed against `?\[@contract-v[0-9]+\]' _bmad-output/implementation-artifacts/
       # Bump-side: downstream drafts that acked a SPECIFIC upstream AC (pipe to sprint-status lookup)
       grep -rnE 'confirmed against `?\[@contract-v[0-9]+\]' _bmad-output/implementation-artifacts/ | grep -E 'Story <X>-<Y>.*AC #<N>'
  🔎 Backtick-tolerance (added retro-19-P3, 2026-06-02 — origin bugfix-19-4b-1-followup CR H1):
     The two "confirmed against" helpers above match the literal `[` immediately
     after `confirmed against `. An ack written with the TOKEN wrapped in inline-code
     backticks — `confirmed against `[@contract-v1]` (Story X-Y AC #N)` — inserts a
     backtick between `against ` and `[`, which defeats the un-tolerant pattern and
     makes the downstream ack INVISIBLE to the bump-side stale-mark grep → the exact
     silent-strand class Rule 20's 🔁 obligation exists to prevent. The `` `? ``
     (zero-or-one literal backtick; safe inside the single-quoted regex) absorbs that
     case. The bare-token helpers above need no change — `\[@contract-v[0-9]+\]`
     already matches `[@contract-v1]` regardless of surrounding backticks; only the
     `confirmed against`-adjacent form has the adjacency break.
  📌 Precedent (Epic 10 Retro AI-5, spike 2026-04-22): Pattern #2 from
     Epic 10 retro — cross-story AC drift recurred 3 times across 3 epics.
     retro-10-AI2 AC Drift Check caught story-ID references; this rule
     closes the contract-shape gap. Spike doc:
     `_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md`.
     CR follow-up (2026-04-22): grep helpers hoisted into Rule 20 body
     (was AC #4 partial), Change Log format unified to `{what changed, what
     breaks downstream}` across all three canonical sources, v0 fallback
     documented for pre-Rule-20 upstream references.
  📌 Precedent (retro-19-P2, 2026-06-02): the 🔁 bump-side stale-mark sub-section.
     Origin: 19-6's `[@contract-v1→v2]` bump (live per-case credit check dropped
     for pre-run slicing) silently stranded 19-7's already-drafted story, which had
     acked 19-6 [@contract-v1] AC #2/#4/#5/#6 — the stale draft surfaced only at
     19-7 implementation and forced a full pre-impl re-architecture via correct-course
     (`_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-20.md`). The
     consumer-side ack rule could not catch it because nothing re-scans a draft when
     ITS upstream moves. CR sync: the code-review workflow gained a MANDATORY
     "RULE 20 CONTRACT BUMP STALE-MARK CHECK" action in Step 3 (mirrors the Rule 7
     wire-format check pattern) — last synced 2026-06-02.
```

### Rule 21: Component-to-Design Node Traceability

```typescript
// ✅ CORRECT — component file header references .pen node
// Implements: Component/PosterCardHover (MQbvp)
// Source: ux-design.pen (Pencil app)
export function PosterCard({ ... }: PosterCardProps) { ... }

// ❌ WRONG — no link from code to design source of truth
export function PosterCard({ ... }: PosterCardProps) { ... }
```

```
Every file under apps/web/src/components/ that renders a designed UI
element MUST include a header comment referencing its .pen node ID:

  // Implements: Component/{Name} ({pen-node-id})

Where:
  {Name}        = the .pen reusable-component name (e.g., PosterCardHover)
  {pen-node-id} = the unique Pencil node identifier (e.g., MQbvp)

Lookup: query .pen via Pencil MCP `get_editor_state` — every reusable
component is listed with its node ID under "Reusable Components".

Multi-component: a file rendering more than one designed component joins
them with ` + ` on one line:
  // Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)

Exemptions / placeholders (annotate explicitly so absence is intentional, not accidental):
  - Pure layout/utility components:  // Implements: <utility — no .pen counterpart>
  - One-off route-level wrappers:     // Implements: <route-only>
  - Renders a section of a designed *screen frame* (not a Reusable Component):
                                      // Design ref: ux-design.pen Screen {ScreenName} ({nodeId})
    (Phase-2 form, LIVE since story 19-8 — a 19-3 [@contract-v3] grammar bump; the
    ESLint rule accepts it. A design-coverage-gap variant covers components whose
    feature postdates the design:
                                      // Design ref: ux-design.pen — no current screen frame; {reason})
  - First-pass backfill placeholder (kept for any future backfill before a screen
    mapping is decided):           // Implements: <screen-section — pending epic-{N}-{M} mapping>
    As of story 19-8 (2026-05-20) NO `components/` file should still carry the
    pending placeholder — the 19-8 sweep upgraded all 94 such files to the
    `// Design ref:` form; see `_bmad-output/audit/drift-sweep-2026-05.md`.
    Do NOT use either screen-section form for components that genuinely have no
    design — those are `<utility — no .pen counterpart>`.
  - Tests, hooks, services, stores:   exempt (no annotation required)

Enforcement:
  Phase 1 (manual, ships with Rule 21): SM /create-story template REQUIRES
  the Implements: line in story Dev Notes. Stories without it are bounced.
  Phase 2 (automated, LIVE since story 19-3): the custom ESLint rule
  `local/implements-pen-node-id` (apps/web/src/eslint-rules/implements-pen-node-id.js,
  wired in eslint.config.mjs) errors on missing or malformed `// Implements:`
  headers under apps/web/src/components/**/*.{ts,tsx} (spec/test files and
  index.ts barrels excluded). Runs inside `eslint .` ⇒ `pnpm lint:all` ⇒ CI.
```

📌 Precedent (Party Mode 2026-05-08, bugfix-10-4 root cause):
HoverPreviewCard.tsx was independently invented, diverging from the
.pen designed Component/PosterCardHover (node MQbvp). The drift went
undetected for months because there was no link from code back to the
design source of truth. Sally + Bob + Winston + Amelia + Murat
consensus: epic-19 (cross-cutting Design-Implementation Drift Audit
initiative) makes Rule 21 an enforceable code-level invariant. .pen
file is the design contract; this rule is the bridge from code to that
contract.

### Rule 22: Epic Retro Design-Drift Audit

```
Every epic retrospective MUST include a Design-Drift Audit action item:

  1. SAMPLE: pick ≥5 components changed during the epic
  2. COMPARE: render each component (storybook/dev) → screenshot →
     diff against .pen node via Pencil MCP `get_screenshot`
  3. CLASSIFY each component:
     - ✅ exact-match    : pixel diff < 0.5%
     - ⚠️  minor drift    : 0.5–5% diff (typography, spacing micro-shifts)
                          — log only
     - ❌ material drift  : >5% diff OR structural change (different
                          layout, missing elements) — file as bugfix-N-X
  4. RECORD: write findings to `_bmad-output/audit/drift-{epic}-{YYYY-MM}.md`
  5. ESCALATE: any material drift becomes a tracked bugfix story
     (use `bugfix-10-6-polish-ux-visual-pass` precedent to bundle small drifts)

Sample-pick policy:
  - Always include the most-touched component of the epic
  - Always include any component with hover/focus state changes
  - Random-pick remaining slots from the epic's File List
  - OVERRIDE — full sweep: story 19-8 (2026-05-20) examined ALL 131
    components/ files, not a ≥5 sample. Future retros MUST NOT cite 19-8
    as the ≥5-sample precedent — it is the explicit full-sweep capstone
    exception (see `_bmad-output/audit/drift-sweep-2026-05.md`).

Audit history: each epic's audit doc is the durable record for
post-mortem and trend tracking. If 3 consecutive epics show <2 material
drifts, audit moves to spot-check mode (random 3 components per epic).

Tooling (LIVE since story 19-4; wired into PR-blocking CI via the
`Visual Regression` GitHub Actions workflow since story 19-5): the
Playwright `visual` project (`playwright.config.ts`) + `pnpm run test:visual`
automate diff calculation — it drives the dev-only component gallery route
`apps/web/src/routes/test/gallery.tsx` (`/test/gallery`), screenshots each
component's default/hover/focus state, and compares against committed
baselines under `tests/visual/components.visual.spec.ts-snapshots/`
(`maxDiffPixelRatio` ≈0.1%). Each baseline carries a `data-pen-node` link to
its `ux-design.pen` node (per the `// Implements:` header — Rule 21).
Coverage rollout: 19-4 = harness + ~25 reference components; 19-4b = the
remaining ~99 (122 components / 123 fixtures / 262 PNGs at 19-4b close); see
`_bmad-output/audit/visual-baseline-19-4.md`. CI enforcement: story 19-5
landed `.github/workflows/visual-regression.yml` — PR job fails on any
visual diff (PR-blocking once the branch-protection rule is enabled), main
push runs the full suite, first main-push self-bootstraps the `-linux`
baseline set via a `requires-manual-review` PR (Sally re-engagement gate).
Classification (exact / minor / material per Step 3) + escalation remain
human judgment. Full-sweep precedent: story 19-8 (2026-05-20) applied this
tooling to ALL 131 `components/` files — audit doc
`_bmad-output/audit/drift-sweep-2026-05.md`. Outcome: 0 material drift,
2 minor (log-only) — confirming the bugfix-10-4 drift was isolated, not
systemic.

Rule 22 covers the SPATIAL dimension of visual-baseline correctness
(design-vs-code). For the TEMPORAL dimension (clock-window-vs-fixture) see
Rule 23 below — every `components/**` file that reads the wall clock MUST
be paired with a clock-mocked fixture so a moving real-world date can't
silently stale a committed baseline.
```

📌 Precedent: Party Mode 2026-05-08. bugfix-10-4 root cause showed
design-implementation drift was systemic, not isolated. Rule 22 turns
drift detection from reactive (user reports a bug) to proactive
(caught at retro). Mirrors retro-10-AI checklist precedents but
specifically targets the design ↔ code gap.

---

### Rule 23: Time-Dependent Component Fixture Stability

```
Every file under `apps/web/src/components/**/*.{ts,tsx}` whose source body
contains an unwrapped wall-clock read MUST be paired with a deterministic
clock at visual-baseline capture time. The wall clock moves; committed
PNG baselines don't. Without the pairing, every baseline silently stales
the moment the real-world date crosses a state boundary the component
reacts to (e.g. `isWithin7Days`, `isOverdue`, `daysSinceCreated`).

Trigger criteria (file is in scope if ANY is true):
  (a) source body contains `Date.now()` / `Date.UTC()` / `Date.parse()`
      MemberExpression OR `new Date()` NewExpression with zero args.
  (b) file calls a utility that transitively does (a) AND the utility's
      return value affects rendered output. (Judgment call documented per
      file in the AC #3 audit doc `_bmad-output/audit/time-bomb-fixtures-*.md`;
      pure formatter utilities like `formatDate(d)` that take their date
      as an argument are NOT in scope — only "ambient now" reads.)
  (c) file branches conditionally on a date comparison whose result
      changes as wall-clock time advances (the `library-recently-added`
      `isWithin7Days` precedent — 19-8 PR #8 visual-regression class).
  (d) NOT in scope: hooks/services/stores under `components/` that never
      render to the DOM (e.g. `parse/useParseProgress.ts` — Murat's call
      per party-mode 2026-05-26). Rule 23 is component-UI-only.

Requirement (one of three accepted forms; declared via leading-comment
header BEFORE imports, alongside the Rule 21 marker — two-line header):

  (i) `// Clock-mocked: gallery fixture {fixture-id} uses page.clock.setFixedTime`
      The component reads ambient time; the gallery fixture for it
      sets `clockTime: '{iso}'` and `tests/visual/components.visual.spec.ts`
      pins the clock via the Rule-23-canonical helper
      `withFixedClock(page, iso)` (Story 19-9 AC #4).

  (ii) `// Clock-injected: component accepts `clock` prop; no fixture-side mock needed`
      The component itself dodges the time-bomb via dependency injection
      — accepts a `clock` (or `now`) prop/context whose default is
      `Date.now()` but whose tests pass a fixed value. No fixture-side
      mock required; single baseline is sufficient.

  (iii) `// Time-bomb-exempt: <one-line rationale>`
      Explicit acknowledged exemption (e.g. debug-only display that
      doesn't affect visual baselines; ambient timestamp shown but not
      pixel-asserted). Rationale MUST identify the reviewer (Sally for
      visual-state calls, Murat for test-architecture calls).

Coverage — ≥2 fixture states per component with time-dependent branching:
  Components whose render OUTPUT differs across the boundary (the
  `library-recently-added` case: `recent` shows the 綠 "新增" badge,
  `stale` doesn't) MUST be visually baselined in BOTH branches. The
  19-4/19-4b harness's "one state per fixture" practice is the loophole
  this requirement closes. State names follow the `{gallery-id}/{state}-visual-{platform}.png`
  convention (Story 19-4 AC #5); the canonical default state pair for
  time-dependent components is `recent` / `stale`, though component-
  specific names are allowed when the branching is naturally named
  differently (e.g. `overdue` / `on-time`).

Enforcement (LIVE since story 19-9):
  ESLint rule `local/time-dependent-fixture-stability` at
  `apps/web/src/eslint-rules/time-dependent-fixture-stability.js`,
  wired in `eslint.config.mjs` scoped to `apps/web/src/components/**/*.{ts,tsx}`
  (excluding `*.spec.*` + `index.ts` barrels — same scoping as Rule 21
  ESLint rule). Rule errors on any in-scope file containing the AST
  shapes from criterion (a) unless one of the three header forms above
  is present as a leading comment.

Tooling:
  - Helper `tests/visual/clock-mock.ts` exports `withFixedClock(page, iso)`
    wrapping Playwright `page.clock.install({ time, shouldAdvance: false })`
    (Playwright ≥1.45; this repo pins 1.57.0).
  - Fixture-row optional `clockTime: '{iso}'` field at
    `apps/web/src/routes/test/-gallery.fixtures.tsx`; when present,
    `components.visual.spec.ts` calls the helper before each `toHaveScreenshot`.
  - Audit doc `_bmad-output/audit/time-bomb-fixtures-{YYYY-MM}.md` —
    durable record of every candidate file's classification + disposition.
    Future re-scans may compare against this baseline to surface "new
    time-bomb candidates added since 19-9 closed".
```

📌 Precedent: Party Mode 2026-05-26 (Murat + Sally + Winston consensus
3-0, Alexyu picked option D). Origin: Story 19-8 PR #8 `Visual Regression
/ PR` check failure on `components/library-recently-added/default`
baseline — the fixture's `createdAt: '2026-05-12'` was within 7 days
when the baseline PNG was captured, but the next baseline run crossed
the 7-day boundary and the green "新增" badge disappeared. The drift
wasn't design (Rule 21+22 caught nothing — Sally approved the rendered
state, the `.pen` design matches) and wasn't a regression in the
component — it was the harness assuming a frozen wall clock when the
component reads `new Date()` ambient time. Three lessons compounded:
(a) 19-4b CountdownTimer learned "pin the date" via `nextAttemptAt:
'2020-...'` but didn't generalize; (b) 19-4/19-4b only baselined ONE
state per fixture (Sally's admission: "I only saw one state when I
approved"); (c) 19-8 sweep examined 131 files for design-vs-code drift
and found zero, but didn't grep for `Date.now()`. Rule 23 closes all
three: AST-level ESLint enforcement (catches "I forgot to mock the
clock"), ≥2-state baseline requirement (catches "I only saw one
state"), audit-doc trend tracking (catches "we knew about this class
of bug but it crept back"). Three orthogonal Rules now cover all
observed visual-baseline drift classes: Rule 21 (spatial — design-vs-
code), Rule 22 (cadence — per-epic retro classification), Rule 23
(temporal — wall-clock-vs-fixture).

---

### Rule 24: Discovery Triage

```
The MOMENT a story (dev-story, quick-dev, bugfix, or PR review) discovers
work that is NOT in its current scope, the discoverer MUST classify it
into EXACTLY ONE of three lanes — at the moment of discovery, not "later":

  ① expand-scope-in-place
       Absorb into the CURRENT story. Allowed only when the work is small,
       on the same surface, and within the story's acceptance-criteria
       spirit. REQUIRED side-effect: add an AC or sub-task to the current
       story file so the absorbed work is itself tracked — "I just fixed it
       quietly" is NOT this lane, it is an untracked change.

  ② spawn-blocking-story
       The discovered work BLOCKS this story's correct completion. Create a
       new tracked story entry in `sprint-status.yaml` immediately, mark the
       current story `blocked` (or note `blocked-by: {new-id}`), and sequence
       the new story ahead. The current story does NOT close until ② resolves.

  ③ backlog-with-carry-forward-link
       The work is real but out-of-scope AND non-blocking. File a `backlog`
       (or `bugfix-N`) entry in `sprint-status.yaml` AT THE MOMENT OF
       DISCOVERY, with a bidirectional link: the entry names the discovering
       story, and the discovering story's Completion Notes names the entry ID.

THE BAN — "mentioned in prose but not in sprint-status":
  Any out-of-scope finding that appears ANYWHERE in narrative — story Dev
  Notes, Completion Notes, a PR description, a retro write-up, a code comment
  TODO — MUST have a corresponding `sprint-status.yaml` entry (lane ② or ③)
  or be absorbed under lane ①. A finding that lives ONLY in prose is invisible
  to sprint planning and becomes a deferred-discovery time-bomb. If it is worth
  writing down, it is worth a tracked entry. No exceptions.

Recording requirement:
  A story that triages ANY discovery MUST enumerate, in its Completion Notes,
  every discovery + its lane (①/②/③) + the resulting tracked entry ID
  (for ② / ③) or the added AC number (for ①). A retro that finds a bugfix
  traceable to an un-triaged prose mention files it as a process miss.

Enforcement:
  Phase 1 (manual, ships with Rule 24): the SM `/create-story` template gains
  a "Discovery Triage" field in the Completion / Dev Notes section — the dev
  records each in-flight discovery's lane + tracked-entry ID before the story
  can be marked done. Stories that mention out-of-scope work in prose without
  a matching sprint-status entry are bounced at review.  [SM Bob — template
  edit is the paired retro-19-P1 deliverable; pending at Rule-24 authoring.]
```

📌 Precedent: epic-19 retro 2026-05-29 (retro-19-P1), Alexyu pain point B —
"the gap between discovery and resolution spanned a whole story." Every one
of epic-19's 3 bugfix follow-ups traced to a finding the originating story
flagged but did not fully resolve (the find→fix gap). The canonical chain:
19-4b CountdownTimer learned the half-lesson → 19-7/19-8 surfaced the
time-bomb class in PR-stage prose → it became actual work only in 19-9 →
which then spawned bugfix-19-9. Alexyu's ruling: adopt a FORCED
classification discipline (not the softer "lean toward absorbing in place").
Rule 24 generalizes retro-8-P1 ("ALL retro action items become tracked
entries, no exceptions") from end-of-epic retros to the moment of mid-story
discovery — the find→fix gap is the dominant velocity drag, and forced triage
at discovery time is the direct counter.

📌 Superseded-mechanism corollary (bugfix-20-1): when a migration introduces a
REPLACEMENT for an existing mechanism (a new table/column/service that supersedes
an old one), the SAME forced triage applies AT THAT MOMENT — the old mechanism's
retirement AND re-pointing every reader to the new one must be classified into a
lane (①/②/③), never left dual-living. Two storage mechanisms for one concept is a
deferred-discovery time-bomb. Origin: the `seasons` table (mig 015) superseded the
`series.seasons` JSON column (mig 006), but `GetSeasons` was never re-pointed and
the dead column lingered, so the season accordion silently broke until the Phase-2
real-data test surfaced it (bugfix-20-1).

### Rule 25: Mega-line Rebase Conflict Resolution

```
The "Last Updated:" line at the TOP of this file (project-context.md L7) is a
SINGLE physical line — a newest-first running log where every story / retro /
rule change PREPENDS one `(...)` entry, demotes the former lead to `Prior: ...`,
and the oldest tail uses `Earlier: ...`. Because it is ONE line that almost
every branch touches, parallel branches routinely conflict on it at rebase /
merge time. This rule governs ONLY how that conflict is resolved.

THE BAN — no whole-side takeover:
  NEVER resolve a conflict on this line by accepting one entire side. That
  means NONE of:
    - `git checkout --ours project-context.md` / `--theirs`
    - editor "Accept Current Change" / "Accept Incoming Change"
    - deleting one of the `<<<<<<<` / `=======` / `>>>>>>>` halves wholesale
  Each side has PREPENDED its own entry; taking one side silently DROPS the
  other side's entry. That is the exact 19-8 failure (see Precedent).

THE RULE — union, newest-first:
  The resolution MUST keep BOTH sides' new entries:
    1. The branch being rebased ON TOP keeps its entry as the `Last Updated:`
       lead (it is chronologically later).
    2. The already-landed branch's former lead entry is DEMOTED to a
       `Prior: {date} (...)` entry, inserted immediately after the new lead.
    3. The shared older `Prior:` / `Earlier:` tail (everything both sides
       inherited from the merge base) is kept EXACTLY ONCE — do not duplicate it.
  If both entries carry the same date, order them by logical sequence
  (the change last-authored / rebased-on-top → lead).

VERIFICATION (before staging the resolved file):
  - Entry count after merge MUST be ≥ max(count on each side). It can never
    shrink — a shrink means an entry was dropped.
  - grep the resolved line for BOTH conflicting story / retro IDs; both MUST
    be present.
  - Re-run `pnpm exec prettier --check project-context.md`. Keep every entry
    English-only: a CJK character anywhere in this mega-line makes prettier
    reflow the entire line, masking a dropped entry inside a noisy diff.
```

📌 Precedent: epic-19 retro 2026-05-29 (retro-19-P4). Story 19-8's
"Last Updated" entry was silently dropped during a 2026-05-28 rebase-conflict
resolution — the conflict was settled by a whole-side takeover, discarding the
other branch's prepended entry. It was caught only by a later adversarial CR
(H1), not at resolution time. Rule 25 turns that into a banned anti-pattern
plus a count-and-grep verification, and the CR workflow gains a paired
mega-line merge check (`code-review/instructions.xml` Step 3).

### Rule 26: TanStack Router Search-Param Coercion (lone-numeric trap)

```typescript
// TanStack Router's DEFAULT search parser JSON-parses each query value. A param
// holding a SINGLE numeric value (`?genre=16`, `?platform=8`) is parsed into a
// `number`, NOT a string. A validateSearch guard written `typeof x === 'string'`
// is then false for the lone-numeric case and SILENTLY DROPS the param — the
// deep link looks unfiltered. A multi-value form (`?genre=16,28`) stays a string
// and slips through, so the bug shows ONLY on single-value deep links and is
// easy to miss in manual testing.

// ✅ CORRECT — coerce number → string before the guard (CSV-string params)
function toCsvString(value: unknown): string | undefined {
  if (typeof value === 'string') return value || undefined;
  if (typeof value === 'number' && Number.isFinite(value)) return String(value);
  return undefined;
}
validateSearch: (search) => ({ genre: toCsvString(search.genre) }),

// ❌ WRONG — lone `?genre=16` arrives as number 16, guard false, param dropped
validateSearch: (search) => ({
  genre: typeof search.genre === 'string' ? search.genre : undefined,
}),
```

Applies to every `validateSearch` under `apps/web/src/routes/**` whose param can
be written as a bare number in a deep link (CSV id-lists: genre, platform, person
ids; any single-id filter). Canonical helper:
`apps/web/src/routes/discover.tsx::toCsvString`. Genuinely-numeric params use a
`toOptionalNumber`-style coercion instead; a string-enum guard (e.g.
`subtitleStatus`) is only safe when the value
can NEVER be all-digits.

📌 Precedent: recurred twice. Story 11-2 (persistent filter chip UI) — `?genre=16`
/ `?platform=8` single-value deep links silently dropped the filter; fixed with
`toCsvString()` + defensive `String()` coercion, E2E-guarded (CR caught it as a
HIGH). Story 8-11 (batch subtitle UI) — same class on the `subtitleStatus` param.
Two strikes across two epics → codified here per Epic 11 Retro Insight 3 / Rule 22
(codify a framework gotcha the moment it hits twice).

---

### Rule 27: External Integration Standard (the Five Pillars)

Any code path that calls a third-party network service — or surfaces third-party
data to a story — MUST provide all FIVE pillars. This is the convention every
Epic 12 external integration (F-3 recommendations, F-4 watch providers, F-5
trailers, F-6 Douban) and every future external integration follows. It codifies
the shape the existing `internal/tmdb` / `internal/douban` / `internal/wikipedia`
clients already implement — it is a convention, NOT a new shared package (YAGNI;
see ADR Decision 3).

```
① RATE LIMIT   one *rate.Limiter per upstream, built once at client init, reused
               for process life (Rule 14). limiter.Wait(ctx) is the FIRST line of
               the request method. Limit = upstream's published/observed ceiling,
               named + commented (e.g. tmdb requestsPerInterval=40 / 10s).
② CACHE        tiered per AD #4, key {source}:{type}:{id}:{version}. Cache is
               checked BEFORE the limiter (a hit never waits) — this is what keeps
               the detail page < 1.5s on warm content. TTL by volatility (TMDB
               recs/providers 24h; Douban 24h).
③ DEGRADE      external data is ENRICHMENT, not core content → fail-soft, never
               fail-page. Per-section isolation (one dead source hides only its
               section); bounded exp-backoff retry (1s→2s→4s→8s, ctx-aware,
               non-retryable 404/auth fail fast); stale-on-error; scrapers reuse
               the existing client's robots.txt / UA-rotation / `enabled` switch.
④ ERROR CODES  Rule 7 {SOURCE}_{ERROR_TYPE} for backend-originated failures.
               REUSE existing prefixes — F-3/F-4 → TMDB_*, F-6 → DOUBAN_*. Add new
               codes only under the EXISTING prefix + sync code-review Rule-7 grep.
               NO new prefix for Epic 12. (F-5 makes no backend call → no code.)
⑤ KEYS         API keys via ClientConfig from settings/env, never hardcoded/
               committed. Log only through slog (inherits sanitizeAttr — strips
               api_key/key/token, AD #6). Epic 12 adds NO new secret.
```

📌 **Reuse over re-invent.** Three of four Epic 12 integrations ride the
already-shared `internal/tmdb` client (F-3/F-4 add endpoint wrappers; F-5's
`GetMovieVideos`/`GetTVShowVideos` already exist) — they add ZERO new limiter,
key, quota, or error prefix. F-6 routes through the existing `internal/douban`
scraper. **F-5 YouTube = client-side `youtube-nocookie.com/embed/{key}` iframe
using the key TMDB already returns — no backend YouTube call, no YouTube Data API
key** (ADR Decision 4). A NEW external client is justified only when no existing
one fits; a shared `internal/externalapi` base is deferred until a third
independent client re-hand-rolls the limiter+cache+fallback triplet.

📌 Spec: `_bmad-output/planning-artifacts/architecture/adr-external-api-integration-standard.md`
(retro-11-AI3). Origin: Epic 11 Retro AI-3 — F-3..F-6 each integrate a third-party
API and would otherwise re-invent rate-limit / cache / degradation / key-mgmt four
times; codified once here per Epic 11 Retro Insight 1 (codified checks stick,
passive docs rot).

---

## 🧪 Known dev-mode artifacts

Behaviors that look like bugs in `pnpm nx serve web` but DO NOT reproduce in
`pnpm nx run web:preview` (production build). Do not chase these in dev —
verify in preview first. Each entry must link to a spike that proves the
prod-vs-dev diff.

### Homepage skeleton "flicker" on cold load

```
Symptom (dev only):
  Homepage section skeletons (HeroBanner, ExploreBlocksList,
  RecentMediaPanel, DownloadPanel) appear to flash twice during initial
  load in `pnpm nx serve web`.

Cause:
  React 18 <StrictMode> wrapper at apps/web/src/main.tsx:11 intentionally
  double-invokes component bodies, useState initializers, and useEffect
  setup+cleanup in dev to surface side-effect bugs. The fiber-level
  double-render can produce a visible double-paint of pre-commit elements.
  StrictMode is a no-op at runtime in production builds.

Verification:
  Spike (2026-05-07) ran a Playwright probe with deterministic 100ms-mocked
  endpoints against both `nx serve web` (port 4200, StrictMode active) and
  `vite preview` (port 4201, prod build). Two probes:
    1. 50-450ms snapshot poller of skeleton testid counts
    2. Frame-level MutationObserver log of every mount/unmount
  Result: every tracked testid showed IDENTICAL mount/unmount counts in
  dev and prod (Δ=0). All skeleton sequences were monotonically
  non-increasing (0→1→0, no re-mount) in BOTH modes. Verdict: Bucket A —
  dev-mode-only artifact, no real prod regression.

Fix:
  None. This entry IS the fix per AC #10 of the spike-gated story.
  Do not add a regression test for a non-existent prod bug — it would be
  permanent dead weight.

Reference:
  _bmad-output/implementation-artifacts/spike-bugfix-10-3-findings.md
  _bmad-output/implementation-artifacts/spike-bugfix-10-3-{dev,prod}-{snapshots,mutations}.json
  _bmad-output/implementation-artifacts/bugfix-10-3-skeleton-flicker-on-load.md
```

How to add a new entry: when investigating a "looks like a bug" report,
ALWAYS verify in `web:preview` before opening a story. If the symptom
disappears in prod, file a doc entry here with a spike artifact and skip
the fix code. Adding speculative fixes for dev-only artifacts pollutes the
codebase with permanent test scaffolding for non-bugs.

---

## 🏗️ Project Structure

```
vido/
├── apps/
│   ├── api/                    # ⭐ SINGLE BACKEND (unified)
│   │   ├── main.go
│   │   ├── internal/
│   │   │   ├── handlers/       # HTTP handlers (Gin)
│   │   │   ├── services/       # Business logic
│   │   │   ├── repository/     # Data access (Repository pattern)
│   │   │   ├── models/         # Domain models
│   │   │   ├── middleware/     # HTTP middleware
│   │   │   ├── tmdb/           # TMDb API client
│   │   │   ├── ai/             # AI provider abstraction
│   │   │   ├── parser/         # Filename parser
│   │   │   ├── cache/          # Cache manager
│   │   │   ├── tasks/          # Background task queue
│   │   │   ├── plugins/        # Plugin interfaces and manager
│   │   │   ├── sse/            # Server-Sent Events hub
│   │   │   ├── subtitle/       # Subtitle engine pipeline
│   │   │   ├── scanner/        # Media library scanner
│   │   │   ├── errors/         # Unified AppError
│   │   │   └── logger/         # slog config
│   │   ├── migrations/         # SQLite migrations
│   │   └── .air.toml
│   │
│   └── web/                    # Frontend (React)
│       ├── src/
│       │   ├── routes/         # TanStack Router
│       │   ├── components/     # Feature-organized
│       │   │   ├── search/
│       │   │   ├── library/
│       │   │   ├── downloads/
│       │   │   └── ui/         # Shared UI
│       │   ├── hooks/          # Custom hooks
│       │   ├── services/       # API clients
│       │   ├── stores/         # Zustand (UI state only)
│       │   └── utils/
│       └── tailwind.config.js
│
├── libs/
│   └── shared-types/           # TypeScript types
│
├── archive/                    # ⚠️ DEPRECATED (old root backend)
│   ├── cmd/
│   └── internal/
│
├── project-context.md          # ⭐ THIS FILE
└── _bmad-output/
    └── planning-artifacts/
        └── architecture/       # Complete architecture doc (sharded)
            └── index.md
```

---

## 📝 Naming Conventions Quick Reference

### Database (SQLite)

| Element     | Pattern                | Example                       | ❌ Anti-pattern         |
| ----------- | ---------------------- | ----------------------------- | ----------------------- |
| Tables      | snake_case plural      | `movies`, `media_files`       | `Movies`, `movie`       |
| Columns     | snake_case             | `tmdb_id`, `created_at`       | `tmdbId`, `createdAt`   |
| Primary Key | `id`                   | `id TEXT PRIMARY KEY`         | `movie_id`              |
| Foreign Key | `{table}_id`           | `library_id`, `movie_id`      | `fk_library`, `movieId` |
| Indexes     | `idx_{table}_{column}` | `idx_movies_tmdb_id`          | `movies_tmdb_index`     |
| Migrations  | `{seq}_{desc}.sql`     | `001_create_movies_table.sql` | `create-movies.sql`     |

### Backend (Go)

| Element    | Pattern              | Example                         | ❌ Anti-pattern             |
| ---------- | -------------------- | ------------------------------- | --------------------------- |
| Packages   | lowercase singular   | `tmdb`, `parser`, `cache`       | `tmdb_client`, `Middleware` |
| Structs    | PascalCase           | `Movie`, `TMDbClient`           | `movie`, `tmdbClient`       |
| Interfaces | PascalCase           | `Repository`, `Cache`           | `IRepository`               |
| Functions  | PascalCase/camelCase | `GetMovieByID`, `parseFilename` | `get_movie_by_id`           |
| Files      | snake_case.go        | `tmdb_client.go`                | `TMDbClient.go`             |

### Frontend (TypeScript/React)

| Element          | Pattern         | Example                       | ❌ Anti-pattern           |
| ---------------- | --------------- | ----------------------------- | ------------------------- |
| Components       | PascalCase      | `SearchBar`, `MovieCard`      | `searchBar`, `search-bar` |
| Component Files  | PascalCase.tsx  | `SearchBar.tsx`               | `search-bar.tsx`          |
| Hooks            | use + camelCase | `useSearch`, `useLibrary`     | `UseSearch`, `searchHook` |
| Hook Files       | use{Name}.ts    | `useSearch.ts`                | `search.hook.ts`          |
| Types/Interfaces | PascalCase      | `Movie`, `ApiResponse<T>`     | `IMovie`, `movieType`     |
| Constants        | SCREAMING_SNAKE | `API_BASE_URL`, `MAX_RETRIES` | `apiBaseUrl`              |

### API Endpoints

| Element | Pattern                    | Example                        | ❌ Anti-pattern              |
| ------- | -------------------------- | ------------------------------ | ---------------------------- |
| Paths   | /api/v{version}/{resource} | `/api/v1/movies`               | `/movie`, `/getMovies`       |
| Methods | RESTful                    | `GET`, `POST`, `PUT`, `DELETE` | `POST /api/v1/movies/update` |
| Params  | {param_name}               | `/api/v1/movies/{id}`          | `/api/v1/movies/:id`         |
| Query   | snake_case                 | `?sort_by=release_date`        | `?sortBy=releaseDate`        |

---

## 🔧 Error Handling Pattern

### Backend (Go)

```go
// Step 1: Create AppError
func (s *MovieService) GetMovieByID(ctx context.Context, id string) (*Movie, error) {
    movie, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, NewDBNotFoundError(err) // AppError
        }
        return nil, NewDBQueryError(err)
    }
    return movie, nil
}

// Step 2: Log with slog
func (h *MovieHandler) GetMovie(c *gin.Context) {
    movie, err := h.service.GetMovieByID(c.Request.Context(), id)
    if err != nil {
        var appErr *AppError
        if !errors.As(err, &appErr) {
            appErr = NewInternalError(err)
        }

        slog.Error("Failed to get movie",
            "error_code", appErr.Code,
            "movie_id", id,
            "error", err,
        )

        ErrorResponse(c, appErr)
        return
    }

    SuccessResponse(c, movie)
}
```

### Frontend (TypeScript)

```typescript
const { data, error, isError } = useQuery({
  queryKey: ['movies', 'detail', movieId],
  queryFn: () => movieService.getMovie(movieId),
  onError: (error: ApiError) => {
    toast.error(error.message, {
      description: error.suggestion,
    });
    console.error(`[${error.code}]`, error.details);
  },
});

if (isError) {
  return <ErrorMessage error={error} />;
}
```

---

## 🔄 State Management Pattern

### Server State (TanStack Query) ✅

```typescript
// Query keys with hierarchy
const movieKeys = {
  all: ['movies'] as const,
  lists: () => [...movieKeys.all, 'list'] as const,
  list: (filters: string) => [...movieKeys.lists(), { filters }] as const,
  details: () => [...movieKeys.all, 'detail'] as const,
  detail: (id: string) => [...movieKeys.details(), id] as const,
};

// Usage
const { data: movie } = useQuery({
  queryKey: movieKeys.detail(movieId),
  queryFn: () => fetchMovie(movieId),
});
```

### Global Client State (Zustand) - UI State ONLY

```typescript
// ✅ ONLY for UI state, NOT server data
interface UIState {
  sidebarOpen: boolean;
  viewMode: 'grid' | 'list';
  toggleSidebar: () => void;
  setViewMode: (mode: 'grid' | 'list') => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: true,
  viewMode: 'grid',
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
  setViewMode: (mode) => set({ viewMode: mode }),
}));
```

### Local Component State (useState)

```typescript
// ✅ Form inputs, toggles, local UI state
const [isOpen, setIsOpen] = useState(false);
const [searchTerm, setSearchTerm] = useState('');
```

---

## 🧪 Testing Patterns

### Backend (Go)

```go
// movie_handler_test.go (co-located with movie_handler.go)

func TestMovieHandler_GetMovie(t *testing.T) {
    tests := []struct {
        name       string
        movieID    string
        wantStatus int
        wantError  string
    }{
        {
            name:       "success",
            movieID:    "valid-id",
            wantStatus: http.StatusOK,
        },
        {
            name:       "not found",
            movieID:    "invalid-id",
            wantStatus: http.StatusNotFound,
            wantError:  "DB_NOT_FOUND",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Frontend (TypeScript)

```typescript
// MovieCard.spec.tsx (co-located with MovieCard.tsx)

import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MovieCard } from './MovieCard';

describe('MovieCard', () => {
  it('renders movie title', async () => {
    const queryClient = new QueryClient();
    render(
      <QueryClientProvider client={queryClient}>
        <MovieCard movieId="test-id" />
      </QueryClientProvider>
    );

    expect(await screen.findByText('Test Movie')).toBeInTheDocument();
  });
});
```

---

## 🧹 Test Process Cleanup

### Process Lifecycle Rule

**All test-related child processes MUST terminate when the parent process exits.** This applies to:

- **Unit tests (Vitest):** Uses `pool: 'forks'` so workers are child processes that can be force-killed on exit. `teardownTimeout: 5000` prevents indefinite hangs from uncleaned timers/listeners.
- **E2E tests (Playwright):** `globalSetup`/`globalTeardown` track and clean up spawned servers (Go backend, Vite dev server) per session.
- **Go backend:** Started as background process during E2E; cleaned up by teardown.
- **Vite dev server:** Started as background process during E2E; cleaned up by teardown.

### Automatic Cleanup (Built-in)

- Vitest `pool: 'forks'` ensures workers exit even with open handles
- Playwright `globalTeardown` cleans up only processes from the current session
- Safe for multiple Claude Code sessions running tests in parallel
- `nx run web:test` automatically runs `test:cleanup:all` after vitest exits (configured in `apps/web/project.json`)

### Developer Responsibility (MANDATORY)

After **every** test execution — whether via `nx run web:test`, direct `vitest`, or any other method:

1. Run `pnpm run test:cleanup` to verify no orphaned processes remain
2. If orphaned processes are found, run `pnpm run test:cleanup:all` immediately
3. Test execution is NOT considered complete until cleanup verification passes
4. This rule applies regardless of test pass/fail outcome

### Manual Cleanup Commands

```bash
# List orphaned test processes
pnpm run test:cleanup

# Force kill ALL test processes (use with caution)
pnpm run test:cleanup:all
```

**Session Files Location:** `node_modules/.cache/vido-test-sessions/`

**What Gets Cleaned Up:**

- Go backend (`go run ./cmd/api`)
- Vite dev server (`nx serve web`)
- Vitest workers (`node (vitest N)`)
- Playwright test runners
- Processes on ports 8080, 4200

---

## 🧪 TestSprite Journey Test Workflow

### Manual Trigger (After NAS Deploy)

1. **Start localhost proxy:** `node -e "const n=require('net');const s=n.createServer(c=>{const r=n.connect(8088,'192.168.50.52');c.pipe(r);r.pipe(c);c.on('error',()=>r.destroy());r.on('error',()=>c.destroy())});s.listen(8088,'127.0.0.1',()=>console.log('Proxy ready'))" &`
2. **Verify proxy:** `curl -s -o /dev/null -w "%{http_code}" http://localhost:8088/` (expect 200)
3. **Run TestSprite:** `node $(npm root)/.cache/@testsprite/testsprite-mcp/dist/index.js generateCodeAndExecute` or use TestSprite MCP tools in Claude Code
4. **Review results:** Check `testsprite_tests/tmp/raw_report.md` and TestSprite dashboard links
5. **Compare with baseline:** Check `testsprite_tests/testsprite-mcp-test-report.md` for expected pass/fail
6. **Kill proxy when done:** `kill $(lsof -ti:8088)`

### Baseline Strategy

- **Current baseline:** 2026-03-28, 14/30 passed (46.7%)
- After bugfix-1 + bugfix-3: expected ~73%
- When a previously-passing TC fails after a deploy → **regression**, investigate immediately
- When a previously-failing TC passes after a bugfix → **intentional change**, update baseline report

### Key Files

- Test plan: `testsprite_tests/testsprite_frontend_test_plan.json` (40 TCs)
- Test report: `testsprite_tests/testsprite-mcp-test-report.md`
- Raw results: `testsprite_tests/tmp/raw_report.md`
- Config: `testsprite_tests/tmp/config.json`
- Credits: 150/month (Free plan), check via TestSprite MCP `testsprite_check_account_info`

### Monthly Cron Workflow (story 19-6)

The manual flow above is still useful for ad-hoc post-deploy verification (the 30-credit reserve exists exactly for this). Most monthly coverage now runs unattended in CI:

- **Workflow file:** `.github/workflows/testsprite-monthly.yml` (`TestSprite Monthly` workflow, single job `TestSprite Monthly / Consume`).
- **Queue source of truth:** `_bmad-output/audit/testsprite-queue.yaml` — `schema_version: 2` ([@contract-v2] Rule 20 surface after CR 2026-05-19; 19-7 readers MUST check `schema_version == 2`). Each run rotates consumed entries from `queue:` to `history:` (capped at 200, FIFO prune); empty `queue:` triggers refill from `history:` oldest-first. Quota constants (`monthly_budget`, `consumption_cap_pct`, `reserved_credits`) live in the YAML — owner edits the file to switch plan tier, no workflow code change. Per-entry `last_status` enum is strictly `"pass" | "fail" | "error" | null` (BLOCKED report state and unparseable status both land as `error`; missing-report defaults the case to `null`).
- **Schedule:** `cron: '0 3 1 * *'` UTC (every month 1st, 03:00) — consumes early in the calendar month to maximise lead time for follow-up rebless / regression fix before next billing window.
- **Manual trigger entry point:** Actions tab → `TestSprite Monthly` → Run workflow (`workflow_dispatch`). Useful for testing the workflow itself, mid-month catch-up, or first-merge end-to-end validation.
- **Budget ceiling (AC #3 [@contract-v2] — pre-run testIds slicing, not LIVE per-case):** the workflow picks `N = MIN(queue_len, PER_RUN_CAP / credits_per_case) = MIN(queue_len, 24)` testIds upfront from the queue head and feeds them into a single `generateCodeAndExecute` CLI invocation. The math: `consumption_cap_pct: 80` × `monthly_budget: 150` ÷ `credits_per_case: 5` ≈ 24 cases per run = 120 credits = the per-run cap; the remaining 30 credits stay reserved for the manual ad-hoc lane. Tradeoff vs the original v1 design: a human consuming credits between cron-start and cron-finish doesn't trigger a mid-run abort (v1 would have, via LIVE `testsprite_check_account_info` per case — but that's an MCP-server-only tool, not a bare-CLI subcommand, so v1 was unimplementable). Race acceptable for monthly cadence. The three `last_run.credits_*` schema fields stay `null` in v2 pending a future MCP-stdio follow-up.
- **Secret + Variable setup (owner one-time, AC #5 / AC #9):**
  - Settings → Secrets and variables → Actions → **Secrets** tab → New repository secret `TESTSPRITE_API_KEY` = value from local `testsprite_tests/tmp/config.json` `executionArgs.envs.API_KEY` (or freshly issued if rotated). The local file is double-gitignored (`.gitignore:8` + `:74`).
  - Settings → Secrets and variables → Actions → **Variables** tab → New repository variable `TESTSPRITE_TARGET_URL` = the chosen target. Three viable paths: (1) cloudflared/ngrok tunnel pinned to the NAS at `http://192.168.50.52:8088` (simplest), (2) cloud-staging deploy (most robust), (3) runner-local docker-compose with TestSprite's own proxy (no LAN dependency). The workflow makes all three work — pick one and document the choice when wiring.
  - Branch-protection: `main` rule must allow `github-actions[bot]` bypass for the commit-back push (Settings → Branches → Branch protection rules → "Allow specified actors to bypass required pull requests" → add `github-actions[bot]`). Without bypass, the AC #6 (iii) failure path fires on every monthly run.
- **Commit-message convention** (AC #4 [@contract-v2] — `git log --grep='chore(testsprite): monthly run'` is the audit-trail filter):

  ```
  chore(testsprite): monthly run {YYYY-MM} — {N} cases consumed

  Status: {success | budget-exhausted | test-failures-only | api-failure}
  Test IDs run: TC001, TC002, ...
  Run URL: {github-actions-run-url}

  Auto-generated by .github/workflows/testsprite-monthly.yml

  Co-Authored-By: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>
  ```

  v2 dropped the `, {credits} credits` heading suffix and the `Credits: {start} → {end} (Δ {consumed})` body line — those depended on LIVE per-case `testsprite_check_account_info` queries the bare CLI doesn't expose (only MCP server mode does). For credits-trend analysis read `_bmad-output/audit/testsprite-queue.yaml`'s `history:` block instead (each entry carries the per-case `last_run_at` / `last_status` / `last_run_url`).

- **Failure semantics (AC #6):** workflow exit ≠ 0 only for "human must intervene" — API auth failure, target URL unreachable, or `git push` rejected after 3-retry rebase. Single-case test `fail` is data (recorded in `history.last_status: fail`, exit 0) so Actions email isn't flooded by regressions. For regression alerting, see 19-7 (`testsprite-quota-warning.yml`) or a future per-case-fail Issue-opener workflow.

- **Month-end run watchdog (story 19-7).** A sibling workflow `.github/workflows/testsprite-quota-warning.yml` runs `0 3 28 * *` UTC and opens a GitHub Issue tagged `testsprite-quota-warning` when 19-6's `last_run` is unhealthy — `status` ∈ {api-failure, test-failures-only}, or `last_run` stale (never run / >35 days old), or queue `schema_version != 2`. Issue auto-closes when `last_run` is healthy again. It reads the queue file only — no live TestSprite API call, no `TESTSPRITE_API_KEY`. See `_bmad-output/implementation-artifacts/19-7-github-actions-quota-warning.md` for the dedup contract + Issue body schema.

---

## ✅ Pre-Commit Checklist

Before committing code, verify:

**Format & Lint (MANDATORY):**

- [ ] Run `pnpm lint:all` — runs `go vet` + `staticcheck` + `eslint` + `prettier --check` (mirrors CI). Fix formatting with `pnpm exec prettier --write <files>`

**Code Location & Architecture:**

- [ ] All new code is in `/apps/api` (backend) or `/apps/web` (frontend)
- [ ] No code added to deprecated `/cmd` or root `/internal`
- [ ] Handler → Service → Repository layering respected
- [ ] Interfaces defined in correct package (Rule 11)

**Code Quality:**

- [ ] Logging uses `slog` (NOT zerolog, fmt.Println, or log.Print)
- [ ] API responses use `ApiResponse<T>` wrapper format
- [ ] Error codes follow `{SOURCE}_{ERROR_TYPE}` pattern
- [ ] Dates are ISO 8601 strings in JSON
- [ ] Naming conventions followed (see tables above)
- [ ] Frontend service POST/PUT bodies use `camelToSnake()` (Rule 18)
- [ ] Frontend service responses use `snakeToCamel()` (Rule 18)
- [ ] No swallowed errors — every error propagated or logged+returned (Rule 13)
- [ ] In-memory maps/caches have upper bounds (Rule 14)
- [ ] Background goroutines honor context cancellation (Rule 14)

**Testing (Definition of Done):**

- [ ] `go test ./...` passes with no failures
- [ ] Services test coverage ≥ 80%
- [ ] Handlers test coverage ≥ 70%
- [ ] Tests co-located with source files (`*_test.go`, `*.spec.tsx`)
- [ ] Assertions use specific matchers — `toBeInTheDocument`, `toBeAttached` (Rule 16)

**Integration (Definition of Done):**

- [ ] New Services/Handlers wired up in `main.go` (Rule 15)
- [ ] New model fields reflected in migration SQL and repository (Rule 15)
- [ ] Swagger annotations updated for new/changed endpoints (Rule 15)
- [ ] No binary files or sensitive data staged
- [ ] TanStack Query used for server state (NOT Zustand)

---

## 🤝 Team Agreements (Epic 1 Retrospective)

**Established: 2026-01-17**

These agreements were established during Epic 1 retrospective to improve development quality:

### Agreement 1: 標記完成 = 驗證完成

> "Marking a task complete means it has been **verified**, not just implemented."

- Before marking a task `[x]`, run the code and confirm it works
- Don't rely solely on Code Review to catch unfinished work
- If unsure, test it manually before marking complete

### Agreement 2: 左移品質檢查

> "Shift quality checks LEFT - catch issues during implementation, not review."

- Run `go test -cover` during implementation, not just before commit
- Check coverage targets (Services ≥80%, Handlers ≥70%) while coding
- Code Review should focus on architecture and design, not basic issues

### Agreement 3: project-context.md 是聖經

> "This file is the single source of truth. Read it before implementing."

- All Rules (1-17) must be followed
- When in doubt, check this file first
- Update this file when new patterns are established

---

## 🎯 Quick Decision Guide

### When to use what?

| Use Case               | Technology/Pattern                                    |
| ---------------------- | ----------------------------------------------------- |
| Backend HTTP framework | Gin                                                   |
| Backend logging        | `log/slog` (NOT zerolog)                              |
| Backend testing        | Go testing + testify                                  |
| Backend ORM            | **None** - Use repository pattern with `database/sql` |
| Database               | SQLite with WAL mode                                  |
| API documentation      | Swaggo (OpenAPI/Swagger)                              |
| Frontend framework     | React 19 + TypeScript                                 |
| Frontend routing       | TanStack Router                                       |
| Server state           | TanStack Query v5                                     |
| Client state (UI only) | Zustand                                               |
| Frontend styling       | Tailwind CSS v3.x                                     |
| Frontend testing       | Vitest + React Testing Library                        |
| Build tool (frontend)  | Vite                                                  |
| Monorepo               | Nx                                                    |

---

## 🔗 Complete Documentation

**For full details, see:**

- **Architecture Decisions:** `_bmad-output/planning-artifacts/architecture/index.md`
- **PRD:** `_bmad-output/planning-artifacts/prd.md`
- **UX Design:** `_bmad-output/planning-artifacts/ux-design-specification.md`

**Key Sections in architecture/:**

- Core Architectural Decisions (Step 4)
- Implementation Patterns & Consistency Rules (Step 5)
- Current Implementation Analysis (Brownfield Assessment)
- Consolidation & Refactoring Plan (5 Phases)

---

## ✅ Architecture Validation Summary

**Validation Status:** COMPLETE (2026-01-12)

The complete architecture has been validated for:

- ✅ **Coherence:** All 9 architectural decisions work together without conflicts
- ✅ **Coverage:** All 94 functional requirements are architecturally supported
- ✅ **Readiness:** 47 implementation patterns ensure AI agent consistency

**Key Deliverables:**

- 9 architectural decisions documented with versions and rationale
- 47 implementation patterns preventing AI agent conflicts (see architecture/)
- 400+ files/directories defined in complete project structure
- 5-phase consolidation roadmap from current to target state

**Confidence Level:** HIGH - Ready for implementation with comprehensive guidance.

---

## 🚀 Implementation Workflow

1. **Read this file FIRST** before implementing any feature
2. **Check architecture/** for specific pattern details if needed
3. **Follow the consolidation plan** (Phase 1-5) for refactoring
4. **Verify checklist** before committing code
5. **Write tests** alongside implementation (TDD encouraged)

---

**Questions or clarifications?** Refer to the full architecture document or ask the user.

**Last reminder:** ALL new backend code goes to `/apps/api`. The root backend is deprecated.
