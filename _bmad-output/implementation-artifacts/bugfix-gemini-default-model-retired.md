# Story bugfix: Gemini default model was shut down by Google

Status: done

## Story

As anyone deploying vido with the default AI settings,
I want the default Gemini model to be a model that still exists,
so that AI-assisted filename parsing works instead of 404-ing on every call.

## Evidence

- `apps/api/internal/ai/gemini.go:18` — `DefaultGeminiModel = "gemini-2.0-flash"`.
- Verified 2026-08-24 against https://ai.google.dev/gemini-api/docs/pricing: *"Gemini 2.0 Flash is deprecated and has been shut down June 1, 2026."* So the default has been pointing at a dead endpoint for ~3 months.
- `AI_PROVIDER` defaults to `gemini`, and there is no `GEMINI_MODEL` override — unlike Claude, which got `CLAUDE_MODEL` in story 9R-1 after the identical incident (a retired Claude model id shipped as a default). So a Gemini deployment has **no way to route around this without a code change**.
- Cheapest current flash-tier model is `gemini-2.5-flash-lite` at **$0.10 in / $0.40 out per 1M tokens** — *identical* to the retired model's final published rate, so the bump carries no cost change. `defaultLLMPricing` (`budget.go:33`) already carries that exact row.
- Live NAS check: `AI_PROVIDER=gemini` **and `GEMINI_API_KEY` is empty**, so that particular instance never reached the API at all — the retired default is latent there, not active. It becomes active the moment anyone sets a key. Also confirms this bug is *not* what made `bugfix-scanner-bracket-prefix-filenames-dropped`'s AI hand-off useless on that box; the missing key was.

## Acceptance Criteria

1. `DefaultGeminiModel` is `gemini-2.5-flash-lite` — a model verified present on the official pricing page on 2026-08-24, at the same price tier as the retired default so no deployment sees a cost change.
2. A `GEMINI_MODEL` env var overrides the model id, mirroring `CLAUDE_MODEL` exactly: `Config.GeminiModel` loaded via `loadString("GEMINI_MODEL", "")`, a `GetGeminiModel()` accessor, and empty meaning "use the provider default".
3. `ai.FactoryConfig` gains `GeminiModel`, and `NewProvider` applies it via the existing `WithGeminiModel` option — same shape as the `ClaudeModel` branch, including the `model_override` slog attribute.
4. The override actually reaches the provider (Rule 15: a config field no wiring reads is a silent no-op). **⚠️ AC corrected during implementation:** the wiring point is `services/ai_service.go:88` (the sole `ai.FactoryConfig` construction site), **not** `main.go`. `main.go`'s two `GetClaudeModel()` calls are Claude-specific (the provider holder and the subtitle translation model) and have no Gemini counterpart.
5. `defaultLLMPricing` covers every model a `GEMINI_MODEL` override can now realistically select, with prices verified on 2026-08-24: the existing `gemini-2.5-flash` / `gemini-2.5-flash-lite` rows plus `gemini-3.5-flash-lite`, `gemini-3.6-flash`, `gemini-3.7-flash`. The retired `gemini-2.0-flash` row **stays** (a still-configured deployment must meter honestly rather than fall to the fallback tier).
6. `.env.example` documents `GEMINI_MODEL` next to the existing AI vars.
7. Tests: the default is asserted to be the new id; `GEMINI_MODEL` override reaches the provider; a pricing lookup is asserted for each new row; the retired-model row still resolves (not the fallback).
8. Gates: `pnpm nx test api` green, `gofmt -l` clean on touched files, `pnpm run format:check` green. Zero frontend changes.

## Tasks / Subtasks

