# Retrospective — Subtitle Pipeline M2.5（成本同意與批次控制）

**Date:** 2026-08-12 · **Facilitator:** Bob (SM) · **Participants:** Alexyu (Project Lead), Winston (Architect), Amelia (Dev), Murat (TEA), Sally (UX), John (PM)

---

## 1. Epic Summary

### M2.5 — epic-subtitle-pipeline-m2-5（3 stories，2026-08-09 → 2026-08-11）

Origin: the 2026-08-07 production incident（掃描一鍵自動派工整庫 1026 筆、~2/3 落付費 ASR、零金額提示）→ Alexyu 三件一體裁定（掃描只 metadata＋篩選畫面＋總預算上限）。Delivered as a design-first arc:

- **Design（PR #211/#216）**: 8 screens F14–F20 over THREE Sally MCP review rounds — round 2 closed the over-limit/mobile-toolbar/empty-state gaps, round 3 (§5-sexies ruling) replaced the dishonest「免費」extract label with verbatim small translation fees. D1 ruled mid-design: batches support episodes.
- **sub-4-1（讀側，PR #213）**: scan decoupling（`scan_callback.go` deleted outright + `cost_consent_test.go` repo guard）、probe-only route prediction（Rule 19 adapter）、exported pricing accessors、dual-source candidate enumeration、analysis job + SSE progress。CR (Fable 審 Opus) 1H/1M/2L — H1 cancel/restart generation-token race.
- **sub-4-2（寫側，PR #215）**: `budget_usd` WYSIWYG ceiling、mixed movie+episode explicit-id batches（reject-not-filter）、mode-dependent engine（pipeline mode rides D2 `ProcessItem` directly — route-honest; legacy keeps Route C + `WithMediaType`）、`WorkerPool.TryReserve/Release` dedup bridge。9R-16 AC #1 `[@contract-v2→v3]`。CR (Opus 審 Fable) 1H/3M/2L — H1 skip-counted-as-success。
- **sub-4-3（前端，PR #218/#219）**: consent flow replaces the dialog idle branch（F14 analyze → F15 list → F16/F19 confirm → existing F8）、F17 scan-toast entry + Rule 26-safe deep link、dual-family per-item SSE join、D1 尾款（`excludedSeriesCount` retired）。CR (Opus 審 Fable) 2H/5M/2L — H1 workspace un-consented resume、H2 permanently-frozen candidate snapshot。

**Metrics:** 3/3 stories done · 9 PRs merged（#211–#219）· 19 CR findings（4H/9M/6L）all handled in-session · ~110 new FE tests + 26 new BE tests（web suite 228/2547 → 233/2612）· zero production incidents · alternating-model adversarial CR on every story.

---

## 2. What Went Well

1. **Design-first with in-flight rulings（Winston/Sally 點名）.** D1（影集支援）與 §5-sexies（據實金額）都在 FE 動工前裁定完成 — sub-4-3 實作期間零設計返工。The「免費」label problem, had it survived into implementation, would have meant redoing the entire amount-rendering path; instead it cost one prompt round.
2. **Alternating-model adversarial CR caught product-level holes.** 19 findings across three CRs, all fixed in-session. The two sub-4-3 Highs（workspace 下次繼續 bypassing consent; candidate snapshot frozen forever）were violations of the consent model itself — the exact blind spots same-model self-review tends to share with the implementation.
3. **Product rulings became executable assertions.** The「免費」ban is a test（`queryByText(/免費/) → null`）、skip≠success is a sentinel、WYSIWYG budget is a wire assertion、三處金額同源 is one selector — consent honesty moved from copywriting into the regression net.
4. **Rule 24 at full speed, including same-story closure.** Six tracked discoveries filed at authoring/implementation time; `backlog-batch-pool-dedup-overlap` was filed at sub-4-2 authoring and RESOLVED by that story's own CR (M4), with its impact statement corrected en route.
5. **retro-m2-AI1 paid for itself immediately.** Warm Docker PR builds all epic（39s–3m23s vs the old 11–16 min cold builds that thrice hit the 30-min cap）— the strongest possible advert for actually executing retro action items.
6. **Seam-reach practice (retro-m2-AI3) applied manually and it worked.** sub-4-1/sub-4-2 story files carried explicit「seam 資料層觸及」sections; sub-4-2's pre-identified the `WithMediaType` wrong-table risk before implementation touched it.

## 3. Challenges

1. **Unexecuted action items charged compound interest.** retro-m2-AI2（gh-account guard）: the account flip broke merges twice more（#215、#216）. retro-m2-AI4（InstantSearchBar flake）: two more rerun cycles on #218. Neither cost was new information — both were priced in the previous retro and left unpaid.
2. **Drift-sweep blind spot: `tests/e2e/`.** sub-4-3's AC-drift grep covered `apps/web/src` but not the Playwright E2E tree — the old idle-contract suite（`batch-subtitle.spec.ts`）went red only in CI. Real regression, late discovery, one avoidable CI round trip.
3. **`infra-vr-pr-bootstrap-gap` verified twice more, with a new facet.** The PR-side visual check cannot pre-merge-bootstrap new `-linux` baselines (known), AND the bot-pushed bootstrap PR triggers no checks at all（GITHUB_TOKEN push）— both #218 and #219 needed `--admin` merges. The delete-and-bootstrap path for *changed* linux baselines worked as documented precedent.
4. **Raw `tsc -p` is unusable as a local gate**（149 pre-existing route-type errors on main）— noted, not actioned; the real gates (vite build, vitest, eslint) are green and CI-enforced.

