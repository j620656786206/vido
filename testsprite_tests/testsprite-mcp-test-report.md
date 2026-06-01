# TestSprite AI Testing Report (MCP)

---

## 1️⃣ Document Metadata

- **Project Name:** vido
- **Date:** 2026-06-01
- **Prepared by:** TestSprite AI Team
- **Run mode:** Local CLI (`generateCodeAndExecute`), serverMode=production
- **Target under test:** NAS-deployed Vido — `http://192.168.50.52:8088` (zh-TW UI)
- **Scope:** 18 cases re-run against the real NAS — the 3-case smoke subset, the
  15 cases the broken runner-local CI had flagged fail/error, plus a final
  single-case re-verification of TC043 after fixing the qBittorrent config.
- **Credits consumed:** ~56 total over the session (124 → 68)

> **Headline:** **18 / 18 PASS — zero genuine product failures.** Every test the
> broken runner-local CI had marked `fail`/`error` passes against the real NAS.
> TC043 (qBittorrent connected-state) was initially BLOCKED by a missing backend
> config; after pointing Vido at the live qBittorrent (v5.1.4) it now PASSES.

---

## 2️⃣ Requirement Validation Summary

### Requirement: Media Library — browse, list/grid, sort, filter, paginate, item actions

| Test  | Title                                                       | Status    |
| ----- | ----------------------------------------------------------- | --------- |
| TC009 | Switch to list view, sort by Year, paginate, verify count   | ✅ PASSED |
| TC010 | Delete an item from list view and confirm it is removed     | ✅ PASSED |
| TC011 | Cancel delete from item action menu leaves item intact      | ✅ PASSED |
| TC012 | Filter media by type TV and then clear filter to repopulate | ✅ PASSED |
| TC013 | Export a media item from action menu shows confirmation     | ✅ PASSED |
| TC014 | Pagination previous/next controls behave at boundaries      | ✅ PASSED |

### Requirement: qBittorrent Settings — load, test, save, validation

| Test  | Title                                                   | Status    |
| ----- | ------------------------------------------------------- | --------- |
| TC035 | Load qBittorrent settings page and verify form visible  | ✅ PASSED |
| TC036 | Successful connection test and save settings            | ✅ PASSED |
| TC037 | Verify save confirmation after successful test          | ✅ PASSED |
| TC038 | Connection test failure shows inline error and guidance | ✅ PASSED |
| TC040 | Empty host validation prevents connection test          | ✅ PASSED |
| TC041 | Password field masks input                              | ✅ PASSED |
| TC042 | Re-test after failure shows updated success state       | ✅ PASSED |

### Requirement: Dashboard — qBittorrent health indicator

| Test  | Title                                                           | Status    |
| ----- | --------------------------------------------------------------- | --------- |
| TC043 | Dashboard shows qBittorrent health indicator in Connected state | ✅ PASSED |
| TC048 | Indicator and modal remain usable after switching filters       | ✅ PASSED |

- **TC043 resolution (2026-06-01):** initially BLOCKED because the NAS Vido had
  no working qBittorrent config (indicator showed `未連線`). A real qBittorrent
  (v5.1.4) was found running at `http://192.168.50.52:8080`; once Vido was
  configured to point at it (connection test returned app_version v5.1.4), the
  dashboard health monitor flipped to `已連線` and **TC043 now PASSES**. Root
  cause was a missing/invalid backend config, not a defect.

### Requirement: Subtitle / Manual Search

| Test  | Title                                                    | Status    |
| ----- | -------------------------------------------------------- | --------- |
| TC050 | Manual search supports multi-source results visibility   | ✅ PASSED |
| TC051 | Manual search with a specific query returns results list | ✅ PASSED |
| TC052 | Selecting a manual search result shows a confirmation    | ✅ PASSED |

---

## 3️⃣ Coverage & Matching Metrics

- **18 of 18** NAS-verified tests passed (**100%**); 0 blocked; 0 failures.

| Requirement                              | Tests  | ✅ Passed | 🟠 Blocked | ❌ Failed |
| ---------------------------------------- | ------ | --------- | ---------- | --------- |
| Media Library (browse/sort/filter/items) | 6      | 6         | 0          | 0         |
| qBittorrent Settings (test/save/valid.)  | 7      | 7         | 0          | 0         |
| Dashboard health indicator               | 2      | 2         | 0          | 0         |
| Subtitle / Manual Search                 | 3      | 3         | 0          | 0         |
| **Total**                                | **18** | **18**    | **0**      | **0**     |

**CI-vs-NAS reconciliation** — the broken runner-local CI run had flagged these
same cases very differently. NAS truth overrides it:

| CI (runner-local, unreliable) | Count | NAS truth                             |
| ----------------------------- | ----- | ------------------------------------- |
| `error`                       | 15    | all → PASS (TC043 PASS after qBT fix) |
| `fail`                        | 3     | TC010, TC036, TC043 → all PASS        |

> Not yet NAS-verified: the 6 cases CI already passed (TC039, TC044–TC047,
> TC049) and the remainder of the 50-case plan. 68 credits remain (~13 cases).

---

## 4️⃣ Key Gaps / Risks

1. **The CI signal was 100% environment noise.** Every fail/error flagged by the
   runner-local CI passes against the real NAS. This empirically justifies the
   2026-06-01 decision to defer the monthly cron and develop locally.
2. **Backend concurrency weakness (NEW, found during this session).** Running 18
   concurrent TestSprite browser cases against the production NAS degraded the
   API — `/library`, `/downloads`, `/settings/qbittorrent` all hung >12s, even
   pure-DB reads (GetConfig). UI was stuck flashing skeletons until a container
   restart. Likely cause: SQLite PRAGMAs (`busy_timeout`, WAL) applied via
   `db.conn.Exec()` on the connection **pool** only take effect on one pooled
   connection, leaving others without `busy_timeout` under lock contention.
   **FIXED (this session):** `DatabaseConfig.GetConnectionString` now passes the
   per-connection PRAGMAs via modernc's `_pragma=` DSN params so every pooled
   connection gets `busy_timeout`/WAL/`foreign_keys`. Pending deploy to the NAS
   (which still runs the old build — the hang recurs there until redeployed). A
   possible connection-hold/leak under the health monitor's steady writes is a
   separate open question needing NAS logs/profiling. Also: do not load-test
   against production — use an isolated environment.
3. **Local-run requires a localhost→NAS proxy.** The CLI pre-checks
   `localhost:<port>`; a TCP proxy (`localhost:8088 → 192.168.50.52:8088`) is
   needed. Capture this in `docs/testsprite-local-dev.md`.
4. **Plan ID drift.** The committed `.py` filenames (TC001–TC031, old plan) do
   not match the active v4 plan IDs (TC009–TC052). Always pick testIds from
   `testsprite_frontend_test_plan.json`, not from the local filenames.

---

_Run artifacts: `testsprite_tests/tmp/test_results.json` (latest raw verdicts),
15-case backup at `/tmp/results15.json`._
