# Story sub-1.1: Migrate the Claude client to the official Go SDK

Status: done

**Epic:** `epic-subtitle-pipeline-m1` — Automatic Traditional-Chinese subtitles for English media (M1) · **Risk: 🔴 HIGH** · **BACKEND-ONLY**
**Source:** `_bmad-output/planning-artifacts/epics-subtitle-pipeline.md` § Story 1.1 · architecture Step 3 + **D8** + **P10**
**Depends on:** nothing — this is implementation step **1 of 7** (architecture § Decision Impact Analysis). Everything downstream calls it.
**Blocks:** sub-1-5a/1.5b (translate + quality gate / cache + delivery — need prompt caching + caller-visible usage), and indirectly every later story that spends LLM tokens.
**Split note (2026-07-27, IR-r2 M1):** sub-1-5 was size-split into **sub-1-5a** (translate core + quality gate) and **sub-1-5b** (cache key + serialization + delivery). References to "sub-1-5" elsewhere in this file read as that pair; **AC #5's `CachingCompleter` primary consumer is sub-1-5a**.
**Cross-stack split check:** backend tasks = 4, frontend tasks = **0** → single story (threshold is >3 on *both* sides). No split.

---

## Story

As a maintainer,
I want `apps/api/internal/ai/claude.go` re-implemented on `github.com/anthropics/anthropic-sdk-go`,
so that the translation path gains prompt caching and caller-visible token usage without hand-rolling the Messages wire protocol.

---

## 🚨 Why this story is HIGH risk (read before touching code)

`claude.go` is **not** a subtitle file. It also backs `Provider.Parse` → `AIService` → **AI filename parsing used by media scanning**. A regression here breaks *scanning*, not just subtitles. The existing test suite is the only guardrail, and keeping it green is the completion bar — not a nice-to-have.

The three **NAILs** (epic-level, citable regardless of AC formatting):

| # | NAIL | Where it is enforced |
|---|---|---|
| 1 | Every existing Claude test stays green (media-scanning regression guard) | AC #1 + AC #2 |
| 2 | SDK retries **disabled**; `retryTransient` + Governor budget gate remain the sole retry/throttle path (D8) | AC #3 |
| 3 | *(quality gate before OpenCC)* | **not this story** — sub-1-5 |

---

## Acceptance Criteria

### AC #1 — Exported surface is byte-identical; consumers are untouched

**Given** the re-implemented client, **when** any existing consumer calls it, **then** behaviour is unchanged and **zero** call-site edits are required.

The following exported symbols keep their exact signatures and semantics:

```go
const DefaultClaudeModel = "claude-haiku-4-5"      // unchanged
const ClaudeAPIVersion   = "2023-06-01"            // unchanged (still asserted as a request header)
const ClaudeMaxTokens    = 1024                    // unchanged

type ClaudeProvider struct{ … }                    // fields apiKey/baseURL/model/httpClient/timeout/governor RETAINED
type ClaudeProviderOption func(*ClaudeProvider)

func NewClaudeProvider(apiKey string, opts ...ClaudeProviderOption) *ClaudeProvider
func WithClaudeBaseURL(url string) ClaudeProviderOption
func WithClaudeModel(model string) ClaudeProviderOption
func WithClaudeHTTPClient(client *http.Client) ClaudeProviderOption
func WithClaudeTimeout(timeout time.Duration) ClaudeProviderOption
func WithClaudeGovernor(g *Governor) ClaudeProviderOption

func (p *ClaudeProvider) Name() ProviderName
func (p *ClaudeProvider) Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error)
func (p *ClaudeProvider) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error)

var _ Provider      = (*ClaudeProvider)(nil)
var _ TextCompleter = (*ClaudeProvider)(nil)
```

**Files that MUST NOT change** (Rule 15 self-verification — if you find yourself editing one of these, the exported surface has drifted; **stop and re-read this AC**):

- `apps/api/cmd/api/main.go:532-538` (`claudeOpts` → `NewClaudeProvider` → terminology + translation services)
- `apps/api/internal/ai/factory.go:45-54` (`NewProvider` claude branch)
- `apps/api/internal/ai/provider.go` (`Provider` / `TextCompleter` — **the moat**)
- `apps/api/internal/ai/gemini.go`, `governor.go`, `budget.go`, `retry.go`
- `apps/api/internal/services/terminology_service.go`, `translation_service.go`, `transcription_service.go`
- `apps/api/cmd/route-c-poc/main.go:158-159`

The **one** permitted exported-value change is `DefaultClaudeBaseURL` — see AC #6.

### AC #2 — [NAIL 1] Test-suite regression guard: all 26 Claude tests green, none deleted

**Given** the migration, **then** `go test ./internal/ai/...` passes with **26** Claude-touching test functions present and green.

Exact inventory (verified 2026-07-27 via `go test -list`) — **24 in `claude_test.go` + 2 in `retry_test.go`**:

`claude_test.go` (24): `TestNewClaudeProvider` · `TestClaudeProvider_Name` · `TestClaudeProvider_Parse_Success` · `TestClaudeProvider_Parse_MovieSuccess` · `TestClaudeProvider_Parse_ValidationError` · `TestClaudeProvider_Parse_Timeout` · `TestClaudeProvider_Parse_QuotaExceeded` · `TestClaudeProvider_Parse_ServerError` · `TestClaudeProvider_Parse_InvalidJSONResponse` · `TestClaudeProvider_Parse_EmptyResponse` · `TestClaudeProvider_CompleteText_Success` · `TestClaudeProvider_CompleteText_SystemFieldSerialization` · `TestClaudeProvider_CompleteText_MaxTokensDefaulting` · `TestClaudeProvider_CompleteText_Timeout` · `TestClaudeProvider_CompleteText_QuotaExceeded` · `TestClaudeProvider_CompleteText_ServerError` · `TestClaudeProvider_CompleteText_EmptyResponse` · `TestClaudeProvider_CompleteText_MalformedJSON` · `TestClaudeResponse_GetText` · `TestDefaultClaudeModel_CurrentAndCarriedInRequestBody` · `TestClaudeProvider_NotFoundGuard_NamesBadModel` · `TestNewProvider_ClaudeModelOverride` · `TestClaudeProvider_MetersUsageToBudget` · `TestClaudeProvider_BudgetCutoffStopsCall`

