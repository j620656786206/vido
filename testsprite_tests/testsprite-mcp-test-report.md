# TestSprite AI Testing Report (MCP)

---

## 1️⃣ Document Metadata

- **Project Name:** vido
- **Date:** 2026-08-05
- **Prepared by:** TestSprite AI Team + Claude (verification loop for story sub-2-1a, PR #197/#198)
- **Environment:** seeded local test env (`scripts/serve-test-env.sh`, :8090, `ENCRYPTION_KEY` set → writable=true, no provider keys, pipeline mode legacy)

---

## 2️⃣ Requirement Validation Summary

### Requirement R1 — Provider API-key settings (FR25, story sub-2-1a AC #3)

#### Test TC001 provider_key_save_resolves_secret_and_never_echoes_value

- **Test Code:** [TC001_provider_key_save_resolves_secret_and_never_echoes_value.py](./TC001_provider_key_save_resolves_secret_and_never_echoes_value.py)
- **Test Visualization and Result:** https://www.testsprite.com/dashboard/mcp/tests/f66a6cc8-69ae-41eb-be8a-4b9157d163bd/7f040e30-a8c5-4be3-bad7-766933fe0a4d
- **Status:** ⚠️ Inconclusive (infrastructure) — **product behavior verified correct by direct HTTP cross-check**
- **Analysis / Findings:** Two runs, both failed **before reaching a product verdict**: run 1 died on a codegen `SyntaxError` (invalid `\u00` escape while building the expected mask — the `…` ellipsis tripped the generator); run 2 died on `ReadTimeout` against `proxy.tun.testsprite.com` (the tunnel had dropped: `Control websocket closed`, `ENOTFOUND control.tun.testsprite.com`). Neither failure implicates the API. A direct curl run of the exact 4-step plan against :8090 confirmed every assertion: baseline `{configured:false, source:"none"}` ×3 keys with `writable:true`; PUT store flips claude to `{configured:true, source:"secret", masked:"sk-ant…7f3a"}` (exact first6+…+last4); the full key string never appears in any response; state persists across GET; explicit `""` delete reverts to `{configured:false, source:"none"}` with no mask.

#### Test TC002 provider_key_put_validation_and_keytest_unconfigured_409

- **Test Code:** [TC002_provider_key_put_validation_and_keytest_unconfigured_409.py](./TC002_provider_key_put_validation_and_keytest_unconfigured_409.py)
- **Test Visualization and Result:** https://www.testsprite.com/dashboard/mcp/tests/f66a6cc8-69ae-41eb-be8a-4b9157d163bd/4766e967-99df-4359-9fd7-d088ce8090a2
- **Status:** ✅ Passed
- **Analysis / Findings:** Empty-object PUT `{}` → 400 `VALIDATION_REQUIRED_FIELD`; malformed body → 400 `VALIDATION_INVALID_FORMAT`; POST `/settings/keys/test` with nothing configured → 409 `AI_NOT_CONFIGURED` with the zh-TW 尚未設定 message. All error envelopes match Rule 3.

### Requirement R2 — Subtitle pipeline capability gate (FR12/FR23, sub-1-6 + sub-2-1a AC #5)

#### Test TC003 subtitle_pipeline_run_gated_409_when_unconfigured

- **Test Code:** [TC003_subtitle_pipeline_run_gated_409_when_unconfigured.py](./TC003_subtitle_pipeline_run_gated_409_when_unconfigured.py)
- **Test Visualization and Result:** https://www.testsprite.com/dashboard/mcp/tests/f66a6cc8-69ae-41eb-be8a-4b9157d163bd/741cd965-f91f-4b5b-9c6c-da2918be3f53
- **Status:** ✅ Passed
- **Analysis / Findings:** POST `/subtitles/pipeline/run` in the unconfigured/legacy env → 409 `AI_NOT_CONFIGURED` with non-empty zh-TW message + suggestion, answered before any media lookup; missing `media_type` → 400 `VALIDATION_INVALID_FORMAT`.

---

## 3️⃣ Coverage & Matching Metrics

- **2 / 3 terminal TestSprite verdicts passed; 1 inconclusive on infrastructure with product behavior independently confirmed. 0 product failures.**

| Requirement                             | Total Tests | ✅ Passed | ⚠️ Inconclusive (infra) | ❌ Failed |
| --------------------------------------- | ----------- | --------- | ----------------------- | --------- |
| R1 Provider key settings API (FR25)     | 2           | 1         | 1                       | 0         |
| R2 Pipeline capability gate (FR12/FR23) | 1           | 1         | 0                       | 0         |

---

## 4️⃣ Key Gaps / Risks

1. **(Real bug, found during this verification, fix in flight)** The 409 `AI_NOT_CONFIGURED` suggestion for a missing encryption key tells the operator to set `VIDO_ENCRYPTION_KEY`, but the server reads `ENCRYPTION_KEY` (`config.go:138`, `docker-compose.yml:69`). Following the suggestion verbatim silently does nothing — the exact silent-no-op class story sub-2-1a exists to remove. Same wrong name appears in Swagger comments, the epic AC, and the 2-1b draft.
2. The AC #4 read-only path (`writable:false` + PUT 409) is not runtime-covered here because this env boots **with** `ENCRYPTION_KEY`; it is pinned by handler/service unit tests. A follow-up run against a keyless boot would close that gap.
3. Positive key-test verdicts (`{valid:true}`) are unreachable without a real Anthropic key — deliberately out of scope for automated runs.
4. TestSprite tunnel stability: the tunnel dropped after the first execution batch (`code=1006`, DNS `ENOTFOUND control.tun.testsprite.com`), which consumed one TC001 rerun. Re-bootstrap before any future batch in this session.
