# Story sub-2.1a: Key resolution + provider hot-reload + settings API

Status: review

**Epic:** `epic-subtitle-pipeline-m1-5` (M1.5) · **Risk: 🔴 HIGH (a silent-no-op trap + live client re-wiring)** · **BACKEND-ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 2.1 · PRD **FR25** · architecture **D9 / NFR-S3** · spec §5
**Split:** 2-1 was **cross-stack** — backend 5 tasks / frontend 4 tasks, **both > 3** ⇒ **mandatory split** (Epic 8 retro story-splitting rule; 13-1a/13-1b precedent). **2-1a = backend** · **2-1b = frontend** (depends on this).
**Depends on:** **sub-1-7a merged** (the three `.pen` copy revisions that block Epic 2 — carried by its AC #7).
**Blocks:** sub-2-1b.
**Cross-stack split check:** backend tasks = 5, frontend tasks = **0** (post-split) → single story.

---

## Story

As a NAS owner,
I want the keys I enter in the app to actually be the keys the pipeline uses,
so that configuring Claude in the UI enables translation without editing env-vars and restarting the container.

---

## 🚨 The finding that makes this story HIGH risk, not a CRUD page

**A key-config page written against the epic AC as literally stated would save successfully and change nothing.** Two independent breaks, both verified 2026-07-27:

### Break 1 — the key the pipeline reads is **env-only**. Secrets are not consulted.

```go
// config.go:122
cfg.ClaudeAPIKey = cfg.loadString("CLAUDE_API_KEY", "")
// api_keys.go:14,54 — the ONLY readers the pipeline uses
func (c *Config) HasClaudeKey() bool   { return c.ClaudeAPIKey != "" }
func (c *Config) GetClaudeAPIKey() string { return c.ClaudeAPIKey }
```

`sub-1-6` AC #5 wires the capability gate from `cfg.HasClaudeKey()`; `main.go:536` builds the provider from `cfg.GetClaudeAPIKey()`. **Nothing anywhere reads `secretsService` for an AI key.** Store a key from a settings page into the encrypted secrets table and `HasClaudeKey()` still returns `false` — the user sets a key, sees "saved", and translation stays gated with 「尚未設定翻譯服務金鑰」. That is the worst class of bug: a working UI over a disconnected backend.

### Break 2 — the provider is constructed **once, at startup**, from that env value.

`main.go:531-538` builds `ai.NewClaudeProvider(cfg.GetClaudeAPIKey(), …)` inside `if cfg.HasClaudeKey()`. A key supplied at runtime reaches a client that no longer exists to receive it — and if the key was empty at boot, `terminologyService`/`translationService` are **nil** and were never wired at all.

**Therefore this story is: a resolver, a re-buildable provider holder, and the API — with the page itself in 2-1b.** The epic AC's "persists to the existing encrypted secrets service" is necessary but **not sufficient**, and saying so is the whole point of drafting this before someone builds the inert version.

---

## Acceptance Criteria

### AC #1 — `[@contract-v1]` `KeyResolver`: secrets > env, one source of truth

**Given** two key sources, **then** a new `internal/config` (or `internal/services`) **`KeyResolver`** is the single reader every consumer uses:

```go
// [@contract-v1] — consumed by sub-2-1b (via the API), the pipeline capability
// gate (sub-1-6 AC #5 re-point), and the provider holder (AC #2).
// Resolution order is FIXED: encrypted secret (runtime, user-set) wins over
// env-var (deploy-time). Changing the order = Rule 20 bump.
type KeyResolver interface {
    Get(ctx context.Context, name KeyName) (value string, source KeySource, err error)
    Has(ctx context.Context, name KeyName) bool
}
type KeyName string   // KeyClaude | KeyTMDb | KeyOpenAI  (ASR — optional per FR21)
type KeySource string // KeySourceSecret | KeySourceEnv | KeySourceNone
```

- **Order ruling — secret wins:** the UI is the newer, more specific intent; an env-var that permanently overrode a user's in-app edit would make the page silently useless (the same failure class this story exists to prevent). `source` is surfaced so 2-1b can render 「目前由環境變數提供」 honestly.
- Secret names reuse the established convention (the qBittorrent/DVR precedent): `claude.api_key`, `tmdb.api_key`, `openai.api_key`.
- **Fail-soft:** a secrets-service error (e.g. decryption failure) logs at `slog.Error` and falls through to env — never a hard failure that takes down an already-working env-configured deployment.

### AC #2 — `[@contract-v1]` Provider holder: a runtime key change takes effect without a restart

**Given** Break 2, **then** a `ClaudeProviderHolder` owns the client and rebuilds it **only when the resolved key or model changes** — the established **Rule 14 config-fingerprint** pattern (`download_service.go:24-60`; `plugins.Manager`'s `URL|APIKey` fingerprint, 13-4a):

```go
// Fingerprint = resolvedKey + "|" + resolvedModel. Get() returns the cached
// provider when unchanged, rebuilds when it differs, and returns nil +
// ErrAINotConfigured when the key resolves empty.
func (h *ClaudeProviderHolder) Get(ctx context.Context) (ai.TextCompleter, error)
```

- **The nil-service problem (Break 2's second half):** `TranslationService` / `TerminologyCorrectionService` currently take a provider at construction and are skipped entirely when the key is absent at boot. Ruling: **construct them unconditionally** with the *holder* behind the existing `ai.TextCompleter` interface (a thin `holderCompleter` that resolves per call). `CompleteText` on an unconfigured holder returns `ErrAINotConfigured` — which every consumer already handles (fail-soft, NFR-R1). **`main.go`'s `if cfg.HasClaudeKey()` guard at :531 is removed**; the services always exist, they just decline while unconfigured.
- **Interfaces unchanged** — `ai.Provider`, `ai.TextCompleter`, `ai.CachingCompleter` (sub-1-1) are untouched; the holder implements the last two by delegation. Rule 20: no bump on sub-1-1's contract.
- Rebuild is cheap (an SDK client over a shared `http.Client`); the Governor is injected once and **shared across rebuilds** — a new client must never mean a new budget pool (NAIL-2 adjacent).

### AC #3 — Settings triad, mirroring the qBittorrent/DVR precedent

**Given** `qbittorrent_handler.go:51,68,101,143`'s `GetConfig`/`SaveConfig`/`TestConnection`/`RegisterRoutes` shape, **then** a new `handlers/keys_settings_handler.go`:

| Route | Behaviour |
|---|---|
| `GET /api/v1/settings/keys` | Per key: `{name, configured: bool, source: "secret"\|"env"\|"none", masked: "sk-ant-…7f3a"}`. **Never the value.** `masked` shows first 6 + last 4 only when `source == "secret"`; env-sourced keys report `configured` + `source` with **no mask** (we don't re-expose deploy secrets through an API). |
| `PUT /api/v1/settings/keys` | `{claude?, tmdb?, openai?}` — omitted fields untouched; **explicit `""` deletes the secret** (falls back to env, `source` flips) — the "clear my key" path, needed or a mistyped key is unrecoverable from the UI. Validates non-blank-after-trim, then `secretsService.Store`. |
| `POST /api/v1/settings/keys/test` | Tests the body's key if present, else the resolved one (`TestConnectionWithConfig` pattern). Claude: a **minimal `max_tokens: 1` Messages call** — cheap, real, and exercises the exact auth path. Maps 401/403 → `AI_PROVIDER_ERROR` with 「金鑰無效或已撤銷」; 404 → the sub-1-1 model diagnostic; timeout → `AI_TIMEOUT`. |

- **No test-before-save guard** (deliberate deviation from 13-4a's `DVR_TEST_FAILED` 409): an offline NAS must still be able to save a key it will use later, and unlike a DVR URL a key has no "reachable" precondition. 2-1b's UX is test-then-save by *affordance*, not by server refusal. Recorded so a reviewer reads it as a decision.
- **Zero new Rule 7 codes** — reuse `AI_NOT_CONFIGURED` / `AI_PROVIDER_ERROR` / `AI_TIMEOUT` / `VALIDATION_REQUIRED_FIELD`. Registry and `code-review/instructions.xml` untouched (prefix count stays 16).
- Rule 3 envelope · Rule 10 versioning · **Swagger + `swag init`** · `RegisterRoutes` wired in main.go and **verified** (Rule 15).

### AC #4 — Encryption-key pre-flight (the honest-degradation guard)

**Given** `Config.HasEncryptionKey()` and that the secrets service is AES-256-GCM over `VIDO_ENCRYPTION_KEY`, **when** that key is absent, **then** `GET /settings/keys` returns `{..., writable: false, reason: "encryption_key_missing"}` and `PUT` returns **409** with 「未設定加密金鑰，無法安全儲存 API 金鑰」 + suggestion 「請設定 VIDO_ENCRYPTION_KEY 後重啟」.

Without this the page would accept input and fail at the storage layer with an opaque error — the same silent-no-op family as Break 1. Read-only mode (showing env-sourced state) still works.

### AC #5 — Consumers re-pointed to the resolver

`cfg.HasClaudeKey()` / `cfg.GetClaudeAPIKey()` call sites that gate or build **runtime** behaviour move to the resolver/holder:

- `main.go:531-538` provider construction → holder (AC #2).
- `sub-1-6` AC #5's pipeline `configured` field → `resolver.Has(ctx, KeyClaude)`. **This is a sub-1-6 contact point**: if 1-6 has already merged, this story edits that one wiring line; if not, 1-6 wires the resolver directly. Either way **no Rule 20 bump** — `configured` is a `func() bool` struct field, not a stamped surface.
- `factory.go` / `NewProvider` stay 🔒 (they take an explicit key argument — the holder supplies it).
- **`GetTMDbAPIKey` is deliberately NOT re-pointed in this story.** TMDb is read at startup by many services and a runtime swap has its own blast radius; the resolver *exposes* TMDb (so 2-1b can display and store it, satisfying spec §5) while runtime consumption stays env-based until a follow-up. Recorded as lane ③ → `backlog-tmdb-runtime-key-resolution`.

### AC #6 — Tests (Rule 9/16)

1. `KeyResolver` — secret-wins-over-env; env fallback; both absent → `KeySourceNone`; secrets error → falls through to env + logs (fail-soft assertion).
2. Holder — same fingerprint returns the *same instance* (pointer equality); changed key rebuilds; empty key → `ErrAINotConfigured` via `errors.Is`; Governor identity preserved across a rebuild.
3. Handler table — GET shape incl. masking rules per source; PUT partial update / explicit-`""` delete / blank-trim rejection; POST test 401→invalid-key mapping; **AC #4's 409 when the encryption key is missing**; envelope shape.
4. **Integration on a real `:memory:` DB with the real secrets service** (Rule 15 / bugfix-20-1): store → resolve → holder rebuild → `Has()` flips true, all in one test. This is the end-to-end proof that Break 1 is closed.
5. `go test ./...` + `pnpm lint:all` green.

### AC #7 — Scope fence

- ❌ **No frontend** — the page, the HTTPS warning, and the dead-end fix are **2-1b**.
- ❌ No ASR/local-worker-URL settings beyond the `openai` key slot (spec §5's worker URL is Tier-2).
- ❌ No provider/model *selection* UI or API (FR24, Tier-2) — model stays `CLAUDE_MODEL`/default.
- ❌ No changes to `ai.Provider`/`TextCompleter`/`CachingCompleter`/`factory.go`/`gemini.go`.
- ❌ No new Rule 7 codes, no migrations (settings + secrets tables exist), no SSE.
- ❌ No TMDb **runtime** re-point (AC #5) — filed as backlog.

---

## Tasks / Subtasks

- [x] **Task 1 — Resolver (AC #1):** `KeyResolver` + names/sources + fail-soft fallthrough + tests.
- [x] **Task 2 — Holder (AC #2):** fingerprint cache, `holderCompleter` delegation, shared Governor; remove main.go's `if cfg.HasClaudeKey()` guard so services always construct.
- [x] **Task 3 — Handler triad (AC #3):** GET/PUT/POST-test + masking + Swagger + routes + main.go wiring + Rule 15 verification.
- [x] **Task 4 — Encryption pre-flight (AC #4):** `writable`/`reason` + 409 path.
- [x] **Task 5 — Re-point + gates (AC #5, #6):** pipeline `configured` (coordinate with 1-6's merge state), file `backlog-tmdb-runtime-key-resolution`, integration test, full gates.

---

## Dev Notes

- **Precedents to mirror, not invent:** `qbittorrent_handler.go:51-143` (triad + `has_password`-style masking) · `qbittorrent_service.go:33-55` (`secretsService.Exists` gating) · `dvr_settings_service.go:56-68` (settings-table + secrets composition) · `download_service.go:24-60` (fingerprint-cached client).
- **`secrets.SecretsServiceInterface`** = `Store/Retrieve/Delete/Exists` (`secrets_service.go:49-61`) — already injected in main.go for qBT/DVR; reuse that instance.
- **Rule 20:** stamps AC #1 + AC #2 (`[@contract-v1]`, consumer sub-2-1b); acks nothing (sub-1-6's `configured` is unstamped).
- **Rule 13/14/15** all in play — especially Rule 14 (one holder, fingerprint-rebuilt, never per request) and Rule 15 (main.go wiring + Swagger + route verification).

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Backend-only Go. Rule 23 does not apply.

### References

- [Source: `epics-subtitle-pipeline.md`#Story 2.1 · PRD FR25 · spec §5 · architecture #D9/NFR-S3]
- [`apps/api/internal/config/config.go`:122 · `api_keys.go`:14,54 — **the Break-1 evidence**]
- [`apps/api/cmd/api/main.go`:531-538 — **the Break-2 evidence**]
- [`apps/api/internal/secrets/secrets_service.go`:49-61 · `handlers/qbittorrent_handler.go`:51-143 · `services/dvr_settings_service.go`:56-68]
- [`sub-1-6-wire-triggering-gating.md`#AC #5 — the `configured` contact point]

---

## Dev Agent Record

### Agent Model Used

Amelia (Developer Agent) · Claude Fable 5, effort xhigh · 2026-08-05

### Debug Log References

RED verified before each task; each load-bearing guarantee falsified afterwards.

| Task | RED signal |
|---|---|
| 1 | `vet: undefined: NewKeyResolver / EnvKeys / KeyClaude / SecretNameClaude` |
| 2 | `vet: undefined: ClaudeProviderHolder / NewClaudeProviderHolder` |
| 3–4 | service + handler written test-first against the fakes |
| 5 | `go vet ./internal/repository/` → **import cycle not allowed in test** (see below) |

| Guarantee | Falsification | Result |
|---|---|---|
| secret > env (the whole contract) | resolution order inverted so env returns first | `TestKeyResolver_SecretWinsOverEnv` **FAILS** |
| the holder delegates `CachingCompleter` | `CompleteTextWithUsage` renamed + the `var _` assertion removed | **BUILD FAILS** — a compile-time guard, stronger than a runtime test |
| the shared Governor survives a rebuild | `opts` replaced with `nil` on rebuild | `TestClaudeProviderHolder_GovernorSurvivesRebuild` **FAILS** |

**Import cycle (Task 5).** AC #6.4 asked for the integration test on a real DB, and the natural home looked like `internal/repository` (that is where `setupTestDB` lives). `go vet` rejected it: `repository` importing `services` is the exact edge Rule 19 forbids, and a test file counts. Moved to `internal/services` with the migration-005 schema inlined verbatim — `services → repository` is the legal direction. The `srt_parity_test.go` external-test-package precedent is the same lesson.

### Completion Notes List

- 🎯 **Both Breaks are closed, and the integration test proves it against real encryption.** `TestKeyResolution_StoredKeyReachesTheResolver` runs store → resolve → gate-flip through a real `:memory:` DB and the real AES-256-GCM secrets service; `TestKeyResolution_HolderRebuildsAfterAStore` boots keyless, saves a key, and gets a working provider with no restart. Those two are the story's reason for existing — a settings page over an env-only reader would have saved successfully and changed nothing.
- ⚠️ **The single most dangerous thing I found: `TranslationService` type-asserts `ai.CachingCompleter` and degrades SILENTLY.** At `translation_service.go:323` it checks the assertion and, on failure, logs one line (`:335`) and translates on **without prompt caching or usage reporting** — which would have voided sub-1-5b's entire caching design and paid full token price on every cue, with nothing failing anywhere. A holder implementing only `TextCompleter` (which is all AC #2 literally requires to satisfy the constructors) would have compiled and passed every other test. The holder therefore implements **both**, with a `var _ ai.CachingCompleter` compile-time assertion and a test whose comment explains the trap.
- 🔴 **A regression I introduced, caught by the pre-ship self-review — worth reading before the next unconditional-construction change.** Removing `if cfg.HasClaudeKey()` makes the services always exist. Both consumers gate on `!= nil && IsConfigured()` (`subtitle/engine.go:191`, `transcription_service.go:416`) and `IsConfigured()` was `s.provider != nil` — which the holder makes permanently true. So on a **keyless** install the subtitle engine would have attempted AI terminology correction and the transcription service would have attempted translation, both failing with `ErrAINotConfigured` where they previously skipped cleanly. Nothing in AC #2 mentions this; the story's own text would have shipped it. Fixed by making `IsConfigured()` probe the provider for a key-aware `IsConfigured(ctx)` and fall back to the old `provider != nil` semantics for a plain provider (Gemini, a direct Claude client) — so nothing else changes behaviour. Pinned by three tests; falsified by restoring the old one-liner (`TestServices_ReportUnconfiguredWhenTheHolderHasNoKey` FAILS).
- 🧩 **AC #5 re-point landed on sub-1-6's actual shipped line.** The story anticipated this ("if 1-6 has already merged, this story edits that one wiring line") — it had, so `subtitleCapabilityGate := cfg.HasClaudeKey` became a closure over `keyResolver.Has(...)`. Still a plain `func() bool`, so **no Rule 20 bump** is owed, as the story predicted. The story's cited line numbers (`main.go:531-538`) had drifted to `537-544` because sub-1-6 rewrote that region; the facts held, the coordinates did not.
- ➕ **One additive change beyond the story text, with a reason: `ai.ErrAIUnauthorized`.** AC #3 wants the key-test endpoint to say 「金鑰無效或已撤銷」 on 401/403, but `claude.go` collapsed every non-429/404 status into a generic `ErrAIProviderError` whose only distinguishing feature was the substring `status 401`. String-matching an error message to decide what to tell a user is the kind of thing that breaks silently on an SDK bump. The new sentinel is wrapped **alongside** `ErrAIProviderError` (`fmt.Errorf("%w: %w: …")`), so every existing `errors.Is(err, ErrAIProviderError)` check is unaffected — verified by `go test ./internal/ai/` and a dedicated ordering test (`TestTestKey_UnauthorizedBeatsGenericProviderError`) that pins *why* the unauthorized branch must be checked first.
- 🔍 **429 deliberately does NOT report an invalid key.** A rate-limited account has a *working* key; saying 「金鑰無效」 there would send the user to regenerate a key that is fine. It answers 「金鑰有效，但目前已達用量上限」.
- 🔒 **Masking rule is stricter than "mask it".** Secret-sourced keys show `head6…tail4`; **env-sourced keys carry no mask at all** — an env value is a deploy secret the operator put in the environment precisely to keep out of the app, so re-exposing even a slice of it through an API would be a leak. Short values are fully bulleted rather than mostly revealed. Pinned by tests on both the fake and the real encrypted round trip.
- ✅ **Zero new Rule 7 codes** — `AI_NOT_CONFIGURED` / `AI_PROVIDER_ERROR` / `AI_QUOTA_EXCEEDED` / `AI_TIMEOUT` / `VALIDATION_*` / `DB_QUERY_FAILED` all reused. Prefix count stays 16; `project-context.md` and `code-review/instructions.xml` need **zero** edits — verified, not assumed. (`ai.ErrAIUnauthorized` is a Go sentinel for internal classification, not a wire code; nothing emits `AI_UNAUTHORIZED`.)
- 🎭 **A11y Pre-Flight: N/A** (100% backend — zero `apps/web/` files; the page is 2-1b).
- 🎨 **UX Verification: SKIPPED** — no UI in this story.
- 📘 **`swag init` NOT run — still a no-op in this repo.** `apps/api` has no swaggo dependency and no `docs` package (backend-consolidation Phase 1 Step 1.2 remains open). Full annotations are written and will be picked up when Swagger lands. Same finding sub-1-6 recorded; no new entry filed.
- ✅ **Gates:** `go build ./...` · `go vet ./...` · `go test ./...` **all packages green** · `nx run api:lint` (staticcheck) green · `gofmt -l` clean on every touched file.

### Discovery Triage

- **Pre-recorded at authoring:**
  - **① expand-scope-in-place → AC #1/#2.** The epic AC ("persists to the encrypted secrets service") is necessary but insufficient — env-only resolution (Break 1) and startup-only provider construction (Break 2) would make the page inert. Absorbed as the resolver + holder ACs.
  - **③ TMDb runtime resolution** → `backlog-tmdb-runtime-key-resolution` (filed with this story; AC #5 states the boundary). ✅ **Confirmed at implementation:** the resolver *exposes* `KeyTMDb` (so 2-1b can display and store it) but `GetTMDbAPIKey`'s runtime consumers are untouched — a TMDb swap has its own blast radius across many startup-wired services.
- **Triaged AT implementation (2026-08-05):**
  - **① expand-scope-in-place → `ai.ErrAIUnauthorized`.** AC #3's 「金鑰無效或已撤銷」 was not implementable without string-matching an error message: `claude.go` mapped every non-429/404 status to a generic `ErrAIProviderError`. Absorbed as an additive sentinel wrapped *alongside* the existing one, so no consumer changes behaviour. Not deferred, because the alternative shipping today would have been the fragile string match.
  - **① Rule 19 forced the integration test to move.** `internal/repository` cannot import `services`; the test lives in `internal/services` with the migration-005 schema inlined. Recorded because the next person will have the same instinct.
  - **Not a discovery — anticipated by the story and confirmed:** sub-1-6 had merged, so AC #5's re-point edited the shipped `subtitleCapabilityGate` line (`main.go:572`) rather than being wired by 1-6.
- Reference: `project-context.md` Rule 24.

### Change Log

| Date | Change |
|---|---|
| 2026-08-05 | **Tasks 1–5 — RED first on every task.** `KeyResolver` (secret > env, fail-soft through a secrets failure, blank-secret-never-shadows-env) · `ClaudeProviderHolder` (fingerprint-cached rebuild, shared Governor preserved, `ErrAINotConfigured` when unconfigured, **implements `CachingCompleter` as well as `TextCompleter`** — the assertion `TranslationService` makes at `:323`, whose silent failure mode would have voided sub-1-5b's prompt caching) · `KeySettingsService` (masking that never echoes env values, explicit-`""` delete reverting to env) · the GET/PUT/POST-test triad with the AC #4 encryption pre-flight · main.go rewiring: services now construct **unconditionally** behind the holder (the old `if cfg.HasClaudeKey()` guard left them nil forever on a keyless boot) and the pipeline capability gate reads the resolver instead of the boot-time env snapshot. Added `ai.ErrAIUnauthorized` (wrapped alongside `ErrAIProviderError`) so a 401 is distinguishable from a transient fault without string-matching. AC #6.4's integration test runs the whole chain on a real `:memory:` DB with real AES-256-GCM. Gates: `go vet ./...`, `go test ./...`, staticcheck all green; gofmt clean. |

### File List

| File | Change |
|---|---|
| `apps/api/internal/services/key_resolver.go` | **new** — AC #1 `[@contract-v1]` `KeyResolver`: closed `KeyName` set, `KeySource`, secret-first resolution, fail-soft fallthrough on a secrets error, blank-secret-treated-as-absent, nil-secrets-service degrades to env-only |
| `apps/api/internal/services/key_resolver_test.go` | **new** — 10 tests incl. the secret-wins contract, the decryption-failure fallthrough, and unknown-key-name-is-an-error (distinct from unconfigured) |
| `apps/api/internal/services/claude_provider_holder.go` | **new** — AC #2 `[@contract-v1]` holder: `key\|model` fingerprint cache (Rule 14), options replayed on rebuild so the Governor survives, `IsConfigured`, `TestKey`, and delegation of **both** `TextCompleter` and `CachingCompleter` with compile-time `var _` assertions |
| `apps/api/internal/services/claude_provider_holder_test.go` | **new** — 9 tests incl. pointer-identity caching, rebuild-on-key-change, keyless-boot recovery, Governor identity across a rebuild, and the CachingCompleter guard |
| `apps/api/internal/services/key_settings_service.go` | **new** — AC #3/#4 state + save: per-source masking rules, explicit-`""` delete, idempotent clear, `Writable`/`Reason` pre-flight |
| `apps/api/internal/services/key_settings_service_test.go` | **new** — 13 tests incl. env-keys-are-never-masked-echoed, short-value full masking, whitespace-clears, partial-update isolation |
| `apps/api/internal/services/key_resolution_integration_test.go` | **new** — AC #6.4 on a real `:memory:` DB + real AES-256-GCM secrets service: store → resolve → gate flips; keyless boot → save → holder rebuilds; clear reverts to env; masking holds on the real round trip. Lives in `services` because Rule 19 forbids `repository → services` |
| `apps/api/internal/handlers/key_settings_handler.go` | **new** — AC #3 triad `GET/PUT /api/v1/settings/keys` + `POST /test`; pointer fields distinguish omitted-vs-cleared; `classifyKeyTestError` with unauthorized checked before the generic provider error; Swagger annotations; Rule 3 envelope |
| `apps/api/internal/handlers/key_settings_handler_test.go` | **new** — 14 tests incl. the 5-case error-classification table, the omitted-vs-`""` distinction, the AC #4 409, and the sentinel-ordering guard |
| `apps/api/internal/services/terminology_service.go` · `translation_service.go` | **modified** — `IsConfigured()` now probes a key-aware provider instead of testing `provider != nil`; without this, unconditional construction silently re-enables AI paths on a keyless install |
| `apps/api/internal/ai/types.go` | **modified** — `ErrAIUnauthorized` sentinel (additive) |
| `apps/api/internal/ai/claude.go` | **modified** — 401/403 wrapped as `ErrAIProviderError` **and** `ErrAIUnauthorized` (existing `errors.Is` consumers unaffected); `Governor()` accessor so a rebuild's budget-pool identity is assertable |
| `apps/api/cmd/api/main.go` | **modified** — resolver + holder construction; terminology/translation services built **unconditionally** behind the holder (the `if cfg.HasClaudeKey()` guard removed); AC #5 capability-gate re-point; `keySettingsHandler` construction + `RegisterRoutes` (Rule 15 verified) |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | **modified** — `sub-2-1a` → `review`; `backlog-tmdb-runtime-key-resolution` filed |

---

## Open Questions for Alexyu (non-blocking — dev proceeds on the stated rulings)

1. **Resolution order (AC #1):** **secret > env**. The alternative (env wins, as an operator override) would mean an in-app edit silently does nothing on env-configured deployments — the exact trap this story removes. Confirm.
2. **No test-before-save (AC #3):** deliberate deviation from 13-4a's DVR guard — an offline NAS must still be able to save. Say if you'd rather refuse un-tested keys.
