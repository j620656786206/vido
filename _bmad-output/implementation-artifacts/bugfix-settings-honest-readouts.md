# Bugfix: settings honest readouts (連線頁的謊言 + 狀態頁沒時間軸)

Status: ready-for-review

> Rule-24 lane ① from `/impeccable critique settings` (2026-08-25, dual-agent,
> score **19/40**). Alexyu picked the 誠實度 lane first, scope = this item only,
> plus the J7-D width loose end folded in.

## Problem

Three defects, one theme: **the settings surface reports a state that is not the
real state.** PRODUCT.md's north star is「會諂媚的讀數比沒有讀數更糟」— these are
sycophantic readouts.

### 1. `/settings/connection` renders a reassuring lie

`QBittorrentForm.tsx:20` is `const { data: config, isLoading } = useQBittorrentConfig();`
— `isError` is never destructured or handled. When `GET /api/v1/settings/qbittorrent`
returns **500**, the component falls through to its normal render and paints a
**clean, empty form with placeholders**.

Confirmed at runtime by the critique's detector pass on the seeded local env:
HTTP 500 on every load, **no error word anywhere in the DOM**, both CTAs
`disabled=true`. The app's own log viewer, two clicks away, shows
`Failed to get qBittorrent config` and `Failed to decrypt qBittorrent password`.
`/settings/status` correctly shows qBittorrent 「已斷線」 — **the app contradicts
itself across two tabs.**

User consequence: someone whose config exists but whose password will not decrypt
is told, visually, that they never configured it. They retype credentials that
were already there; the decrypt failure stays hidden and the retype does not fix it.

### 2. The backend throws away the one fact that would help

`qbittorrent_service.go:60` logs `Failed to decrypt qBittorrent password` and
returns `fmt.Errorf("failed to decrypt password: %w", err)` — an opaque error.
`qbittorrent_handler.go:55` then flattens **every** GetConfig failure into
`InternalServerError(c, "Failed to retrieve qBittorrent configuration")` →
code `INTERNAL_ERROR`, suggestion `"Please try again later or contact support."`

So even a frontend that *did* handle `isError` could only say "something broke".
The actionable cause — ENCRYPTION_KEY changed or is missing — is knowable
server-side and discarded before it reaches the client. Fixing the frontend alone
would produce an honest-but-useless banner.

### 3. `服務狀態` has no time axis

`ServiceStatusCard.tsx:63` — `showDetail = status !== 'connected' && status !== 'unconfigured'`,
and `lastCheckAt` renders **only inside** the 顯示詳情 panel. So for `connected`
and `unconfigured` — the two states a returning user checks most — there is no
toggle at all and freshness is unreachable.

PRODUCT.md's operating context is「使用者會關掉視窗走開，之後回來查」. A green dot
that cannot say whether it was verified 2 seconds or 6 hours ago is asking to be
taken on faith, which is the exact thing the product exists to eliminate.

### 4 (folded in). J7-D width ruling is partly wrong on the record

Measured at 1440px: `connection.tsx:10` caps the **whole page** at `max-w-3xl`
(h1 measured **768px** wide); `ApiKeysForm.tsx:215` does the same; every other
settings route inherits the layout's `max-w-5xl` (**928px**). So the page
*heading* jumps 160px as you move between settings tabs, which is what makes a
deliberate rule read as a bug.

J7-D's substance (form cards ≤768px for field scannability) is right. Its
*application* is wrong: it narrowed the page header too.

## Decision

- **Backend surfaces the cause.** New sentinel + error code so the client can
  distinguish "decrypt failed" from every other GetConfig failure. Code follows
  the existing `QBITTORRENT_*` family in `apps/api/internal/qbittorrent/types.go`
  (Rule 7).
- **Frontend never renders an empty form as a load-failure state.** An empty form
  is an assertion that nothing is saved. On `isError` the form is replaced by a
  fault banner naming the real condition and pointing at 金鑰設定.
- **`ApiKeysForm.tsx` is the reference implementation** for the banner
  (`--error-tint` + `--error-text` + `role="status" aria-live="polite"`), per the
  critique's finding that it is the only settings component honoring the tokens.
  Not a new pattern — a second use of the existing one.
- **Freshness is unconditional.** `最後檢查` moves out of the detail panel and onto
  the card for every status, as a relative string (`剛剛` / `N 分鐘前` / `N 小時前`).
  Absolute time stays in the detail panel for the states that have one.
- **J7-D: cap the form, not the page.** The width cap moves off the page wrapper
  onto the form body in both form pages, so all nine settings pages anchor their
  h1 at the same 928px edge and only the fields are narrowed.

## Acceptance Criteria

1. **`GET /settings/qbittorrent` distinguishes decrypt failure.** A decrypt error
   returns error code `QBITTORRENT_CONFIG_DECRYPT_FAILED` with an actionable
   suggestion naming `ENCRYPTION_KEY`; every other failure keeps `INTERNAL_ERROR`.
   Detection is `errors.Is` against an exported sentinel, not string matching.
2. **The web service layer preserves `code` and `suggestion`.** `fetchApi` throws a
   typed error carrying them; existing `error.message` consumers keep working
   unchanged (the class extends `Error`).
3. **`QBittorrentForm` handles `isError`.** On load failure it renders
   `data-testid="qb-config-load-error"` — an `--error-tint` banner, `role="status"`,
   `aria-live="polite"` — that states the condition, states the cause when the code
   is `QBITTORRENT_CONFIG_DECRYPT_FAILED`, and links to `/settings/keys`.
   **The form fields are not rendered in this state.**
4. **`ServiceStatusCard` shows freshness for every status**, including `connected`
   and `unconfigured`, without expanding anything —
   `data-testid="last-check-{name}"`. Relative wording; `尚未檢查` when absent.
5. **J7-D applied as ruled.** `connection.tsx` and `ApiKeysForm.tsx` cap the form
   body, not the page header. A test asserts the page header is not inside the
   `max-w-3xl` wrapper, so the regression cannot return silently.
6. **Tests.** Go: decrypt-failure path returns the new code, non-decrypt failures
   do not. Web: the three component behaviors above, each failing before the fix.
7. **Rule 24.** Every critique finding NOT fixed here gets a `sprint-status.yaml`
   entry — no prose-only findings.

## Out of scope (filed, not fixed)

The other three P1s and the P2/P3 from the same critique: contrast tokens,
destructive-action confirmation, mobile tab overflow + 44px targets, the
12-component token retrofit, `border-l-4`. See sprint-status entries.