- [x] Task 1 — Bump the default (AC: #1, #7)
  - [x] 1.1 `DefaultGeminiModel` → `gemini-2.5-flash-lite`, with the retirement recorded in the comment
  - [x] 1.2 Test pinning the default id
- [x] Task 2 — `GEMINI_MODEL` override, mirroring `CLAUDE_MODEL` (AC: #2, #3, #4, #6, #7)
  - [x] 2.1 `Config.GeminiModel` + `loadString("GEMINI_MODEL", "")` + `GetGeminiModel()`
  - [x] 2.2 `FactoryConfig.GeminiModel` + `NewProvider` branch
  - [x] 2.3 Wire `cfg.GetGeminiModel()` at the `ai.FactoryConfig` site in `services/ai_service.go` (see AC #4 correction — not `main.go`)
  - [x] 2.4 `.env.example` entry
  - [x] 2.5 Tests for override reaching the provider
- [x] Task 3 — Pricing table (AC: #5, #7)
  - [x] 3.1 Add the three verified 3.x rows; keep the retired 2.0 row
  - [x] 3.2 Test each row resolves and does not hit the fallback
- [x] Task 4 — Gates (AC: #8)

## Dev Notes

- **Mirror `CLAUDE_MODEL`, do not invent a new shape.** The 9R-1 precedent is the whole point: same env-var naming, same empty-means-default semantics, same factory branch, same `model_override` log attribute. A second convention for the same problem is worse than the bug.
- **Do not delete the `gemini-2.0-flash` pricing row.** `budget.go`'s comment already explains why: an existing deployment that pinned it must still meter at its final published rate rather than silently inherit the Haiku-tier fallback and record a fabricated number.
- The fallback tier ($1.00/$5.00) is *higher* than every Gemini rate, so an unknown model over-counts rather than under-counts. That is the safe direction and is deliberate — the new rows are about accuracy, not about preventing under-billing.
- Rule 7 / 10 / 20: **N/A** — no error codes, no routes, no wire contract. Rule 23: N/A (no frontend).
- Out of scope: whether `AI_PROVIDER` should default to `gemini` at all, and the empty-key-means-silently-disabled behaviour. Both are real questions; neither is this bug.

### Time-dependent visual coverage

- N/A — backend only, no `apps/web/src/components/**` touched.

### References

- [Source: apps/api/internal/ai/gemini.go:18 — the retired default]
- [Source: apps/api/internal/ai/budget.go:27-33 — pricing rows + the retirement note]
- [Source: apps/api/internal/ai/factory.go:17-19, 49-58 — the ClaudeModel branch to mirror]
- [Source: apps/api/internal/config/config.go:59-60, 129 + api_keys.go:68-72 — the CLAUDE_MODEL precedent]
- [Source: https://ai.google.dev/gemini-api/docs/pricing — verified 2026-08-24]

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Claude Fable 5)

### Debug Log References

- Facts confirmed against the live official page before writing code (CLAUDE.md "Confirm Before Coding"): https://ai.google.dev/gemini-api/docs/pricing states verbatim *"Gemini 2.0 Flash is deprecated and has been shut down June 1, 2026."* Current flash tier: `gemini-2.5-flash-lite` $0.10/$0.40, `gemini-2.5-flash` $0.30/$2.50, `gemini-3.5-flash-lite` $0.30/$2.50, `gemini-3.6-flash` / `gemini-3.7-flash` $0.75/$3.75 (promotional through 2026-12-31).
- RED: `gemini_model_test.go` failed to compile on `unknown field GeminiModel in FactoryConfig`, then failed on the default-id assertion.
- GREEN: 10 new tests pass; `pnpm nx test api` green; `go vet` clean on the three touched packages.

### Completion Notes List

- **The bump is cost-neutral by construction** — `gemini-2.5-flash-lite` carries the exact rate the retired model had ($0.10 in / $0.40 out per 1M), so no deployment sees a price change from the default moving.
- **Live NAS finding that reframes the urgency:** `AI_PROVIDER=gemini` but **`GEMINI_API_KEY` is empty** on `192.168.50.52`, so that instance never reached the API — the retired default was latent there, not active. Two consequences worth recording: (a) this bug is *not* what killed the AI hand-off in `bugfix-scanner-bracket-prefix-filenames-dropped` — the missing key was; (b) AI-assisted parsing has been silently off on that box, and turning it on is a config decision for the owner (set a Gemini key, or switch `AI_PROVIDER=claude` with a Claude key). Surfaced to Alexyu rather than decided here.
- **Adversarial sweep before shipping, all clean:** (1) no remaining hardcoded `gemini-2.0-flash` outside deliberate comments, the retained pricing row, and the tests that pin it; (2) `NewGeminiProvider` has exactly one production construction site (`factory.go`), and both its branches now honour the override — no path bypasses it; (3) install guides document only must-set vars and list neither `CLAUDE_MODEL` nor `GEMINI_MODEL`, so they were deliberately left alone — adding only the Gemini one would have been inconsistent, and it would have pulled in the Rule 17 bilingual obligation for a var whose sibling is undocumented. `.env.example` is the canonical env reference and does carry both.
- **Pricing rows widened, not narrowed.** Three 3.x rows added because `GEMINI_MODEL` can now select them and an unpriced model meters at the fallback tier — a fabricated number. `TestFallbackTierIsNotCheaperThanAnyGeminiRow` pins the invariant that makes the fallback safe: it is dearer than every real Gemini rate, so an unknown model over-counts rather than under-counts.
- **`gemini-2.0-flash` row deliberately retained** with its final published rate, per `budget.go`'s existing rationale — a deployment that pins it via `GEMINI_MODEL` must still meter honestly.
- **Pre-existing gofmt drift** in `internal/ai/gemini.go` (struct-tag alignment at ~line 273, unrelated to this change) fixed in passing since the file was already being edited. Tracked generally by `backlog-repo-wide-gofmt-drift`.
- 🔗 AC Drift: **FOUND** — this story's own AC #4, corrected above (wiring site is `ai_service.go`, not `main.go`). No prior story's AC contradicted; this follows the 9R-1 `CLAUDE_MODEL` precedent rather than changing it.
- 📎 Contract Stamps: NONE — no `[@contract-v*]` in this story or its upstream refs. `GEMINI_MODEL` is a new opt-in env var with an empty default, so no existing behaviour changes for anyone who does not set it.
- 🎭 A11y Pre-Flight: N/A (100% backend).
- 🎨 UX Verification: SKIPPED — no UI changes.

### Discovery Triage

- **YES — one discovery, surfaced not filed:** `GEMINI_API_KEY` is empty on the live NAS while `AI_PROVIDER=gemini`, so AI parsing is silently disabled there. This is a **deployment-configuration** state, not a code defect, so it does not meet the Rule 24 "work" bar — there is nothing to implement. Reported to Alexyu as a decision (set a key, or switch provider). If the *silence* itself is judged a defect (a configured provider with no key should warn loudly at boot rather than log once at Info), that would be a separate story; not filed unilaterally.
- Separately noted, no entry filed: an `OPENAI_API_KEY` sits in plaintext in the container's env, readable by anyone with `docker inspect` access, while the app already ships an AES-256 `secrets` table (migration 005) as the intended home for keys. Infra/config observation for the owner, not a repo change; verified the key is **not** present anywhere in git history or tracked files.

### File List

- apps/api/internal/ai/gemini.go
- apps/api/internal/ai/factory.go
- apps/api/internal/ai/budget.go
- apps/api/internal/ai/gemini_model_test.go (new)
- apps/api/internal/config/config.go
- apps/api/internal/config/api_keys.go
- apps/api/internal/services/ai_service.go
- .env.example
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/bugfix-gemini-default-model-retired.md

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | Task 1: `DefaultGeminiModel` → `gemini-2.5-flash-lite` (verified live on the official pricing page; same rate as the retired default, so cost-neutral). |
| 2026-08-24 | Task 2: `GEMINI_MODEL` override mirroring `CLAUDE_MODEL` — `Config.GeminiModel` + `GetGeminiModel()` + `FactoryConfig.GeminiModel` + the `ai_service.go` wiring + `.env.example`. AC #4 corrected: the wiring site is `ai_service.go`, not `main.go`. |
| 2026-08-24 | Task 3: pricing table gains `gemini-3.5-flash-lite` / `gemini-3.6-flash` / `gemini-3.7-flash`; retired `gemini-2.0-flash` row retained on purpose. |
| 2026-08-24 | Task 4: gates green — `pnpm nx test api`, `go vet`, `gofmt` clean on touched files, `format:check`. |
