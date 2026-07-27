# Story sub-2.1a: Key resolution + provider hot-reload + settings API

Status: ready-for-dev

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

- [ ] **Task 1 — Resolver (AC #1):** `KeyResolver` + names/sources + fail-soft fallthrough + tests.
- [ ] **Task 2 — Holder (AC #2):** fingerprint cache, `holderCompleter` delegation, shared Governor; remove main.go's `if cfg.HasClaudeKey()` guard so services always construct.
- [ ] **Task 3 — Handler triad (AC #3):** GET/PUT/POST-test + masking + Swagger + routes + main.go wiring + Rule 15 verification.
- [ ] **Task 4 — Encryption pre-flight (AC #4):** `writable`/`reason` + 409 path.
- [ ] **Task 5 — Re-point + gates (AC #5, #6):** pipeline `configured` (coordinate with 1-6's merge state), file `backlog-tmdb-runtime-key-resolution`, integration test, full gates.

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

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Pre-recorded at authoring:**
  - **① expand-scope-in-place → AC #1/#2.** The epic AC ("persists to the encrypted secrets service") is necessary but insufficient — env-only resolution (Break 1) and startup-only provider construction (Break 2) would make the page inert. Absorbed as the resolver + holder ACs.
  - **③ TMDb runtime resolution** → `backlog-tmdb-runtime-key-resolution` (filed with this story; AC #5 states the boundary).
- Reference: `project-context.md` Rule 24.

### File List

---

## Open Questions for Alexyu (non-blocking — dev proceeds on the stated rulings)

1. **Resolution order (AC #1):** **secret > env**. The alternative (env wins, as an operator override) would mean an in-app edit silently does nothing on env-configured deployments — the exact trap this story removes. Confirm.
2. **No test-before-save (AC #3):** deliberate deviation from 13-4a's DVR guard — an offline NAS must still be able to save. Say if you'd rather refuse un-tested keys.