## 4. Previous-Retro Follow-Through（M1+M2 combined retro, 2026-08-07）

| Item | Status | Evidence in M2.5 |
| --- | --- | --- |
| retro-m2-AI1 docker-pr-cache-auth | ✅ done (PR #210) | Every M2.5 PR Docker build warm (39s–3m23s) |
| retro-m2-AI2 ship-gh-account-guard | ❌ not executed | Recurred ×2 (#215, #216 merges) → **escalated MED→HIGH this retro** |
| retro-m2-AI3 seam-reach-check | ⏳ practiced, not institutionalized | Manual sections in sub-4-1/sub-4-2 worked; create-story template still unchanged |
| retro-m2-AI4 instantsearchbar-flake | ❌ not executed | Flaked ×2 on #218 → **escalated LOW→MED this retro** |

Process rules held: Rule 20 machinery ran a real producer-side bump（9R-16 AC #1 v2→v3, all ackers frozen）and a deferred-stamp fulfillment（sub-4-1 AC #7/#8 stamped by sub-4-3 per its own deferral clause）; Rule 24, checkbox audits, a11y/AC-drift/contract-stamp bindings appear in every story record.

## 5. Readiness Assessment

- **Testing & Quality:** ✅ All suites green（web 233 檔/2612、api、E2E ×4 shards、visual incl. bootstrapped `-linux` set）.
- **Deployment:** ✅ **NAS pipeline mode re-enabled**（Alexyu confirmed in-session）. The 2026-08-07 freeze reason is structurally removed: scanning is metadata-only by construction（repo-level guard test）, generation requires explicit consent.
- **Stakeholder acceptance:** ✅ N/A-single-user; Alexyu participated and approved（「我覺得都做不錯」）.
- **Technical health:** Sound. Carried debt all tracked: `backlog-consent-toast-count-episodes`、`backlog-budget-default-config-endpoint`、`preexisting-fail-parse-progress-darwin-baseline`、`backlog-selfhosted-asr-actual-cost`、`backlog-gemini-cost-metering`、`backlog-assrt-search-response-parsing`、`backlog-compute-aware-asr-default`、`infra-vr-pr-bootstrap-gap`.
- **Blockers:** None.

## 6. Action Items（all tracked in sprint-status.yaml — Agreement 4; Alexyu approved all four）

| ID | Priority | Owner lane | Summary |
| --- | --- | --- | --- |
| retro-m2-AI2-ship-gh-account-guard | **MED→HIGH**（escalated in place） | SM/QD | Unchanged scope; now three recurrences across two epics. Execute before the next ship. |
| retro-m25-AI1-dev-story-e2e-drift-scope | MED | SM | dev-story AC-drift/spec-update sweep must explicitly include `tests/e2e/**`（and `tests/visual` fixture referrers）— the sub-4-3 miss class. |
| retro-m25-AI2-bootstrap-admin-merge-path | MED | SM/QD | Ship skill: document the sanctioned visual-baseline path — delete-and-bootstrap for changed `-linux`, verify-content-then-`--admin`-merge for the bot-pushed bootstrap PR（checks never trigger on GITHUB_TOKEN pushes）; annotate `infra-vr-pr-bootstrap-gap` with the two new occurrences. |
| retro-m2-AI4-instantsearchbar-flake | **LOW→MED**（escalated in place） | TEA | Unchanged scope; now three CI occurrences. |

## 7. Next Steps

1. **Verify one consent round on the NAS**（Alexyu）: scan → F17 入口 → F15 報價 → 小額批次確認 → F8 執行 — the first real-money exercise of the shipped flow.
2. Execute retro-m2-AI2 before the next ship（HIGH）.
3. Next epic: **M3（產品化 + Tier-2 本地）** per spec §8 — still unplanned in sprint-status; prep inputs unchanged from the previous retro（compute-aware default、restart-free ASR key、Assrt manual-dialog story）plus the new consent-flow backlog pair（toast count、budget default endpoint）as candidate M3 stories.
4. **No significant-discovery epic-update alert**: M2.5 was itself the plan correction born from the 2026-08-07 incident; spec §8's M3 sketch remains valid.

## 8. Epic Closure

Per the retro-gates-done ruling（2026-08-05, standing for the subtitle-pipeline lane）: with this retrospective run, `epic-subtitle-pipeline-m2-5` transitions to **done** and `epic-subtitle-pipeline-m2-5-retrospective` is recorded **done**. The 2026-08-07 incident that created this epic is closed: 掃描零成本、生成必經同意，且同意的數字就是被強制的數字。