`retry_test.go` (2 — **these exercise `ClaudeProvider` and are part of the guard**): `TestClaudeProvider_RetriesTransientThenSucceeds` · `TestClaudeProvider_NoRetryOnPermanent4xx`

**The `httptest.NewServer` fakes are KEPT** — the SDK accepts a base-URL override, so the servers, the response fixtures, and every behavioural assertion survive. Only *client construction* and *request-body assertions* change (architecture § Scope and regression surface).

**Per-test disposition — this is the contract. `-1 test` is a story failure, not a cleanup:**

| Test | Disposition |
|---|---|
| All `Parse_*`, `CompleteText_*` behavioural tests, `MetersUsageToBudget`, `BudgetCutoffStopsCall`, `NotFoundGuard`, both `retry_test.go` Claude tests | **Unchanged.** They assert observable behaviour through an httptest server; the SDK swap must be invisible to them. |
| `TestNewClaudeProvider` | **Unchanged** (asserts struct fields `apiKey`/`baseURL`/`model`/`timeout`/`httpClient` — all retained per AC #1). |
| `TestClaudeProvider_CompleteText_SystemFieldSerialization` | **Rewritten in place, same name, ≥ same coverage.** It currently marshals the deleted hand-rolled `claudeRequest` struct. Rewrite it to assert the *wire body the SDK actually sends*, captured by an httptest server: (a) non-empty system → `system` present in the decoded body; (b) empty system → `system` **absent** from the decoded body. Do not delete it, do not weaken it to a construction-only check. |
| `TestClaudeResponse_GetText` | **Rewritten in place, same name.** Retarget it at whatever helper now extracts the first `text` block from `*anthropic.Message` (see AC #4 `textFromMessage`). Keep all four table cases: text content · empty content · non-text content · multiple blocks → first text. |
| `TestNewProvider_ClaudeModelOverride` | **Unchanged** (factory-level, asserts `cp.model`). |

**Deleting a test to make the suite green is an automatic reject.** If a test genuinely cannot survive, that is a Rule 24 discovery → triage it, do not silently drop it.

### AC #3 — [NAIL 2] SDK retries OFF; Governor + `retryTransient` remain the sole retry/throttle path (D8)

**Given** the SDK retries by default (max 2, on 408/409/429/5xx and connection errors) and `doRequest` already wraps `retryTransient` (max 3 attempts), **when** the client is constructed, **then** the SDK's retries are **disabled** so a logical call makes at most 3 real HTTP requests — never 2×3 = 6.

**Why this ruling and not the inverse** (architecture D8, verbatim): the Governor's budget pre-check (`governed(...)`, Story 9R-11) wraps *outside* `retryTransient`. Moving retries into the SDK would put them *below* the budget gate, so a retry storm would bypass cost control entirely. Keeping the existing nesting preserves the existing guarantee.

The nesting order is **load-bearing and must be preserved exactly**:

```
governed(ctx, p.governor, "claude.messages", …)   ← budget pre-check + rate token + concurrency slot (ONCE per logical call)
  └── retryTransient(ctx, "claude.messages", …)   ← 3 attempts, 1s→2s→4s→8s backoff
        └── p.client.Messages.New(ctx, params)    ← SDK, retries DISABLED
```

**Proof obligation (new test, AC #7):** a test that returns 503 on every request and asserts the httptest server was hit **exactly `retryMaxAttempts` (3)** times. Without SDK retries disabled this test observes 6+ and fails. This is the only mechanical guard against a silent double-retry regression, so it is mandatory.

`TestClaudeProvider_BudgetCutoffStopsCall` (asserts `hits == 0` when the budget is blown) must stay green — proving `governed` still short-circuits **before** any HTTP call.

### AC #4 — Error mapping, the 404 diagnostic, and budget metering are preserved

**Given** an upstream error, **when** it surfaces from the SDK, **then** it is re-mapped to the existing sentinels before it leaves the package — SDK error types must **not** leak upward.

| Upstream condition | Mapped error | Retryable? |
|---|---|---|
| HTTP 429 | `ErrAIQuotaExceeded` | **yes** |
| HTTP 404 | `ErrAIProviderError` wrapped with the **verbatim 9R-1 diagnostic** (below) | **no** |
| Other 4xx (not 429) | `ErrAIProviderError` + `status %d` | **no** |
| 5xx | `ErrAIProviderError` + `status %d` | **yes** |
| Request timeout / context deadline | `ErrAITimeout` (+ the existing `slog.Warn("Claude API timeout", "timeout_seconds", …)`) | **yes** |
| Connection error | `ErrAIProviderError` wrapping the cause | **yes** |
| **Malformed JSON in a 200 body** | `ErrAIInvalidResponse` | **no** — see the trap below |
| 200 with zero text content blocks | `ErrAIInvalidResponse` | n/a (post-call check) |

**The 404 diagnostic is hard-won (9R-1 incident) and must be re-created character-for-character:**

```go
slog.Error("Claude model not found — the configured model id is deprecated or invalid", "model", p.model)
return fmt.Errorf("%w: status 404: model %q not found (deprecated or invalid model id — set CLAUDE_MODEL to a current model)", ErrAIProviderError, p.model)
```

`TestClaudeProvider_NotFoundGuard_NamesBadModel` asserts `err.Error()` contains the bad model id via **both** `CompleteText` and `Parse`.

**⚠️ TRAP — the malformed-JSON retryability flip.** Today `json.Unmarshal` runs *outside* `retryTransient` (after `doRequest` returns), so a malformed 200 body fails **once**. With the SDK, decoding happens *inside* `Messages.New`, i.e. *inside* the retry loop. Classifying it as retryable would turn 1 request into 3 — a silent 3× cost regression on a garbage response. **Classify decode failures as NOT retryable.** Detection, in order of preference:

1. `errors.As(err, &syntaxErr)` for `*json.SyntaxError` / `*json.UnmarshalTypeError` (the SDK's `apijson` decoder wraps `encoding/json`) — **verify this actually surfaces before relying on it.**
2. Fallback if (1) can't discriminate: `option.WithMiddleware` that reads/validates the response body and returns a package-local sentinel the classifier recognises.

AC #7 adds a hit-count assertion to `TestClaudeProvider_CompleteText_MalformedJSON` locking this at **exactly 1**.

**Timeout detection must cover both paths** — `http.Client.Timeout` and a caller-supplied `ctx` deadline:

```go
func isTimeoutErr(err error) bool {
    if errors.Is(err, context.DeadlineExceeded) { return true }
    var netErr net.Error
    return errors.As(err, &netErr) && netErr.Timeout()
}
```

**Budget metering (9R-11) still fires on every successful call**, from the SDK's typed usage — replacing the current re-unmarshal of the raw body:

```go
if b := BudgetFromContext(ctx); b != nil {
    b.RecordLLM(p.model, msg.Usage.InputTokens, msg.Usage.OutputTokens)
}
```

`TestClaudeProvider_MetersUsageToBudget` asserts `1000` in / `500` out / `1` call / `$0.0035` — unchanged.

**Text extraction helper** (replaces `claudeResponse.GetText()`, retargeted by `TestClaudeResponse_GetText`):

```go
// textFromMessage returns the text of the first text content block, or "".
func textFromMessage(msg *anthropic.Message) string {
    for _, block := range msg.Content {
        if t, ok := block.AsAny().(anthropic.TextBlock); ok { return t.Text }
    }
    return ""
}
```

### AC #5 — `[@contract-v1]` Prompt caching + caller-visible usage (additive; consumer = sub-1-5)

**Given** a translation-oriented call, **when** it runs, **then** `system` content blocks carrying `cache_control` are supported **and** usage (input / output / **cache-creation** / **cache-read** tokens) is returned to the caller.

`CompleteText` returns only `(string, error)` and **must not change** (it is the `TextCompleter` moat). The capability is therefore **additive**: a new optional interface + one new method, implemented only by `ClaudeProvider`. Gemini does not implement it, so consumers type-assert and degrade — the existing multi-provider degradation shape is preserved.

**Stamped shape — changing it later is a Rule 20 bump + downstream stale-mark on sub-1-5:**

```go
// provider.go — additive, sits beside TextCompleter. Gemini does NOT implement it.
type CachingCompleter interface {
    CompleteTextWithUsage(ctx context.Context, req CompletionRequest) (CompletionResult, error)
}

type CacheTTL string
const (
    CacheTTLNone CacheTTL = ""    // block is not a cache breakpoint
    CacheTTL5m   CacheTTL = "5m"
    CacheTTL1h   CacheTTL = "1h"  // the season-batch TTL (architecture Step 3 fact 7)
)

type SystemBlock struct {
    Text     string
    CacheTTL CacheTTL             // non-empty ⇒ emit cache_control on THIS block
}

type CompletionRequest struct {
    System     []SystemBlock      // ordered; stable content FIRST (prefix match)
    UserPrompt string
    MaxTokens  int                // <=0 ⇒ ClaudeMaxTokens
}

type CompletionUsage struct {
    InputTokens              int64
    OutputTokens             int64
    CacheCreationInputTokens int64
    CacheReadInputTokens     int64
}

type CompletionResult struct {
    Text  string
    Usage CompletionUsage
}

var _ CachingCompleter = (*ClaudeProvider)(nil)
```

Behavioural requirements:

1. Blocks are emitted **in slice order** — caching is a prefix match, so order is semantic, not cosmetic.
2. A block with `CacheTTL5m` / `CacheTTL1h` emits `cache_control` on **that block**; `CacheTTLNone` emits none.
3. `CompletionResult.Usage` is populated from `msg.Usage` including both cache fields.
4. `CompleteTextWithUsage` goes through the **same** `governed → retryTransient → Messages.New` path and the same error mapping as `CompleteText` (AC #3/#4). No second code path.
5. `RecordLLM` still fires (input/output tokens only — `Budget` has no cache-token dimension; **do not extend `budget.go`** in this story).

**Do NOT implement in this story** (sub-1-5 owns them): the ≥4096-token prefix rule and the "disable caching + record it" ruling (D4), the per-show first-request serialization gate (D10), the versioned cache key, and any prompt text. This story ships the *capability*; 1.5 ships the *policy*. Exposing `CacheCreationInputTokens`/`CacheReadInputTokens` is precisely what lets 1.5 detect a silently-inert cache.

> 📌 **Context for 1.5, not an obligation here:** on `claude-haiku-4-5` (the repo default) the minimum cacheable prefix is **4096 tokens**. Below it, `cache_control` silently does nothing — `cache_creation_input_tokens: 0`, no error.

### AC #6 — Base-URL path semantics (the `/v1` double-prefix trap)

**Given** the SDK appends its own version path segment (`v1/messages`) to the configured base URL, **when** the client is constructed, **then** `DefaultClaudeBaseURL` **must drop its `/v1` suffix** or every production call goes to `…/v1/v1/messages` and 404s.

```go
// BEFORE — correct for the hand-rolled client, which did fmt.Sprintf("%s/messages", baseURL)
const DefaultClaudeBaseURL = "https://api.anthropic.com/v1"

// AFTER — the SDK appends "v1/messages" itself
const DefaultClaudeBaseURL = "https://api.anthropic.com"
```

This is the single permitted exported-value change (AC #1). Add a comment saying *why*, or the next reader will "fix" it back.

**This trap is invisible to the current suite**: every test asserts `assert.Contains(t, r.URL.Path, "/messages")`, which passes for `/v1/messages`, `/v1/v1/messages`, and `/messages` alike, and `TestNewClaudeProvider` asserts the const against itself. AC #7 adds an **exact-path** assertion to lock it.

### AC #7 — New tests (4) covering what the existing suite structurally cannot see

Co-located in `apps/api/internal/ai/claude_test.go` (Rule 9), specific matchers (Rule 16):

1. **`TestClaudeProvider_SDKRetriesDisabled`** — [NAIL 2 proof] server returns 503 on every request; assert `hits == retryMaxAttempts` (3) exactly, and the error is `ErrAIProviderError`. Fails at 6+ if SDK retries are left on.
2. **`TestClaudeProvider_RequestPathIsV1Messages`** — assert `r.URL.Path == "/v1/messages"` exactly (not `Contains`). Locks AC #6.
3. **`TestClaudeProvider_MalformedJSONNotRetried`** — 200 + `not json at all`; assert `ErrAIInvalidResponse` **and** `hits == 1`. Locks the AC #4 trap. *(May instead be folded into `TestClaudeProvider_CompleteText_MalformedJSON` as an added hit-count assertion — either satisfies this AC.)*
4. **`TestClaudeProvider_CompleteTextWithUsage_CacheControlAndUsage`** — table/subtests over AC #5:
   - two system blocks, second marked `CacheTTL1h` → decoded wire body has `system` as an **array of 2 blocks in order**, `cache_control` on the **second only**, with the 1-hour TTL;
   - `CacheTTLNone` on all blocks → no `cache_control` anywhere;
   - a canned response carrying `cache_creation_input_tokens` / `cache_read_input_tokens` → surfaced verbatim on `CompletionResult.Usage`.

### AC #8 — Dependency pinned; module + build hygiene

**Given** the new dependency, **then** `apps/api/go.mod` requires **`github.com/anthropics/anthropic-sdk-go v1.59.0`** exactly (architecture § Core Architectural Decisions + **P10**), `go.sum` is updated, and `pnpm lint:all` is green (`go vet` → `staticcheck@2026.1` → `eslint` → `prettier --check`).

```bash
cd apps/api && go get github.com/anthropics/anthropic-sdk-go@v1.59.0 && go mod tidy
```

- No `vendor/` directory in this repo — `apps/api/Dockerfile`'s `go mod download` picks it up with no Dockerfile edit.
- `internal/ai` is a **declared leaf package** (`boundaries_test.go:64` — `{"ai","models","sse","retry","cache"}`), but that test only forbids `internal/…` imports. An **external** module dependency does not violate it. `TestLeafPackagesHaveNoInternalDeps` stays green with no edit.
- **Not the latest, and that is confirmed.** `go list -m -versions` on 2026-07-27 shows up to **v1.61.0**. The pin is a deliberate, dated architectural decision, and **Alexyu confirmed on 2026-07-27: ship v1.59.0.** The bump is filed as `backlog-anthropic-sdk-go-version-bump` (see Discovery Triage). Do not "helpfully" upgrade.

### AC #9 — Housekeeping deliberately NOT done (scope fence)

**Given** the temptation to tidy, **then** this story ships **none** of the following, and a reviewer seeing them should push back:

- ❌ No new Rule 7 error codes — the `AI_*` sentinels in `types.go` are reused verbatim. **No `code-review/instructions.xml` sync**, no prefix-count change (stays 16).
- ❌ No migrations, no schema, no SSE stage values, no `subtitle_status` change — those are sub-1-2 / sub-1-3.
- ❌ No prompt text and no prompt-version constant (P11) — sub-1-5.
- ❌ No `internal/subtitle/**` file touched, no feature flag, no `batch.go:244` edit — sub-1-6.
- ❌ No `Budget` cache-token dimension, no FR14 cost-estimate aggregation — P2.
- ❌ No frontend, no `.pen`, no visual baselines.
- ❌ No model-ladder change (`defaultLLMPricing` untouched) — sub-1-5 owns Haiku → Sonnet → Opus escalation.

---

## Tasks / Subtasks

- [x] **Task 1 — Add and pin the SDK; confirm the symbol set (AC #8, P10)**
  - [x] 1.1 `go get github.com/anthropics/anthropic-sdk-go@v1.59.0` from `apps/api/`; `go mod tidy`; confirm go.mod line reads exactly `v1.59.0`.
  - [x] 1.2 **P10 gate — verify BEFORE writing.** These are used by this story but are **not** in the architecture's confirmed-bindings list, so per P10 they must be checked against the SDK source/godoc first: `option.WithMaxRetries`, `option.WithHTTPClient`, `option.WithBaseURL`, the `anthropic.NewClient` return type (value vs pointer), `anthropic.Model` accepting a plain string, and `msg.Usage.CacheCreationInputTokens` / `CacheReadInputTokens` field names. Record what you verified in the Debug Log. **Do not infer Go symbols from cURL or Python shapes.**
  - [x] 1.3 Confirm `apps/api/Dockerfile` needs no change (`go mod download` path).

- [x] **Task 2 — Re-implement `claude.go` internals (AC #1, #3, #4, #6)**
  - [x] 2.1 Add an SDK client field to `ClaudeProvider`; keep **every** existing field (`TestNewClaudeProvider` asserts them). Build the client at the **end** of `NewClaudeProvider`, after all options are applied, so `WithClaudeBaseURL`/`WithClaudeTimeout`/`WithClaudeHTTPClient` are honoured.
  - [x] 2.2 Change `DefaultClaudeBaseURL` to drop `/v1`, with a why-comment (AC #6).
  - [x] 2.3 Replace `doRequest` with a `send(ctx, params) (*anthropic.Message, error)` that preserves the exact `governed → retryTransient → Messages.New` nesting and disables SDK retries (AC #3).
  - [x] 2.4 Write the error classifier: `*anthropic.Error` → `StatusCode` switch; timeout via `isTimeoutErr`; decode failure → `ErrAIInvalidResponse`, **not retryable** (AC #4). Re-create the 404 diagnostic verbatim.
  - [x] 2.5 Move `RecordLLM` onto `msg.Usage`; add `textFromMessage`; delete `claudeRequest`/`claudeMessage`/`claudeResponse`/`claudeContentBlock`/`claudeUsage` and `GetText()`.
  - [x] 2.6 Rewrite `Parse` and `CompleteText` over `anthropic.MessageNewParams`. `CompleteText` with an empty `systemPrompt` must emit **no** `system` field (existing `omitempty` semantics — asserted by the rewritten `SystemFieldSerialization` test).
  - [x] 2.7 Run `go build ./...` + `go vet ./...` and confirm **zero** edits landed in the AC #1 "must not change" file list (`git diff --name-only`).

- [x] **Task 3 — Additive caching + usage API (AC #5)**
  - [x] 3.1 Add `CachingCompleter`, `CacheTTL`, `SystemBlock`, `CompletionRequest`, `CompletionUsage`, `CompletionResult` to `provider.go` (Rule 11 — interfaces live with the package that owns them). Stamp `[@contract-v1]` in the doc comment naming sub-1-5 as the consumer.
  - [x] 3.2 Implement `CompleteTextWithUsage` on `*ClaudeProvider`, routed through the **same** `send` helper — no second request path.
  - [x] 3.3 Map `SystemBlock` → `[]anthropic.TextBlockParam`, preserving order; set `CacheControl` only on blocks with a non-empty `CacheTTL` (1h via the TTL param variant, 5m via the default ephemeral param).
  - [x] 3.4 Add `var _ CachingCompleter = (*ClaudeProvider)(nil)`. Confirm `gemini.go` compiles untouched (it must **not** implement the new interface).

- [x] **Task 4 — Tests (AC #2, #7)**
  - [x] 4.1 Rewrite `TestClaudeProvider_CompleteText_SystemFieldSerialization` against the captured wire body — same name, both cases (present / absent).
  - [x] 4.2 Rewrite `TestClaudeResponse_GetText` against `textFromMessage` — same name, all four table cases.
  - [x] 4.3 Add the 4 new tests from AC #7.
  - [x] 4.4 `go test ./internal/ai/... -count=1` green; `go test -list` shows **26 + 4 = 30** Claude-touching test functions (26 preserved, 4 added; see the [@contract-v1] caveat if #3 was folded in → 29).
  - [x] 4.5 Full backend suite `go test ./...` green (scanning/AIService regression check — NAIL 1's real blast radius).
  - [x] 4.6 `pnpm lint:all` green from the repo root.

---

## Dev Notes

### Current implementation map (read these before writing)

| File | Lines | What matters here |
|---|---|---|
| `apps/api/internal/ai/claude.go` | 345 | The whole target. `doRequest` at :110-177 is the retry/governor/metering core. |
| `apps/api/internal/ai/claude_test.go` | 545 | 24 tests, all on `httptest.NewServer`. Keep the servers. |
| `apps/api/internal/ai/retry_test.go` | 113 | `TestMain` shrinks `retryBaseDelay`→5ms / `retryMaxDelay`→20ms for the whole package, so retry tests are fast. **2 of the 26 guard tests live here.** |
| `apps/api/internal/ai/governor.go` | 80 | `governed[T]` — budget pre-check → rate token → concurrency slot. Generic, so `T = *anthropic.Message` works unchanged. |
| `apps/api/internal/ai/retry.go` | 75 | `retryTransient[T]`, `retryMaxAttempts = 3`, `isTransientStatus(code)` (429 or ≥500). Reuse `isTransientStatus` in the new classifier. |
| `apps/api/internal/ai/budget.go` | 159 | `RecordLLM(model string, in, out int64)` — SDK usage fields are already `int64`. No conversion. |
| `apps/api/internal/ai/types.go` | 105 | The `AI_*` sentinels. **Reused, not extended.** |
| `apps/api/internal/ai/provider.go` | 79 | `Provider` / `TextCompleter` — the moat. New interface goes **beside** them. |
| `apps/api/internal/ai/factory.go` | 89 | `NewProvider` claude branch. Untouched. |
| `apps/api/cmd/api/main.go` | 490, 532-538 | Shared `aiGovernor` → `WithClaudeGovernor`. Untouched. |

### SDK bindings — confirmed by the architecture record (P10 green-list, safe to use directly)

```go
client := anthropic.NewClient(option.WithAPIKey(key), option.WithBaseURL(base))

msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
    Model:     anthropic.Model(p.model),          // anthropic.Model is a string alias
    MaxTokens: int64(maxTokens),
    System: []anthropic.TextBlockParam{{
        Text:         stableInstructions,
        CacheControl: anthropic.NewCacheControlEphemeralParam(),                                  // 5m
        // 1h: anthropic.CacheControlEphemeralParam{TTL: anthropic.CacheControlEphemeralTTLTTL1h}
    }},
    Messages: []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
    },
})

// usage
msg.Usage.InputTokens, msg.Usage.OutputTokens
msg.Usage.CacheCreationInputTokens, msg.Usage.CacheReadInputTokens

// content
for _, block := range msg.Content {
    if t, ok := block.AsAny().(anthropic.TextBlock); ok { _ = t.Text }
}

// errors
var apierr *anthropic.Error
if errors.As(err, &apierr) { switch apierr.StatusCode { /* … */ } }

// options
option.WithRequestTimeout(d)   // confirmed
option.WithMiddleware(fn)      // confirmed
```

**P10 red-list — verify against the SDK repo before writing** (Task 1.2): `option.WithMaxRetries`, `option.WithHTTPClient`, `anthropic.NewClient`'s return type, and the exact `Usage` cache-field names.

### Headers the SDK sets for you

The suite asserts `x-api-key`, `anthropic-version: 2023-06-01`, and `Content-Type: application/json` on the request. The SDK sets all three automatically (`anthropic-version` is pinned to `2023-06-01`, matching the existing `ClaudeAPIVersion` const). **Keep the `ClaudeAPIVersion` const** — the tests reference it as the expected header value even though the client no longer sets it by hand.

### Pitfalls specific to this migration

1. **`/v1` double prefix** (AC #6) — the #1 way to ship a client that passes every test and 404s in production.
2. **Double retry** (AC #3) — silent 2× cost + 2× rate-limit pressure, invisible without a hit-count assertion.
3. **Malformed-JSON retryability flip** (AC #4) — the decode moved *inside* the retry loop.
4. **Deleting `claudeRequest`/`claudeResponse` breaks two tests** — they are rewritten, not dropped (AC #2).
5. **Client built before options applied** — `NewClaudeProvider` applies options in a loop; build the SDK client *after* it, or `WithClaudeBaseURL` is silently ignored and every test hits the real API.
6. **`p.httpClient`** already carries `Timeout: p.timeout` when the caller supplies no custom client. Pass it via `option.WithHTTPClient` and the timeout is enforced there; adding `option.WithRequestTimeout` on top is redundant — pick one and be deliberate, don't stack two competing deadlines.

### Rule compliance for this story

- **Rule 1** — all code under `apps/api/`. ✅
- **Rule 2** — `slog` only; preserve the existing `slog.Warn`/`slog.Error` call sites verbatim (the 404 and timeout lines are asserted behaviour in spirit and hard-won in fact).
- **Rule 9** — tests co-located in `internal/ai/`.
- **Rule 11** — new interface in `provider.go`, beside `TextCompleter`. Never in `handlers`.
- **Rule 13** — every error propagated or logged-and-returned. No swallowed errors in the classifier.
- **Rule 14** — one SDK client per `ClaudeProvider`, built once in the constructor and reused (Rule 27 ①). Never per request.
- **Rule 15** — main.go wiring **unchanged**; verify with `git diff --name-only` (Task 2.7). No DB columns, no Swagger (no HTTP surface changes).
- **Rule 16** — `assert.ErrorIs` for sentinels, exact `assert.Equal` for hit counts and paths, never `assert.True(err != nil)`.
- **Rule 19** — `ai` stays a leaf; external deps are permitted (AC #8).
- **Rule 20** — `[@contract-v1]` stamped on **AC #5 only**. No upstream contract is consumed (claude.go's existing surface is unstamped/v0). Downstream consumer: **sub-1-5**. Producer-side grep obligation is nil at authoring time (sub-1-5 is not yet drafted).
- **Rule 27** — Five Pillars for the Claude integration: ① rate limit = the shared `Governor` (unchanged) · ② cache = prompt caching capability added here, policy in 1.5 · ③ degrade = the `AI_*` sentinels + `NewProviderWithFallback` (unchanged) · ④ error codes = existing `AI_` prefix, none added · ⑤ keys = `CLAUDE_API_KEY` via `config`, `maskSecret` in logs (unchanged).

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** This story changes zero files under `apps/web/src/components/**`; it is backend-only Go. Rule 23 does not apply. No visual baselines, no `-linux`/`-darwin` regeneration.

### References

- [Source: `_bmad-output/planning-artifacts/epics-subtitle-pipeline.md`#Story 1.1] — the four epic ACs and the three NAILs.
- [Source: `_bmad-output/planning-artifacts/subtitle-pipeline-architecture.md`#Foundation Evaluation (Step 3)] — Option B ruling, verified SDK facts 1–8, scope/regression table, the 5 migration hazards.
- [Source: `subtitle-pipeline-architecture.md`#D8 — Retry strategy] — NAIL 2's *why* (Governor wraps outside `retryTransient`).
- [Source: `subtitle-pipeline-architecture.md`#P10 — Go SDK symbol discipline] — verify-before-writing rule and the v1.59.0 pin.
- [Source: `subtitle-pipeline-architecture.md`#D4 — Chunking and cache key] — 4096-token prefix minimum (context for AC #5; policy is 1.5's).
- [Source: `subtitle-pipeline-architecture.md`#Decision Impact Analysis] — this is step 1 of the 7-step sequence.
- [Source: `_bmad-output/planning-artifacts/prd.md`] — FR10/FR11 (translate, preserve timing) are downstream consumers; FR14 (cost estimate, P2) is why usage must reach the caller.
- [Source: `project-context.md`#Rule 27] — Five Pillars; #Rule 20 — contract stamping; #Rule 24 — Discovery Triage; #Rule 15 — pre-commit self-verification.
- [Source: `apps/api/internal/ai/claude.go`:110-177] — `doRequest`, the exact structure being replaced.
- [Source: `apps/api/internal/ai/governor.go`:65-80] — `governed`, the budget pre-check that must stay outermost.

---

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (`claude-fable-5`) — BMAD dev agent (Amelia)

### Debug Log References

**Task 1.2 — P10 symbol verification.** All red-list symbols checked against the SDK source in the module cache (`$(go env GOMODCACHE)/github.com/anthropics/anthropic-sdk-go@v1.59.0`), never inferred:

| Symbol | Verified signature | Source |
|---|---|---|
| `option.WithMaxRetries` | `func WithMaxRetries(retries int) RequestOption` | `option/requestoption.go:219` |
| `option.WithHTTPClient` | `func WithHTTPClient(client HTTPClient) RequestOption`, `HTTPClient = interface{ Do(*http.Request) (*http.Response, error) }` — `*http.Client` satisfies it | `option/requestoption.go:169,178` |
| `option.WithBaseURL` | `func WithBaseURL(base string) RequestOption`; **auto-appends a trailing `/` when the path is non-empty** | `option/requestoption.go:148` |
| `anthropic.NewClient` | `func NewClient(opts ...option.RequestOption) (r Client)` — **returns a VALUE, not a pointer** | `client.go:207` |
| `anthropic.Model` | `type Model = string` — a true **alias**, so a plain string needs no conversion helper | `message.go:4282` |
| `Usage` cache fields | `CacheCreationInputTokens int64` / `CacheReadInputTokens int64` (plus `InputTokens`/`OutputTokens int64`) | `message.go:8443-8456` |
| `anthropic.Error` | `type Error = apierror.Error` with `StatusCode int` | `aliases.go:17`, `internal/apierror/apierror.go:19-27` |
| 1h TTL | `CacheControlEphemeralTTLTTL1h CacheControlEphemeralTTL = "1h"` (and `…TTL5m = "5m"`); `TextBlockParam.CacheControl` is `json:"cache_control,omitzero"` | `message.go:502,528-529,5636-5644` |
| **AC #6 confirmation** | SDK default base URL is `https://api.anthropic.com/` and the endpoint path is the relative `"v1/messages"` — the `/v1` double-prefix trap is real | `option/requestoption.go:601`, `message.go:71,97` |
| Decode path | `json.NewDecoder(...).Decode(...)` — standard `encoding/json`, so malformed bodies surface as `*json.SyntaxError`, making AC #4 detection option (1) viable (option (2) middleware fallback not needed) | `internal/requestconfig/requestconfig.go:560` |

**Task 1.3** — `apps/api/Dockerfile` unchanged; it runs `go mod download` and picks up the new module with no edit.

### Completion Notes List

**🔗 AC Drift:** NONE (checked: `grep -rn "CompleteText\|claudeRequest\|GetText\|DefaultClaudeBaseURL" _bmad-output/implementation-artifacts/*.md` — hits are all sub-1-* stories that CONSUME this contract downstream, none re-specify prior shipped behaviour. `claude.go`'s pre-existing exported surface carries no `[@contract-vN]` stamp, i.e. implicit v0 under the Rule 20 forward-only retrofit, and AC #1 preserves it byte-for-byte.)

**📎 Contract Stamps:** FOUND (1 stamped AC in this story — AC #5 `[@contract-v1]` on `CachingCompleter` + its request/result types, declared in `provider.go`. Consumer sub-1-5a is drafted but not yet implemented, so no downstream ack is owed yet. No upstream stamps consumed: this story sits at the head of the dependency chain.)

**🎭 A11y Pre-Flight:** N/A (100% backend — no `apps/web/` files touched)

**🎨 UX Verification:** SKIPPED — no UI changes in this story

**What was actually implemented.** `claude.go` internals fully replaced with `anthropic-sdk-go v1.59.0`. The exported surface is byte-identical apart from the one permitted change (`DefaultClaudeBaseURL`, AC #6). `doRequest` became `send()`, preserving the `governed → retryTransient → Messages.New` nesting exactly; `classifyErr()` maps every SDK failure onto the pre-existing `AI_*` sentinels so no SDK type leaks past the package boundary; the 9R-1 404 diagnostic is re-created verbatim. `CompleteText` is now a thin wrapper over the new `CompleteTextWithUsage`, so both entry points share one request path (AC #5.4) rather than duplicating it.

**Deviation from AC #1's file list, and why it is not a drift.** AC #1 lists `provider.go` under "must not change" with the parenthetical "(`Provider` / `TextCompleter` — **the moat**)", while Task 3.1 *mandates* adding the new types to that same file. Read together, the intent is unambiguous: the two existing interfaces must not change. They did not — `Provider` and `TextCompleter` are untouched, and the new `CachingCompleter` sits beside them per Rule 11. Every other file on the must-not-change list has zero diff (verified via `git diff --name-only`).

**🔍 Pre-existing fixture drift found and fixed (Rule 24 lane ① — absorbed).** The migration surfaced a latent defect in the test fakes, not in the story's own scope: **the httptest servers never sent `Content-Type: application/json`**. The real Messages API always sends it and the SDK refuses to decode a body claiming any other type, so 12 fakes in `internal/ai` plus the shared `newTestClaudeProvider` helper in `internal/services` were silently unrealistic — the hand-rolled client simply never checked. Fixed by setting the header in the fakes (behavioural assertions untouched). This is fixture *fidelity*, not test weakening: notably, `TestClaudeProvider_CompleteText_MalformedJSON` only tests the intended thing once the server actually claims JSON — before the fix it was exercising a content-type rejection, not a decode failure.

**Request-body assertions updated (explicitly in AC #2's contract: "only client construction and the request-body assertions change").** Three assertions moved from the hand-rolled wire shape to the Messages API's canonical shape: `system` is now an ordered **array of content blocks** (this is precisely what makes `cache_control` expressible — the old plain-string field structurally could not carry it), and `messages[].content` is likewise a block array. Behaviour asserted is unchanged; only the shape being asserted moved.

**Pre-existing fix:** none applied in-place. **Pre-existing failure filed:** `preexisting-flake-scanner-sse-scan-cancelled` — `TestScannerService_SSEBroadcast_ScanCancelled` fails ~2 of 3 *full-package* runs and passes 100% in isolation. Proven pre-existing by stashing this story's changes and re-running the full package three times: identical failure rate. Non-trivial (a timing/interference race), so filed per the dev-story rule rather than fixed in-flight.

**Test count:** `go test -list` reports **30** Claude-touching test functions = the 26 required by AC #2 (all present, none deleted) + the 4 new AC #7 guards. Full backend suite green apart from the filed flake; `pnpm lint:all` green (0 errors).

### Discovery Triage

- **Did this story discover any work outside its current scope?** — **YES**, one item, known at authoring time:
  - **③ backlog-with-carry-forward-link** → tracked entry **`backlog-anthropic-sdk-go-version-bump`** (filed in `sprint-status.yaml` on 2026-07-27, at discovery time; that entry names this story back). `anthropic-sdk-go` has shipped **v1.60.0** and **v1.61.0** since the architecture pinned **v1.59.0** (`go list -m -versions`, 2026-07-27). This story **ships the pinned v1.59.0** per architecture § Core Architectural Decisions + P10 — the pin is a dated decision, not an oversight, and re-pinning mid-story would invalidate the P10 confirmed-bindings list the ACs rely on. The bump (changelog review + re-run of the 26-test guard + the 4 new tests + P10 red-list re-verification) is evaluated after M1 lands. Non-blocking: v1.59.0 supplies every binding M1 needs.
- Reference: `project-context.md` Rule 24; origin retro-19-P1.

### File List

| File | Change |
|---|---|
| `apps/api/go.mod` | +`github.com/anthropics/anthropic-sdk-go v1.59.0` (direct) + transitive deps |
| `apps/api/go.sum` | updated |
| `apps/api/internal/ai/claude.go` | **rewritten internals** — SDK client, `send()`, `classifyErr()`, `isTimeoutErr()`, `isDecodeErr()`, `textFromMessage()`, `CompleteTextWithUsage()`; hand-rolled wire types deleted |
| `apps/api/internal/ai/provider.go` | **added** `CachingCompleter` `[@contract-v1]`, `CacheTTL`, `SystemBlock`, `CompletionRequest`, `CompletionUsage`, `CompletionResult`. `Provider`/`TextCompleter` untouched |
| `apps/api/internal/ai/claude_test.go` | test-only wire fixtures; 2 tests rewritten in place (AC #2); request-body assertions updated; 4 new tests (AC #7); `Content-Type` added to 10 fakes |
| `apps/api/internal/ai/retry_test.go` | `Content-Type` added to 2 fakes (behaviour untouched) |
| `apps/api/internal/services/terminology_service_test.go` | `Content-Type` set centrally in `newTestClaudeProvider`; system-prompt assertion updated to the block-array shape |
| `_bmad-output/implementation-artifacts/sub-1-1-claude-sdk-migration.md` | this file — task checkboxes, Dev Agent Record, File List, Change Log, Status |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | status → review; filed `preexisting-flake-scanner-sse-scan-cancelled` |

### Senior Developer Review (AI) — 2026-07-28, adversarial CR

**Verdict: APPROVED after fixes.** All 9 ACs verified as implemented (mechanically, not by sampling): 30 Claude tests green pre-review, full backend suite green, `go vet` + `pnpm lint:all` green, AC #1 must-not-change list zero-diff (`provider.go` purely additive), D8 nesting / 404 diagnostic / `/v1` guard / malformed-JSON-not-retried all locked by tests. Rule 7: PASS (0 new codes). Rule 20: N/A (new stamp, no bump). Rule 25: N/A. Git-vs-story discrepancies: 0.

**Findings (2 Medium, 3 Low) — all FIXED in-review:**

| # | Sev | Finding | Fix |
|---|---|---|---|
| M1 | MEDIUM | `WithClaudeTimeout` silently ignored when combined with `WithClaudeHTTPClient` — old `doRequest` wrapped **every** attempt in `context.WithTimeout(ctx, p.timeout)` regardless of client; the SDK build only carried the timeout on the default-built client. Proven by probe: 50 ms timeout + custom client + 300 ms server → request ran to completion. AC #1 semantic drift (latent — production passes no custom client). | `option.WithRequestTimeout(p.timeout)` added on the custom-client branch of `NewClaudeProvider`; locked by new test `TestClaudeProvider_TimeoutEnforcedWithCustomHTTPClient`. |
| M2 | MEDIUM | 2xx + non-JSON `Content-Type` (broken-proxy HTML page with status 200) was **retried 3×** — the SDK's content-type rejection error is neither `*anthropic.Error` nor `*json.SyntaxError`, so it fell into the connection-error retryable fallback. Same permanent class as the AC #4 malformed-JSON trap, different shape. (Probe also confirmed HTML-body **5xx** classifies correctly — retried as provider error, no bug there.) | `rejectNonJSONSuccess` middleware (AC #4's documented option (2)) + `errNonJSONResponse` sentinel → `ErrAIInvalidResponse`, non-retryable; locked by new test `TestClaudeProvider_NonJSONSuccessNotRetried`. |
| L1 | LOW | Unknown `CacheTTL` values silently emitted no `cache_control` — the exact "silently-inert cache" failure mode AC #5 exists to surface. | Explicit `CacheTTLNone` case + `default:` branch with `slog.Warn`. |
| L2 | LOW | Error-log fidelity regression: old client logged the raw response body; new logged only `apiErr.Error()` (SDK-formatted). | `"raw_json", apiErr.RawJSON()` added to the `slog.Warn` (symbol P10-verified: `apierror.go:39`). |
| L3 | LOW | `CompleteTextWithUsage` (the sub-1-5a entry point) emitted no request-level debug log — only the `CompleteText` wrapper did. | Debug log moved into `CompleteTextWithUsage` (the shared funnel), now with `system_blocks` + `max_tokens`. |

**Review-verified P10 additions:** `option.WithMiddleware` / `Middleware` / `MiddlewareNext` (`option/requestoption.go:198-207`), `apierror.Error.RawJSON()` (`internal/apierror/apierror.go:39`) — checked against the module-cache source before use.

**Test count after review: 32** Claude-touching test functions (30 from dev + 2 review guards). Gates re-run post-fix: `go test ./internal/ai/... ./internal/services/` green · `go vet` clean · touched files `gofmt` clean · `pnpm lint:all` 0 errors.

### Change Log

| Date | Change |
|---|---|
| 2026-07-28 | **Task 1** — pinned `anthropic-sdk-go v1.59.0`; P10 red-list verified against SDK source (see Debug Log); Dockerfile confirmed unchanged. |
| 2026-07-28 | **Task 2** — `claude.go` internals re-implemented on the SDK: SDK client built after options (Rule 14), `DefaultClaudeBaseURL` drops `/v1` (AC #6), `send()` preserves the D8 nesting with `WithMaxRetries(0)`, `classifyErr()` re-maps every failure to the existing `AI_*` sentinels incl. the verbatim 9R-1 404 diagnostic, metering moved onto typed `msg.Usage`. |
| 2026-07-28 | **Task 3** — additive `CachingCompleter` `[@contract-v1]` + supporting types in `provider.go`; `CompleteTextWithUsage` implemented over the same `send()` path; `gemini.go` untouched and deliberately not implementing it. |
| 2026-07-28 | **Task 4** — 2 tests rewritten in place, 4 AC #7 guards added (30 total); pre-existing `Content-Type` fixture drift fixed in 13 fakes across 2 packages; request-body assertions moved to the canonical block-array shape; full backend suite + `pnpm lint:all` green; pre-existing scanner flake filed. |
| 2026-07-28 | **Code review (adversarial CR)** — 2 Medium + 3 Low findings, all fixed: custom-client timeout enforcement restored (`WithRequestTimeout`), non-JSON 2xx made permanent via `rejectNonJSONSuccess` middleware, unknown-`CacheTTL` warn, `raw_json` error logging, debug log moved to the shared funnel; +2 lock-in tests (32 total). Story → done. |

---

## Open Questions for Alexyu (non-blocking — saved for after the story, per the workflow)

1. ~~**SDK pin.**~~ ✅ **RESOLVED 2026-07-27 (Alexyu): ship v1.59.0.** The bump to v1.60.0/v1.61.0 is triaged to `backlog-anthropic-sdk-go-version-bump` and evaluated after M1 lands.
2. **Model ladder + pricing table.** `budget.go:19-24` prices `claude-haiku-4-5` / `claude-sonnet-5` / `claude-opus-4-8` / `claude-sonnet-4-6`. Since the architecture's ladder was set, **Claude Opus 5** (`claude-opus-5`, $5/$25 — same tier price as Opus 4.8) has shipped. Out of scope here; flagging it for **sub-1-5**, which owns the Haiku → Sonnet → Opus escalation.
3. **`CachingCompleter` naming.** Happy with a second optional interface beside `TextCompleter` (type-asserted by consumers, Gemini degrades), or would you rather sub-1-5 consume `*ai.ClaudeProvider` concretely? The interface keeps the moat intact and is my recommendation.
